package broadcasts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

const testDBPkg = "broadcasts"

// feedServer serves body, or a 404 while dead is set, and counts requests.
type feedServer struct {
	*httptest.Server
	mu   sync.Mutex
	body string
	dead bool
	hits atomic.Int32
}

func newFeedServer(t *testing.T, body string) *feedServer {
	fs := &feedServer{body: body}
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.hits.Add(1)
		fs.mu.Lock()
		defer fs.mu.Unlock()
		if fs.dead {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(fs.body))
	}))
	t.Cleanup(fs.Close)
	return fs
}

func (fs *feedServer) set(body string, dead bool) {
	fs.mu.Lock()
	fs.body, fs.dead = body, dead
	fs.mu.Unlock()
}

// testBroadcast is a broadcast row for the tests to poll.
type testBroadcast struct {
	id        int32
	uuid      string
	slug      string
	url       string
	creatorID int32
}

func setupFeedDB(t *testing.T) (*pgxpool.Pool, *models.Queries) {
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

func createTestBroadcast(t *testing.T, pool *pgxpool.Pool, q *models.Queries, slug, url string) testBroadcast {
	t.Helper()
	ctx := context.Background()
	var userID int32
	err := pool.QueryRow(ctx, `INSERT INTO users (uuid, username, email, password) VALUES ($1, $2, $3, '') RETURNING id`,
		"creator-uuid", "creator", "creator@example.com").Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	u := uuid.New()
	row, err := q.CreateBroadcast(ctx, models.CreateBroadcastParams{
		Uuid:                u,
		Slug:                slug,
		Name:                slug,
		BroadcastUrl:        url,
		BroadcastUrlFormat:  "tsh_newt_json",
		PollIntervalSeconds: 60,
		Lexicon:             "CSW24",
		BoardLayout:         "CrosswordGame",
		LetterDistribution:  "english",
		CreatorID:           userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return testBroadcast{id: row.ID, uuid: u.String(), slug: slug, url: url, creatorID: userID}
}

func (bs *BroadcastService) pollTest(t *testing.T, b testBroadcast) (bool, error) {
	t.Helper()
	return bs.pollBroadcast(context.Background(), b.id, b.uuid, b.slug, b.url, "tsh_newt_json")
}

func newTestService(q *models.Queries) *BroadcastService {
	return NewBroadcastService(nil, q, nil, nil, nil)
}

func playerNames(f *parsedFeed) []string {
	var names []string
	for _, p := range f.division("").Players {
		names = append(names, p.Name)
	}
	return names
}

// The poller stores what it fetches. Another server reads that, without
// fetching, and keeps serving it after the feed URL dies.
func TestStoredFeedOutlivesItsURL(t *testing.T) {
	pool, q := setupFeedDB(t)
	ctx := context.Background()
	fs := newFeedServer(t, minimalTSHFeed)
	b := createTestBroadcast(t, pool, q, "stored", fs.URL)

	poller := newTestService(q)
	changed, err := poller.pollTest(t, b)
	if err != nil || !changed {
		t.Fatalf("first poll: changed=%v err=%v", changed, err)
	}

	other := newTestService(q)
	f := other.feed(ctx, b.slug)
	if f == nil || len(f.division("").Players) != 3 {
		t.Fatalf("other server: want the stored feed with 3 players, got %+v", f)
	}
	if n := fs.hits.Load(); n != 1 {
		t.Fatalf("feed fetched %d times, want only the poll's 1", n)
	}

	changed, err = poller.pollTest(t, b)
	if err != nil || changed {
		t.Fatalf("same feed again: changed=%v err=%v, want unchanged", changed, err)
	}

	fs.set("", true)
	if _, err := poller.pollTest(t, b); err == nil {
		t.Fatal("poll of a dead URL succeeded")
	}
	st, err := q.GetBroadcastFeedStatus(ctx, b.id)
	if err != nil {
		t.Fatal(err)
	}
	if st.ConsecutiveFailures != 1 || st.LastError == "" {
		t.Fatalf("failure not recorded: %+v", st)
	}

	// A fresh server, after a restart say, still has the stored feed.
	if f := newTestService(q).feed(ctx, b.slug); f == nil || len(f.division("").Players) != 3 {
		t.Fatalf("after the URL died: want the stored feed, got %+v", f)
	}

	// Recovering clears the error.
	fs.set(minimalTSHFeed, false)
	if _, err := poller.pollTest(t, b); err != nil {
		t.Fatal(err)
	}
	if st, _ := q.GetBroadcastFeedStatus(ctx, b.id); st.ConsecutiveFailures != 0 || st.LastError != "" {
		t.Fatalf("error not cleared after a good poll: %+v", st)
	}
}

// A server that parsed an older feed picks up the one another server stored.
func TestFeedChangeReachesOtherServers(t *testing.T) {
	pool, q := setupFeedDB(t)
	ctx := context.Background()
	fs := newFeedServer(t, minimalTSHFeed)
	b := createTestBroadcast(t, pool, q, "changing", fs.URL)

	poller, viewer := newTestService(q), newTestService(q)
	if _, err := poller.pollTest(t, b); err != nil {
		t.Fatal(err)
	}
	if got := playerNames(viewer.feed(ctx, b.slug)); len(got) != 3 {
		t.Fatalf("before: %v", got)
	}

	fs.set(undefinedTSHFeed, false)
	if changed, err := poller.pollTest(t, b); err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if got := playerNames(viewer.feed(ctx, b.slug)); len(got) != 2 {
		t.Fatalf("viewer still has the old feed: %v", got)
	}
}

// A feed that can't be parsed is recorded as an error and doesn't replace
// the stored one.
func TestUnparseableFeedKeepsStoredFeed(t *testing.T) {
	pool, q := setupFeedDB(t)
	ctx := context.Background()
	fs := newFeedServer(t, minimalTSHFeed)
	b := createTestBroadcast(t, pool, q, "garbled", fs.URL)

	bs := newTestService(q)
	if _, err := bs.pollTest(t, b); err != nil {
		t.Fatal(err)
	}
	fs.set("<html>maintenance</html>", false)
	if _, err := bs.pollTest(t, b); err == nil {
		t.Fatal("unparseable feed accepted")
	}
	if f := newTestService(q).feed(ctx, b.slug); f == nil || len(f.division("").Players) != 3 {
		t.Fatalf("stored feed replaced: %+v", f)
	}
}

// Each due broadcast is claimed by one poller, and isn't due again until its
// interval has passed.
func TestClaimBroadcastForPolling(t *testing.T) {
	pool, q := setupFeedDB(t)
	ctx := context.Background()
	b := createTestBroadcast(t, pool, q, "claimed", "http://unused.invalid/")

	var wg sync.WaitGroup
	var claimed atomic.Int32
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if row, err := q.ClaimBroadcastForPolling(ctx); err == nil && row.ID == b.id {
				claimed.Add(1)
			}
		}()
	}
	wg.Wait()
	if n := claimed.Load(); n != 1 {
		t.Fatalf("claimed %d times, want 1", n)
	}

	// Changing the URL makes it due again.
	if err := q.UpdateBroadcast(ctx, models.UpdateBroadcastParams{
		Slug:                b.slug,
		Name:                b.slug,
		BroadcastUrl:        "http://other.invalid/",
		BroadcastUrlFormat:  "tsh_newt_json",
		PollIntervalSeconds: 60,
		Lexicon:             "CSW24",
		BoardLayout:         "CrosswordGame",
		LetterDistribution:  "english",
		Active:              true,
	}); err != nil {
		t.Fatal(err)
	}
	row, err := q.ClaimBroadcastForPolling(ctx)
	if err != nil || row.ID != b.id {
		t.Fatalf("after a URL change: row=%+v err=%v, want it due", row, err)
	}
}

// Changing claims or slots changes the version the response caches are
// keyed by, on every server.
func TestGamesChangedChangesVersion(t *testing.T) {
	pool, q := setupFeedDB(t)
	ctx := context.Background()
	b := createTestBroadcast(t, pool, q, "versioned", "http://unused.invalid/")

	bs := newTestService(q)
	before, err := bs.version(ctx, b.slug)
	if err != nil {
		t.Fatal(err)
	}
	bs.gamesChanged(ctx, b.id)
	after, err := newTestService(q).version(ctx, b.slug)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatalf("version unchanged: %+v", after)
	}
}
