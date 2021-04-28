-- Arguments to this Lua script:
-- uuid

local userpresencekey = "userpresence:"..ARGV[1]

-- get current set of channels.
local setchannel = {}
for _, simpleuserkey in ipairs(redis.call("ZRANGE", userpresencekey, 0, -1)) do
  -- simpleuserkey looks like conn_id#channel
  local conn_id, chan = string.match(simpleuserkey, "^([%a%d]+)#([%a%.%d]+)$")
  if conn_id and chan then
    setchannel[chan] = true
  end
end

-- make sorted set.
local arraychannel = {}
for chan in pairs(setchannel) do
  table.insert(arraychannel, chan)
end
table.sort(arraychannel)

-- return channels.
return { arraychannel }
