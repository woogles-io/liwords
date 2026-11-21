package chat

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/user"
)

const pkg = "chatstore"

func setup(t *testing.T) (*DBStore, *pgxpool.Pool) {
	t.Helper()
	if err := common.RecreateTestDB(pkg); err != nil {
		t.Fatal(err)
	}
	db, err := common.OpenTestingDB(pkg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	for _, u := range []string{"cesar", "mina", "zed"} {
		if _, err := db.Exec(context.Background(),
			`INSERT INTO users (username, uuid, email, password, created_at)
			 VALUES ($1::text, 'uuid-' || $1::text, $1::text || '@x.test', '', now())`, u); err != nil {
			t.Fatal(err)
		}
	}
	return NewDBStore(db), db
}

// add stores n unregulated messages in channel from cesar and returns them.
func add(t *testing.T, s *DBStore, channel string, n int) []string {
	t.Helper()
	ids := make([]string, n)
	for i := range ids {
		cm, err := s.AddChat(context.Background(), "cesar", "uuid-cesar", fmt.Sprintf("m%d", i), channel, false)
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = cm.Id
	}
	return ids
}

func texts(p *user.ChatPage) []string {
	out := make([]string, len(p.Messages))
	for i, m := range p.Messages {
		out[i] = m.Message
	}
	return out
}

// A message gets a UUIDv7 carrying its time, and comes back with its sender.
// A server line has no sender.
func TestAddChat(t *testing.T) {
	is := is.New(t)
	s, _ := setup(t)
	ctx := context.Background()

	cm, err := s.AddChat(ctx, "cesar", "uuid-cesar", "hello", "chat.game.g1", false)
	is.NoErr(err)
	id, err := uuid.Parse(cm.Id)
	is.NoErr(err)
	is.Equal(int(id.Version()), 7)
	sec, nsec := id.Time().UnixTime()
	is.Equal(sec*1000+nsec/1e6, cm.Timestamp)

	_, err = s.AddChat(ctx, "", "", "Challenged play was valid", "chat.game.g1", false)
	is.NoErr(err)

	msgs, err := s.OldChats(ctx, "chat.game.g1", 10)
	is.NoErr(err)
	is.Equal(len(msgs), 2)
	is.Equal(msgs[0].Message, "hello")
	is.Equal(msgs[0].UserId, "uuid-cesar")
	is.Equal(msgs[0].Username, "cesar")
	is.Equal(msgs[0].Channel, "chat.game.g1")
	is.Equal(msgs[1].UserId, "")
}

// A regulated message is refused within 5 s of the sender's previous one,
// or if it repeats it within 30 minutes. Private and game chats don't count
// toward it, and unregulated messages aren't checked.
func TestCooldown(t *testing.T) {
	is := is.New(t)
	s, db := setup(t)
	ctx := context.Background()

	_, err := s.AddChat(ctx, "cesar", "uuid-cesar", "hi", "chat.tournament.t1", true)
	is.NoErr(err)
	_, err = s.AddChat(ctx, "cesar", "uuid-cesar", "again", "chat.tournament.t1", true)
	is.Equal(err, ErrTooFast)
	_, err = s.AddChat(ctx, "cesar", "uuid-cesar", "unregulated", "chat.tournament.t1", false)
	is.NoErr(err)
	_, err = s.AddChat(ctx, "mina", "uuid-mina", "someone else", "chat.tournament.t1", true)
	is.NoErr(err)

	// Ten seconds later, the cooldown is over but the repeat check isn't.
	_, err = db.Exec(ctx, `UPDATE chat_messages SET created_at = created_at - interval '10 seconds'`)
	is.NoErr(err)
	_, err = s.AddChat(ctx, "cesar", "uuid-cesar", "unregulated", "chat.tournament.t1", true)
	is.Equal(err, ErrRepeated) // its last message, though that one wasn't regulated
	_, err = s.AddChat(ctx, "cesar", "uuid-cesar", "something new", "chat.tournament.t1", true)
	is.NoErr(err)

	// A private message just now doesn't hold up the next regulated one.
	_, err = db.Exec(ctx, `UPDATE chat_messages SET created_at = created_at - interval '10 seconds'`)
	is.NoErr(err)
	_, err = s.AddChat(ctx, "cesar", "uuid-cesar", "psst", "chat.pm.uuid-cesar_uuid-mina", false)
	is.NoErr(err)
	_, err = s.AddChat(ctx, "cesar", "uuid-cesar", "to everyone", "chat.tournament.t1", true)
	is.NoErr(err)
}

// History pages through a channel by message ID in both directions, and
// around a message.
func TestHistory(t *testing.T) {
	is := is.New(t)
	s, _ := setup(t)
	ctx := context.Background()
	ids := add(t, s, "chat.league.l1", 10) // m0 … m9
	add(t, s, "chat.league.other", 3)

	p, err := s.History(ctx, "chat.league.l1", user.ChatQuery{Limit: 4})
	is.NoErr(err)
	is.Equal(texts(p), []string{"m6", "m7", "m8", "m9"})
	is.True(p.HasOlder)
	is.True(!p.HasNewer)

	p, err = s.History(ctx, "chat.league.l1", user.ChatQuery{Before: ids[6], Limit: 4})
	is.NoErr(err)
	is.Equal(texts(p), []string{"m2", "m3", "m4", "m5"})
	is.True(p.HasOlder)
	is.True(p.HasNewer)

	p, err = s.History(ctx, "chat.league.l1", user.ChatQuery{Before: ids[2], Limit: 4})
	is.NoErr(err)
	is.Equal(texts(p), []string{"m0", "m1"})
	is.True(!p.HasOlder)

	p, err = s.History(ctx, "chat.league.l1", user.ChatQuery{After: ids[5], Limit: 3})
	is.NoErr(err)
	is.Equal(texts(p), []string{"m6", "m7", "m8"})
	is.True(p.HasNewer)
	p, err = s.History(ctx, "chat.league.l1", user.ChatQuery{After: ids[7], Limit: 3})
	is.NoErr(err)
	is.Equal(texts(p), []string{"m8", "m9"})
	is.True(!p.HasNewer)

	p, err = s.History(ctx, "chat.league.l1", user.ChatQuery{Around: ids[4], Limit: 5})
	is.NoErr(err)
	is.Equal(texts(p), []string{"m2", "m3", "m4", "m5", "m6"})
	is.True(p.HasOlder)
	is.True(p.HasNewer)

	_, err = s.History(ctx, "chat.league.l1", user.ChatQuery{Before: "nonsense"})
	is.True(err != nil)
}

// A moderator's delete hides a message from history and keeps it, with who
// deleted it. A Redis stream ID, which a page loaded before the move may
// still hold, finds the message copied from Redis.
func TestDeleteAndLegacyIDs(t *testing.T) {
	is := is.New(t)
	s, db := setup(t)
	ctx := context.Background()
	ids := add(t, s, "chat.gametv.g1", 3)

	is.NoErr(s.DeleteChat(ctx, "chat.gametv.g1", ids[1], "uuid-mina"))
	p, err := s.History(ctx, "chat.gametv.g1", user.ChatQuery{})
	is.NoErr(err)
	is.Equal(texts(p), []string{"m0", "m2"})
	var by string
	is.NoErr(db.QueryRow(ctx, `SELECT u.username FROM chat_messages m JOIN users u ON u.id = m.deleted_by
		WHERE m.id = $1`, ids[1]).Scan(&by))
	is.Equal(by, "mina")
	is.True(s.DeleteChat(ctx, "chat.gametv.g1", ids[1], "uuid-mina") != nil) // already gone
	_, err = s.GetChat(ctx, "chat.gametv.g1", ids[1])
	is.True(err != nil)

	// A message copied from Redis, found by its stream ID.
	lid, err := parseLegacyID("1700000000000-2")
	is.NoErr(err)
	_, err = db.Exec(ctx, `INSERT INTO chat_messages (id, channel, sender_username, message, created_at)
		VALUES ($1, 'chat.gametv.g1', 'zed', 'from redis', to_timestamp(1700000000))`, lid.messageID("chat.gametv.g1"))
	is.NoErr(err)
	m, err := s.GetChat(ctx, "chat.gametv.g1", "1700000000000-2")
	is.NoErr(err)
	is.Equal(m.Message, "from redis")
	p, err = s.History(ctx, "chat.gametv.g1", user.ChatQuery{})
	is.NoErr(err)
	is.Equal(texts(p), []string{"from redis", "m0", "m2"}) // older, so first
	is.NoErr(s.DeleteChat(ctx, "chat.gametv.g1", "1700000000000-2", "uuid-mina"))
}

// The read position only moves forward, and only to a message in that
// channel. Sending marks your own message read.
func TestReadPosition(t *testing.T) {
	is := is.New(t)
	s, _ := setup(t)
	ctx := context.Background()
	ids := add(t, s, "chat.tournament.t1", 3)
	other := add(t, s, "chat.tournament.t2", 1)

	got, err := s.LastRead(ctx, "uuid-mina", "chat.tournament.t1")
	is.NoErr(err)
	is.Equal(got, "")
	got, err = s.LastRead(ctx, "uuid-cesar", "chat.tournament.t1")
	is.NoErr(err)
	is.Equal(got, ids[2]) // cesar sent them

	is.NoErr(s.MarkRead(ctx, "uuid-mina", "chat.tournament.t1", ids[1]))
	is.NoErr(s.MarkRead(ctx, "uuid-mina", "chat.tournament.t1", ids[0])) // back: ignored
	is.NoErr(s.MarkRead(ctx, "uuid-mina", "chat.tournament.t1", other[0]))
	got, err = s.LastRead(ctx, "uuid-mina", "chat.tournament.t1")
	is.NoErr(err)
	is.Equal(got, ids[1])

	p, err := s.History(ctx, "chat.tournament.t1", user.ChatQuery{Around: got, Limit: 2})
	is.NoErr(err)
	is.Equal(texts(p), []string{"m0", "m1"})
	is.True(p.HasNewer)
}

// The chat list holds the user's private chats, most recent first, unread
// only if someone else wrote after the user's read position.
func TestLatestChannels(t *testing.T) {
	is := is.New(t)
	s, _ := setup(t)
	ctx := context.Background()
	cm := "chat.pm.uuid-cesar_uuid-mina"
	cz := "chat.pm.uuid-cesar_uuid-zed"

	_, err := s.AddChat(ctx, "cesar", "uuid-cesar", "hi mina", cm, false)
	is.NoErr(err)
	time.Sleep(2 * time.Millisecond)
	z, err := s.AddChat(ctx, "zed", "uuid-zed", "hi cesar", cz, false)
	is.NoErr(err)
	add(t, s, "chat.tournament.t1", 1) // not listed

	ch, err := s.LatestChannels(ctx, 20, 0, "uuid-cesar")
	is.NoErr(err)
	is.Equal(len(ch.Channels), 2)
	is.Equal(ch.Channels[0].Name, cz)
	is.Equal(ch.Channels[0].DisplayName, "pm:cesar:zed")
	is.Equal(ch.Channels[0].LastMessage, "hi cesar")
	is.True(ch.Channels[0].HasUpdate) // zed wrote; cesar hasn't read it
	is.Equal(ch.Channels[1].Name, cm)
	is.True(!ch.Channels[1].HasUpdate) // cesar's own message

	is.NoErr(s.MarkRead(ctx, "uuid-cesar", cz, z.Id))
	ch, err = s.LatestChannels(ctx, 20, 0, "uuid-cesar")
	is.NoErr(err)
	is.True(!ch.Channels[0].HasUpdate)

	// mina is a member of her chat with cesar, with it unread.
	ch, err = s.LatestChannels(ctx, 20, 0, "uuid-mina")
	is.NoErr(err)
	is.Equal(len(ch.Channels), 1)
	is.Equal(ch.Channels[0].DisplayName, "pm:cesar:mina")
	is.True(ch.Channels[0].HasUpdate)

	ch, err = s.LatestChannels(ctx, 1, 1, "uuid-cesar") // paging
	is.NoErr(err)
	is.Equal(ch.Channels[0].Name, cm)
}

// The migration makes both people of each existing private chat members,
// with it read up to its newest message.
func TestPrivateChatMembersMigration(t *testing.T) {
	is := is.New(t)
	s, db := setup(t)
	ctx := context.Background()
	// Messages as the backfill left them: no chat_reads rows.
	for i, l := range []string{"1700000000000-0", "1700000000001-0"} {
		lid, err := parseLegacyID(l)
		is.NoErr(err)
		_, err = db.Exec(ctx, `INSERT INTO chat_messages (id, channel, sender_id, sender_username, message, created_at)
			VALUES ($1, 'chat.pm.uuid-mina_uuid-zed', (SELECT id FROM users WHERE username='mina'), 'mina', $2, now())`,
			lid.messageID("chat.pm.uuid-mina_uuid-zed"), fmt.Sprint("old ", i))
		is.NoErr(err)
	}
	sql, err := os.ReadFile("../../../db/migrations/202611210001_chat_pm_members.up.sql")
	is.NoErr(err)
	_, err = db.Exec(ctx, string(sql))
	is.NoErr(err)

	for _, u := range []string{"uuid-mina", "uuid-zed"} {
		ch, err := s.LatestChannels(ctx, 20, 0, u)
		is.NoErr(err)
		is.Equal(len(ch.Channels), 1)
		is.Equal(ch.Channels[0].LastMessage, "old 1")
		is.True(!ch.Channels[0].HasUpdate)
	}
	_, err = db.Exec(ctx, string(sql)) // re-running changes nothing
	is.NoErr(err)
}

// Moderator chat review scans channel names by prefix.
func TestScanChannels(t *testing.T) {
	is := is.New(t)
	s, _ := setup(t)
	ctx := context.Background()
	for _, ch := range []string{"chat.game.g1", "chat.game.g2", "chat.gametv.g1", "chat.pm.uuid-cesar_uuid-mina"} {
		add(t, s, ch, 2)
	}
	var all []string
	cursor := ""
	for {
		chans, next, err := s.ScanChannels(ctx, cursor, "chat.game.", 1)
		is.NoErr(err)
		all = append(all, chans...)
		if next == "0" {
			break
		}
		cursor = next
	}
	is.Equal(all, []string{"chat.game.g1", "chat.game.g2"})
	chans, next, err := s.ScanChannels(ctx, "", "", 100)
	is.NoErr(err)
	is.Equal(len(chans), 4)
	is.Equal(next, "0")
}
