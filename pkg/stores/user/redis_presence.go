package user

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/zerolog/log"
)

const (
	NullPresenceChannel = "NULL"
)

// RedisPresenceStore implements a Redis store for user presence.
type RedisPresenceStore struct {
	redisPool *redis.Pool

	setPresenceScript      *redis.Script
	clearPresenceScript    *redis.Script
	renewPresenceScript    *redis.Script
	getChannelsScript      *redis.Script
	updateActiveGameScript *redis.Script

	eventChan chan *entity.EventWrapper
}

// SetPresenceScript is a Lua script that handles presence in an atomic way.
// We may want to move this to a separate file if we start adding more Lua
// scripts.
//go:embed set_presence.lua
var SetPresenceScript string

// ClearPresenceScript clears the presence and returns the channel(s) it was in.
//go:embed clear_presence.lua
var ClearPresenceScript string

// RenewPresenceScript renews the presence
//go:embed renew_presence.lua
var RenewPresenceScript string

// GetChannelsScript gets the channels
//go:embed get_channels.lua
var GetChannelsScript string

// updateActiveGameScript gets the channels
//go:embed update_active_game.lua
var UpdateActiveGameScript string

func NewRedisPresenceStore(r *redis.Pool) *RedisPresenceStore {

	return &RedisPresenceStore{
		redisPool:              r,
		setPresenceScript:      redis.NewScript(0, SetPresenceScript),
		clearPresenceScript:    redis.NewScript(0, ClearPresenceScript),
		renewPresenceScript:    redis.NewScript(0, RenewPresenceScript),
		getChannelsScript:      redis.NewScript(0, GetChannelsScript),
		updateActiveGameScript: redis.NewScript(0, UpdateActiveGameScript),
		eventChan:              nil,
	}
}

func fromRedisToArrayOfLengthN(ret interface{}, n int) ([]interface{}, error) {
	arr, ok := ret.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result: %T", ret)
	}
	if len(arr) != n {
		return nil, fmt.Errorf("unexpected length (want %v, got %v): %v", n, len(arr), arr)
	}
	return arr, nil
}

func fromRedisArrayToArrayOfString(v interface{}) ([]string, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected subresult: %T", v)
	}
	cvt := make([]string, len(arr))
	for i, el := range arr {
		barr, ok := el.([]byte)
		if !ok {
			return nil, fmt.Errorf("unexpected interface type: %T", el)
		}
		cvt[i] = string(barr)
	}
	return cvt, nil
}

func (s *RedisPresenceStore) getChannelsForUUID(conn redis.Conn, uuid string) ([]string, error) {
	ret, err := s.getChannelsScript.Do(conn, uuid)
	if err != nil {
		return nil, err
	}
	arr, err := fromRedisToArrayOfLengthN(ret, 1)
	if err != nil {
		return nil, err
	}

	channels, err := fromRedisArrayToArrayOfString(arr[0])
	if err != nil {
		return nil, err
	}

	return channels, nil
}

// SetPresence sets the user's presence channel.
func (s *RedisPresenceStore) SetPresence(ctx context.Context, uuid, username string, anon bool,
	channel, connID string) ([]string, []string, error) {
	// We try to map channels closely to the pubsub NATS channels (and realms),
	// with some exceptions.
	// If the user is online in two different tabs, we go in priority order,
	// as we only want to show them in one place.
	// Priority (from lowest to highest):
	// 	- lobby - The "base" channel.
	//  - usertv.<user_id> - Following a user's games
	//  - gametv.<game_id> - Watching a game
	//  - game.<game_id> - Playing in a game

	conn := s.redisPool.Get()
	defer conn.Close()
	log.Debug().Str("username", username).Str("connID", connID).Msg("set-presence")

	authUser := "auth"
	if anon {
		authUser = "anon"
	}

	ts := time.Now().Unix()
	ret, err := s.setPresenceScript.Do(conn, uuid, username, authUser, connID, channel, ts)
	if err != nil {
		return nil, nil, err
	}
	arr, err := fromRedisToArrayOfLengthN(ret, 2)
	if err != nil {
		return nil, nil, err
	}

	oldChannels, err := fromRedisArrayToArrayOfString(arr[0])
	if err != nil {
		return nil, nil, err
	}
	newChannels, err := fromRedisArrayToArrayOfString(arr[1])
	if err != nil {
		return nil, nil, err
	}

	return oldChannels, newChannels, nil
}

func (s *RedisPresenceStore) ClearPresence(ctx context.Context, uuid, username string,
	anon bool, connID string) ([]string, []string, []string, error) {
	conn := s.redisPool.Get()
	defer conn.Close()

	authUser := "auth"
	if anon {
		authUser = "anon"
	}

	log.Debug().Str("username", username).Str("connID", connID).Msg("clear-presence")

	ts := time.Now().Unix()
	ret, err := s.clearPresenceScript.Do(conn, uuid, username, authUser, connID, ts)
	if err != nil {
		return nil, nil, nil, err
	}
	arr, err := fromRedisToArrayOfLengthN(ret, 3)
	if err != nil {
		return nil, nil, nil, err
	}

	oldChannels, err := fromRedisArrayToArrayOfString(arr[0])
	if err != nil {
		return nil, nil, nil, err
	}
	newChannels, err := fromRedisArrayToArrayOfString(arr[1])
	if err != nil {
		return nil, nil, nil, err
	}
	removedChannels, err := fromRedisArrayToArrayOfString(arr[2])
	if err != nil {
		return nil, nil, nil, err
	}

	return oldChannels, newChannels, removedChannels, nil
}

func (s *RedisPresenceStore) GetInChannel(ctx context.Context, channel string) ([]*entity.User, error) {

	conn := s.redisPool.Get()
	defer conn.Close()

	key := "channelpresence:" + channel

	vals, err := redis.Strings(conn.Do("ZRANGE", key, 0, -1))
	if err != nil {
		return nil, err
	}
	users := make([]*entity.User, len(vals))

	for idx, member := range vals {
		splitmember := strings.Split(member, "#")
		anon := false
		if len(splitmember) > 2 {
			anon = splitmember[2] == "anon"
		}

		users[idx] = &entity.User{
			UUID:      splitmember[0],
			Username:  splitmember[1],
			Anonymous: anon,
		}
	}

	return users, nil
}

// Get the current channels the given user is in. Return empty for no channel.
func (s *RedisPresenceStore) GetPresence(ctx context.Context, uuid string) ([]string, error) {
	conn := s.redisPool.Get()
	defer conn.Close()

	key := "userpresence:" + uuid

	m, err := redis.Strings(conn.Do("ZRANGE", key, 0, -1))
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, nil
	}

	ret := make([]string, len(m))
	for i, el := range m {
		// Format should be: conn_id#channel
		p := strings.Split(el, "#")
		if len(p) != 2 {
			return nil, fmt.Errorf("unexpected presence member: %v (%v)", el, m)
		}
		ret[i] = p[1]
	}
	return ret, nil
}

func (s *RedisPresenceStore) RenewPresence(ctx context.Context, userID, username string,
	anon bool, connID string) ([]string, []string, error) {

	conn := s.redisPool.Get()
	defer conn.Close()

	log.Debug().Str("username", username).Str("connID", connID).Msg("renew-presence")

	authUser := "auth"
	if anon {
		authUser = "anon"
	}

	ts := time.Now().Unix()
	ret, err := s.renewPresenceScript.Do(conn, userID, username, authUser, connID, ts)
	if err != nil {
		return nil, nil, err
	}
	arr, err := fromRedisToArrayOfLengthN(ret, 2)
	if err != nil {
		return nil, nil, err
	}

	oldChannels, err := fromRedisArrayToArrayOfString(arr[0])
	if err != nil {
		return nil, nil, err
	}
	newChannels, err := fromRedisArrayToArrayOfString(arr[1])
	if err != nil {
		return nil, nil, err
	}

	return oldChannels, newChannels, nil
}

func (s *RedisPresenceStore) CountInChannel(ctx context.Context, channel string) (int, error) {

	conn := s.redisPool.Get()
	defer conn.Close()

	key := "channelpresence:" + channel

	val, err := redis.Int(conn.Do("ZCARD", key))
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (s *RedisPresenceStore) BatchGetPresence(ctx context.Context, users []*entity.User) ([]*entity.User, error) {
	return nil, nil
}

func (s *RedisPresenceStore) LastSeen(ctx context.Context, uuid string) (int64, error) {

	conn := s.redisPool.Get()
	defer conn.Close()

	key := "lastpresences"

	val, err := redis.Int(conn.Do("ZSCORE", key, uuid))
	if err != nil {
		return 0, err
	}
	return int64(val), nil
}

func (s *RedisPresenceStore) SetEventChan(c chan *entity.EventWrapper) {
	s.eventChan = c
}

func (s *RedisPresenceStore) EventChan() chan *entity.EventWrapper {
	return s.eventChan
}

func (s *RedisPresenceStore) BatchGetChannels(ctx context.Context, uuids []string) ([][]string, error) {
	conn := s.redisPool.Get()
	defer conn.Close()

	ret := make([][]string, 0, len(uuids))
	for _, uuid := range uuids {
		ret1, err := s.getChannelsForUUID(conn, uuid)
		if err != nil {
			return nil, err
		}
		ret = append(ret, ret1)
	}

	return ret, nil
}

func (s *RedisPresenceStore) UpdateFollower(ctx context.Context, followee, follower *entity.User, following bool) error {
	var channels []string

	if following {
		var err error
		func() {
			conn := s.redisPool.Get()
			defer conn.Close()

			channels, err = s.getChannelsForUUID(conn, followee.UUID)
		}()
		if err != nil {
			return err
		}
	}

	// when following, send the (possibly empty) current list of channels.
	// when unfollowing, always send [].

	evtChan := s.EventChan()
	if evtChan != nil {
		wrapped := entity.WrapEvent(&pb.PresenceEntry{
			Username: followee.Username,
			UserId:   followee.UUID,
			Channel:  channels,
		}, pb.MessageType_PRESENCE_ENTRY)
		wrapped.AddAudience(entity.AudUser, follower.UUID)
		evtChan <- wrapped
	}

	return nil
}

func (s *RedisPresenceStore) UpdateActiveGame(ctx context.Context, activeGameEntry *pb.ActiveGameEntry) ([][][]string, error) {
	args := make([]interface{}, 0, 2+len(activeGameEntry.Player))
	args = append(args, activeGameEntry.Id)
	args = append(args, activeGameEntry.Ttl)
	for _, player := range activeGameEntry.Player {
		args = append(args, player.UserId)
	}

	conn := s.redisPool.Get()
	defer conn.Close()

	ret, err := s.updateActiveGameScript.Do(conn, args...)
	if err != nil {
		return nil, err
	}

	// Script returns an array of [before, after] channels, both are sorted unique arrays.
	arr, err := fromRedisToArrayOfLengthN(ret, len(activeGameEntry.Player))
	if err != nil {
		return nil, err
	}
	channels := make([][][]string, 0, len(arr))
	for _, redisElt := range arr {
		row, err := fromRedisToArrayOfLengthN(redisElt, 2)
		if err != nil {
			return nil, err
		}
		oldChannels, err := fromRedisArrayToArrayOfString(row[0])
		if err != nil {
			return nil, err
		}
		newChannels, err := fromRedisArrayToArrayOfString(row[1])
		if err != nil {
			return nil, err
		}
		channels = append(channels, [][]string{oldChannels, newChannels})
	}

	return channels, nil
}
