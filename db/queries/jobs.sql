-- name: UpsertJobSchedule :exec
-- Registers a schedule. A row whose schedule is unchanged keeps its run_at,
-- so a restart does not run every schedule again; a changed schedule starts
-- over from the new first run.
INSERT INTO jobs (kind, key, schedule, run_at)
VALUES (@kind, '', @schedule, @first_run_at)
ON CONFLICT (kind, key) DO UPDATE
SET schedule = EXCLUDED.schedule,
    run_at = CASE WHEN jobs.schedule IS DISTINCT FROM EXCLUDED.schedule
                  THEN EXCLUDED.run_at ELSE jobs.run_at END,
    done_at = NULL;

-- name: EnqueueJob :exec
-- Adds a one-off job, or makes a finished one with the same kind and key due
-- again. A job not yet finished is left as it is.
INSERT INTO jobs (kind, key, run_at)
VALUES (@kind, @key, @run_at)
ON CONFLICT (kind, key) DO UPDATE
SET run_at = EXCLUDED.run_at, done_at = NULL, attempts = 0, last_error = ''
WHERE jobs.done_at IS NOT NULL;

-- name: ClaimDueJobs :many
-- Claims up to max_jobs due rows of the given kinds. A row is due when its
-- run_at has passed and nobody holds its lease.
UPDATE jobs
SET lease_until = NOW() + make_interval(secs => @lease_seconds),
    lease_token = @lease_token,
    attempts = attempts + 1
WHERE id IN (
    SELECT id FROM jobs
    WHERE done_at IS NULL
      AND kind = ANY(@kinds::text[])
      AND run_at <= NOW()
      AND (lease_until IS NULL OR lease_until < NOW())
    ORDER BY run_at
    LIMIT @max_jobs
    FOR UPDATE SKIP LOCKED
)
RETURNING id, kind, key, schedule, attempts;

-- name: RenewJobLease :execrows
-- Returns 0 rows when the lease was lost: it expired and another runner
-- claimed the row.
UPDATE jobs
SET lease_until = NOW() + make_interval(secs => @lease_seconds)
WHERE id = @id AND lease_token = @lease_token;

-- name: FinishScheduledRun :execrows
-- Records a schedule run, successful or not, and sets its next run.
UPDATE jobs
SET lease_until = NULL, lease_token = NULL,
    run_at = @next_run_at,
    attempts = 0,
    last_run_at = NOW(), last_ms = @last_ms, last_error = @last_error
WHERE id = @id AND lease_token = @lease_token;

-- name: FinishJob :execrows
-- Marks a one-off job done.
UPDATE jobs
SET lease_until = NULL, lease_token = NULL, done_at = NOW(),
    last_run_at = NOW(), last_ms = @last_ms, last_error = @last_error
WHERE id = @id AND lease_token = @lease_token;

-- name: RetryJob :execrows
-- Records a one-off job failed attempt, and when to try again.
UPDATE jobs
SET lease_until = NULL, lease_token = NULL,
    run_at = NOW() + make_interval(secs => @retry_in_seconds),
    last_run_at = NOW(), last_ms = @last_ms, last_error = @last_error
WHERE id = @id AND lease_token = @lease_token;

-- name: NextJobDueInMs :one
-- How long until a row of the given kinds can next be claimed, in
-- milliseconds, or a day when there are none. Computed in the database, so
-- the runner clock does not matter.
SELECT COALESCE(
    (EXTRACT(EPOCH FROM (MIN(GREATEST(run_at, COALESCE(lease_until, run_at))) - NOW())) * 1000)::bigint,
    86400000)::bigint AS due_in_ms
FROM jobs
WHERE done_at IS NULL AND kind = ANY(@kinds::text[]);

-- name: GetJob :one
SELECT id, kind, key, schedule, run_at, lease_until, done_at, attempts,
       last_run_at, last_ms, last_error
FROM jobs
WHERE kind = @kind AND key = @key;
