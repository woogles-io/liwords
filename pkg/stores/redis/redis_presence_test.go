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
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
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

	_, _, _, err = ps.RenewPresence(ctx, "uuid1", "cesar", false, conn, nil)
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

// A store whose clock the test moves.
func clockedStore(t *testing.T) (*RedisPresenceStore, *redis.Pool, *time.Time) {
	t.Helper()
	pool := newPool(RedisURL)
	flushTestDB(pool)
	ps := NewRedisPresenceStore(pool)
	now := time.Now()
	ps.now = func() time.Time { return now }
	return ps, pool, &now
}

// Games past their expiry don't count as being played, whoever asks, and
// the key goes with its last game.
func TestExpiredActiveGamesIgnored(t *testing.T) {
	is := is.New(t)
	ps, pool, _ := clockedStore(t)
	ctx := context.Background()
	c := pool.Get()
	defer c.Close()
	_, err := c.Do("ZADD", "activeusergames:uuid1", time.Now().Unix()-10, "oldgame")
	is.NoErr(err)

	chans, err := ps.BatchGetChannels(ctx, []string{"uuid1", "uuid2"})
	is.NoErr(err)
	is.Equal(chans, [][]string{{}, {}})

	ret, err := ps.UpdateActiveGame(ctx, &pb.ActiveGameEntry{Id: "newgame", Ttl: 3600,
		Player: []*pb.ActiveGamePlayer{{UserId: "uuid1", Username: "cesar"}}})
	is.NoErr(err)
	is.Equal(ret[0][0], []string{})
	is.Equal(ret[0][1], []string{"activegame:newgame"})
	games, err := redis.Strings(c.Do("ZRANGE", "activeusergames:uuid1", 0, -1))
	is.NoErr(err)
	is.Equal(games, []string{"newgame"}) // the expired one is gone
	ttl, err := redis.Int(c.Do("TTL", "activeusergames:uuid1"))
	is.NoErr(err)
	is.True(ttl > 3500 && ttl <= 3600)

	_, _, err = ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "conn1")
	is.NoErr(err)
	chans, err = ps.BatchGetChannels(ctx, []string{"uuid1"})
	is.NoErr(err)
	is.Equal(chans, [][]string{{"activegame:newgame", "chat.lobby"}})
}

// A connection that went away without clearing its presence is swept once
// its entries expire, and the user is said to have left only the channels
// no other connection of theirs is in.
func TestSweepPresence(t *testing.T) {
	is := is.New(t)
	ps, _, now := clockedStore(t)
	ctx := context.Background()

	_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "gone")
	is.NoErr(err)
	_, _, err = ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.tournament.t1", "gone")
	is.NoErr(err)
	_, _, err = ps.SetPresence(ctx, "uuid2", "mina", true, "chat.lobby", "gone2")
	is.NoErr(err)

	// Nothing has expired yet.
	n, users, err := ps.SweepPresence(ctx, 100)
	is.NoErr(err)
	is.Equal(n, 0)
	is.Equal(len(users), 0)

	// Two minutes on, cesar opens another tab in the lobby; the first tab
	// stops renewing.
	*now = now.Add(2 * time.Minute)
	_, _, err = ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "here")
	is.NoErr(err)

	*now = now.Add(2 * time.Minute)
	n, users, err = ps.SweepPresence(ctx, 100)
	is.NoErr(err)
	is.Equal(n, 3)
	sort.Slice(users, func(i, j int) bool { return users[i].UUID < users[j].UUID })
	is.Equal(len(users), 2)
	is.Equal(users[0].UUID, "uuid1")
	is.Equal(users[0].Removed, []string{"chat.tournament.t1"}) // still in the lobby
	is.Equal(users[0].Before, []string{"chat.lobby", "chat.tournament.t1"})
	is.Equal(users[0].After, []string{"chat.lobby"})
	is.Equal(users[1].UUID, "uuid2")
	is.True(users[1].Anon)
	is.Equal(users[1].Removed, []string{"chat.lobby"})

	in, err := ps.GetInChannel(ctx, "chat.lobby")
	is.NoErr(err)
	is.Equal(len(in), 1)
	is.Equal(in[0].Username, "cesar")
	ct, err := ps.CountInChannel(ctx, "chat.tournament.t1")
	is.NoErr(err)
	is.Equal(ct, 0)

	// Nothing more to sweep.
	n, _, err = ps.SweepPresence(ctx, 100)
	is.NoErr(err)
	is.Equal(n, 0)
}

// The sweep leaves a connection that renewed in time, and takes expired
// entries a batch at a time.
func TestSweepPresenceRenewedAndBatched(t *testing.T) {
	is := is.New(t)
	ps, _, now := clockedStore(t)
	ctx := context.Background()

	_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "alive")
	is.NoErr(err)
	for _, u := range []string{"a", "b", "c"} {
		_, _, err = ps.SetPresence(ctx, "uuid-"+u, u, false, "chat.lobby", "conn-"+u)
		is.NoErr(err)
	}
	*now = now.Add(2 * time.Minute)
	_, _, _, err = ps.RenewPresence(ctx, "uuid1", "cesar", false, "alive", nil)
	is.NoErr(err)

	*now = now.Add(2 * time.Minute)
	n, _, err := ps.SweepPresence(ctx, 2)
	is.NoErr(err)
	is.Equal(n, 2)
	n, _, err = ps.SweepPresence(ctx, 2)
	is.NoErr(err)
	is.Equal(n, 1)
	in, err := ps.GetInChannel(ctx, "chat.lobby")
	is.NoErr(err)
	is.Equal(len(in), 1)
	is.Equal(in[0].Username, "cesar")
}

// Every live active-game entry is listed, with when it lapses.
func TestListActiveGames(t *testing.T) {
	is := is.New(t)
	ps, pool, _ := clockedStore(t)
	ctx := context.Background()
	c := pool.Get()
	defer c.Close()
	_, err := ps.UpdateActiveGame(ctx, &pb.ActiveGameEntry{Id: "g1", Ttl: 3600,
		Player: []*pb.ActiveGamePlayer{{UserId: "uuid1"}, {UserId: "uuid2"}}})
	is.NoErr(err)
	_, err = c.Do("ZADD", "activeusergames:uuid3", time.Now().Unix()-10, "expired")
	is.NoErr(err)

	// A key from before keys had a TTL, with an expired and a live game.
	_, err = c.Do("ZADD", "activeusergames:uuid4", time.Now().Unix()-10, "old", time.Now().Unix()+600, "g2")
	is.NoErr(err)

	games, err := ps.ListActiveGames(ctx)
	is.NoErr(err)
	sort.Slice(games, func(i, j int) bool { return games[i].UserID < games[j].UserID })
	is.Equal(len(games), 3)
	is.Equal(games[0].UserID, "uuid1")
	is.Equal(games[0].GameID, "g1")
	is.True(time.Until(games[0].Expires) > 59*time.Minute)
	is.Equal(games[1].UserID, "uuid2")
	is.Equal(games[2].GameID, "g2")

	// The expired entries are gone, and so is the key that had only one.
	n, err := redis.Int(c.Do("EXISTS", "activeusergames:uuid3"))
	is.NoErr(err)
	is.Equal(n, 0)
	left, err := redis.Strings(c.Do("ZRANGE", "activeusergames:uuid4", 0, -1))
	is.NoErr(err)
	is.Equal(left, []string{"g2"})
	ttl, err := redis.Int(c.Do("TTL", "activeusergames:uuid4"))
	is.NoErr(err)
	is.True(ttl > 590 && ttl <= 600)
}

// Clearing a connection reports the channels left from live entries only.
func TestClearIgnoresExpiredEntries(t *testing.T) {
	is := is.New(t)
	ps, pool, _ := clockedStore(t)
	ctx := context.Background()
	c := pool.Get()
	defer c.Close()
	_, err := c.Do("ZADD", "userpresence:uuid1", time.Now().Unix()-10, "oldconn#chat.gametv.x")
	is.NoErr(err)
	_, _, err = ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "conn1")
	is.NoErr(err)
	// Setting purged the expired entry; put it back to see clear ignore it.
	_, err = c.Do("ZADD", "userpresence:uuid1", time.Now().Unix()-10, "oldconn#chat.gametv.x")
	is.NoErr(err)

	before, after, removed, err := ps.ClearPresence(ctx, "uuid1", "cesar", false, "conn1")
	is.NoErr(err)
	is.Equal(before, []string{"chat.lobby"})
	is.Equal(after, []string{})
	is.Equal(removed, []string{"chat.lobby"})
}

// Given its channels, a renewal puts back a connection's presence that was
// lost, and says where; without them, it can only extend what's there.
func TestRenewRestoresLostPresence(t *testing.T) {
	is := is.New(t)
	ps, pool, now := clockedStore(t)
	ctx := context.Background()

	_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "conn1")
	is.NoErr(err)
	// Renewing what's there restores nothing.
	before, after, restored, err := ps.RenewPresence(ctx, "uuid1", "cesar", false, "conn1", []string{"chat.lobby"})
	is.NoErr(err)
	is.Equal(before, []string{"chat.lobby"})
	is.Equal(after, []string{"chat.lobby"})
	is.Equal(restored, []string{})

	// Redis loses its data.
	flushTestDB(pool)
	*now = now.Add(time.Minute)

	// An older socket server sends no channels: nothing to renew.
	_, after, restored, err = ps.RenewPresence(ctx, "uuid1", "cesar", false, "conn1", nil)
	is.NoErr(err)
	is.Equal(after, []string{})
	is.Equal(restored, []string{})

	before, after, restored, err = ps.RenewPresence(ctx, "uuid1", "cesar", false, "conn1",
		[]string{"chat.lobby", "chat.tournament.t1"})
	is.NoErr(err)
	is.Equal(before, []string{})
	is.Equal(after, []string{"chat.lobby", "chat.tournament.t1"})
	is.Equal(restored, []string{"chat.lobby", "chat.tournament.t1"})
	in, err := ps.GetInChannel(ctx, "chat.tournament.t1")
	is.NoErr(err)
	is.Equal(len(in), 1)
	is.Equal(in[0].Username, "cesar")

	// The sweep's index has it too: it goes if the tab does.
	*now = now.Add(4 * time.Minute)
	n, users, err := ps.SweepPresence(ctx, 100)
	is.NoErr(err)
	is.Equal(n, 2)
	is.Equal(users[0].Removed, []string{"chat.lobby", "chat.tournament.t1"})
}

// A renewal handled after its tab closed (a pong sent just before the close,
// handled later by another goroutine or server) doesn't put the tab back.
func TestRenewAfterCloseRestoresNothing(t *testing.T) {
	is := is.New(t)
	ps, _, _ := clockedStore(t)
	ctx := context.Background()

	_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "conn1")
	is.NoErr(err)
	_, _, _, err = ps.ClearPresence(ctx, "uuid1", "cesar", false, "conn1")
	is.NoErr(err)

	for _, chans := range [][]string{{"chat.lobby"}, nil} {
		before, after, restored, err := ps.RenewPresence(ctx, "uuid1", "cesar", false, "conn1", chans)
		is.NoErr(err)
		is.Equal(before, []string{})
		is.Equal(after, []string{})
		is.Equal(restored, []string{})
	}
	ct, err := ps.CountInChannel(ctx, "chat.lobby")
	is.NoErr(err)
	is.Equal(ct, 0)

	// Another tab of the same user is unaffected.
	_, _, err = ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "conn2")
	is.NoErr(err)
	_, after, _, err := ps.RenewPresence(ctx, "uuid1", "cesar", false, "conn2", []string{"chat.lobby"})
	is.NoErr(err)
	is.Equal(after, []string{"chat.lobby"})
}

// A refresh: the old tab's socket goes and the new page's socket, with a new
// connection ID, joins the same channel. Whichever comes first, the closed
// tab's tombstone doesn't touch the new one.
func TestRefresh(t *testing.T) {
	ctx := context.Background()

	t.Run("new tab joins before the old one closes", func(t *testing.T) {
		is := is.New(t)
		ps, _, _ := clockedStore(t)
		_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "old")
		is.NoErr(err)
		before, after, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "new")
		is.NoErr(err)
		is.Equal(before, after) // followers hear nothing
		before, after, removed, err := ps.ClearPresence(ctx, "uuid1", "cesar", false, "old")
		is.NoErr(err)
		is.Equal(before, after)
		is.Equal(removed, []string{}) // no leave notice
		_, after, restored, err := ps.RenewPresence(ctx, "uuid1", "cesar", false, "new", []string{"chat.lobby"})
		is.NoErr(err)
		is.Equal(after, []string{"chat.lobby"})
		is.Equal(restored, []string{})
	})

	t.Run("old tab closes before the new one joins", func(t *testing.T) {
		is := is.New(t)
		ps, _, _ := clockedStore(t)
		_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "old")
		is.NoErr(err)
		_, after, removed, err := ps.ClearPresence(ctx, "uuid1", "cesar", false, "old")
		is.NoErr(err)
		is.Equal(after, []string{})
		is.Equal(removed, []string{"chat.lobby"}) // a leave notice, then a join
		before, after, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "new")
		is.NoErr(err)
		is.Equal(before, []string{})
		is.Equal(after, []string{"chat.lobby"})
		// The old tab's last pong, handled late, changes nothing.
		_, after, restored, err := ps.RenewPresence(ctx, "uuid1", "cesar", false, "old", []string{"chat.lobby"})
		is.NoErr(err)
		is.Equal(after, []string{"chat.lobby"})
		is.Equal(restored, []string{})
		in, err := ps.GetInChannel(ctx, "chat.lobby")
		is.NoErr(err)
		is.Equal(len(in), 1)
	})

	t.Run("old socket dies without closing", func(t *testing.T) {
		is := is.New(t)
		ps, _, now := clockedStore(t)
		_, _, err := ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "old")
		is.NoErr(err)
		_, _, err = ps.SetPresence(ctx, "uuid1", "cesar", false, "chat.lobby", "new")
		is.NoErr(err)
		*now = now.Add(2 * time.Minute)
		_, _, _, err = ps.RenewPresence(ctx, "uuid1", "cesar", false, "new", []string{"chat.lobby"})
		is.NoErr(err)
		*now = now.Add(2 * time.Minute)
		n, users, err := ps.SweepPresence(ctx, 100)
		is.NoErr(err)
		is.Equal(n, 1)
		is.Equal(users[0].Removed, []string{}) // still here: no leave notice
		is.Equal(users[0].Before, users[0].After)
	})
}
