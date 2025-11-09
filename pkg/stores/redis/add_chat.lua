-- this code was translated from golang.

local senderUsername = ARGV[1]
local senderUID = ARGV[2]
local msg = ARGV[3]
local channel = ARGV[4]
local channelFriendly = ARGV[5]
local tsNow = tonumber(ARGV[6])
local regulateChat = ARGV[7] -- "regulated", "unregulated"

local MessageCooldownTime = 5
local DuplicateMessageCooldownTime = 30 * 60
local LongChannelExpiration = 86400 * 14
local GameChatChannelExpiration = 86400 * 14
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

local userCooldownKey = "userchatcooldown:" .. senderUID

if regulateChat == "regulated" then
  local hmgetRet = redis.call("HMGET", userCooldownKey,
    "ts",
    "msg")
  local lastMessageTimeString = hmgetRet[1]
  local lastMessage = hmgetRet[2]

  if lastMessage then
    -- Check if the message is identical to the last one
    if msg == lastMessage then
      return { "err", "you cannot send a message identical to the previous one" }
    end
  end

  if lastMessageTimeString then
    local lastMessageTime = tonumber(lastMessageTimeString)

    -- Check if the cooldown is over
    local cooldownFinishedTime = lastMessageTime + MessageCooldownTime
    if cooldownFinishedTime > tsNow then
      return { "err", "you cannot send messages that quickly" }
    end
  end

  redis.call("HMSET", userCooldownKey,
    "ts", tsNow,
    "msg", msg)
  redis.call("EXPIRE", userCooldownKey, DuplicateMessageCooldownTime)
elseif regulateChat == "unregulated" then
else
  return { "err", "invalid parameter" }
end

local redisKey = "chat:" .. trimPrefix(channel, "chat.")

local ret = redis.call("XADD", redisKey, "MAXLEN", "~", "500", "*",
  "username", senderUsername, "message", msg, "userID", senderUID)

local tsString = string.match(ret, "^(%d+)-%d+$")
local ts = tonumber(tsString)
local tsSeconds = math.floor(ts / 1000)

if channel ~= LobbyChatChannel then
  local exp
  if hasPrefix(channel, "chat.tournament.") or hasPrefix(channel, "chat.league.") or hasPrefix(channel, "chat.pm.") then
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
elseif hasPrefix(channel, "chat.league.") then
  storeLatestChat(msg, senderUID, channel, channelFriendly, tsSeconds)
end

return { "ok", ts, ret }
