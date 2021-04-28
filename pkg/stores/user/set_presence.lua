-- Arguments to this Lua script:
-- uuid, username, authOrAnon, connID, channel string, timestamp  (ARGV[1] through [6])

local userpresencekey = "userpresence:"..ARGV[1]
local channelpresencekey = "channelpresence:"..ARGV[5]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4] -- uuid#username#auth#connID
local ts = tonumber(ARGV[6])

-- 3 minutes. We will renew these keys constantly.
local expiry = 180

-- get current set of channels.
local setbefore = {}
for _, simpleuserkey in ipairs(redis.call("ZRANGE", userpresencekey, 0, -1)) do
  -- simpleuserkey looks like conn_id#channel
  local conn_id, chan = string.match(simpleuserkey, "^([%a%d]+)#([%a%.%d]+)$")
  if conn_id and chan then
    setbefore[chan] = true
  end
end

-- set presence.
local chan = ARGV[5]
local simpleuserkey = ARGV[4].."#"..chan -- just conn_id#channel
redis.call("ZADD", channelpresencekey, ts + expiry, userkey)
redis.call("EXPIRE", channelpresencekey, expiry)
redis.call("ZADD", userpresencekey, ts + expiry, simpleuserkey)
redis.call("EXPIRE", userpresencekey, expiry)
redis.call("ZADD", "userpresences", ts + expiry, userkey.."#"..chan)

-- Do not modify lastpresences here. We only modify that when people leave, or on
-- the renew presence here. We'd like there to be a little time (a few seconds)
-- when the user logs in to determine how long it's been since they last logged in.

-- inefficient, but easier to write.
-- an efficient algorithm just adds the new channel to setbefore.
local setafter = {}
for _, simpleuserkey in ipairs(redis.call("ZRANGE", userpresencekey, 0, -1)) do
  -- simpleuserkey looks like conn_id#channel
  local conn_id, chan = string.match(simpleuserkey, "^([%a%d]+)#([%a%.%d]+)$")
  if conn_id and chan then
    setafter[chan] = true
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

-- return channels before and channels after.
return { arraybefore, arrayafter }
