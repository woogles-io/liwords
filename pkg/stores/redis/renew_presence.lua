-- Arguments to this Lua script:
-- uuid, username, auth, connID, ts (ARGV[1] through [5])

local activeusergameskey = "activeusergames:"..ARGV[1]
local userpresencekey = "userpresence:"..ARGV[1]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4] -- uuid#username#auth#connID
local ts = tonumber(ARGV[5])
-- Each member's score is when it expires; it's live while that's after now.
local live = "("..ts

-- 3 minutes. We will renew these keys constantly.
local expiry = 180

-- The user's channels, from their live entries and active games.
local function channels()
  local set = {}
  for _, simpleuserkey in ipairs(redis.call("ZRANGEBYSCORE", userpresencekey, live, "+inf")) do
    -- simpleuserkey looks like conn_id#channel
    local conn_id, chan = string.match(simpleuserkey, "^([^#]+)#([^#]+)$")
    if conn_id and chan then
      set[chan] = true
    end
  end
  for _, gameuuid in ipairs(redis.call("ZRANGEBYSCORE", activeusergameskey, live, "+inf")) do
    set["activegame:"..gameuuid] = true
  end
  return set
end

local setbefore = channels()

-- renew where conn_id matches, including an entry that has just expired:
-- the connection is still here.
local setpurged = {[activeusergameskey]=true}
for _, simpleuserkey in ipairs(redis.call("ZRANGE", userpresencekey, 0, -1)) do
  local conn_id, chan = string.match(simpleuserkey, "^([^#]+)#([^#]+)$")
  if conn_id == ARGV[4] and chan then
    local channelpresencekey = "channelpresence:"..chan
    redis.call("ZADD", channelpresencekey, ts + expiry, userkey)
    redis.call("EXPIRE", channelpresencekey, expiry)
    setpurged[channelpresencekey] = true
    redis.call("ZADD", userpresencekey, ts + expiry, simpleuserkey)
    redis.call("EXPIRE", userpresencekey, expiry)
    setpurged[userpresencekey] = true
    redis.call("ZADD", "userpresences", ts + expiry, userkey.."#"..chan)
  end
end

-- remove all subkeys inside the zsets that have expired. The global
-- userpresences is left to the sweep, which tells others about the
-- connections that went away.
for k in pairs(setpurged) do
  redis.call("ZREMRANGEBYSCORE", k, "-inf", ts)
end

local setafter = channels()

-- make sorted sets.
local function sorted(set)
  local arr = {}
  for chan in pairs(set) do
    table.insert(arr, chan)
  end
  table.sort(arr)
  return arr
end

-- return channels before and channels after.
return { sorted(setbefore), sorted(setafter) }
