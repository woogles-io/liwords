package user

import (
	"context"
	"errors"
	"strings"

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
}

// SetPresenceScript is a Lua script that handles presence in an atomic way.
// We may want to move this to a separate file if we start adding more Lua
// scripts.
const SetPresenceScript = `
-- Arguments to this Lua script:
-- uuid, username, authOrAnon, channel string  (ARGV[1] through [4])

local presencekey = "presence:user:"..ARGV[1]
local channelpresencekey = "presence:channel:"..ARGV[4]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3]  -- uuid#username#auth

-- get the current channel that this presence is in.
local curchannel = redis.call("HGET", presencekey, "channel")

-- compare with false; Lua converts redis nil reply to false
if curchannel ~= false then
    -- the presence is already somewhere else. we must delete it from the right SET
    redis.call("SREM", "presence:channel:"..curchannel, userkey)
end

redis.call("HSET", presencekey, "username", ARGV[2], "channel", ARGV[4])
-- and add to the channel presence
redis.call("SADD", channelpresencekey, userkey)

`

// ClearPresenceScript clears the presence and returns the channel it was in.
const ClearPresenceScript = `
-- Arguments to this Lua script:
-- uuid, username, authOrAnon

local presencekey = "presence:user:"..ARGV[1]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3]  -- uuid#username#anon

-- get the current channel that this presence is in.
local curchannel = redis.call("HGET", presencekey, "channel")
if curchannel ~= false then
    -- figure out what channel we are in
    redis.call("SREM", "presence:channel:"..curchannel, userkey)
end
redis.call("DEL", presencekey)

-- return the channel where the user used to be.
return curchannel
`

func NewRedisPresenceStore(r *redis.Pool) *RedisPresenceStore {

	return &RedisPresenceStore{
		redisPool:           r,
		setPresenceScript:   redis.NewScript(0, SetPresenceScript),
		clearPresenceScript: redis.NewScript(0, ClearPresenceScript),
	}
}

// SetPresence sets the user's presence channel.
func (s *RedisPresenceStore) SetPresence(ctx context.Context, uuid, username string, anon bool,
	channel string) error {
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

	authUser := "auth"
	if anon {
		authUser = "anon"
	}

	_, err := s.setPresenceScript.Do(conn, uuid, username, authUser, channel)
	return err
}

func (s *RedisPresenceStore) ClearPresence(ctx context.Context, uuid, username string,
	anon bool) (string, error) {
	conn := s.redisPool.Get()
	defer conn.Close()

	authUser := "auth"
	if anon {
		authUser = "anon"
	}

	ret, err := s.clearPresenceScript.Do(conn, uuid, username, authUser)
	if err != nil {
		return "", err
	}
	log.Debug().Interface("ret", ret).Msg("clear-presence-return")

	switch v := ret.(type) {
	case bool:
		// can only be false. User was not found anywhere, so nowhere to leave from.
		if !v {
			return "", nil
		}
		return "", errors.New("unexpected bool")
	case []byte:
		return string(v), nil
	}
	return "", errors.New("unexpected clear presence result")
}

func (s *RedisPresenceStore) GetInChannel(ctx context.Context, channel string) ([]*entity.User, error) {

	conn := s.redisPool.Get()
	defer conn.Close()

	key := "presence:channel:" + channel

	vals, err := redis.Strings(conn.Do("SMEMBERS", key))
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

// Get the current channel the given user is in. Return empty for no channel.
func (s *RedisPresenceStore) GetPresence(ctx context.Context, uuid string) (string, error) {
	conn := s.redisPool.Get()
	defer conn.Close()

	key := "presence:user:" + uuid

	m, err := redis.StringMap(conn.Do("HGETALL", key))
	if err != nil {
		return "", err
	}

	if m == nil {
		return "", nil
	}

	return m["channel"], nil
}

func (s *RedisPresenceStore) CountInChannel(ctx context.Context, channel string) (int, error) {

	conn := s.redisPool.Get()
	defer conn.Close()

	key := "presence:channel:" + channel

	val, err := redis.Int(conn.Do("SCARD", key))
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (s *RedisPresenceStore) BatchGetPresence(ctx context.Context, users []*entity.User) ([]*entity.User, error) {
	return nil, nil
}
