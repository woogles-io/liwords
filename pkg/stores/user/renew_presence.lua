-- Arguments to this Lua script:
-- uuid, username, auth, connID, ts (ARGV[1] through [5])

local userpresencekey = "userpresence:"..ARGV[1]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4] -- uuid#username#auth#connID
local ts = tonumber(ARGV[5])

-- 3 minutes. We will renew these keys constantly.
local expiry = 180

-- renew where conn_id matches.
local shouldtouch = false
local setbefore = {}
local setpurged = {}
for _, simpleuserkey in ipairs(redis.call("ZRANGE", userpresencekey, 0, -1)) do
  -- simpleuserkey looks like conn_id#channel
  local conn_id, chan = string.match(simpleuserkey, "^([%a%d]+)#([%a%.%d]+)$")
  if conn_id and chan then
    setbefore[chan] = true
    if conn_id == ARGV[4] then
      local channelpresencekey = "channelpresence:"..chan
      redis.call("ZADD", channelpresencekey, ts + expiry, userkey)
      redis.call("EXPIRE", channelpresencekey, expiry)
      setpurged[channelpresencekey] = true
      redis.call("ZADD", userpresencekey, ts + expiry, simpleuserkey)
      redis.call("EXPIRE", userpresencekey, expiry)
      setpurged[userpresencekey] = true
      redis.call("ZADD", "userpresences", ts + expiry, userkey.."#"..chan)
      setpurged["userpresences"] = true
      shouldtouch = true
    end
  end
end

-- update the last known presence time.
if shouldtouch and ARGV[3] == "auth" then
  redis.call("ZADD", "lastpresences", ts, ARGV[1])
end

-- remove all subkeys inside the zsets that have expired.
for k in pairs(setpurged) do
  redis.call("ZREMRANGEBYSCORE", k, -math.huge, ts)
end

-- inefficient, but easier to write.
-- an efficient algorithm counts each channel and decrements those counts during purge.
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
