package chat

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	redisstore "github.com/woogles-io/liwords/pkg/stores/redis"
)

const pkg = "chatstore"

// redisTestDB is a Redis database no other package's tests use.
const redisTestDB = 14

func setup(t *testing.T) (*pgxpool.Pool, *redis.Pool) {
	t.Helper()
	if err := common.RecreateTestDB(pkg); err != nil {
		t.Fatal(err)
	}
	db, err := common.OpenTestingDB(pkg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	rp := &redis.Pool{Dial: func() (redis.Conn, error) {
		return redis.DialURL(os.Getenv("REDIS_URL"), redis.DialDatabase(redisTestDB))
	}}
	t.Cleanup(func() { rp.Close() })
	conn := rp.Get()
	defer conn.Close()
	if _, err := conn.Do("FLUSHDB"); err != nil {
		t.Fatal(err)
	}
	for _, u := range [][2]string{{"cesar", "uuid-cesar"}, {"mina", "uuid-mina"}} {
		if _, err := db.Exec(context.Background(),
			`INSERT INTO users (username, uuid, email, password, created_at) VALUES ($1::text, $2, $1::text || '@x.test', '', now())`,
			u[0], u[1]); err != nil {
			t.Fatal(err)
		}
	}
	return db, rp
}

type row struct {
	id                         uuid.UUID
	channel, username, message string
	senderID                   *string // the sender's UUID, through sender_id
	createdAt                  time.Time
	deleted                    bool
	deletedBy                  *string
}

// rowFor reads the row of the message with Redis stream ID streamID.
func rowFor(t *testing.T, db *pgxpool.Pool, channel, streamID string) row {
	t.Helper()
	lid, err := parseLegacyID(streamID)
	if err != nil {
		t.Fatal(err)
	}
	var r row
	err = db.QueryRow(context.Background(), `
		SELECT m.id, m.channel, m.sender_username, m.message, s.uuid, m.created_at,
		       m.deleted_at IS NOT NULL, d.uuid
		FROM chat_messages m
		LEFT JOIN users s ON s.id = m.sender_id
		LEFT JOIN users d ON d.id = m.deleted_by
		WHERE m.channel = $1 AND m.id = $2`, channel, lid.messageID(channel)).
		Scan(&r.id, &r.channel, &r.username, &r.message, &r.senderID, &r.createdAt, &r.deleted, &r.deletedBy)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func count(t *testing.T, db *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM chat_messages`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// A message sent through the mirror is in Redis, as before, and copied to
// Postgres with its sender, time and stream ID. A moderator's delete removes
// it from Redis and marks it deleted, by whom, in Postgres.
func TestMirror(t *testing.T) {
	is := is.New(t)
	db, rp := setup(t)
	ctx := context.Background()
	m := NewMirror(redisstore.NewRedisChatStore(rp, nil, nil, nil), models.New(db))

	cm, err := m.AddChat(ctx, "cesar", "uuid-cesar", "hello", "chat.game.g1", "", false)
	is.NoErr(err)
	r := rowFor(t, db, "chat.game.g1", cm.Id)
	is.Equal(r.username, "cesar")
	is.Equal(r.message, "hello")
	is.Equal(*r.senderID, "uuid-cesar")
	is.Equal(r.createdAt.UnixMilli(), cm.Timestamp)
	is.Equal(int(r.id.Version()), 7)
	sec, nsec := r.id.Time().UnixTime()
	is.Equal(sec*1000+nsec/1e6, cm.Timestamp) // the ID carries the message's time
	is.True(!r.deleted)

	// A line the server posts has no sender.
	line, err := m.AddChat(ctx, "", "", "challenged off", "chat.game.g1", "", false)
	is.NoErr(err)
	is.Equal(rowFor(t, db, "chat.game.g1", line.Id).senderID, nil)

	old, err := m.OldChats(ctx, "chat.game.g1", 10) // still read from Redis
	is.NoErr(err)
	is.Equal(len(old), 2)

	is.NoErr(m.DeleteChat(ctx, "chat.game.g1", cm.Id, "uuid-mina"))
	r = rowFor(t, db, "chat.game.g1", cm.Id)
	is.True(r.deleted)
	is.Equal(*r.deletedBy, "uuid-mina")
	old, err = m.OldChats(ctx, "chat.game.g1", 10)
	is.NoErr(err)
	is.Equal(len(old), 1)
}

// The backfill copies what is only in Redis, including messages from before
// user IDs were stored, leaves what the mirror already copied, and copies
// nothing on a second run.
func TestBackfill(t *testing.T) {
	is := is.New(t)
	db, rp := setup(t)
	ctx := context.Background()
	q := models.New(db)
	m := NewMirror(redisstore.NewRedisChatStore(rp, nil, nil, nil), q)

	conn := rp.Get()
	defer conn.Close()
	_, err := conn.Do("XADD", "chat:lobby", "1600000000000-0", "username", "cesar", "message", "old, no id")
	is.NoErr(err)
	_, err = conn.Do("XADD", "chat:pm.uuid-cesar_uuid-mina", "1700000000000-0",
		"username", "mina", "message", "hi cesar", "userID", "uuid-mina")
	is.NoErr(err)
	_, err = conn.Do("SET", "chat:notastream", "x") // skipped
	is.NoErr(err)
	mirrored, err := m.AddChat(ctx, "cesar", "uuid-cesar", "hi mina", "chat.pm.uuid-cesar_uuid-mina", "", false)
	is.NoErr(err)

	st, err := Backfill(ctx, rp, q, 1)
	is.NoErr(err)
	is.Equal(st.Channels, 2)
	is.Equal(st.Messages, 3)
	is.Equal(st.Inserted, int64(2)) // the mirrored one was already there
	is.Equal(count(t, db), 3)

	r := rowFor(t, db, "chat.lobby", "1600000000000-0")
	is.Equal(r.senderID, nil)
	is.Equal(r.username, "cesar")
	is.Equal(r.createdAt.UnixMilli(), int64(1600000000000))
	r = rowFor(t, db, "chat.pm.uuid-cesar_uuid-mina", "1700000000000-0")
	is.Equal(*r.senderID, "uuid-mina")
	is.Equal(rowFor(t, db, "chat.pm.uuid-cesar_uuid-mina", mirrored.Id).message, "hi mina")

	// Ordering by ID is ordering by time, though the backfill copied the
	// older messages after the mirror copied the newest.
	rows, err := db.Query(ctx, `SELECT message FROM chat_messages ORDER BY id`)
	is.NoErr(err)
	var order []string
	for rows.Next() {
		var m string
		is.NoErr(rows.Scan(&m))
		order = append(order, m)
	}
	is.NoErr(rows.Err())
	is.Equal(order, []string{"old, no id", "hi cesar", "hi mina"})

	st, err = Backfill(ctx, rp, q, 1000)
	is.NoErr(err)
	is.Equal(st.Inserted, int64(0))
	is.Equal(count(t, db), 3)
}
