package user

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/domino14/liwords/pkg/entity"
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
	}
}

// SetPresence sets the user's presence channel.
func (s *RedisPresenceStore) SetPresence(ctx context.Context, uuid, username string, anon bool,
	channel, connID string) error {
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

	_, err := s.setPresenceScript.Do(conn, uuid, username, authUser, connID, channel, ts)
	return err
}

func (s *RedisPresenceStore) ClearPresence(ctx context.Context, uuid, username string,
	anon bool, connID string) ([]string, error) {
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
		return nil, err
	}
	log.Debug().Interface("ret", ret).Msg("clear-presence-return")

	switch v := ret.(type) {
	case bool:
		// can only be false. User was not found anywhere, so nowhere to leave from.
		if !v {
			return nil, nil
		}
		return nil, errors.New("unexpected bool")
	case []interface{}:
		cvt := make([]string, len(v))
		for i, el := range v {
			barr, ok := el.([]byte)
			if !ok {
				return nil, fmt.Errorf("unexpected interface type: %T", el)
			}
			// This is just the channel now. No more conn_id#channel.
			// We only need an array of channels.
			cvt[i] = string(barr)
		}
		return cvt, nil
	}
	return nil, fmt.Errorf("unexpected clear presence result: %T", ret)
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
	anon bool, connID string) error {

	conn := s.redisPool.Get()
	defer conn.Close()

	log.Debug().Str("username", username).Str("connID", connID).Msg("renew-presence")

	authUser := "auth"
	if anon {
		authUser = "anon"
	}

	ts := time.Now().Unix()
	_, err := s.renewPresenceScript.Do(conn, userID, username, authUser, connID, ts)
	return err
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
