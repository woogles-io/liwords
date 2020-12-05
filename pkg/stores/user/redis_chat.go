package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	upb "github.com/domino14/liwords/rpc/api/proto/user_service"

	"github.com/rs/zerolog/log"

	"github.com/gomodule/redigo/redis"
)

// Expire all non-lobby channels after this many seconds. Lobby channel doesn't expire.
// (We may have other non-expiring channels as well later?)

// These should expire at the same time:
const TournamentChannelExpiration = 86400 * 14
const PMChannelExpiration = 86400 * 14

const GameChatChannelExpiration = 86400

const LobbyChatChannel = "chat.lobby"

const ChatsOnReload = 100

// RedisChatStore implements a Redis store for chats.
type RedisChatStore struct {
	redisPool *redis.Pool
}

// NewRedisChatStore instantiates a new store for chats, based on Redis.
func NewRedisChatStore(r *redis.Pool) *RedisChatStore {
	return &RedisChatStore{
		redisPool: r,
	}
}

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
// Additionally, a user-friendly name for the channel should be provided.
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

	ts, err := redisStreamTS(ret)
	if err != nil {
		return 0, err
	}

	if channel != LobbyChatChannel {
		var exp int
		if strings.HasPrefix(channel, "chat.tournament") {
			exp = TournamentChannelExpiration
		} else if strings.HasPrefix(channel, "chat.pm") {
			exp = PMChannelExpiration
		} else {
			exp = GameChatChannelExpiration
		}
		_, err = conn.Do("EXPIRE", redisKey, exp)
		if err != nil {
			return 0, err
		}
	}

	// Add to the relevant "latest channels" key
	lcKeyPrefix := "latestchannel:"

	if strings.HasPrefix(channel, "chat.pm.") {
		users := strings.Split(strings.TrimPrefix(channel, "chat.pm."), "_")
		for _, user := range users {
			key := lcKeyPrefix + user
			// Update the entry for each latestchannel key for each user in this
			// private-message channel.
			_, err := conn.Do("ZADD", key, ts+PMChannelExpiration, channel+":"+channelFriendly)
			if err != nil {
				return 0, err
			}
			_, err = conn.Do("EXPIRE", key, TournamentChannelExpiration)
			if err != nil {
				return 0, err
			}
		}
	} else if strings.HasPrefix(channel, "chat.tournament.") {
		key := lcKeyPrefix + senderUID
		_, err := conn.Do("ZADD", key, ts+TournamentChannelExpiration, channel+":"+channelFriendly)
		if err != nil {
			return 0, err
		}
		_, err = conn.Do("EXPIRE", key, TournamentChannelExpiration)
		if err != nil {
			return 0, err
		}
	}

	return ts, nil
}

func (r *RedisChatStore) OldChats(ctx context.Context, channel string) ([]*pb.ChatMessage, error) {
	redisKey := "chat:" + strings.TrimPrefix(channel, "chat.")
	log.Debug().Str("redisKey", redisKey).Msg("get-old-chats")
	conn := r.redisPool.Get()
	defer conn.Close()

	// Get the latest 50 chats to display to the user.
	vals, err := redis.Values(conn.Do("XREVRANGE", redisKey, "+", "-", "COUNT", ChatsOnReload))
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

// LatestChannels returns a list of channel names for the given user. These
// channels should be sorted in time order starting with the most recent.
// The time associated with each channel is the time of the latest message sent in
// that channel.
func (r *RedisChatStore) LatestChannels(ctx context.Context, count, offset int, uid string) (*upb.ActiveChatChannels, error) {
	conn := r.redisPool.Get()
	defer conn.Close()

	lcKey := "latestchannel:" + uid

	// First expire anything that needs expiring.
	ts := time.Now().Unix()
	_, err := conn.Do("ZREMRANGEBYSCORE", lcKey, 0, ts)
	if err != nil {
		return nil, err
	}

	vals, err := redis.Strings(
		conn.Do("ZREVRANGEBYSCORE", lcKey, "+inf", "-inf", "LIMIT", offset, count))
	if err != nil {
		return nil, err
	}
	chans := make([]*upb.ActiveChatChannels_Channel, len(vals))
	for idx, member := range vals {
		chanName := strings.Split(member, ":")
		if len(chanName) != 2 {
			return nil, fmt.Errorf("unexpected channel name: %v", chanName)
		}
		chans[idx] = &upb.ActiveChatChannels_Channel{
			Name:        chanName[0],
			DisplayName: chanName[1],
		}
	}
	return &upb.ActiveChatChannels{Channels: chans}, nil
}
