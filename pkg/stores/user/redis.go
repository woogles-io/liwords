package user

import (
	"context"
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
const SetPresenceScript = `
-- Arguments to this Lua script:
-- uuid, username, authOrAnon, connID, channel string, timestamp  (ARGV[1] through [6])

local userpresencekey = "userpresence:"..ARGV[1]
local channelpresencekey = "channelpresence:"..ARGV[5]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4] -- uuid#username#auth#connID
-- 3 minutes. We will renew these keys constantly.
local expiry = 180
local ts = tonumber(ARGV[6])
local simpleuserkey = ARGV[4].."#"..ARGV[5] -- just conn_id#channel
-- Set user presence:
redis.call("ZADD", userpresencekey, ts + expiry, simpleuserkey)
redis.call("ZADD", "userpresences", ts + expiry, userkey.."#"..ARGV[5])

-- Set channel presence:
redis.call("ZADD", channelpresencekey, ts + expiry, userkey)
-- Expire ephemeral presence keys:

redis.call("EXPIRE", userpresencekey, expiry)
redis.call("EXPIRE", channelpresencekey, expiry)
`

// ClearPresenceScript clears the presence and returns the channel(s) it was in.
const ClearPresenceScript = `
-- Arguments to this Lua script:
-- uuid, username, authOrAnon, connID

local userpresencekey = "userpresence:"..ARGV[1]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4]  -- uuid#username#anon#conn_id

-- get the current channels that this presence is in.

local curchannels = redis.call("ZRANGE", userpresencekey, 0, -1)

local deletedfrom = {}
local deletedcount = 0
local totalcount = 0

-- only delete the users where the conn_id actually matches
for i, v in ipairs(curchannels) do
	-- v looks like conn_id#channel
	local chan = string.match(v, ARGV[4].."#([%a%.%d]+)")
	if chan then
		table.insert(deletedfrom, v)
		-- delete from the relevant channel key
		redis.call("ZREM", "channelpresence:"..chan, userkey)
		redis.call("ZREM", userpresencekey, v)
		redis.call("ZREM", "userpresences", userkey.."#"..chan)
	end
end

-- return the channel(s) where this user connection used to be.
return deletedfrom
`

const RenewPresenceScript = `
-- Arguments to this Lua script:
-- uuid, username, auth, connID, ts (ARGV[1] through [5])
local userpresencekey = "userpresence:"..ARGV[1]
local userkey = ARGV[1].."#"..ARGV[2].."#"..ARGV[3].."#"..ARGV[4] -- uuid#username#auth#connID

local expiry = 180
local ts = tonumber(ARGV[5])

local purgeold = {}

-- Get all members of userpresencekey
local curchannels = redis.call("ZRANGE", userpresencekey, 0, -1)

-- For every channel that we are in, we renew that channel, only for this conn id.
for i, v in ipairs(curchannels) do
	-- v looks like conn_id#channel
	local chan = string.match(v, ARGV[4].."#([%a%.%d]+)")
	if chan then
		-- extend expiries of the channelpresence...
		redis.call("ZADD", "channelpresence:"..chan, ts + expiry, userkey)
		redis.call("EXPIRE", "channelpresence:"..chan, expiry)
		-- and of the userpresence
		redis.call("ZADD", userpresencekey, ts + expiry, v)
		redis.call("EXPIRE", userpresencekey, expiry)
		-- and the overall set of user presences.
		redis.call("ZADD", "userpresences", ts + expiry, userkey.."#"..chan)

		table.insert(purgeold, "channelpresence:"..chan)
		table.insert(purgeold, userpresencekey)
		table.insert(purgeold, "userpresences")
	end
end

-- remove all subkeys inside the zsets that have expired.
for i, v in ipairs(purgeold) do
	redis.call("ZREMRANGEBYSCORE", v, 0, ts)
end

return purgeold
`

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

	ts := time.Now().UTC().Unix()

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
			// The channels look like conn_id#channel for each user.
			// We only need an array of channels.
			sp := strings.Split(string(barr), "#")
			if len(sp) != 2 {
				return nil, fmt.Errorf("unexpected presence format: %v", string(barr))
			}
			cvt[i] = sp[1]
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

	ts := time.Now().UTC().Unix()
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
