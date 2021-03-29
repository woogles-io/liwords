-- Arguments to this Lua script:
-- uuid, username, authOrAnon, connID, timestamp

local userpresencekey = "userpresence:"..ARGV[1]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4]  -- uuid#username#anon#conn_id

-- get the current channels that this presence is in.

local curchannels = redis.call("ZRANGE", userpresencekey, 0, -1)

local deletedfrom = {}
local stillconnectedto = {}
local ts = tonumber(ARGV[5])

-- only delete the users where the conn_id actually matches
for i, v in ipairs(curchannels) do
	-- v looks like conn_id#channel
	local conn_id, chan = string.match(v, "^([%a%d]+)#([%a%.%d]+)$")
	if not (conn_id and chan) then
		-- should not happen
	elseif conn_id == ARGV[4] then
		table.insert(deletedfrom, chan)
		-- delete from the relevant channel key
		redis.call("ZREM", "channelpresence:"..chan, userkey)
		redis.call("ZREM", userpresencekey, v)
		redis.call("ZREM", "userpresences", userkey.."#"..chan)
		-- update the last known presence time.
		if ARGV[3] == "auth" then
			redis.call("ZADD", "lastpresences", ts, ARGV[1])
		end
	else
		-- user is still in chan through another conn, do not delete yet
		stillconnectedto[chan] = true
	end
end

local totallydisconnectedfrom = {}
for _, chan in ipairs(deletedfrom) do
	if not stillconnectedto[chan] then
		table.insert(totallydisconnectedfrom, chan)
	end
end

-- return the channel(s) this user totally disconnected from.
return totallydisconnectedfrom