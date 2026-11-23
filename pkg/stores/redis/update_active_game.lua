-- Arguments to this Lua script:
-- gameuuid expiry p0uuid p1uuid ...
-- (the script should work with any number of players)
-- An expiry of 0 removes the game; otherwise the game is added, or kept,
-- with that expiry (in seconds) from now.

local gameuuid = ARGV[1]
local expiry = tonumber(ARGV[2]) -- seconds

local ts = tonumber(redis.call("TIME")[1])
-- Each member's score is when it expires; it's live while that's after now.
local live = "("..ts

local ret = {}

for idx = 3, #ARGV do
  local useruuid = ARGV[idx]
  local activeusergameskey = "activeusergames:"..useruuid
  local userpresencekey = "userpresence:"..useruuid

  local function channels()
    local set = {}
    for _, simpleuserkey in ipairs(redis.call("ZRANGEBYSCORE", userpresencekey, live, "+inf")) do
      -- simpleuserkey looks like conn_id#channel
      local conn_id, chan = string.match(simpleuserkey, "^([^#]+)#([^#]+)$")
      if conn_id and chan then
        set[chan] = true
      end
    end
    for _, othergameuuid in ipairs(redis.call("ZRANGEBYSCORE", activeusergameskey, live, "+inf")) do
      set["activegame:"..othergameuuid] = true
    end
    return set
  end

  local setbefore = channels()

  -- perform update, dropping games that have expired.
  if expiry > 0 then
    redis.call("ZADD", activeusergameskey, ts + expiry, gameuuid)
    -- The key goes once its last game has expired.
    if redis.call("TTL", activeusergameskey) < expiry then
      redis.call("EXPIRE", activeusergameskey, expiry)
    end
  else
    redis.call("ZREM", activeusergameskey, gameuuid)
  end
  redis.call("ZREMRANGEBYSCORE", activeusergameskey, "-inf", ts)

  local setafter = channels()

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

return ret
