package user

import (
	"context"
	_ "embed"
	"fmt"
	"sort"
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

	setPresenceScript   *redis.Script
	clearPresenceScript *redis.Script
	renewPresenceScript *redis.Script

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

func NewRedisPresenceStore(r *redis.Pool) *RedisPresenceStore {

	return &RedisPresenceStore{
		redisPool:           r,
		setPresenceScript:   redis.NewScript(0, SetPresenceScript),
		clearPresenceScript: redis.NewScript(0, ClearPresenceScript),
		renewPresenceScript: redis.NewScript(0, RenewPresenceScript),
		eventChan:           nil,
	}
}

// nextToken("abc#def#ghi") = ("abc", "def#ghi")
// nextToken("ghi") = ("ghi", "")
func nextToken(s string) (string, string) {
	// Use this instead of strings.Split to reduce garbage.
	p := strings.IndexByte(s, '#')
	if p >= 0 {
		return s[:p], s[p+1:]
	} else {
		return s, s[len(s):]
	}
}

func sortDedupStrings(parr *[]string) {
	if len(*parr) > 1 {
		sort.Strings(*parr)
		w := 1
		for r := 1; r < len(*parr); r++ {
			if (*parr)[r] != (*parr)[r-1] {
				(*parr)[w] = (*parr)[r]
				w++
			}
		}
		*parr = (*parr)[:w]
	}
}

func getChannelsForUUID(conn redis.Conn, uuid string) ([]string, error) {
	key := "userpresence:" + uuid

	m, err := redis.Strings(conn.Do("ZRANGE", key, 0, -1))
	if err != nil {
		return nil, err
	}

	channels := make([]string, 0, len(m))
	for _, el := range m {
		st := el
		connID, st := nextToken(st)
		channel, _ := nextToken(st)
		if len(connID)+1+len(channel) != len(el) {
			return nil, fmt.Errorf("unexpected presence member for %v: %v (%v)", uuid, el, m)
		}
		channels = append(channels, channel)
	}
	sortDedupStrings(&channels)

	return channels, nil
}

func getChannels(conn redis.Conn, uuid string, anon bool) ([]string, error) {
	if anon {
		return nil, nil
	}
	return getChannelsForUUID(conn, uuid)
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

	oldChannels, err := getChannels(conn, uuid, anon)
	if err != nil {
		return nil, nil, err
	}

	ts := time.Now().Unix()
	_, err = s.setPresenceScript.Do(conn, uuid, username, authUser, connID, channel, ts)
	if err != nil {
		return nil, nil, err
	}

	newChannels, err := getChannels(conn, uuid, anon)
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

	oldChannels, err := getChannels(conn, uuid, anon)
	if err != nil {
		return nil, nil, nil, err
	}

	ts := time.Now().Unix()
	ret, err := s.clearPresenceScript.Do(conn, uuid, username, authUser, connID, ts)
	if err != nil {
		return nil, nil, nil, err
	}
	log.Debug().Interface("ret", ret).Msg("clear-presence-return")

	newChannels, err := getChannels(conn, uuid, anon)
	if err != nil {
		return nil, nil, nil, err
	}

	switch v := ret.(type) {
	case []interface{}:
		cvt := make([]string, len(v))
		for i, el := range v {
			barr, ok := el.([]byte)
			if !ok {
				return nil, nil, nil, fmt.Errorf("unexpected interface type: %T", el)
			}
			// This is just the channel now. No more conn_id#channel.
			// We only need an array of channels.
			cvt[i] = string(barr)
		}
		return oldChannels, newChannels, cvt, nil
	}
	return nil, nil, nil, fmt.Errorf("unexpected clear presence result: %T", ret)
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

	oldChannels, err := getChannels(conn, userID, anon)
	if err != nil {
		return nil, nil, err
	}

	ts := time.Now().Unix()
	_, err = s.renewPresenceScript.Do(conn, userID, username, authUser, connID, ts)
	if err != nil {
		return nil, nil, err
	}

	newChannels, err := getChannels(conn, userID, anon)
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
		ret1, err := getChannelsForUUID(conn, uuid)
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

			channels, err = getChannelsForUUID(conn, followee.UUID)
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
