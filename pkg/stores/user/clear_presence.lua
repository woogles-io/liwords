-- Arguments to this Lua script:
-- uuid, username, authOrAnon, connID, timestamp

local userpresencekey = "userpresence:"..ARGV[1]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4]  -- uuid#username#anon#conn_id
local ts = tonumber(ARGV[5])

-- delete where conn_id matches.
local shouldtouch = false
local setbefore = {}
for _, simpleuserkey in ipairs(redis.call("ZRANGE", userpresencekey, 0, -1)) do
  -- simpleuserkey looks like conn_id#channel
  local conn_id, chan = string.match(simpleuserkey, "^([%a%d]+)#([%a%.%d]+)$")
  if conn_id and chan then
    setbefore[chan] = true
    if conn_id == ARGV[4] then
      redis.call("ZREM", "channelpresence:"..chan, userkey)
      redis.call("ZREM", userpresencekey, simpleuserkey)
      redis.call("ZREM", "userpresences", userkey.."#"..chan)
      shouldtouch = true
    end
  end
end

-- update the last known presence time.
if shouldtouch and ARGV[3] == "auth" then
  redis.call("ZADD", "lastpresences", ts, ARGV[1])
end

-- inefficient, but easier to write.
-- an efficient algorithm sets setafter for kept keys during earlier iteration.
local setafter = {}
for _, simpleuserkey in ipairs(redis.call("ZRANGE", userpresencekey, 0, -1)) do
  -- simpleuserkey looks like conn_id#channel
  local conn_id, chan = string.match(simpleuserkey, "^([%a%d]+)#([%a%.%d]+)$")
  if conn_id and chan then
    setafter[chan] = true
  end
end

-- inefficient, but easier to write.
-- an efficient algorithm is linear over arraybefore and arrayafter.
local setremoved = {}
for chan in pairs(setbefore) do
  if not setafter[chan] then
    setremoved[chan] = true
  end
end

-- make sorted sets.
local arraybefore = {}
for chan in pairs(setbefore) do
  table.insert(arraybefore, chan)
end
table.sort(arraybefore)
local arrayafter = {}
for chan in pairs(setafter) do
  table.insert(arrayafter, chan)
end
table.sort(arrayafter)
local arrayremoved = {}
for chan in pairs(setremoved) do
  table.insert(arrayremoved, chan)
end
table.sort(arrayremoved)

-- return channels before, channels after, and channels removed.
return { arraybefore, arrayafter, arrayremoved }
