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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

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

func TestCronNext(t *testing.T) {
	at := func(s string) time.Time {
		tm, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return tm
	}
	cases := []struct {
		expr  string
		after string
		want  string
	}{
		{"*/10 * * * * *", "2026-11-14T10:00:00Z", "2026-11-14T10:00:10Z"},
		{"*/10 * * * * *", "2026-11-14T10:00:03Z", "2026-11-14T10:00:10Z"},
		{"*/30 * * * * *", "2026-11-14T10:00:59Z", "2026-11-14T10:01:00Z"},
		{"* * * * *", "2026-11-14T10:00:30Z", "2026-11-14T10:01:00Z"},
		{"*/10 * * * *", "2026-11-14T10:05:00Z", "2026-11-14T10:10:00Z"},
		{"1 * * * *", "2026-11-14T10:00:00Z", "2026-11-14T10:01:00Z"},
		{"1 * * * *", "2026-11-14T10:01:00Z", "2026-11-14T11:01:00Z"},
		{"0 5 * * *", "2026-11-14T04:59:59Z", "2026-11-14T05:00:00Z"},
		{"0 5 * * *", "2026-11-14T05:00:00Z", "2026-11-15T05:00:00Z"},
		// UTC, whatever zone the time is given in.
		{"0 5 * * *", "2026-11-14T01:00:00-05:00", "2026-11-15T05:00:00Z"},
		{"0 0 * * *", "2026-12-31T12:00:00Z", "2027-01-01T00:00:00Z"},
	}
	for _, c := range cases {
		s := Cron(c.expr)
		if s.String() != c.expr {
			t.Errorf("String() = %q, want %q", s.String(), c.expr)
		}
		if got := s.Next(at(c.after)); !got.Equal(at(c.want)) {
			t.Errorf("%q after %s: got %s, want %s", c.expr, c.after, got.UTC().Format(time.RFC3339), c.want)
		}
	}
}

// A server whose clock is behind the database's finishes a run before, by
// its own clock, the time it was due. The next run must still be the next
// slot, not the one just run.
func TestNextRunWithSlowClock(t *testing.T) {
	s := Cron("*/10 * * * * *")
	due := time.Date(2026, 11, 15, 12, 0, 10, 0, time.UTC)

	slowNow := due.Add(-150 * time.Millisecond)
	if got, want := nextRun(s, due, slowNow), due.Add(10*time.Second); !got.Equal(want) {
		t.Errorf("slow clock: next run %v, want %v", got, want)
	}
	// A run that went past later slots skips them.
	lateNow := due.Add(23 * time.Second)
	if got, want := nextRun(s, due, lateNow), due.Add(30*time.Second); !got.Equal(want) {
		t.Errorf("long run: next run %v, want %v", got, want)
	}
}

func TestCronEvery(t *testing.T) {
	type in struct{ d, offset time.Duration }
	good := map[in]string{
		{10 * time.Second, 0}:                "*/10 * * * * *",
		{30 * time.Second, 0}:                "*/30 * * * * *",
		{30 * time.Second, 17 * time.Second}: "17/30 * * * * *",
		{time.Minute, 0}:                     "0 */1 * * * *",
		{5 * time.Minute, 0}:                 "0 */5 * * * *",
		{5 * time.Minute, 17 * time.Second}:  "17 */5 * * * *",
		{time.Hour, 59 * time.Second}:        "59 */60 * * * *",
	}
	for in, want := range good {
		got, err := CronEvery(in.d, in.offset)
		if err != nil || got != want {
			t.Errorf("CronEvery(%v, %v) = %q, %v; want %q", in.d, in.offset, got, err, want)
		}
		Cron(got) // parses
	}
	bad := []in{
		{0, 0}, {7 * time.Second, 0}, {90 * time.Second, 0}, {7 * time.Minute, 0},
		{2 * time.Hour, 0}, {1500 * time.Millisecond, 0},
		{30 * time.Second, 30 * time.Second}, // not less than d
		{5 * time.Minute, time.Minute},       // not under a minute
		{5 * time.Minute, 1500 * time.Millisecond},
		{5 * time.Minute, -time.Second},
	}
	for _, in := range bad {
		if got, err := CronEvery(in.d, in.offset); err == nil {
			t.Errorf("CronEvery(%v, %v) = %q, want an error", in.d, in.offset, got)
		}
	}

	// The offset moves every run, not just the first.
	next := func(expr string, from time.Time) []time.Time {
		var ts []time.Time
		for range 3 {
			from = Cron(expr).Next(from)
			ts = append(ts, from)
		}
		return ts
	}
	from := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	if got := next("17/30 * * * * *", from); !got[0].Equal(from.Add(17*time.Second)) ||
		!got[1].Equal(from.Add(47*time.Second)) || !got[2].Equal(from.Add(77*time.Second)) {
		t.Errorf("17/30 runs at %v", got)
	}
	if got := next("17 */5 * * * *", from); !got[0].Equal(from.Add(17*time.Second)) ||
		!got[1].Equal(from.Add(5*time.Minute+17*time.Second)) {
		t.Errorf("17 */5 runs at %v", got)
	}
}

func TestCronRejectsBadExpression(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("no panic")
		}
	}()
	Cron("every ten seconds")
}

// A schedule runs, and runs again after each run.
func TestScheduleRunsAgain(t *testing.T) {
	_, q := setupDB(t)
	var runs atomic.Int32
	r := NewRunner(q, fast)
	r.Schedule("tick", Cron("* * * * * *"), func(ctx context.Context, job Job) error {
		runs.Add(1)
		return nil
	})
	start(t, r)
	eventually(t, 5*time.Second, "3 runs", func() bool { return runs.Load() >= 3 })

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
	register(Cron("0 5 * * *"))
	first := getJob(t, q, "daily", "").RunAt.Time
	time.Sleep(10 * time.Millisecond)
	register(Cron("0 5 * * *"))
	if again := getJob(t, q, "daily", "").RunAt.Time; !again.Equal(first) {
		t.Fatalf("same schedule moved its run: %v → %v", first, again)
	}
	register(Cron("1 * * * *"))
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
	// The job has returned by the time Drain does, so the events it sent
	// are in the bus before the bus drains.
	select {
	case <-cancelled:
	default:
		t.Fatal("Drain returned before the cancelled job did")
	}
	// Handed back as it was: due now, unleased, the attempt not counted,
	// no run recorded.
	j := getJob(t, q, "stuck", "a")
	if j.DoneAt.Valid || j.LeaseUntil.Valid || j.Attempts != 0 || j.LastRunAt.Valid ||
		j.RunAt.Time.After(time.Now()) {
		t.Fatalf("cut-off row: %+v", j)
	}
	// The next runner takes it at once, not after a lease or a backoff.
	ran := make(chan struct{})
	r2 := NewRunner(q, fast)
	r2.Handle("stuck", func(ctx context.Context, job Job) error {
		if job.Attempt != 1 {
			t.Errorf("attempt %d, want 1", job.Attempt)
		}
		close(ran)
		return nil
	})
	start(t, r2)
	select {
	case <-ran:
	case <-time.After(time.Second):
		t.Fatal("handed-back job not run")
	}
}

// Drain doesn't wait for a claim query stuck on the database.
func TestDrainDoesNotWaitForStuckClaim(t *testing.T) {
	pool, q := setupDB(t)
	r := NewRunner(q, fast)
	r.Handle("any", func(ctx context.Context, job Job) error { return nil })
	done := make(chan error, 1)
	go func() { done <- r.Run(context.Background()) }()
	eventually(t, 2*time.Second, "runner polling", func() bool {
		_, err := q.NextJobDueInMs(context.Background(), []string{"any"})
		return err == nil
	})
	time.Sleep(100 * time.Millisecond) // past the schedule upsert

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(context.Background(), "LOCK TABLE jobs IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}
	r.Wake()
	time.Sleep(200 * time.Millisecond) // the claim is now waiting on the lock

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	drained := make(chan error, 1)
	go func() { drained <- r.Drain(ctx) }()
	select {
	case err := <-drained:
		if err != nil {
			t.Fatalf("Drain: %v", err)
		}
	case <-time.After(time.Second):
		tx.Rollback(context.Background()) // let the claim and Drain finish
		<-drained
		t.Fatal("Drain waited for the stuck claim")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return")
	}
}

func TestDefaultBackoff(t *testing.T) {
	for attempt, want := range map[int32]time.Duration{
		1: 30 * time.Second, 2: time.Minute, 3: 2 * time.Minute, 5: 8 * time.Minute,
		6: 10 * time.Minute, 40: 10 * time.Minute, 1000: 10 * time.Minute,
	} {
		if got := defaultBackoff(attempt); got != want {
			t.Errorf("defaultBackoff(%d) = %v, want %v", attempt, got, want)
		}
	}
}

// A job that ignores its context holds Drain only cancelGrace past its
// deadline.
func TestDrainDoesNotWaitForStuckJobs(t *testing.T) {
	_, q := setupDB(t)
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	r := NewRunner(q, fast)
	r.Handle("ignores-ctx", func(ctx context.Context, job Job) error {
		close(started)
		<-release
		return nil
	})
	go r.Run(context.Background())
	enqueue(t, q, "ignores-ctx", "a")
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	begin := time.Now()
	if err := r.Drain(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Drain: %v", err)
	}
	if took := time.Since(begin); took > 100*time.Millisecond+cancelGrace+500*time.Millisecond {
		t.Fatalf("Drain took %v", took)
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

// Each poll is a jobs.poll trace; each run is its own job.run trace, linked
// to the poll that claimed it, with how it ended.
func TestSpans(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	defer otel.SetTracerProvider(prev)

	_, q := setupDB(t)
	r := NewRunner(q, fast)
	r.Handle("once", func(ctx context.Context, job Job) error {
		if job.Key == "fails" {
			return errors.New("lambda timed out")
		}
		return nil
	}, MaxAttempts(2), Backoff(func(int32) time.Duration { return 20 * time.Millisecond }))
	start(t, r)
	enqueue(t, q, "once", "ok")
	enqueue(t, q, "once", "fails")
	eventually(t, 3*time.Second, "both done", func() bool {
		return getJob(t, q, "once", "ok").DoneAt.Valid && getJob(t, q, "once", "fails").DoneAt.Valid
	})

	polls := map[trace.SpanID]bool{}
	var runs []sdktrace.ReadOnlySpan
	eventually(t, time.Second, "three job.run spans", func() bool {
		runs = runs[:0]
		for _, s := range rec.Ended() {
			switch s.Name() {
			case "jobs.poll":
				polls[s.SpanContext().SpanID()] = true
			case "job.run":
				runs = append(runs, s)
			}
		}
		return len(runs) == 3
	})

	outcomes := map[string]int{}
	for _, s := range runs {
		attrs := map[attribute.Key]attribute.Value{}
		for _, a := range s.Attributes() {
			attrs[a.Key] = a.Value
		}
		if s.Parent().IsValid() {
			t.Errorf("job.run has a parent; want its own trace")
		}
		if len(s.Links()) != 1 || !polls[s.Links()[0].SpanContext.SpanID()] {
			t.Errorf("job.run links %v, want the jobs.poll that claimed it", s.Links())
		}
		if _, ok := attrs["job.lag_ms"]; !ok {
			t.Errorf("job.run has no job.lag_ms")
		}
		key, outcome := attrs["job.key"].AsString(), attrs["job.outcome"].AsString()
		outcomes[key+":"+outcome]++
		if (outcome == outcomeDone) != (s.Status().Code == codes.Unset) {
			t.Errorf("%s/%s has status %v", key, outcome, s.Status())
		}
	}
	want := map[string]int{"ok:done": 1, "fails:retry": 1, "fails:given_up": 1}
	if fmt.Sprint(outcomes) != fmt.Sprint(want) {
		t.Errorf("outcomes %v, want %v", outcomes, want)
	}
}
