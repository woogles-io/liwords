package bus

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/matryer/is"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
	presence "github.com/woogles-io/liwords/pkg/stores/redis"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// presenceBus is a bus with real stores, presence in Redis database 14 and,
// when one is running locally (CI has none), a real NATS server, whose
// messages on subjects come back on the returned channel.
func presenceBus(t *testing.T, subjects ...string) (*Bus, *stores.Stores, redis.Conn, chan *nats.Msg) {
	t.Helper()
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set")
	}
	nc, err := nats.Connect(nats.DefaultURL, nats.Timeout(time.Second))
	if err != nil {
		t.Skip("no local NATS server:", err)
	}
	t.Cleanup(nc.Close)
	msgs := make(chan *nats.Msg, 64)
	for _, s := range subjects {
		sub, err := nc.ChanSubscribe(s, msgs)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { sub.Unsubscribe() })
	}
	if err := nc.Flush(); err != nil {
		t.Fatal(err)
	}

	pool := &redis.Pool{Dial: func() (redis.Conn, error) {
		c, err := redis.DialURL(url)
		if err == nil {
			_, err = c.Do("SELECT", 14)
		}
		return c, err
	}}
	conn := pool.Get()
	t.Cleanup(func() { conn.Do("FLUSHDB"); conn.Close() })
	if _, err := conn.Do("FLUSHDB"); err != nil {
		t.Fatal(err)
	}

	st := recreateInternalTestDB()
	t.Cleanup(st.DBPool.Close)
	st.PresenceStore = presence.NewRedisPresenceStore(pool)
	return &Bus{stores: st, natsconn: nc}, st, conn, msgs
}

// drain returns the messages that arrive within a short wait, by subject.
func drain(msgs chan *nats.Msg) map[string][]*nats.Msg {
	got := map[string][]*nats.Msg{}
	timeout := time.After(300 * time.Millisecond)
	for {
		select {
		case m := <-msgs:
			got[m.Subject] = append(got[m.Subject], m)
		case <-timeout:
			return got
		}
	}
}

func presenceEntry(t *testing.T, m *nats.Msg) *pb.PresenceEntry {
	t.Helper()
	pe := &pb.PresenceEntry{}
	if err := proto.Unmarshal(m.Data, pe); err != nil {
		t.Fatal(err)
	}
	return pe
}

// The active-game entries are made to match the started real-time games:
// missing ones added, those of ended games removed, and entries written
// since the games were read left alone. Followers hear of each change.
func TestReconcileActiveGames(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	rec := recordSpans(t)
	b, st, conn, msgs := presenceBus(t, "presence.changed.>")

	ids := map[string]string{} // username to uuid
	for _, name := range []string{"pres1", "pres2", "pres3"} {
		u := &entity.User{Username: name, Email: name + "@example.com", UUID: "uuid-" + name}
		is.NoErr(st.UserStore.New(ctx, u))
		ids[name] = u.UUID
	}
	ongoing := func(gid string, mode int, started bool, p0, p1 string) {
		_, err := st.DBPool.Exec(ctx, `INSERT INTO ongoing_games
			(game_uuid, lexicon, letter_distribution, board_layout, variant, timers,
			 game_mode, started, player0_id, player1_id)
			VALUES ($1, 'NWL23', 'english', 'CrosswordGame', 'classic', '{}', $2, $3,
			 (SELECT id FROM users WHERE uuid = $4), (SELECT id FROM users WHERE uuid = $5))`,
			gid, mode, started, p0, p1)
		is.NoErr(err)
	}
	ongoing("live", 0, true, ids["pres1"], ids["pres2"])
	ongoing("correspondence", 1, true, ids["pres1"], ids["pres3"])
	ongoing("notstarted", 0, false, ids["pres2"], ids["pres3"])

	ttl := int64(entity.ActiveGameTTL / time.Second)
	now := time.Now().Unix()
	// An ended game's entry, written two hours ago; and a game that started
	// just now, after the games were read.
	_, err := conn.Do("ZADD", "activeusergames:"+ids["pres3"],
		now+ttl-2*3600, "ended", now+ttl+60, "juststarted")
	is.NoErr(err)

	is.NoErr(b.ReconcileActiveGames(ctx))

	games, err := st.PresenceStore.ListActiveGames(ctx)
	is.NoErr(err)
	var got []string
	for _, g := range games {
		got = append(got, g.UserID+":"+g.GameID)
	}
	sort.Strings(got)
	is.Equal(got, []string{
		"uuid-pres1:live", "uuid-pres2:live", "uuid-pres3:juststarted"})

	span := endedSpan(is, rec, "presence.reconcile_active_games")
	for k, want := range map[string]int64{"presence.games": 1, "presence.entries": 2,
		"presence.added": 2, "presence.removed": 1} {
		v, ok := attr(span, k)
		is.True(ok)
		is.Equal(v.AsInt64(), want)
	}

	sent := drain(msgs)
	is.Equal(len(sent), 3)
	is.Equal(presenceEntry(t, sent["presence.changed.uuid-pres1"][0]).Channel, []string{"activegame:live"})
	is.Equal(presenceEntry(t, sent["presence.changed.uuid-pres3"][0]).Channel, []string{"activegame:juststarted"})

	// Nothing left to change.
	is.NoErr(b.ReconcileActiveGames(ctx))
	is.Equal(len(drain(msgs)), 0)
}

// The sweep removes connections that went away without clearing their
// presence, says they left the chat they were in, and tells followers.
func TestSweepPresenceAnnounces(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	rec := recordSpans(t)
	b, _, conn, msgs := presenceBus(t, "chat.>", "presence.changed.>")

	expired := time.Now().Unix() - 10
	for _, kv := range [][2]string{
		{"channelpresence:chat.lobby", "uuid-z#zed#auth#gone"},
		{"userpresence:uuid-z", "gone#chat.lobby"},
		{"userpresences", "uuid-z#zed#auth#gone#chat.lobby"},
	} {
		_, err := conn.Do("ZADD", kv[0], expired, kv[1])
		is.NoErr(err)
	}

	is.NoErr(b.SweepPresence(ctx))

	span := endedSpan(is, rec, "presence.sweep")
	for k, want := range map[string]int64{"presence.swept": 1, "presence.users": 1,
		"presence.left_notices": 1, "presence.batches": 1} {
		v, ok := attr(span, k)
		is.True(ok)
		is.Equal(v.AsInt64(), want)
	}
	// The store's own span is under it.
	store := endedSpan(is, rec, "presence.Sweep")
	is.Equal(store.Parent().SpanID(), span.SpanContext().SpanID())

	sent := drain(msgs)
	is.Equal(len(sent["chat.lobby"]), 1)
	// A short frame: two bytes of length, the type, then the message.
	data := sent["chat.lobby"][0].Data
	is.Equal(data[2], byte(pb.MessageType_USER_PRESENCE))
	left := &pb.UserPresence{}
	is.NoErr(proto.Unmarshal(data[3:], left))
	is.Equal(left.Username, "zed")
	is.True(left.Deleting)
	is.Equal(len(sent["presence.changed.uuid-z"]), 1)
	is.Equal(presenceEntry(t, sent["presence.changed.uuid-z"][0]).Channel, []string(nil))

	for _, k := range []string{"channelpresence:chat.lobby", "userpresence:uuid-z", "userpresences"} {
		n, err := redis.Int(conn.Do("ZCARD", k))
		is.NoErr(err)
		is.Equal(n, 0)
	}
}
