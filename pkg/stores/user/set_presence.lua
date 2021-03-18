-- Arguments to this Lua script:
-- uuid, username, authOrAnon, connID, channel string, timestamp  (ARGV[1] through [6])

local userpresencekey = "userpresence:"..ARGV[1]
local channelpresencekey = "channelpresence:"..ARGV[5]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4] -- uuid#username#auth#connID
-- 3 minutes. We will renew these keys constantly.
local expiry = 180
local ts = tonumber(ARGV[6])
local simpleuserkey = ARGV[4].."#"..ARGV[5] -- just conn_id#channel
-- Set user presence:
redis.call("ZADD", userpresencekey, ts + expiry, simpleuserkey)
redis.call("ZADD", "userpresences", ts + expiry, userkey.."#"..ARGV[5])
-- Do not modify lastpresences here. We only modify that when people leave, or on
-- the renew presence here. We'd like there to be a little time (a few seconds)
-- when the user logs in to determine how long it's been since they last logged in.
-- Set channel presence:
redis.call("ZADD", channelpresencekey, ts + expiry, userkey)
-- Expire ephemeral presence keys:

redis.call("EXPIRE", userpresencekey, expiry)
redis.call("EXPIRE", channelpresencekey, expiry)