-- Arguments
-- UID, count, offset, nowTS (ARGV[1] through [4])

local lckey = "latestchannel:"..ARGV[1]
-- expire anything that needs expiring
local ts = tonumber(ARGV[4])
redis.call("ZREMRANGEBYSCORE", lckey, 0, ts)
-- get all channels
local offset = tonumber(ARGV[3])
local count = tonumber(ARGV[2])
local rresp = redis.call("ZREVRANGEBYSCORE", lckey, "+inf", "-inf", "LIMIT", offset, count)
-- parse through redis results

local results = {}

for i, v in ipairs(rresp) do
	-- channel looks like  channel:friendly_name
	-- capture the channel.
	-- Accepted characters in channel name (the thing after "chat." --
	--  letters, numbers, period, dash and underscore.
	-- Note: the dash is only there to fix a legacy crash. (liwords GH Issue #325)
	-- We can remove this after a few weeks, once any old channels expire. It won't
	-- work because presence channels use dashes as separators (realms).
	local chan = string.match(v, "chat%.([%a%.%d%-_]+):.+")
	if chan then
		-- get the last chat msg
		local chatkey = "chat:"..chan
		local lastchat = redis.call("XREVRANGE", chatkey, "+", "-", "COUNT", 1)
		if lastchat ~= nil and lastchat[1] ~= nil then
			-- lastchat[1][1] is the timestamp, lastchat[1][2] is the bulk reply
			-- lastchat[1][2][4] is always the message
			-- So insert, in order: the full channel name (v), the timestamp of
			-- the last chat, and the last chat.
			table.insert(results, v)
			table.insert(results, lastchat[1][1])
			table.insert(results, lastchat[1][2][4])
		end
	end
end

return results