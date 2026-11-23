-- Arguments to this Lua script:
-- ts, limit
--
-- Sweeps up to limit expired entries of userpresences: connections that went
-- away without clearing their presence (a socket server that crashed or was
-- redeployed, a network that dropped). Removes what is left of each entry
-- in channelpresence and userpresence, unless the connection has renewed it
-- since. For each user swept, returns
--   { uuid, username, auth, removed, before, after }
-- where removed are the channels no live connection of theirs is in any
-- more (the ones to announce they left), and before and after are their
-- channels, active games included, as the other scripts return them.
-- The first element of the result is how many entries were swept.

local ts = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local live = "("..ts

local expired = redis.call("ZRANGEBYSCORE", "userpresences", "-inf", ts, "LIMIT", 0, limit)

local users = {}
local order = {}
for _, member in ipairs(expired) do
  redis.call("ZREM", "userpresences", member)
  -- member looks like uuid#username#auth#conn_id#channel
  local uuid, username, auth, conn_id, chan =
    string.match(member, "^([^#]+)#([^#]+)#([^#]+)#([^#]+)#([^#]+)$")
  if uuid then
    local channelpresencekey = "channelpresence:"..chan
    local userkey = uuid.."#"..username.."#"..auth.."#"..conn_id
    local s = redis.call("ZSCORE", channelpresencekey, userkey)
    if s and tonumber(s) <= ts then
      redis.call("ZREM", channelpresencekey, userkey)
    end
    local userpresencekey = "userpresence:"..uuid
    local simpleuserkey = conn_id.."#"..chan
    s = redis.call("ZSCORE", userpresencekey, simpleuserkey)
    if s and tonumber(s) <= ts then
      redis.call("ZREM", userpresencekey, simpleuserkey)
    end
    if not users[uuid] then
      users[uuid] = {username = username, auth = auth, chans = {}}
      table.insert(order, uuid)
    end
    users[uuid].chans[chan] = true
  end
end

local function sorted(set)
  local arr = {}
  for k in pairs(set) do
    table.insert(arr, k)
  end
  table.sort(arr)
  return arr
end

local ret = { #expired }
for _, uuid in ipairs(order) do
  local u = users[uuid]
  local after = {}
  for _, simpleuserkey in ipairs(redis.call("ZRANGEBYSCORE", "userpresence:"..uuid, live, "+inf")) do
    local conn_id, chan = string.match(simpleuserkey, "^([^#]+)#([^#]+)$")
    if conn_id and chan then
      after[chan] = true
    end
  end
  for _, gameuuid in ipairs(redis.call("ZRANGEBYSCORE", "activeusergames:"..uuid, live, "+inf")) do
    after["activegame:"..gameuuid] = true
  end
  local removed = {}
  local before = {}
  for chan in pairs(after) do
    before[chan] = true
  end
  for chan in pairs(u.chans) do
    if not after[chan] then
      removed[chan] = true
      before[chan] = true
    end
  end
  table.insert(ret, { uuid, u.username, u.auth, sorted(removed), sorted(before), sorted(after) })
end

return ret
