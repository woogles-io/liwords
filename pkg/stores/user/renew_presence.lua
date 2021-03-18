-- Arguments to this Lua script:
-- uuid, username, auth, connID, ts (ARGV[1] through [5])
local userpresencekey = "userpresence:"..ARGV[1]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4] -- uuid#username#auth#connID

local expiry = 180
local ts = tonumber(ARGV[5])

local purgeold = {}

-- Get all members of userpresencekey
local curchannels = redis.call("ZRANGE", userpresencekey, 0, -1)

-- For every channel that we are in, we renew that channel, only for this conn id.
for i, v in ipairs(curchannels) do
	-- v looks like conn_id#channel
	local chan = string.match(v, ARGV[4].."#([%a%.%d]+)")
	if chan then
		-- extend expiries of the channelpresence...
		redis.call("ZADD", "channelpresence:"..chan, ts + expiry, userkey)
		redis.call("EXPIRE", "channelpresence:"..chan, expiry)
		-- and of the userpresence
		redis.call("ZADD", userpresencekey, ts + expiry, v)
		redis.call("EXPIRE", userpresencekey, expiry)
		-- and the overall set of user presences.
		redis.call("ZADD", "userpresences", ts + expiry, userkey.."#"..chan)
		-- set the last known presence time.
		if ARGV[3] == "auth" then
			redis.call("ZADD", "lastpresences", ts, ARGV[1])
		end
		table.insert(purgeold, "channelpresence:"..chan)
		table.insert(purgeold, userpresencekey)
		table.insert(purgeold, "userpresences")
	end
end

-- remove all subkeys inside the zsets that have expired.
for i, v in ipairs(purgeold) do
	redis.call("ZREMRANGEBYSCORE", v, 0, ts)
end

return purgeold