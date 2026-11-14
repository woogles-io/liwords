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
		full, err := r.claimAndStart(ctx)
		if errors.Is(err, errDraining) {
			return nil
		}
		var sleep time.Duration
		switch {
		case err != nil:
			log.Err(err).Msg("jobs-claim-failed")
			sleep = errorPause
		case full:
			// More may be due; a finishing job frees a slot and wakes the
			// runner.
			sleep = r.opts.MaxSleep
		default:
			sleep = r.untilNextDue(ctx)
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

// claimAndStart claims as many due jobs as there are free slots and starts
// them. full is true when every slot is taken afterwards.
func (r *Runner) claimAndStart(ctx context.Context) (full bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.draining {
		return false, errDraining
	}
	free := cap(r.slots) - len(r.slots)
	if free == 0 {
		return true, nil
	}
	token := uuid.New()
	rows, err := r.q.ClaimDueJobs(ctx, models.ClaimDueJobsParams{
		LeaseSeconds: r.opts.Lease.Seconds(),
		LeaseToken:   pgtype.UUID{Bytes: token, Valid: true},
		Kinds:        r.kindNames,
		MaxJobs:      int32(free),
	})
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		r.slots <- struct{}{}
		r.running.Add(1)
		go r.run(ctx, row, token)
	}
	return len(rows) == free, nil
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
	job := Job{ID: row.ID, Kind: row.Kind, Key: row.Key, Attempt: row.Attempts}
	if k.schedule != nil {
		job.Attempt = 1
	}

	jobCtx, cancel := context.WithCancel(r.jobsCtx)
	defer cancel()
	jobCtx, span := otel.Tracer("jobs").Start(jobCtx, "job.run")
	defer span.End()
	span.SetAttributes(
		attribute.String("job.kind", job.Kind),
		attribute.String("job.key", job.Key),
		attribute.Int("job.attempt", int(job.Attempt)))
	logger := zerolog.Ctx(ctx).With().Str("job-kind", job.Kind).Str("job-key", job.Key).
		Int32("job-attempt", job.Attempt).Logger()
	jobCtx = logger.WithContext(jobCtx)

	pgToken := pgtype.UUID{Bytes: token, Valid: true}
	stopRenewing := r.renewLease(jobCtx, cancel, job, pgToken, &logger)

	start := time.Now()
	err := runHandler(jobCtx, k.handler, job)
	elapsed := time.Since(start)
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
		span.SetAttributes(attribute.Bool("job.lease_lost", true))
		return
	}

	finishCtx, cancelFinish := context.WithTimeout(context.WithoutCancel(jobCtx), finishTimeout)
	defer cancelFinish()
	if ferr := r.finish(finishCtx, k, job, pgToken, elapsed, err); ferr != nil {
		logger.Err(ferr).Msg("job-finish-failed")
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
				logger.Err(err).Msg("job-lease-renew-failed")
				continue
			}
			if n == 0 {
				lost.Store(true)
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

func (r *Runner) finish(ctx context.Context, k *kindConfig, job Job, token pgtype.UUID,
	elapsed time.Duration, runErr error) error {

	lastMs := pgtype.Int4{Int32: int32(min(elapsed.Milliseconds(), math.MaxInt32)), Valid: true}
	lastError := ""
	if runErr != nil {
		lastError = runErr.Error()
	}
	var n int64
	var err error
	switch {
	case k.schedule != nil:
		n, err = r.q.FinishScheduledRun(ctx, models.FinishScheduledRunParams{
			NextRunAt:  pgtype.Timestamptz{Time: k.schedule.Next(time.Now()), Valid: true},
			LastMs:     lastMs,
			LastError:  lastError,
			ID:         job.ID,
			LeaseToken: token,
		})
	case runErr == nil || job.Attempt >= k.maxAttempts:
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
		n, err = r.q.RetryJob(ctx, models.RetryJobParams{
			RetryInSeconds: k.backoff(job.Attempt).Seconds(),
			LastMs:         lastMs,
			LastError:      lastError,
			ID:             job.ID,
			LeaseToken:     token,
		})
	}
	if err != nil {
		return err
	}
	if n == 0 {
		zerolog.Ctx(ctx).Warn().Msg("job-lease-lost-before-finish")
	}
	return nil
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

// Drain stops claiming and waits for the running jobs. If ctx ends first,
// it cancels their contexts and returns ctx's error; their rows are taken
// again once their leases expire.
func (r *Runner) Drain(ctx context.Context) error {
	ctx, span := otel.Tracer("jobs").Start(ctx, "jobs.drain")
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
		span.RecordError(ctx.Err())
		span.SetStatus(codes.Error, ctx.Err().Error())
		log.Warn().Int("running", len(r.slots)).Msg("jobs-drain-timed-out")
		return ctx.Err()
	}
}
