package jobs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

const testDBPkg = "jobs"

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

// fast are options for tests: leases that expire within a test, and short
// sleeps.
var fast = Options{Lease: 300 * time.Millisecond, MaxSleep: 50 * time.Millisecond}

// start runs r until the test ends.
func start(t *testing.T, r *Runner) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- r.Run(context.Background()) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		r.Drain(ctx)
		if err := <-done; err != nil {
			t.Errorf("Run: %v", err)
		}
	})
}

func eventually(t *testing.T, within time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(within)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("not within %v: %s", within, what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func getJob(t *testing.T, q *models.Queries, kind, key string) models.GetJobRow {
	t.Helper()
	j, err := q.GetJob(context.Background(), models.GetJobParams{Kind: kind, Key: key})
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func enqueue(t *testing.T, q *models.Queries, kind, key string) {
	t.Helper()
	if err := Enqueue(context.Background(), q, kind, key, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestScheduleNext(t *testing.T) {
	at := func(s string) time.Time {
		tm, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return tm
	}
	cases := []struct {
		s     Schedule
		str   string
		after string
		want  string
	}{
		{Every(10 * time.Second), "@every 10s", "2026-11-14T10:00:00Z", "2026-11-14T10:00:10Z"},
		{Hourly(1), "@hourly :01", "2026-11-14T10:00:00Z", "2026-11-14T10:01:00Z"},
		{Hourly(1), "@hourly :01", "2026-11-14T10:01:00Z", "2026-11-14T11:01:00Z"},
		{Hourly(1), "@hourly :01", "2026-11-14T23:30:00Z", "2026-11-15T00:01:00Z"},
		{Daily(5, 0), "@daily 05:00", "2026-11-14T04:59:59Z", "2026-11-14T05:00:00Z"},
		{Daily(5, 0), "@daily 05:00", "2026-11-14T05:00:00Z", "2026-11-15T05:00:00Z"},
		// Times in other zones are read as the instant they are.
		{Daily(5, 0), "@daily 05:00", "2026-11-14T01:00:00-05:00", "2026-11-15T05:00:00Z"},
		{Daily(0, 0), "@daily 00:00", "2026-12-31T12:00:00Z", "2027-01-01T00:00:00Z"},
	}
	for _, c := range cases {
		if c.s.String() != c.str {
			t.Errorf("String() = %q, want %q", c.s.String(), c.str)
		}
		if got := c.s.Next(at(c.after)); !got.Equal(at(c.want)) {
			t.Errorf("%s after %s: got %s, want %s", c.str, c.after, got.UTC().Format(time.RFC3339), c.want)
		}
	}
}

// A schedule runs, and runs again after each run.
func TestScheduleRunsAgain(t *testing.T) {
	_, q := setupDB(t)
	var runs atomic.Int32
	r := NewRunner(q, fast)
	r.Schedule("tick", Every(100*time.Millisecond), func(ctx context.Context, job Job) error {
		runs.Add(1)
		return nil
	})
	start(t, r)
	eventually(t, 3*time.Second, "3 runs", func() bool { return runs.Load() >= 3 })

	j := getJob(t, q, "tick", "")
	if !j.LastRunAt.Valid || j.LastError != "" || j.DoneAt.Valid || j.Attempts > 1 {
		t.Fatalf("row after runs: %+v", j)
	}
}

// A restart with the same schedule keeps its next run; a changed schedule
// starts over.
func TestScheduleKeepsNextRunAcrossRestart(t *testing.T) {
	_, q := setupDB(t)
	ctx := context.Background()
	register := func(s Schedule) {
		r := NewRunner(q, fast)
		r.Schedule("daily", s, func(context.Context, Job) error { return nil })
		if err := r.upsertSchedules(ctx); err != nil {
			t.Fatal(err)
		}
	}
	register(Daily(5, 0))
	first := getJob(t, q, "daily", "").RunAt.Time
	time.Sleep(10 * time.Millisecond)
	register(Daily(5, 0))
	if again := getJob(t, q, "daily", "").RunAt.Time; !again.Equal(first) {
		t.Fatalf("same schedule moved its run: %v → %v", first, again)
	}
	register(Hourly(1))
	if changed := getJob(t, q, "daily", "").RunAt.Time; changed.Equal(first) {
		t.Fatal("changed schedule kept the old run")
	}
}

// With several runners, every job runs exactly once.
func TestEachJobRunsOnce(t *testing.T) {
	_, q := setupDB(t)
	const n = 60
	var mu sync.Mutex
	runs := map[string]int{}
	for range 3 {
		r := NewRunner(q, fast)
		r.Handle("once", func(ctx context.Context, job Job) error {
			mu.Lock()
			runs[job.Key]++
			mu.Unlock()
			return nil
		})
		start(t, r)
	}
	for i := range n {
		enqueue(t, q, "once", fmt.Sprint(i))
	}
	eventually(t, 5*time.Second, "all jobs run", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(runs) == n
	})
	time.Sleep(200 * time.Millisecond) // time for a duplicate to show up
	mu.Lock()
	defer mu.Unlock()
	for key, c := range runs {
		if c != 1 {
			t.Errorf("job %s ran %d times", key, c)
		}
	}
}

// A server that dies holding a job keeps it until its lease expires; then
// another runner takes it.
func TestDeadRunnersJobIsTakenOver(t *testing.T) {
	_, q := setupDB(t)
	ctx := context.Background()
	enqueue(t, q, "orphan", "a")

	// The dead server's claim: a lease of a second, never renewed.
	claimed, err := q.ClaimDueJobs(ctx, models.ClaimDueJobsParams{
		LeaseSeconds: 1,
		LeaseToken:   pgtype.UUID{Bytes: uuid.New(), Valid: true},
		Kinds:        []string{"orphan"},
		MaxJobs:      1,
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %v %v", claimed, err)
	}
	claimedAt := time.Now()

	var ranAt atomic.Int64
	r := NewRunner(q, fast)
	r.Handle("orphan", func(ctx context.Context, job Job) error {
		ranAt.Store(time.Now().UnixNano())
		if job.Attempt != 2 {
			t.Errorf("attempt %d, want 2 (the dead server's was 1)", job.Attempt)
		}
		return nil
	})
	start(t, r)

	eventually(t, 3*time.Second, "job taken over", func() bool { return ranAt.Load() != 0 })
	if waited := time.Unix(0, ranAt.Load()).Sub(claimedAt); waited < 900*time.Millisecond {
		t.Fatalf("taken after %v, while the lease still held", waited)
	}
	eventually(t, time.Second, "job recorded done", func() bool { return getJob(t, q, "orphan", "a").DoneAt.Valid })
}

// A runner renews the lease of a job that outlasts it, so no other runner
// takes the job.
func TestLeaseIsRenewedWhileRunning(t *testing.T) {
	_, q := setupDB(t)
	var runs atomic.Int32
	for range 2 {
		r := NewRunner(q, fast)
		r.Handle("long", func(ctx context.Context, job Job) error {
			runs.Add(1)
			time.Sleep(4 * fast.Lease)
			return nil
		})
		start(t, r)
	}
	enqueue(t, q, "long", "a")
	eventually(t, 3*time.Second, "job done", func() bool { return getJob(t, q, "long", "a").DoneAt.Valid })
	if n := runs.Load(); n != 1 {
		t.Fatalf("ran %d times", n)
	}
}

// A runner that loses its lease stops the job and doesn't record its
// outcome over the new owner's.
func TestLostLeaseCancelsJob(t *testing.T) {
	pool, q := setupDB(t)
	ctx := context.Background()
	cancelled := make(chan struct{})
	r := NewRunner(q, fast)
	r.Handle("stolen", func(ctx context.Context, job Job) error {
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	})
	start(t, r)
	enqueue(t, q, "stolen", "a")
	eventually(t, 2*time.Second, "job claimed", func() bool { return getJob(t, q, "stolen", "a").LeaseUntil.Valid })

	// Another runner's claim, as after an expired lease.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET lease_token = $1, lease_until = NOW() + INTERVAL '1 hour' WHERE kind = 'stolen'`,
		uuid.New()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("job not cancelled after losing its lease")
	}
	time.Sleep(100 * time.Millisecond)
	if j := getJob(t, q, "stolen", "a"); j.DoneAt.Valid || j.LastError != "" {
		t.Fatalf("lost job's outcome recorded: %+v", j)
	}
}

// A failing job is retried with backoff, and given up after its last
// attempt.
func TestRetryThenGiveUp(t *testing.T) {
	_, q := setupDB(t)
	var attempts []int32
	var mu sync.Mutex
	gaveUp := make(chan Job, 2)
	r := NewRunner(q, fast)
	r.Handle("flaky", func(ctx context.Context, job Job) error {
		mu.Lock()
		attempts = append(attempts, job.Attempt)
		mu.Unlock()
		if job.Key == "recovers" && job.Attempt == 2 {
			return nil
		}
		return errors.New("lambda timed out")
	},
		MaxAttempts(3),
		Backoff(func(int32) time.Duration { return 20 * time.Millisecond }),
		OnGiveUp(func(ctx context.Context, job Job, err error) { gaveUp <- job }),
	)
	start(t, r)
	enqueue(t, q, "flaky", "fails")
	enqueue(t, q, "flaky", "recovers")

	select {
	case job := <-gaveUp:
		if job.Key != "fails" || job.Attempt != 3 {
			t.Fatalf("gave up %+v", job)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("never gave up")
	}
	eventually(t, 2*time.Second, "recovering job done", func() bool { return getJob(t, q, "flaky", "recovers").DoneAt.Valid })

	if j := getJob(t, q, "flaky", "fails"); !j.DoneAt.Valid || j.LastError != "lambda timed out" || j.Attempts != 3 {
		t.Fatalf("given-up row: %+v", j)
	}
	if j := getJob(t, q, "flaky", "recovers"); j.LastError != "" || j.Attempts != 2 {
		t.Fatalf("recovered row: %+v", j)
	}
	select {
	case job := <-gaveUp:
		t.Fatalf("also gave up %+v", job)
	default:
	}
}

// A panic is recorded as the job's error, and the runner carries on.
func TestPanicIsRecorded(t *testing.T) {
	_, q := setupDB(t)
	r := NewRunner(q, fast)
	r.Handle("bad", func(ctx context.Context, job Job) error {
		if job.Key == "panics" {
			var m map[string]int
			m["x"] = 1 // nil map write
		}
		return nil
	}, MaxAttempts(1))
	start(t, r)
	enqueue(t, q, "bad", "panics")
	eventually(t, 2*time.Second, "panicking job recorded", func() bool { return getJob(t, q, "bad", "panics").DoneAt.Valid })
	if j := getJob(t, q, "bad", "panics"); !strings.HasPrefix(j.LastError, "panic:") {
		t.Fatalf("error %q", j.LastError)
	}
	enqueue(t, q, "bad", "fine")
	eventually(t, 2*time.Second, "next job done", func() bool { return getJob(t, q, "bad", "fine").DoneAt.Valid })
}

// A draining runner claims nothing new, and waits for the job in flight.
func TestDrainFinishesInFlightOnly(t *testing.T) {
	_, q := setupDB(t)
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	r := NewRunner(q, fast)
	r.Handle("work", func(ctx context.Context, job Job) error {
		if job.Key == "first" {
			started <- struct{}{}
			<-release
		}
		return nil
	})
	runDone := make(chan error, 1)
	go func() { runDone <- r.Run(context.Background()) }()

	enqueue(t, q, "work", "first")
	<-started
	drained := make(chan error, 1)
	go func() { drained <- r.Drain(context.Background()) }()
	time.Sleep(50 * time.Millisecond)
	enqueue(t, q, "work", "second")
	r.Wake()
	time.Sleep(200 * time.Millisecond)
	select {
	case err := <-drained:
		t.Fatalf("Drain returned (%v) with a job running", err)
	default:
	}
	close(release)
	if err := <-drained; err != nil {
		t.Fatal(err)
	}
	if err := <-runDone; err != nil {
		t.Fatal(err)
	}
	if !getJob(t, q, "work", "first").DoneAt.Valid {
		t.Fatal("in-flight job not finished")
	}
	if j := getJob(t, q, "work", "second"); j.Attempts != 0 || j.DoneAt.Valid {
		t.Fatalf("draining runner claimed a new job: %+v", j)
	}
}

// When Drain runs out of time, the jobs still running are cancelled.
func TestDrainTimeoutCancelsJobs(t *testing.T) {
	_, q := setupDB(t)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	r := NewRunner(q, fast)
	r.Handle("stuck", func(ctx context.Context, job Job) error {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	})
	go r.Run(context.Background())
	enqueue(t, q, "stuck", "a")
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := r.Drain(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Drain: %v", err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("job not cancelled")
	}
}

// A runner without a handler for a kind (an old server during a deploy)
// leaves its rows alone.
func TestUnknownKindIsLeftAlone(t *testing.T) {
	_, q := setupDB(t)
	r := NewRunner(q, fast)
	r.Handle("known", func(context.Context, Job) error { return nil })
	start(t, r)
	enqueue(t, q, "added_later", "a")
	enqueue(t, q, "known", "a")
	eventually(t, 2*time.Second, "known job done", func() bool { return getJob(t, q, "known", "a").DoneAt.Valid })
	time.Sleep(100 * time.Millisecond)
	if j := getJob(t, q, "added_later", "a"); j.Attempts != 0 || j.LeaseUntil.Valid || j.DoneAt.Valid {
		t.Fatalf("unknown kind touched: %+v", j)
	}
}

// Enqueueing a job that isn't finished leaves it as it is; enqueueing a
// finished one makes it due again.
func TestEnqueueDeduplicates(t *testing.T) {
	_, q := setupDB(t)
	ctx := context.Background()
	later := time.Now().Add(time.Hour)
	if err := Enqueue(ctx, q, "pair", "t1/A/3", later); err != nil {
		t.Fatal(err)
	}
	if err := Enqueue(ctx, q, "pair", "t1/A/3", time.Now()); err != nil {
		t.Fatal(err)
	}
	if j := getJob(t, q, "pair", "t1/A/3"); !j.RunAt.Time.Equal(later.Truncate(time.Microsecond)) {
		t.Fatalf("unfinished job changed: run_at %v", j.RunAt.Time)
	}

	var runs atomic.Int32
	r := NewRunner(q, fast)
	r.Handle("again", func(context.Context, Job) error { runs.Add(1); return nil })
	start(t, r)
	enqueue(t, q, "again", "k")
	eventually(t, 2*time.Second, "first run", func() bool { return getJob(t, q, "again", "k").DoneAt.Valid })
	enqueue(t, q, "again", "k")
	r.Wake()
	eventually(t, 2*time.Second, "second run", func() bool { return runs.Load() == 2 })
}
