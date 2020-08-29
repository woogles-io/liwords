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
)

var RedisURL = os.Getenv("REDIS_URL")

func newPool(addr string) *redis.Pool {
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

	err := ps.SetPresence(ctx, "uuid1", "cesar", "lobby")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid2", "mina", "lobby")
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

	err := ps.SetPresence(ctx, "uuid1", "cesar", "lobby")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid2", "mina", "lobby")
	is.NoErr(err)

	val, err := ps.ClearPresence(ctx, "uuid1", "cesar")
	is.NoErr(err)
	is.Equal(val, "lobby")

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

	err := ps.SetPresence(ctx, "uuid1", "cesar", "lobby")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid2", "mina", "lobby")
	is.NoErr(err)

	channel, err := ps.GetPresence(ctx, "uuid2")
	is.NoErr(err)
	is.Equal(channel, "lobby")

	channel, err = ps.GetPresence(ctx, "uuid3")
	is.NoErr(err)
	is.Equal(channel, "")
}

func TestGetInChannel(t *testing.T) {
	is := is.New(t)
	redisPool := newPool(RedisURL)
	ps := NewRedisPresenceStore(redisPool)
	flushTestDB(redisPool)

	ctx := context.Background()

	err := ps.SetPresence(ctx, "uuid1", "cesar", "lobby")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid2", "mina", "lobby")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid5", "jesse", "lobby")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid6", "conrad", "lobby")
	is.NoErr(err)

	_, err = ps.ClearPresence(ctx, "uuid1", "cesar")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid1", "cesar", "gametv:abc")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid3", "josh", "game:abc")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid4", "lola", "game:abc")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid7", "puneet", "lobby")
	is.NoErr(err)

	err = ps.SetPresence(ctx, "uuid2", "mina", "gametv:abc")
	is.NoErr(err)

	ct, err := ps.CountInChannel(ctx, "game:abc")
	is.NoErr(err)
	is.Equal(ct, 2)

	ct, err = ps.CountInChannel(ctx, "gametv:abc")
	is.NoErr(err)
	is.Equal(ct, 2)

	ct, err = ps.CountInChannel(ctx, "lobby")
	is.NoErr(err)
	is.Equal(ct, 3)

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
