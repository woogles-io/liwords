-- Arguments to this Lua script:
-- uuid, username, authOrAnon, connID, timestamp

local activeusergameskey = "activeusergames:"..ARGV[1]
local userpresencekey = "userpresence:"..ARGV[1]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4]  -- uuid#username#anon#conn_id
local ts = tonumber(ARGV[5])
-- Each member's score is when it expires; it's live while that's after now.
local live = "("..ts

-- The user's channels, from their live entries.
local function channels()
  local set = {}
  for _, simpleuserkey in ipairs(redis.call("ZRANGEBYSCORE", userpresencekey, live, "+inf")) do
    -- simpleuserkey looks like conn_id#channel
    local conn_id, chan = string.match(simpleuserkey, "^([^#]+)#([^#]+)$")
    if conn_id and chan then
      set[chan] = true
    end
  end
  return set
end

local setbefore = channels()

-- delete where conn_id matches, expired or not.
for _, simpleuserkey in ipairs(redis.call("ZRANGE", userpresencekey, 0, -1)) do
  local conn_id, chan = string.match(simpleuserkey, "^([^#]+)#([^#]+)$")
  if conn_id == ARGV[4] and chan then
    redis.call("ZREM", "channelpresence:"..chan, userkey)
    redis.call("ZREM", userpresencekey, simpleuserkey)
    redis.call("ZREM", "userpresences", userkey.."#"..chan)
  end
end

local setafter = channels()

local setremoved = {}
for chan in pairs(setbefore) do
  if not setafter[chan] then
    setremoved[chan] = true
  end
end

-- add active games to both sets.
for _, gameuuid in ipairs(redis.call("ZRANGEBYSCORE", activeusergameskey, live, "+inf")) do
  local activegamepseudochan = "activegame:"..gameuuid
  setbefore[activegamepseudochan] = true
  setafter[activegamepseudochan] = true
end

-- make sorted sets.
local function sorted(set)
  local arr = {}
  for chan in pairs(set) do
    table.insert(arr, chan)
  end
  table.sort(arr)
  return arr
end

-- return channels before, channels after, and channels removed.
return { sorted(setbefore), sorted(setafter), sorted(setremoved) }
