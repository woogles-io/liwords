package user

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
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

	cachedPresences atomic.Value
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
	var cachedPresences atomic.Value
	cachedPresences.Store(&user.CachedPresencesType{
		SeenStrings:   make(map[string]string),
		UserPresences: make(map[string]*user.CachedPresenceType),
	})

	return &RedisPresenceStore{
		redisPool:           r,
		setPresenceScript:   redis.NewScript(0, SetPresenceScript),
		clearPresenceScript: redis.NewScript(0, ClearPresenceScript),
		renewPresenceScript: redis.NewScript(0, RenewPresenceScript),
		cachedPresences:     cachedPresences,
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

func (s *RedisPresenceStore) StartCachingPresence(pleaseQuit chan bool) <-chan bool {
	// No mutex. Please run exactly one StartCachingPresence goroutine. I beg/trust you.

	hasQuit := make(chan bool)
	go func() {
	loop:
		for {
			if err := func() error {
				key := "userpresences"
				cursor := int64(0)
				var multiBulk []string

				old := s.cachedPresences.Load().(*user.CachedPresencesType)
				newSeenStrings := make(map[string]string)
				keepString := func(s string) string {
					// Keep one copy of each unique string to reduce garbage.
					if t, ok := newSeenStrings[s]; ok {
						return t
					}
					// Reuse the copy in previous iteration if any.
					if t, ok := old.SeenStrings[s]; ok {
						s = t
					}
					newSeenStrings[s] = s
					return s
				}

				// Get the current state from redis. This will allocate a lot of things.
				newUserPresences := make(map[string]*user.CachedPresenceType)
				conn := s.redisPool.Get()
				defer conn.Close()
				for {
					// Using ZSCAN loop here instead of ZRANGE.
					// ZSCAN does not incur sorting cost.
					// ZSCAN does not block other redis usages.
					// ZSCAN allows filtering away anonymous connections.
					// ZSCAN will not return an unexpected a huge slice.
					vals, err := redis.Values(conn.Do("ZSCAN", key, cursor, "MATCH", "*#auth#*"))
					if err != nil {
						return err
					}
					vals, err = redis.Scan(vals, &cursor, &multiBulk)
					if err != nil {
						return err
					}

					// Save what we need from this batch.
					for i := 0; i < len(multiBulk); i += 2 {
						// [i] is uuid#username#auth#connID#channel
						// [i+1] is ttlEpochSecondsAsString
						s := multiBulk[i]
						uuid, s := nextToken(s)
						username, s := nextToken(s)
						auth, s := nextToken(s)
						if auth != "auth" {
							continue
						}
						_, s = nextToken(s) // connID
						channel, _ := nextToken(s)

						if vp, ok := newUserPresences[uuid]; ok {
							// This appends directly within the map, because vp is a pointer type.
							vp.Channels = append(vp.Channels, keepString(channel))
						} else {
							newUserPresences[keepString(uuid)] = &user.CachedPresenceType{
								Username: keepString(username),
								Channels: []string{keepString(channel)},
							}
						}
					}

					// ZSCAN returns 0 on final scan.
					if cursor == 0 {
						break
					}
				}
				conn.Close() // The defer will still run but redigo no-ops it at the cost of an uncontended mutex.

				// Deduplicate channel names and deduplicate objects against previous iteration.
				userPresencesChanged := len(newUserPresences) != len(old.UserPresences)
				for uuid, vp := range newUserPresences {
					// Multi-tabbing users connect multiple times to the same channel, sort/deduplicate them.
					newChannels := vp.Channels
					if len(newChannels) > 1 {
						sort.Strings(newChannels)
						w := 1
						for r := 1; r < len(newChannels); r++ {
							if newChannels[r] != newChannels[r-1] {
								newChannels[w] = newChannels[r]
								w++
							}
						}
						newChannels = newChannels[:w]
						vp.Channels = newChannels // This may not be necessary because we never append().
					}

					if oldUserPresence, ok := old.UserPresences[uuid]; ok {
						channelsAreUnchanged := false
						oldChannels := oldUserPresence.Channels
					compareChannels:
						for len(newChannels) == len(oldChannels) {
							for ri, r := range newChannels {
								if r != oldChannels[ri] {
									break compareChannels
								}
							}
							newChannels = oldChannels // Reuse the old slice.
							channelsAreUnchanged = true
							break // The for-loop is just an if with a breakable label.
						}

						if channelsAreUnchanged && oldUserPresence.Username == vp.Username {
							// Insert they are the same meme.
							newUserPresences[uuid] = oldUserPresence
						} else {
							vp.Channels = newChannels
							userPresencesChanged = true
						}
					}
				}

				if userPresencesChanged {
					// To reduce garbage, reuse oldSeenStrings if it's the same.
					oldSeenStrings := old.SeenStrings
				compareSeenStrings:
					for len(newSeenStrings) == len(oldSeenStrings) {
						for k := range newSeenStrings {
							if _, ok := oldSeenStrings[k]; !ok {
								break compareSeenStrings
							}
						}
						newSeenStrings = oldSeenStrings
						break // The for-loop is just an if with a breakable label.
					}

					s.cachedPresences.Store(&user.CachedPresencesType{
						SeenStrings:   newSeenStrings,
						UserPresences: newUserPresences,
					})
				}

				return nil
			}(); err != nil {
				log.Err(err).Msg("cache-presence")
			}

			select {
			case <-pleaseQuit:
				break loop
			case <-time.After(5 * time.Second):
			}
		}
		hasQuit <- true
	}()
	return hasQuit
}

// Please treat the returned value as read-only. Golang cannot enforce this, but defensive copying is too expensive.
func (s *RedisPresenceStore) GetCachedPresences() *user.CachedPresencesType {
	return s.cachedPresences.Load().(*user.CachedPresencesType)
}
