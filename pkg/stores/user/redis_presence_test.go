package user

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/gomodule/redigo/redis"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
)

var RedisURL = os.Getenv("REDIS_URL")

func newPool(addr string) *redis.Pool {
	log.Info().Str("addr", addr).Msg("new-redis-pool")
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) { return redis.DialURL(addr) },
	}
}

func flushTestDB(r *redis.Pool) {
	// flush the test DB (1)
	conn := r.Get()
	defer conn.Close()
	conn.Do("FLUSHDB")
}

// test the Redis presence store.
func TestSetPresence(t *testing.T) {
	is := is.New(t)
	redisPool := newPool(RedisURL)
	ps := NewRedisPresenceStore(redisPool)
	flushTestDB(redisPool)

	ctx := context.Background()

	err := ps.SetPresence(ctx, "uuid1", "cesar", false, "lobby", "connx1")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid2", "mina", false, "lobby", "connx2")
	is.NoErr(err)

	ct, err := ps.CountInChannel(ctx, "lobby")
	is.NoErr(err)
	is.Equal(ct, 2)
}

func TestSetPresenceLeaveAndComeback(t *testing.T) {
	is := is.New(t)
	redisPool := newPool(RedisURL)
	ps := NewRedisPresenceStore(redisPool)
	flushTestDB(redisPool)

	ctx := context.Background()

	err := ps.SetPresence(ctx, "uuid1", "cesar", false, "lobby", "connx1")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid2", "mina", false, "lobby", "connx2")
	is.NoErr(err)

	val, err := ps.ClearPresence(ctx, "uuid1", "cesar", false, "connx1")
	is.NoErr(err)
	is.Equal(val, []string{"lobby"})

	ct, err := ps.CountInChannel(ctx, "lobby")
	is.NoErr(err)
	is.Equal(ct, 1)
}

func TestGetPresence(t *testing.T) {
	is := is.New(t)
	redisPool := newPool(RedisURL)
	ps := NewRedisPresenceStore(redisPool)
	flushTestDB(redisPool)

	ctx := context.Background()

	err := ps.SetPresence(ctx, "uuid1", "cesar", false, "lobby", "connx1")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid2", "mina", false, "lobby", "connx2")
	is.NoErr(err)

	channels, err := ps.GetPresence(ctx, "uuid2")
	is.NoErr(err)
	is.Equal(channels, []string{"lobby"})

	channels, err = ps.GetPresence(ctx, "uuid3")
	is.NoErr(err)
	is.Equal(channels, []string{})
}

func TestGetInChannel(t *testing.T) {
	is := is.New(t)
	redisPool := newPool(RedisURL)
	ps := NewRedisPresenceStore(redisPool)
	flushTestDB(redisPool)

	ctx := context.Background()

	err := ps.SetPresence(ctx, "uuid1", "cesar", false, "lobby", "connx1")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid2", "mina", false, "lobby", "connx2")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid5", "jesse", false, "lobby", "connx5")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid6", "conrad", false, "lobby", "connx6")
	is.NoErr(err)

	_, err = ps.ClearPresence(ctx, "uuid1", "cesar", false, "connx1")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid1", "cesar", false, "gametv:abc", "connx11")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid3", "josh", false, "game:abc", "connx3")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid4", "lola", false, "game:abc", "connx4")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid7", "puneet", false, "lobby", "connx7")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid2", "mina", false, "gametv:abc", "connx22")
	is.NoErr(err)

	ct, err := ps.CountInChannel(ctx, "game:abc")
	is.NoErr(err)
	is.Equal(ct, 2)

	ct, err = ps.CountInChannel(ctx, "gametv:abc")
	is.NoErr(err)
	is.Equal(ct, 2)

	ct, err = ps.CountInChannel(ctx, "lobby")
	is.NoErr(err)
	is.Equal(ct, 4)

	users, err := ps.GetInChannel(ctx, "game:abc")
	sort.Slice(users, func(a, b int) bool {
		return users[a].UUID < users[b].UUID
	})
	is.Equal(users, []*entity.User{
		{Username: "josh", UUID: "uuid3"},
		{Username: "lola", UUID: "uuid4"},
	})

	users, err = ps.GetInChannel(ctx, "lobby")
	sort.Slice(users, func(a, b int) bool {
		return users[a].UUID < users[b].UUID
	})
	is.Equal(users, []*entity.User{
		{Username: "mina", UUID: "uuid2"},
		{Username: "jesse", UUID: "uuid5"},
		{Username: "conrad", UUID: "uuid6"},
		{Username: "puneet", UUID: "uuid7"},
	})

	users, err = ps.GetInChannel(ctx, "gametv:abc")
	sort.Slice(users, func(a, b int) bool {
		return users[a].UUID < users[b].UUID
	})
	is.Equal(users, []*entity.User{
		{Username: "cesar", UUID: "uuid1"},
		{Username: "mina", UUID: "uuid2"},
	})

	users, err = ps.GetInChannel(ctx, "gametv:abcd")
	sort.Slice(users, func(a, b int) bool {
		return users[a].UUID < users[b].UUID
	})
	is.Equal(users, []*entity.User{})

}

// func TestRenewPresence(t *testing.T) {
// 	is := is.New(t)
// 	redisPool := newPool(RedisURL)
// 	ps := NewRedisPresenceStore(redisPool)
// 	flushTestDB(redisPool)

// 	ctx := context.Background()

// 	err := ps.SetPresence(ctx, "uuid1", "cesar", false, "tournament.abc", "connx1")
// 	is.NoErr(err)

// 	err = ps.SetPresence(ctx, "uuid2", "mina", false, "lobby", "connx2")
// 	is.NoErr(err)
// 	// cesar is in the tournament abc and in a game with the same conn id.
// 	err = ps.SetPresence(ctx, "uuid1", "cesar", false, "game.bar", "connx1")
// 	is.NoErr(err)

// 	err = ps.RenewPresence(ctx, "uuid1", "cesar", false, "connx1")

// 	// Pretend that mina's connection died a few minutes ago.
// 	ts := time.Now().UTC().Unix() + 300 // 5 minutes (expiry is only 3 minutes)

// 	conn := redisPool.Get()
// 	defer conn.Close()

// 	// Renew just cesar's presence
// 	purged, err := ps.renewPresenceScript.Do(conn, "uuid1", "cesar", "auth", "connx1", ts)

// }
