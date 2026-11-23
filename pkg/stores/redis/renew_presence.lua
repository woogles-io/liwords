-- Arguments to this Lua script:
-- uuid, username, auth, connID, ts, channel... (ARGV[1] through [5], then
-- the connection's channels, if known)
--
-- Renews the connection's presence. Given its channels, it also puts back
-- any entry of the connection that is missing (Redis lost its data, or the
-- sweep took an entry the connection was late to renew); without them, it
-- can only extend the entries there are.
-- Returns the user's channels before and after, and the channels it put the
-- connection back in.

local activeusergameskey = "activeusergames:"..ARGV[1]
local userpresencekey = "userpresence:"..ARGV[1]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4] -- uuid#username#auth#connID
local connID = ARGV[4]
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

-- The channels to renew: those given, or else those the connection has
-- entries for, live or just expired (the connection is still here).
local renew = {}
if #ARGV > 5 then
  for idx = 6, #ARGV do
    renew[ARGV[idx]] = true
  end
else
  for _, simpleuserkey in ipairs(redis.call("ZRANGE", userpresencekey, 0, -1)) do
    local conn_id, chan = string.match(simpleuserkey, "^([^#]+)#([^#]+)$")
    if conn_id == connID and chan then
      renew[chan] = true
    end
  end
end

local setpurged = {[activeusergameskey]=true}
local restored = {}
for chan in pairs(renew) do
  local channelpresencekey = "channelpresence:"..chan
  -- ZADD returns 1 for a member it adds rather than updates.
  if redis.call("ZADD", channelpresencekey, ts + expiry, userkey) == 1 then
    restored[chan] = true
  end
  redis.call("EXPIRE", channelpresencekey, expiry)
  setpurged[channelpresencekey] = true
  redis.call("ZADD", userpresencekey, ts + expiry, connID.."#"..chan)
  redis.call("EXPIRE", userpresencekey, expiry)
  setpurged[userpresencekey] = true
  redis.call("ZADD", "userpresences", ts + expiry, userkey.."#"..chan)
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

return { sorted(setbefore), sorted(setafter), sorted(restored) }
