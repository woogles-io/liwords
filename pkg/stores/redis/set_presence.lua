-- Arguments to this Lua script:
-- uuid, username, authOrAnon, connID, channel string, timestamp  (ARGV[1] through [6])

local activeusergameskey = "activeusergames:"..ARGV[1]
local userpresencekey = "userpresence:"..ARGV[1]
local channelpresencekey = "channelpresence:"..ARGV[5]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4] -- uuid#username#auth#connID
local ts = tonumber(ARGV[6])
-- Each member's score is when it expires; it's live while that's after now.
local live = "("..ts

-- 3 minutes. We will renew these keys constantly.
local expiry = 180

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

-- set presence.
local chan = ARGV[5]
local simpleuserkey = ARGV[4].."#"..chan -- just conn_id#channel
redis.call("ZADD", channelpresencekey, ts + expiry, userkey)
redis.call("EXPIRE", channelpresencekey, expiry)
redis.call("ZADD", userpresencekey, ts + expiry, simpleuserkey)
redis.call("EXPIRE", userpresencekey, expiry)
redis.call("ZADD", "userpresences", ts + expiry, userkey.."#"..chan)
-- Drop entries that have expired: ones whose connections went away without
-- clearing them. Renewals purge only what they touch.
redis.call("ZREMRANGEBYSCORE", channelpresencekey, "-inf", ts)
redis.call("ZREMRANGEBYSCORE", userpresencekey, "-inf", ts)

local setafter = channels()

-- add active games to both sets.
for _, gameuuid in ipairs(redis.call("ZRANGEBYSCORE", activeusergameskey, live, "+inf")) do
  local activegamepseudochan = "activegame:"..gameuuid
  setbefore[activegamepseudochan] = true
  setafter[activegamepseudochan] = true
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
