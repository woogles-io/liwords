-- this code was translated from golang.

local senderUsername = ARGV[1]
local senderUID = ARGV[2]
local msg = ARGV[3]
local channel = ARGV[4]
local channelFriendly = ARGV[5]

local LongChannelExpiration = 86400 * 14
local GameChatChannelExpiration = 86400
local LobbyChatChannel = "chat.lobby"
local LatestChatSeparator = ":"

local function storeLatestChat(msg, userID, channel, channelFriendly, tsSeconds)
  local lchanKeyPrefix = "latestchannel:"
  local key = lchanKeyPrefix .. userID
  redis.call("ZADD", key, tsSeconds + LongChannelExpiration,
    channel .. LatestChatSeparator .. channelFriendly)
  redis.call("EXPIRE", key, LongChannelExpiration)
end

local function hasPrefix(s, sub)
  return string.sub(s, 1, string.len(sub)) == sub
end

local function trimPrefix(s, sub)
  if hasPrefix(s, sub) then
    return string.sub(s, string.len(sub) + 1)
  end
  return s
end

local redisKey = "chat:" .. trimPrefix(channel, "chat.")

local ret = redis.call("XADD", redisKey, "MAXLEN", "~", "500", "*",
  "username", senderUsername, "message", msg, "userID", senderUID)

local tsString = string.match(ret, "^(%d+)-%d+$")
local ts = tonumber(tsString)
local tsSeconds = math.floor(ts / 1000)

if channel ~= LobbyChatChannel then
  local exp
  if hasPrefix(channel, "chat.tournament.") or hasPrefix(channel, "chat.pm.") then
    exp = LongChannelExpiration
  else
    exp = GameChatChannelExpiration
  end
  redis.call("EXPIRE", redisKey, exp)
end

if hasPrefix(channel, "chat.pm.") then
  for user in string.gmatch(trimPrefix(channel, "chat.pm."), "[^_]+") do
    storeLatestChat(msg, user, channel, channelFriendly, tsSeconds)
  end
elseif hasPrefix(channel, "chat.tournament.") then
  storeLatestChat(msg, senderUID, channel, channelFriendly, tsSeconds)
end

return { ts, ret }
