package redis

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/entity"
)

var RedisURL = os.Getenv("REDIS_URL")

// Make sure this doesn't interfere with other test databases
// (for example, the ones used in omgwords/stores/gamedocument_test.go
var RedisPresenceTestDB = 15

func newPool(addr string) *redis.Pool {
	log.Info().Str("addr", addr).Msg("new-redis-pool")
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(addr)
			if err != nil {
				return nil, err
			}
			_, err = c.Do("SELECT", RedisPresenceTestDB)
			if err != nil {
				c.Close()
				return nil, err
			}
			return c, nil
		},
	}
}

func flushTestDB(r *redis.Pool) {
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

	_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "lobby", "connx1")
	is.NoErr(err)

	_, _, err = ps.SetPresence(ctx, "uuid2", "mina", false, "lobby", "connx2")
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

	_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "lobby", "connx1")
	is.NoErr(err)

	_, _, err = ps.SetPresence(ctx, "uuid2", "mina", false, "lobby", "connx2")
	is.NoErr(err)

	_, _, val, err := ps.ClearPresence(ctx, "uuid1", "cesar", false, "connx1")
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

	_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "lobby", "connx1")
	is.NoErr(err)

	_, _, err = ps.SetPresence(ctx, "uuid2", "mina", false, "lobby", "connx2")
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

	_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "lobby", "connx1")
	is.NoErr(err)

	_, _, err = ps.SetPresence(ctx, "uuid2", "mina", false, "lobby", "connx2")
	is.NoErr(err)

	_, _, err = ps.SetPresence(ctx, "uuid5", "jesse", false, "lobby", "connx5")
	is.NoErr(err)

	_, _, err = ps.SetPresence(ctx, "uuid6", "conrad", false, "lobby", "connx6")
	is.NoErr(err)

	_, _, _, err = ps.ClearPresence(ctx, "uuid1", "cesar", false, "connx1")
	is.NoErr(err)

	_, _, err = ps.SetPresence(ctx, "uuid1", "cesar", false, "gametv:abc", "connx11")
	is.NoErr(err)

	_, _, err = ps.SetPresence(ctx, "uuid3", "josh", false, "game:abc", "connx3")
	is.NoErr(err)

	_, _, err = ps.SetPresence(ctx, "uuid4", "lola", false, "game:abc", "connx4")
	is.NoErr(err)

	_, _, err = ps.SetPresence(ctx, "uuid7", "puneet", false, "lobby", "connx7")
	is.NoErr(err)

	_, _, err = ps.SetPresence(ctx, "uuid2", "mina", false, "gametv:abc", "connx22")
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
	is.NoErr(err)
	sort.Slice(users, func(a, b int) bool {
		return users[a].UUID < users[b].UUID
	})
	is.Equal(users, []*entity.User{
		{Username: "josh", UUID: "uuid3"},
		{Username: "lola", UUID: "uuid4"},
	})

	users, err = ps.GetInChannel(ctx, "lobby")
	is.NoErr(err)
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
	is.NoErr(err)
	sort.Slice(users, func(a, b int) bool {
		return users[a].UUID < users[b].UUID
	})
	is.Equal(users, []*entity.User{
		{Username: "cesar", UUID: "uuid1"},
		{Username: "mina", UUID: "uuid2"},
	})

	users, err = ps.GetInChannel(ctx, "gametv:abcd")
	is.NoErr(err)
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

// 	_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "tournament.abc", "connx1")
// 	is.NoErr(err)

// 	_, _, err = ps.SetPresence(ctx, "uuid2", "mina", false, "lobby", "connx2")
// 	is.NoErr(err)
// 	// cesar is in the tournament abc and in a game with the same conn id.
// 	_, _, err = ps.SetPresence(ctx, "uuid1", "cesar", false, "game.bar", "connx1")
// 	is.NoErr(err)

// 	_, _, err = ps.RenewPresence(ctx, "uuid1", "cesar", false, "connx1")

// 	// Pretend that mina's connection died a few minutes ago.
// 	ts := time.Now().UTC().Unix() + 300 // 5 minutes (expiry is only 3 minutes)

// 	conn := redisPool.Get()
// 	defer conn.Close()

// 	// Renew just cesar's presence
// 	purged, err := ps.renewPresenceScript.Do(conn, "uuid1", "cesar", "auth", "connx1", ts)

// }

// A follow or unfollow tells the follower, and the socket servers, which keep
// whom each connected user follows.
func TestUpdateFollower(t *testing.T) {
	is := is.New(t)
	redisPool := newPool(RedisURL)
	ps := NewRedisPresenceStore(redisPool)
	flushTestDB(redisPool)
	evts := make(chan *entity.EventWrapper, 2)
	ps.SetEventChan(evts)

	ctx := context.Background()
	followee := &entity.User{UUID: "uuid1", Username: "cesar"}
	follower := &entity.User{UUID: "uuid2", Username: "mina"}

	is.NoErr(ps.UpdateFollower(ctx, followee, follower, true))
	is.Equal((<-evts).Audience(), []string{"user.uuid2", "follows.added.uuid2.uuid1"})

	is.NoErr(ps.UpdateFollower(ctx, followee, follower, false))
	is.Equal((<-evts).Audience(), []string{"user.uuid2", "follows.removed.uuid2.uuid1"})
}

// Some clients' connection IDs have hyphens. Their presence has to clear
// like anyone's: entries the scripts couldn't parse were never renewed or
// removed, and piled up.
func TestHyphenatedConnID(t *testing.T) {
	is := is.New(t)
	redisPool := newPool(RedisURL)
	ps := NewRedisPresenceStore(redisPool)
	flushTestDB(redisPool)
	ctx := context.Background()
	conn := "123456789-1234567890-1234"

	_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.game.abc", conn)
	is.NoErr(err)
	channels, err := ps.GetPresence(ctx, "uuid1")
	is.NoErr(err)
	is.Equal(channels, []string{"chat.game.abc"})

	_, _, err = ps.RenewPresence(ctx, "uuid1", "cesar", false, conn)
	is.NoErr(err)

	_, _, left, err := ps.ClearPresence(ctx, "uuid1", "cesar", false, conn)
	is.NoErr(err)
	is.Equal(left, []string{"chat.game.abc"})
	ct, err := ps.CountInChannel(ctx, "chat.game.abc")
	is.NoErr(err)
	is.Equal(ct, 0)
}

// Entries that expired aren't counted or listed, and the next presence set
// in their channel removes them.
func TestExpiredPresenceDropped(t *testing.T) {
	is := is.New(t)
	redisPool := newPool(RedisURL)
	ps := NewRedisPresenceStore(redisPool)
	flushTestDB(redisPool)
	ctx := context.Background()
	c := redisPool.Get()
	defer c.Close()
	_, err := c.Do("ZADD", "channelpresence:lobby", time.Now().Unix()-1000, "uuid9#zed#auth#gone")
	is.NoErr(err)

	ct, err := ps.CountInChannel(ctx, "lobby")
	is.NoErr(err)
	is.Equal(ct, 0)
	users, err := ps.GetInChannel(ctx, "lobby")
	is.NoErr(err)
	is.Equal(len(users), 0)

	_, _, err = ps.SetPresence(ctx, "uuid1", "cesar", false, "lobby", "connx1")
	is.NoErr(err)
	n, err := redis.Int(c.Do("ZCARD", "channelpresence:lobby"))
	is.NoErr(err)
	is.Equal(n, 1)
}
