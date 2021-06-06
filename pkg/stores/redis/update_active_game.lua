-- Arguments to this Lua script:
-- gameuuid expiry p0uuid p1uuid ...
-- (the script should work with any number of players)

local gameuuid = ARGV[1]
local expiry = tonumber(ARGV[2]) -- seconds

local ts = tonumber(redis.call("TIME")[1])

local ret = {}

for idx, useruuid in ipairs(ARGV) do
  if idx >= 3 then
    local activeusergameskey = "activeusergames:"..useruuid
    local userpresencekey = "userpresence:"..useruuid

    local setbefore = {}
    local setafter = {}
    for _, simpleuserkey in ipairs(redis.call("ZRANGE", userpresencekey, 0, -1)) do
      -- simpleuserkey looks like conn_id#channel
      local conn_id, chan = string.match(simpleuserkey, "^([%a%d]+)#([%a%.%d]+)$")
      if conn_id and chan then
        setbefore[chan] = true
        setafter[chan] = true
      end
    end

    -- add active games to both sets.
    for _, othergameuuid in ipairs(redis.call("ZRANGE", activeusergameskey, 0, -1)) do
      local activegamepseudochan = "activegame:"..othergameuuid
      setbefore[activegamepseudochan] = true
      setafter[activegamepseudochan] = true
    end

    -- perform update.
    local activegamepseudochan = "activegame:"..gameuuid
    if expiry > 0 then
      redis.call("ZADD", activeusergameskey, ts + expiry, gameuuid)
      setafter[activegamepseudochan] = true
    else
      redis.call("ZREM", activeusergameskey, gameuuid)
      setafter[activegamepseudochan] = nil
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

    table.insert(ret, {arraybefore, arrayafter})
  end
end

return ret
