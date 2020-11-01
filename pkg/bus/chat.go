package bus

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/domino14/liwords/pkg/entity"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
)

// Expire all non-lobby channels after this many seconds. Lobby channel doesn't expire.
// (We may have other non-expiring channels as well later?)
const ChannelExpiration = 86400

const LobbyChannel = "lobby.chat"

const ChatsOnReload = 100

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

// chat-related functionality should be here. Chat should be mostly ephemeral,
// but will use Redis to keep a short history of previous chats in every channel.

func (b *Bus) chat(ctx context.Context, userID string, evt *pb.ChatMessage) error {
	if len(evt.Message) > MaxMessageLength {
		return errors.New("message-too-long")
	}
	username, _, err := b.userStore.Username(ctx, userID)
	if err != nil {
		return err
	}
	conn := b.redisPool.Get()
	defer conn.Close()
	redisKey := "chat:" + evt.Channel

	ret, err := redis.String(conn.Do("XADD", redisKey, "MAXLEN", "~", "500", "*",
		"username", username, "message", evt.Message, "userID", userID))
	if err != nil {
		return err
	}

	ts, err := redisStreamTS(ret)
	if err != nil {
		return err
	}
	chatMessage := &pb.ChatMessage{
		Username:  username,
		UserId:    userID,
		Channel:   evt.Channel, // this info might be redundant
		Message:   evt.Message,
		Timestamp: ts,
	}

	toSend := entity.WrapEvent(chatMessage, pb.MessageType_CHAT_MESSAGE)
	data, err := toSend.Serialize()

	if err != nil {
		return err
	}

	if evt.Channel != LobbyChannel {
		_, err = conn.Do("EXPIRE", redisKey, ChannelExpiration)
		if err != nil {
			return err
		}
	}
	log.Debug().Interface("chat-message", chatMessage).Msg("publish-chat")
	return b.natsconn.Publish(evt.Channel, data)
}

func (b *Bus) sendOldChats(userID, chatChannel string) error {
	// Send chats in a chatChannel to the given user.
	log.Debug().Str("chatChannel", chatChannel).Msg("send-old-chats")
	if chatChannel == "" {
		// No chats for this channel.
		return nil
	}
	redisKey := "chat:" + chatChannel
	log.Debug().Str("redisKey", redisKey).Msg("get-old-chats")
	conn := b.redisPool.Get()
	defer conn.Close()

	// Get the latest 50 chats to display to the user.
	vals, err := redis.Values(conn.Do("XREVRANGE", redisKey, "+", "-", "COUNT", ChatsOnReload))
	if err != nil {
		return err
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
			return err
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

		messages[len(vals)-1-idx] = msg
	}

	toSend := entity.WrapEvent(&pb.ChatMessages{
		Messages: messages,
	}, pb.MessageType_CHAT_MESSAGES)

	log.Debug().Int("num-chats", len(messages)).Msg("sending-chats")
	return b.pubToUser(userID, toSend, chatChannel)
}
