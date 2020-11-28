package user

import (
	"context"
	"errors"
	"fmt"
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
-- uuid, username, authOrAnon, connID, channel string  (ARGV[1] through [5])

local userpresencekey = "fullpresence:user:"..ARGV[1]
local channelpresencekey = "fullpresence:channel:"..ARGV[5]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4] -- uuid#username#auth#connID
local expiry = 259200  -- 3 days

-- Add the connection ID to the channel so we can track presences per connected tab
redis.call("SADD", userpresencekey, ARGV[5].."#"..ARGV[4])
-- and add to the channel presence
redis.call("SADD", channelpresencekey, userkey)

redis.call("EXPIRE", userpresencekey, expiry)
redis.call("EXPIRE", channelpresencekey, expiry)

`

// ClearPresenceScript clears the presence and returns the channel(s) it was in.
const ClearPresenceScript = `
-- Arguments to this Lua script:
-- uuid, username, authOrAnon, connID

local userpresencekey = "fullpresence:user:"..ARGV[1]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4]  -- uuid#username#anon#conn_id

-- get the current channels that this presence is in.
local curchannels = redis.call("SMEMBERS", userpresencekey)

local deletedfrom = {}
local deletedcount = 0
local totalcount = 0
-- only delete the users where the conn_id actually matches.
for i,v in ipairs(curchannels) do
	-- v looks like channel#conn_id, but we only want to remove from the channel
	-- redis.log(redis.LOG_WARNING, "v: "..v.." our_conn_id: "..ARGV[4])
	local chan = string.match(v, "([%a%.%d]+)#"..ARGV[4])
	totalcount = totalcount + 1
	if chan then
		table.insert(deletedfrom, v)
		deletedcount = deletedcount + 1
		-- redis.log(redis.LOG_WARNING, "found, deleting")
		redis.call("SREM", "fullpresence:channel:"..chan, userkey)
		redis.call("SREM", userpresencekey, v)
	end
end

-- only delete the user presence key if we actually deleted it from all channels.
-- note: the set should already be missing because the SREM calls above will have
--  emptied it...
if deletedcount > 0 and deletedcount == totalcount then
	redis.call("DEL", userpresencekey)
end

-- return the channel(s) where this user connection used to be.
return deletedfrom
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

	_, err := s.setPresenceScript.Do(conn, uuid, username, authUser, connID, channel)
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

	ret, err := s.clearPresenceScript.Do(conn, uuid, username, authUser, connID)
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
			// The channels look like channel#conn_id for each user.
			// We only need an array of channels.
			sp := strings.Split(string(barr), "#")
			if len(sp) != 2 {
				return nil, fmt.Errorf("unexpected presence format: %v", string(barr))
			}
			cvt[i] = sp[0]
		}
		return cvt, nil
	}
	return nil, fmt.Errorf("unexpected clear presence result: %T", ret)
}

func (s *RedisPresenceStore) GetInChannel(ctx context.Context, channel string) ([]*entity.User, error) {

	conn := s.redisPool.Get()
	defer conn.Close()

	key := "fullpresence:channel:" + channel

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

// Get the current channels the given user is in. Return empty for no channel.
func (s *RedisPresenceStore) GetPresence(ctx context.Context, uuid string) ([]string, error) {
	conn := s.redisPool.Get()
	defer conn.Close()

	key := "fullpresence:user:" + uuid

	m, err := redis.Strings(conn.Do("SMEMBERS", key))
	if err != nil {
		return nil, err
	}

	if m == nil {
		return nil, nil
	}

	ret := make([]string, len(m))
	for i, el := range m {
		p := strings.Split(el, "#")
		if len(p) != 2 {
			return nil, fmt.Errorf("unexpected presence member: %v (%v)", el, m)
		}
		ret[i] = p[0]
	}
	return ret, nil
}

func (s *RedisPresenceStore) CountInChannel(ctx context.Context, channel string) (int, error) {

	conn := s.redisPool.Get()
	defer conn.Close()

	key := "fullpresence:channel:" + channel

	val, err := redis.Int(conn.Do("SCARD", key))
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (s *RedisPresenceStore) BatchGetPresence(ctx context.Context, users []*entity.User) ([]*entity.User, error) {
	return nil, nil
}
