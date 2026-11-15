// Package jobs runs background work once across all API servers.
//
// Every server runs a Runner. Work is rows in the jobs table, of two sorts:
//
//   - Schedules (Runner.Schedule): periodic work such as adjudication. One
//     row per kind; after each run, successful or not, its run_at moves to
//     the schedule's next time.
//   - One-off jobs (Runner.Handle, Enqueue): work that must happen once,
//     retried with backoff until it succeeds or runs out of attempts.
//
// A runner claims due rows (FOR UPDATE SKIP LOCKED), setting a lease, and
// renews the lease while the job runs. Another runner can take a row only
// once its lease has expired, so each run happens on one server, and the
// work of a server that dies is taken over within a lease. Coordination is
// only through Postgres.
//
// Handlers must be safe to run twice: a runner can lose its lease (a long
// pause, a lost database connection) while the job still runs, and a job
// cut off by a shutdown runs again.
//
// The runner doesn't poll on a fixed interval. It sleeps until the next row
// can be claimed, at most MaxSleep, or until Wake.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// Job is one run of a row, as its handler sees it.
type Job struct {
	ID   int64
	Kind string
	// Key tells apart one-off jobs of the same kind. It is "" for schedules.
	Key string
	// Attempt is 1 for the first run of a one-off job and goes up with each
	// retry. Schedules always get 1.
	Attempt int32

	// runAt is the time the row was due when it was claimed.
	runAt time.Time
}

// Handler does a job's work. A returned error, or a panic, is recorded on
// the row; a one-off job is then retried.
type Handler func(ctx context.Context, job Job) error

// Options configures a Runner. Zero values take the defaults.
type Options struct {
	// Lease is how long a claim lasts without renewal: how long another
	// runner waits before taking over the work of one that died. Renewed
	// every Lease/3 while the job runs. Default 30s.
	Lease time.Duration
	// MaxSleep bounds how long the runner sleeps between claims. Default
	// 30s.
	MaxSleep time.Duration
	// Concurrency is how many jobs run at once. Default 8.
	Concurrency int
}

func (o Options) withDefaults() Options {
	if o.Lease <= 0 {
		o.Lease = 30 * time.Second
	}
	if o.MaxSleep <= 0 {
		o.MaxSleep = 30 * time.Second
	}
	if o.Concurrency <= 0 {
		o.Concurrency = 8
	}
	return o
}

var tracer = otel.Tracer("jobs")

// errorPause is how long the runner waits after the database fails it.
const errorPause = 5 * time.Second

// finishTimeout bounds recording a run's outcome, which is done even when
// the job's context has ended.
const finishTimeout = 5 * time.Second

type kindConfig struct {
	handler     Handler
	schedule    Schedule // nil for one-off jobs
	maxAttempts int32
	backoff     func(attempt int32) time.Duration
	onGiveUp    func(ctx context.Context, job Job, err error)
}

// HandleOption configures a one-off kind.
type HandleOption func(*kindConfig)

// MaxAttempts is how many times a one-off job runs before it is given up.
// Default 5.
func MaxAttempts(n int) HandleOption {
	return func(k *kindConfig) { k.maxAttempts = int32(n) }
}

// Backoff says how long to wait before retrying a one-off job after the
// given failed attempt. Default: 30s, doubling, at most 10 minutes.
func Backoff(f func(attempt int32) time.Duration) HandleOption {
	return func(k *kindConfig) { k.backoff = f }
}

// OnGiveUp is called once a one-off job has failed its last attempt, with
// the last error. The row is then marked done, with that error.
func OnGiveUp(f func(ctx context.Context, job Job, err error)) HandleOption {
	return func(k *kindConfig) { k.onGiveUp = f }
}

func defaultBackoff(attempt int32) time.Duration {
	d := 30 * time.Second * time.Duration(math.Pow(2, float64(attempt-1)))
	return min(d, 10*time.Minute)
}

// Runner claims and runs jobs. Register kinds with Schedule and Handle,
// then call Run.
type Runner struct {
	q    *models.Queries
	opts Options

	kinds     map[string]*kindConfig
	kindNames []string

	wake  chan struct{}
	slots chan struct{}

	// mu makes claiming and draining exclusive: once Drain has set
	// draining, nothing more is claimed or started.
	mu       sync.Mutex
	draining bool
	running  sync.WaitGroup

	// jobsCtx is the parent of every job's context. Drain cancels it when
	// its own context ends with jobs still running.
	jobsCtx    context.Context
	cancelJobs context.CancelFunc

	started atomic.Bool
}

// NewRunner makes a runner that uses q.
func NewRunner(q *models.Queries, opts Options) *Runner {
	ctx, cancel := context.WithCancel(context.Background())
	opts = opts.withDefaults()
	return &Runner{
		q:          q,
		opts:       opts,
		kinds:      map[string]*kindConfig{},
		wake:       make(chan struct{}, 1),
		slots:      make(chan struct{}, opts.Concurrency),
		jobsCtx:    ctx,
		cancelJobs: cancel,
	}
}

// Schedule registers a scheduled kind.
func (r *Runner) Schedule(kind string, s Schedule, h Handler) {
	r.register(kind, &kindConfig{handler: h, schedule: s})
}

// Handle registers a one-off kind, whose jobs are added with Enqueue.
func (r *Runner) Handle(kind string, h Handler, opts ...HandleOption) {
	k := &kindConfig{handler: h, maxAttempts: 5, backoff: defaultBackoff}
	for _, o := range opts {
		o(k)
	}
	r.register(kind, k)
}

func (r *Runner) register(kind string, k *kindConfig) {
	if r.started.Load() {
		panic("jobs: register " + kind + " after Run")
	}
	if _, ok := r.kinds[kind]; ok {
		panic("jobs: " + kind + " registered twice")
	}
	r.kinds[kind] = k
	r.kindNames = append(r.kindNames, kind)
}

// Enqueue adds a one-off job, due at runAt. A job of the same kind and key
// that isn't finished yet is left as it is. Pass a q bound to a transaction
// to add the job with the change that calls for it.
func Enqueue(ctx context.Context, q *models.Queries, kind, key string, runAt time.Time) error {
	return q.EnqueueJob(ctx, models.EnqueueJobParams{
		Kind:  kind,
		Key:   key,
		RunAt: pgtype.Timestamptz{Time: runAt, Valid: true},
	})
}

// Wake makes the runner look for due jobs now, rather than when it next
// wakes on its own. For jobs this server just enqueued.
func (r *Runner) Wake() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

// Run registers the schedules, then claims and runs jobs until Drain is
// called or ctx ends. Handlers get contexts derived from ctx's values, but
// not its cancellation: they end when Drain runs out of time.
func (r *Runner) Run(ctx context.Context) error {
	if r.started.Swap(true) {
		panic("jobs: Run called twice")
	}
	r.mu.Lock()
	r.cancelJobs()
	r.jobsCtx, r.cancelJobs = context.WithCancel(context.WithoutCancel(ctx))
	r.mu.Unlock()
	if err := r.upsertSchedules(ctx); err != nil {
		return err
	}
	log := zerolog.Ctx(ctx)
	log.Info().Strs("kinds", r.kindNames).Msg("jobs-runner-started")
	for {
		sleep, draining := r.poll(ctx)
		if draining {
			return nil
		}
		t := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil
		case <-r.wake:
		case <-t.C:
		}
		t.Stop()
	}
}

var errDraining = errors.New("draining")

// poll claims and starts due jobs, and says how long to sleep before the
// next poll. Each poll is a jobs.poll trace, whose claim and next-due queries
// are its children; each job it starts is a job.run trace linked to it.
func (r *Runner) poll(ctx context.Context) (sleep time.Duration, draining bool) {
	ctx, span := tracer.Start(ctx, "jobs.poll", trace.WithNewRoot())
	defer span.End()
	claimed, full, err := r.claimAndStart(ctx)
	span.SetAttributes(attribute.Int("jobs.claimed", claimed))
	switch {
	case errors.Is(err, errDraining):
		span.SetAttributes(attribute.Bool("jobs.draining", true))
		return 0, true
	case err != nil:
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		zerolog.Ctx(ctx).Err(err).Msg("jobs-claim-failed")
		sleep = errorPause
	case full:
		// More may be due; a finishing job frees a slot and wakes the
		// runner.
		span.SetAttributes(attribute.Bool("jobs.full", true))
		sleep = r.opts.MaxSleep
	default:
		sleep = r.untilNextDue(ctx)
	}
	span.SetAttributes(attribute.Int64("jobs.sleep_ms", sleep.Milliseconds()))
	return sleep, false
}

// claimAndStart claims as many due jobs as there are free slots and starts
// them. full is true when every slot is taken afterwards.
func (r *Runner) claimAndStart(ctx context.Context) (claimed int, full bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.draining {
		return 0, false, errDraining
	}
	free := cap(r.slots) - len(r.slots)
	if free == 0 {
		return 0, true, nil
	}
	token := uuid.New()
	rows, err := r.q.ClaimDueJobs(ctx, models.ClaimDueJobsParams{
		LeaseSeconds: r.opts.Lease.Seconds(),
		LeaseToken:   pgtype.UUID{Bytes: token, Valid: true},
		Kinds:        r.kindNames,
		MaxJobs:      int32(free),
	})
	if err != nil {
		return 0, false, err
	}
	for _, row := range rows {
		r.slots <- struct{}{}
		r.running.Add(1)
		go r.run(ctx, row, token)
	}
	return len(rows), len(rows) == free, nil
}

// untilNextDue is how long until a row can next be claimed, at most
// MaxSleep.
func (r *Runner) untilNextDue(ctx context.Context) time.Duration {
	ms, err := r.q.NextJobDueInMs(ctx, r.kindNames)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("jobs-next-due-failed")
		return errorPause
	}
	d := time.Duration(ms) * time.Millisecond
	// A few milliseconds late, so the row is due when the claim runs.
	d += 5 * time.Millisecond
	return max(min(d, r.opts.MaxSleep), 0)
}

func (r *Runner) upsertSchedules(ctx context.Context) error {
	now := time.Now()
	for _, kind := range r.kindNames {
		k := r.kinds[kind]
		if k.schedule == nil {
			continue
		}
		err := r.q.UpsertJobSchedule(ctx, models.UpsertJobScheduleParams{
			Kind:       kind,
			Schedule:   pgtype.Text{String: k.schedule.String(), Valid: true},
			FirstRunAt: pgtype.Timestamptz{Time: k.schedule.Next(now), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("register schedule %s: %w", kind, err)
		}
	}
	return nil
}

// run runs one claimed row and records the outcome.
func (r *Runner) run(ctx context.Context, row models.ClaimDueJobsRow, token uuid.UUID) {
	defer func() {
		<-r.slots
		r.running.Done()
		r.Wake()
	}()
	k := r.kinds[row.Kind]
	job := Job{ID: row.ID, Kind: row.Kind, Key: row.Key, Attempt: row.Attempts, runAt: row.RunAt.Time}
	if k.schedule != nil {
		job.Attempt = 1
	}

	jobCtx, cancel := context.WithCancel(r.jobsCtx)
	defer cancel()
	start := time.Now()
	jobCtx, span := tracer.Start(jobCtx, "job.run",
		trace.WithNewRoot(),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(trace.LinkFromContext(ctx)),
		trace.WithAttributes(
			attribute.Int64("job.id", job.ID),
			attribute.String("job.kind", job.Kind),
			attribute.String("job.key", job.Key),
			attribute.Int("job.attempt", int(job.Attempt)),
			// How long after it was due the job started: time spent
			// waiting for a runner, a free slot or an expired lease.
			attribute.Int64("job.lag_ms", start.Sub(job.runAt).Milliseconds())))
	if k.schedule != nil {
		span.SetAttributes(attribute.String("job.schedule", k.schedule.String()))
	}
	defer span.End()
	logger := zerolog.Ctx(ctx).With().Str("job-kind", job.Kind).Str("job-key", job.Key).
		Int32("job-attempt", job.Attempt).Logger()
	jobCtx = logger.WithContext(jobCtx)

	pgToken := pgtype.UUID{Bytes: token, Valid: true}
	stopRenewing := r.renewLease(jobCtx, cancel, job, pgToken, &logger)

	handlerStart := time.Now()
	err := runHandler(jobCtx, k.handler, job)
	elapsed := time.Since(handlerStart)
	lostLease := stopRenewing()

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Err(err).Dur("took", elapsed).Msg("job-failed")
	} else {
		logger.Debug().Dur("took", elapsed).Msg("job-done")
	}
	if lostLease {
		// Another runner may have the row now; its outcome is not ours to
		// record.
		span.SetAttributes(attribute.String("job.outcome", outcomeLeaseLost))
		return
	}

	finishCtx, cancelFinish := context.WithTimeout(context.WithoutCancel(jobCtx), finishTimeout)
	defer cancelFinish()
	outcome, next, ferr := r.finish(finishCtx, k, job, pgToken, elapsed, err)
	if ferr != nil {
		span.AddEvent("finish failed", trace.WithAttributes(attribute.String("error", ferr.Error())))
		logger.Err(ferr).Msg("job-finish-failed")
		outcome = outcomeUnrecorded
	}
	span.SetAttributes(attribute.String("job.outcome", outcome))
	if !next.IsZero() {
		span.SetAttributes(attribute.String("job.next_run_at", next.UTC().Format(time.RFC3339Nano)))
	}
}

// renewLease renews the row's lease every Lease/3 until the returned
// function is called. If the lease is lost, it cancels the job. The
// returned function reports whether it was.
func (r *Runner) renewLease(ctx context.Context, cancelJob context.CancelFunc, job Job,
	token pgtype.UUID, logger *zerolog.Logger) func() bool {

	var lost atomic.Bool
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(r.opts.Lease / 3)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
			}
			n, err := r.q.RenewJobLease(context.WithoutCancel(ctx), models.RenewJobLeaseParams{
				LeaseSeconds: r.opts.Lease.Seconds(),
				ID:           job.ID,
				LeaseToken:   token,
			})
			if err != nil {
				// Try again next tick; the lease outlasts two misses.
				trace.SpanFromContext(ctx).AddEvent("lease renew failed",
					trace.WithAttributes(attribute.String("error", err.Error())))
				logger.Err(err).Msg("job-lease-renew-failed")
				continue
			}
			if n == 0 {
				lost.Store(true)
				trace.SpanFromContext(ctx).AddEvent("lease lost")
				logger.Warn().Msg("job-lease-lost")
				cancelJob()
				return
			}
		}
	}()
	return func() bool {
		close(stop)
		<-done
		return lost.Load()
	}
}

// Outcomes of a run, in the job.run span's job.outcome.
const (
	outcomeDone       = "done"       // a one-off job succeeded
	outcomeRan        = "ran"        // a schedule ran (see the span status for errors)
	outcomeRetry      = "retry"      // a one-off job failed and will be retried
	outcomeGivenUp    = "given_up"   // a one-off job failed its last attempt
	outcomeLeaseLost  = "lease_lost" // another runner may have the row; nothing recorded
	outcomeUnrecorded = "unrecorded" // recording the outcome failed; the lease will expire
)

// finish records a run's outcome on its row. next is when the row is due
// again, if it is.
func (r *Runner) finish(ctx context.Context, k *kindConfig, job Job, token pgtype.UUID,
	elapsed time.Duration, runErr error) (outcome string, next time.Time, err error) {

	lastMs := pgtype.Int4{Int32: int32(min(elapsed.Milliseconds(), math.MaxInt32)), Valid: true}
	lastError := ""
	if runErr != nil {
		lastError = runErr.Error()
	}
	var n int64
	switch {
	case k.schedule != nil:
		outcome = outcomeRan
		next = nextRun(k.schedule, job.runAt, time.Now())
		n, err = r.q.FinishScheduledRun(ctx, models.FinishScheduledRunParams{
			NextRunAt:  pgtype.Timestamptz{Time: next, Valid: true},
			LastMs:     lastMs,
			LastError:  lastError,
			ID:         job.ID,
			LeaseToken: token,
		})
	case runErr == nil || job.Attempt >= k.maxAttempts:
		outcome = outcomeDone
		if runErr != nil {
			outcome = outcomeGivenUp
		}
		n, err = r.q.FinishJob(ctx, models.FinishJobParams{
			LastMs:     lastMs,
			LastError:  lastError,
			ID:         job.ID,
			LeaseToken: token,
		})
		if err == nil && n == 1 && runErr != nil && k.onGiveUp != nil {
			zerolog.Ctx(ctx).Error().Err(runErr).Msg("job-given-up")
			k.onGiveUp(ctx, job, runErr)
		}
	default:
		outcome = outcomeRetry
		wait := k.backoff(job.Attempt)
		next = time.Now().Add(wait)
		n, err = r.q.RetryJob(ctx, models.RetryJobParams{
			RetryInSeconds: wait.Seconds(),
			LastMs:         lastMs,
			LastError:      lastError,
			ID:             job.ID,
			LeaseToken:     token,
		})
	}
	if err != nil {
		return "", time.Time{}, err
	}
	if n == 0 {
		zerolog.Ctx(ctx).Warn().Msg("job-lease-lost-before-finish")
		return outcomeLeaseLost, time.Time{}, nil
	}
	return outcome, next, nil
}

// runHandler runs h, turning a panic into an error, so one bad job can't
// take the server down.
func runHandler(ctx context.Context, h Handler, job Job) (err error) {
	defer func() {
		if p := recover(); p != nil {
			zerolog.Ctx(ctx).Error().Interface("panic", p).Str("stack", string(debug.Stack())).
				Msg("job-panicked")
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return h(ctx, job)
}

// cancelGrace is how long Drain waits for cancelled jobs to return, so the
// events they send still reach the bus before it drains.
const cancelGrace = time.Second

// Drain stops claiming and waits for the running jobs. If ctx ends first,
// it cancels their contexts, waits up to cancelGrace for them to return, and
// returns ctx's error. A cancelled run is recorded like a failed one: a
// schedule runs again at its next time, a one-off job is retried.
func (r *Runner) Drain(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "jobs.drain")
	defer span.End()
	r.mu.Lock()
	r.draining = true
	running := len(r.slots)
	r.mu.Unlock()
	span.SetAttributes(attribute.Int("jobs.running", running))
	r.Wake()

	done := make(chan struct{})
	go func() {
		r.running.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		r.mu.Lock()
		r.cancelJobs()
		r.mu.Unlock()
		cancelled := len(r.slots)
		span.RecordError(ctx.Err())
		span.SetStatus(codes.Error, ctx.Err().Error())
		span.SetAttributes(attribute.Int("jobs.cancelled", cancelled))
		select {
		case <-done:
		case <-time.After(cancelGrace):
			// Handlers that ignore their context. Their rows are taken
			// again once their leases expire.
			span.SetAttributes(attribute.Int("jobs.still_running", len(r.slots)))
		}
		log.Warn().Int("cancelled", cancelled).Int("still-running", len(r.slots)).
			Msg("jobs-drain-timed-out")
		return ctx.Err()
	}
}

// nextRun is a schedule's next run after the one due at runAt. Due times are
// compared with the database clock but computed with this server's: a server
// whose clock is behind the database's would otherwise get back the time it
// just ran, and run it again. Counting from the later of the two never
// repeats a run.
func nextRun(s Schedule, runAt, now time.Time) time.Time {
	if runAt.After(now) {
		return s.Next(runAt)
	}
	return s.Next(now)
}
