package redis

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/tournament"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	upb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

//go:embed add_chat.lua
var AddChatScript string

//go:embed latest_channels.lua
var LatestChannelsScript string

// note: some of these consts may have been moved to lua entirely.

// Expire all non-lobby channels after this many seconds. Lobby channel doesn't expire.
// (We may have other non-expiring channels as well later?)

const LongChannelExpiration = 86400 * 14

const GameChatChannelExpiration = 86400 * 14

const LobbyChatChannel = "chat.lobby"

const ChatsOnReload = 100

const LatestChatSeparator = ":"
const ChatPreviewLength = 40

// RedisChatStore implements a Redis store for chats.
type RedisChatStore struct {
	redisPool            *redis.Pool
	presenceStore        user.PresenceStore
	tournamentStore      tournament.TournamentStore
	addChatScript        *redis.Script
	latestChannelsScript *redis.Script
	eventChan            chan *entity.EventWrapper
}

// NewRedisChatStore instantiates a new store for chats, based on Redis.
func NewRedisChatStore(r *redis.Pool, p user.PresenceStore, t tournament.TournamentStore) *RedisChatStore {
	return &RedisChatStore{
		redisPool:            r,
		presenceStore:        p,
		tournamentStore:      t,
		addChatScript:        redis.NewScript(0, AddChatScript),
		latestChannelsScript: redis.NewScript(0, LatestChannelsScript),
		eventChan:            nil,
	}
}

// redisStreamTS returns the timestamp of the stream data object, in MILLISECONDS.
func redisStreamTS(key string) (int64, error) {
	// note: inefficient
	tskey := strings.Split(key, "-")
	if len(tskey) != 2 {
		return 0, errors.New("wrong timestamp format")
	}
	ts, err := strconv.ParseInt(tskey[0], 10, 64)
	if err != nil {
		return 0, err
	}
	return int64(ts), nil
}

// AddChat takes in sender information, the message, and the name of the channel.
// Additionally, a user-readable name for the channel should be provided.
func (r *RedisChatStore) AddChat(ctx context.Context, senderUsername, senderUID, msg,
	channel, channelFriendly string, regulateChat bool) (*pb.ChatMessage, error) {
	conn := r.redisPool.Get()
	defer conn.Close()

	regulateChatString := "unregulated"
	if regulateChat {
		regulateChatString = "regulated"
	}

	tsNow := time.Now().Unix()

	ret, err := r.addChatScript.Do(conn, senderUsername, senderUID, msg, channel, channelFriendly, tsNow, regulateChatString)
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("ret", ret).Msg("add-chat-return")
	arr, ok := ret.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result: %T", ret)
	}

	was_ok, ok := arr[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected type for arr[0]: %T", arr[0])
	}
	switch string(was_ok) {
	case "err":
		if len(arr) != 2 {
			return nil, fmt.Errorf("unexpected length (want %v, got %v): %v", 2, len(arr), arr)
		}
		reasonBytes, ok := arr[1].([]byte)
		if !ok {
			return nil, fmt.Errorf("unexpected type for arr[1]: %T", arr[1])
		}
		return nil, fmt.Errorf("%s", reasonBytes)
	case "ok":
	default:
		return nil, fmt.Errorf("unexpected value for arr[0]: %s", was_ok)
	}

	if len(arr) != 3 {
		return nil, fmt.Errorf("unexpected length (want %v, got %v): %v", 3, len(arr), arr)
	}

	ts, ok := arr[1].(int64)
	if !ok {
		return nil, fmt.Errorf("unexpected type for arr[1]: %T", arr[1])
	}
	retIdBytes, ok := arr[2].([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected type for arr[2]: %T", arr[2])
	}

	return &pb.ChatMessage{
		Username:  senderUsername,
		UserId:    senderUID,
		Channel:   channel,
		Message:   msg,
		Timestamp: ts,
		Id:        string(retIdBytes),
	}, nil
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

	messages := make([]*pb.ChatMessage, len(vals))
	for idx, val := range vals {
		msg, err := rstreamMsgToChatMsg(val)
		if err != nil {
			return nil, err
		}
		msg.Channel = channel
		messages[len(vals)-1-idx] = msg
	}

	return messages, nil
}

func rstreamMsgToChatMsg(v interface{}) (*pb.ChatMessage, error) {
	// This is kind of gross and fragile, but redigo doesn't have stream support yet 😥
	msg := &pb.ChatMessage{}

	val := v.([]interface{})
	// val[0] is the timestamp key
	tskey := string(val[0].([]byte))
	ts, err := redisStreamTS(tskey)
	if err != nil {
		return nil, err
	}
	msg.Timestamp = ts
	msg.Id = tskey

	// val[1] is an array of arrays. ["username", username, "message", message, "userID", userID]
	msgvals := val[1].([]interface{})
	msg.Username = string(msgvals[1].([]byte))
	msg.Message = string(msgvals[3].([]byte))
	if len(msgvals) > 5 {
		// We need this check because we didn't always store userID -- although
		// we can likely remove this once old chats have expired.
		msg.UserId = string(msgvals[5].([]byte))
	}
	return msg, nil
}

// DeleteChat deletes a chat.
func (r *RedisChatStore) DeleteChat(ctx context.Context, channel, msgID string) error {
	redisKey := "chat:" + strings.TrimPrefix(channel, "chat.")
	log.Debug().Str("redisKey", redisKey).Str("msgID", msgID).Msg("delete-chat")
	conn := r.redisPool.Get()
	defer conn.Close()

	val, err := redis.Int(conn.Do("XDEL", redisKey, msgID))
	if err != nil {
		return err
	}
	if val == 0 {
		return errors.New("zero chats deleted")
	}

	return nil
}

func (r *RedisChatStore) GetChat(ctx context.Context, channel, msgID string) (*pb.ChatMessage, error) {
	redisKey := "chat:" + strings.TrimPrefix(channel, "chat.")
	log.Debug().Str("redisKey", redisKey).Str("msgID", msgID).Msg("get-chat")
	conn := r.redisPool.Get()
	defer conn.Close()

	vals, err := redis.Values(conn.Do("XRANGE", redisKey, msgID, msgID))
	if err != nil {
		return nil, err
	}
	if len(vals) != 1 {
		return nil, errors.New("no such message id")
	}
	msg, err := rstreamMsgToChatMsg(vals[0])
	if err != nil {
		return nil, err
	}
	msg.Channel = channel
	return msg, nil
}

func maybeTrim(msg string) string {
	// No, don't trim on the backend.
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

func (r *RedisChatStore) SetEventChan(c chan *entity.EventWrapper) {
	r.eventChan = c
}

func (r *RedisChatStore) EventChan() chan *entity.EventWrapper {
	return r.eventChan
}
