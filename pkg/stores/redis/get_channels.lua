-- Arguments to this Lua script:
-- ts, uuid1, uuid2, ...
-- Returns, for each user, their sorted channels: the ones a live connection
-- of theirs is in, and activegame:<id> for each game they're playing.

local ts = tonumber(ARGV[1])
-- Each member's score is when it expires; it's live while that's after now.
local live = "("..ts

local ret = {}
for idx = 2, #ARGV do
  local uuid = ARGV[idx]
  local setchannel = {}
  for _, simpleuserkey in ipairs(redis.call("ZRANGEBYSCORE", "userpresence:"..uuid, live, "+inf")) do
    -- simpleuserkey looks like conn_id#channel
    local conn_id, chan = string.match(simpleuserkey, "^([^#]+)#([^#]+)$")
    if conn_id and chan then
      setchannel[chan] = true
    end
  end
  for _, gameuuid in ipairs(redis.call("ZRANGEBYSCORE", "activeusergames:"..uuid, live, "+inf")) do
    setchannel["activegame:"..gameuuid] = true
  end

  local arraychannel = {}
  for chan in pairs(setchannel) do
    table.insert(arraychannel, chan)
  end
  table.sort(arraychannel)
  table.insert(ret, arraychannel)
end

return ret
