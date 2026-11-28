-- Checks one request against several rate limit buckets at once (GCRA, a
-- token bucket stored as one timestamp per key).
--
-- KEYS: one key per bucket.
-- ARGV: for each bucket, two numbers: its interval in milliseconds (the time
-- one request's token takes to come back) and its burst (how many requests
-- can come at once).
--
-- The request is allowed only if every bucket has room; then it takes one
-- token from each. A refused request takes nothing.
-- Returns {1, 0} when allowed, or {0, ms} with how long to wait.
--
-- The off switch: while the key rl:off exists, every request is allowed and
-- nothing is counted (valkey-cli SET rl:off 1; DEL rl:off to turn back on).
-- Read as a literal key, which is fine on our single, non-cluster server.

if redis.call("EXISTS", "rl:off") == 1 then
  return { 1, 0 }
end

local t = redis.call("TIME")
-- Milliseconds. Kept under 14 digits so Lua's number formatting stays exact.
local now = tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)

local tats = {}
local wait = 0
for i, key in ipairs(KEYS) do
  local interval = tonumber(ARGV[2 * i - 1])
  local burst = tonumber(ARGV[2 * i])
  -- The bucket's "theoretical arrival time": when it will be full again.
  local tat = tonumber(redis.call("GET", key)) or now
  if tat < now then
    tat = now
  end
  local newtat = tat + interval
  local allowat = newtat - interval * burst
  if allowat > now and allowat - now > wait then
    wait = allowat - now
  end
  tats[i] = newtat
end

if wait > 0 then
  return { 0, wait }
end

for i, key in ipairs(KEYS) do
  -- The key can go once the bucket is full again.
  redis.call("SET", key, tats[i], "PX", tats[i] - now + 1000)
end
return { 1, 0 }
