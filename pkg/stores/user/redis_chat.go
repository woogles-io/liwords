package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/tournament"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	upb "github.com/domino14/liwords/rpc/api/proto/user_service"
)

const LatestChannelsScript = `
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
`

// Expire all non-lobby channels after this many seconds. Lobby channel doesn't expire.
// (We may have other non-expiring channels as well later?)

const LongChannelExpiration = 86400 * 14

const GameChatChannelExpiration = 86400

const LobbyChatChannel = "chat.lobby"

const ChatsOnReload = 100

const LatestChatSeparator = ":"
const ChatPreviewLength = 40

// RedisChatStore implements a Redis store for chats.
type RedisChatStore struct {
	redisPool            *redis.Pool
	presenceStore        user.PresenceStore
	tournamentStore      tournament.TournamentStore
	latestChannelsScript *redis.Script
}

// NewRedisChatStore instantiates a new store for chats, based on Redis.
func NewRedisChatStore(r *redis.Pool, p user.PresenceStore, t tournament.TournamentStore) *RedisChatStore {
	return &RedisChatStore{
		redisPool:            r,
		presenceStore:        p,
		tournamentStore:      t,
		latestChannelsScript: redis.NewScript(0, LatestChannelsScript),
	}
}

// redisStreamTS returns the timestamp of the stream data object, in MILLISECONDS.
func redisStreamTS(key string) (int64, error) {
	tskey := strings.Split(key, "-")
	if len(tskey) != 2 {
		return 0, errors.New("wrong timestamp format")
	}
	ts, err := strconv.Atoi(tskey[0])
	if err != nil {
		return 0, err
	}
	return int64(ts), nil
}

// AddChat takes in sender information, the message, and the name of the channel.
// Additionally, a user-readable name for the channel should be provided.
func (r *RedisChatStore) AddChat(ctx context.Context, senderUsername, senderUID, msg,
	channel, channelFriendly string) (int64, error) {
	conn := r.redisPool.Get()
	defer conn.Close()
	redisKey := "chat:" + strings.TrimPrefix(channel, "chat.")

	ret, err := redis.String(conn.Do("XADD", redisKey, "MAXLEN", "~", "500", "*",
		"username", senderUsername, "message", msg, "userID", senderUID))
	if err != nil {
		return 0, err
	}

	// ts is in milliseconds
	ts, err := redisStreamTS(ret)
	if err != nil {
		return 0, err
	}
	tsSeconds := ts / 1000

	if channel != LobbyChatChannel {
		var exp int
		if strings.HasPrefix(channel, "chat.tournament.") || strings.HasPrefix(channel, "chat.pm.") {
			exp = LongChannelExpiration
		} else {
			exp = GameChatChannelExpiration
		}
		_, err = conn.Do("EXPIRE", redisKey, exp)
		if err != nil {
			return 0, err
		}
	}

	if strings.HasPrefix(channel, "chat.pm.") {
		users := strings.Split(strings.TrimPrefix(channel, "chat.pm."), "_")

		for _, user := range users {
			// Update the entry for each latestchannel key for each user in this
			// private-message channel.
			err := r.storeLatestChat(conn, msg, user, channel, channelFriendly, tsSeconds)
			if err != nil {
				return 0, err
			}
		}
	} else if strings.HasPrefix(channel, "chat.tournament.") {
		err = r.storeLatestChat(conn, msg, senderUID, channel, channelFriendly, tsSeconds)
		if err != nil {
			return 0, err
		}
	}

	return ts, nil
}

func (r *RedisChatStore) storeLatestChat(conn redis.Conn,
	msg, userID, channel, channelFriendly string, tsSeconds int64) error {
	// Add to the relevant "latest channels" key
	lchanKeyPrefix := "latestchannel:"

	key := lchanKeyPrefix + userID

	_, err := conn.Do("ZADD", key, tsSeconds+LongChannelExpiration,
		channel+LatestChatSeparator+channelFriendly)
	if err != nil {
		return err
	}
	_, err = conn.Do("EXPIRE", key, LongChannelExpiration)
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisChatStore) OldChats(ctx context.Context, channel string, n int) ([]*pb.ChatMessage, error) {
	redisKey := "chat:" + strings.TrimPrefix(channel, "chat.")
	log.Debug().Str("redisKey", redisKey).Msg("get-old-chats")
	conn := r.redisPool.Get()
	defer conn.Close()

	// Get the latest chats to display to the user.
	vals, err := redis.Values(conn.Do("XREVRANGE", redisKey, "+", "-", "COUNT", n))
	if err != nil {
		return nil, err
	}

	// This is kind of gross and fragile, but redigo doesn't have stream support yet 😥
	messages := make([]*pb.ChatMessage, len(vals))
	for idx, val := range vals {
		msg := &pb.ChatMessage{}

		val := val.([]interface{})
		// val[0] is the timestamp key
		tskey := string(val[0].([]byte))
		ts, err := redisStreamTS(tskey)
		if err != nil {
			return nil, err
		}
		msg.Timestamp = ts

		// val[1] is an array of arrays. ["username", username, "message", message, "userID", userID]
		msgvals := val[1].([]interface{})
		msg.Username = string(msgvals[1].([]byte))
		msg.Message = string(msgvals[3].([]byte))
		if len(msgvals) > 5 {
			// We need this check because we didn't always store userID -- although
			// we can likely remove this once old chats have expired.
			msg.UserId = string(msgvals[5].([]byte))
		}
		msg.Channel = channel
		messages[len(vals)-1-idx] = msg
	}

	return messages, nil
}

func maybeTrim(msg string) string {
	if len(msg) > ChatPreviewLength {
		msg = msg[:ChatPreviewLength] + "…"
	}
	return msg
}

// LatestChannels returns a list of channel names for the given user. These
// channels should be sorted in time order starting with the most recent.
// The time associated with each channel is the time of the latest message sent in
// that channel.
// If a non-blank tournamentID is passed, we force this function to return
// the chat channel for the given tournament ID, even if the user has not
// yet chatted in it.
func (r *RedisChatStore) LatestChannels(ctx context.Context, count, offset int,
	uid, tournamentID string) (*upb.ActiveChatChannels, error) {

	conn := r.redisPool.Get()
	defer conn.Close()

	ts := time.Now().Unix()

	vals, err := redis.Strings(r.latestChannelsScript.Do(conn, uid, count, offset, ts))
	if err != nil {
		return nil, err
	}

	lastSeen, err := r.presenceStore.LastSeen(ctx, uid)
	if err != nil {
		// Don't die, this key might not yet exist.
		log.Err(err).Str("uid", uid).Msg("last-seen-not-exist")
	}

	log.Debug().Interface("vals", vals).Msg("vals-from-redis")
	chans := make([]*upb.ActiveChatChannels_Channel, len(vals)/3)
	getTournament := tournamentID != ""

	for idx := 0; idx < len(chans); idx++ {

		chanName := strings.SplitN(vals[idx*3], LatestChatSeparator, 2)
		if len(chanName) != 2 {
			return nil, fmt.Errorf("unexpected channel name: %v", chanName)
		}
		// The timestamp looks like millisecond_ts-seqno
		tst := strings.Split(vals[idx*3+1], "-")
		if len(tst) != 2 {
			return nil, fmt.Errorf("malformed timestamp: %v", tst)
		}

		ts, err := strconv.ParseInt(tst[0], 10, 64)
		if err != nil {
			return nil, err
		}
		lastMsg := maybeTrim(vals[idx*3+2])
		lastUpdate := ts / 1000

		chans[idx] = &upb.ActiveChatChannels_Channel{
			Name:        chanName[0],
			DisplayName: chanName[1],
			LastUpdate:  lastUpdate,
			LastMessage: lastMsg,
			HasUpdate:   lastUpdate > lastSeen,
		}
		if tournamentID != "" && chanName[0] == "chat.tournament."+tournamentID {
			getTournament = false
		}
	}

	// If a tournament ID is passed in, we should fetch the tournament channel
	// as well as the latest chat for this tournament.
	if getTournament {
		t, err := r.tournamentStore.Get(ctx, tournamentID)
		if err != nil {
			return nil, err
		}
		// Get the last chat for this tournament channel.
		chatChannel := "chat.tournament." + tournamentID
		cm, err := r.OldChats(ctx, chatChannel, 1)
		if err != nil {
			return nil, err
		}
		lastUpdate := int64(0)
		lastMessage := ""
		if len(cm) == 1 {
			lastUpdate = int64(cm[0].Timestamp / 1000)
			lastMessage = maybeTrim(cm[0].Message)
		}

		chans = append(chans, &upb.ActiveChatChannels_Channel{
			Name:        chatChannel,
			DisplayName: "tournament:" + t.Name,
			LastUpdate:  lastUpdate,
			LastMessage: lastMessage,
			HasUpdate:   lastUpdate > lastSeen && lastMessage != "",
		})
	}

	return &upb.ActiveChatChannels{Channels: chans}, nil
}
