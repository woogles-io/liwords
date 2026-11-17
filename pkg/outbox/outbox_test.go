package outbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

const testDBPkg = "outbox"

func setupDB(t *testing.T) (*pgxpool.Pool, *models.Queries) {
	t.Helper()
	if os.Getenv("TEST_DB_HOST") == "" {
		t.Skip("no test database (TEST_DB_HOST)")
	}
	if err := common.RecreateTestDB(testDBPkg); err != nil {
		t.Fatal(err)
	}
	pool, err := common.OpenTestingDB(testDBPkg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool, models.New(pool)
}

// recorder is a channel that records what it sends, and fails for the
// recipients in fail.
type recorder struct {
	mu   sync.Mutex
	sent []string
	fail map[string]bool
	// after, if set, runs after each message goes out.
	after func(m Message)
}

func (r *recorder) send(_ context.Context, m Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail[m.Recipient] {
		return errors.New("rejected")
	}
	r.sent = append(r.sent, m.Key)
	if r.after != nil {
		r.after(m)
	}
	return nil
}

func (r *recorder) keys() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.sent...)
}

func emails(n int) []Message {
	var msgs []Message
	for i := range n {
		msgs = append(msgs, Message{
			Key: fmt.Sprintf("k%d", i), Channel: ChannelEmail,
			Recipient: fmt.Sprintf("r%d", i), Subject: "s", Body: "b",
		})
	}
	return msgs
}

func newSender(q *models.Queries, r *recorder) *Sender {
	return NewSender(q, map[string]SendFunc{ChannelEmail: r.send}, time.Millisecond)
}

func TestAddSkipsKeysAlreadyThere(t *testing.T) {
	ctx := context.Background()
	_, q := setupDB(t)
	n, err := Add(ctx, q, emails(3))
	if err != nil || n != 3 {
		t.Fatalf("first add: %d, %v", n, err)
	}
	n, err = Add(ctx, q, emails(4))
	if err != nil || n != 1 {
		t.Fatalf("second add: %d, %v; want only the new message", n, err)
	}
	if _, err := Add(ctx, q, []Message{{Channel: ChannelEmail, Body: "b"}}); err == nil {
		t.Fatal("a message without a key was added")
	}
}

func TestSendSendsEachMessageOnce(t *testing.T) {
	ctx := context.Background()
	_, q := setupDB(t)
	if _, err := Add(ctx, q, emails(3)); err != nil {
		t.Fatal(err)
	}
	r := &recorder{}
	s := newSender(q, r)
	if err := s.Send(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Send(ctx); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(r.keys()); got != "[k0 k1 k2]" {
		t.Fatalf("sent %s, want each once, oldest first", got)
	}
}

// A send cut off by a deploy (ctx cancelled) stops after the message in
// flight, which is marked sent; the next run sends only the rest.
func TestSendCutOffCarriesOn(t *testing.T) {
	ctx := context.Background()
	_, q := setupDB(t)
	if _, err := Add(ctx, q, emails(4)); err != nil {
		t.Fatal(err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	r := &recorder{}
	r.after = func(m Message) {
		if m.Key == "k1" {
			cancel()
		}
	}
	if err := newSender(q, r).Send(runCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cut-off send returned %v, want context.Canceled", err)
	}
	r.after = nil
	if err := newSender(q, r).Send(ctx); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(r.keys()); got != "[k0 k1 k2 k3]" {
		t.Fatalf("sent %s, want each once", got)
	}
}

// A message that fails doesn't hold up the rest, is tried again by the
// next run, and is given up after maxAttempts.
func TestSendRetriesThenGivesUp(t *testing.T) {
	ctx := context.Background()
	_, q := setupDB(t)
	if _, err := Add(ctx, q, emails(3)); err != nil {
		t.Fatal(err)
	}
	r := &recorder{fail: map[string]bool{"r1": true}}
	s := newSender(q, r)
	for i := range maxAttempts {
		if err := s.Send(ctx); err == nil {
			t.Fatalf("run %d: a failed message wasn't reported", i+1)
		}
	}
	if err := s.Send(ctx); err != nil {
		t.Fatalf("run after giving up: %v", err)
	}
	if got := fmt.Sprint(r.keys()); got != "[k0 k2]" {
		t.Fatalf("sent %s", got)
	}
	var attempts int32
	var lastError string
	pool, _ := common.OpenTestingDB(testDBPkg)
	defer pool.Close()
	if err := pool.QueryRow(ctx, "SELECT attempts, last_error FROM outbox WHERE dedupe_key = 'k1'").Scan(&attempts, &lastError); err != nil {
		t.Fatal(err)
	}
	if attempts != maxAttempts || lastError != "rejected" {
		t.Fatalf("attempts %d, last_error %q", attempts, lastError)
	}
}

func TestSendFailsUnknownChannel(t *testing.T) {
	ctx := context.Background()
	_, q := setupDB(t)
	if _, err := Add(ctx, q, []Message{{Key: "d", Channel: ChannelLeagueDiscord, Body: "b"}}); err != nil {
		t.Fatal(err)
	}
	if err := newSender(q, &recorder{}).Send(ctx); err == nil {
		t.Fatal("a message on a channel with no sender was reported sent")
	}
}

// Sent messages are deleted once old; unsent ones stay, however old.
func TestSendDeletesOldSentMessages(t *testing.T) {
	ctx := context.Background()
	pool, q := setupDB(t)
	if _, err := Add(ctx, q, emails(2)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE outbox SET created_at = NOW() - INTERVAL '40 days', sent_at = NOW() WHERE dedupe_key = 'k0'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE outbox SET created_at = NOW() - INTERVAL '40 days', attempts = 1 WHERE dedupe_key = 'k1'"); err != nil {
		t.Fatal(err)
	}
	r := &recorder{}
	if err := newSender(q, r).Send(ctx); err != nil {
		t.Fatal(err)
	}
	var keys []string
	rows, err := pool.Query(ctx, "SELECT dedupe_key FROM outbox ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			t.Fatal(err)
		}
		keys = append(keys, k)
	}
	if fmt.Sprint(keys) != "[k1]" || fmt.Sprint(r.keys()) != "[k1]" {
		t.Fatalf("left %v, sent %v", keys, r.keys())
	}
}
