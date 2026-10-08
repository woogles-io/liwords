-- name: ClaimNextJob :one
-- Claims the next available job atomically using FOR UPDATE SKIP LOCKED.
-- A worker already holding @max_active jobs gets none, so one account can't
-- claim the whole queue and let it time out into failure. Two simultaneous
-- claims by the same worker can each see the old count; overshooting by one
-- is harmless.
UPDATE analysis_jobs
SET
    status = 'claimed',
    claimed_by_user_uuid = sqlc.arg(worker),
    claimed_at = NOW(),
    heartbeat_at = NOW()
WHERE id = (
    SELECT id
    FROM analysis_jobs
    WHERE status = 'pending'
    ORDER BY priority DESC, created_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
AND (
    SELECT COUNT(*)
    FROM analysis_jobs
    WHERE claimed_by_user_uuid = sqlc.arg(worker)
      AND status IN ('claimed', 'processing')
) < sqlc.arg(max_active)::INT
RETURNING id, game_id, config_json;

-- name: UpdateHeartbeat :exec
-- Updates heartbeat timestamp and transitions to processing state
UPDATE analysis_jobs
SET
    heartbeat_at = NOW(),
    status = CASE
        WHEN status = 'claimed' THEN 'processing'
        ELSE status
    END
WHERE id = $1 AND claimed_by_user_uuid = $2;

-- name: CompleteJob :one
-- Marks job as completed and returns game_id, processing duration and the
-- job's previous result_s3_key (so the caller can delete a replaced object).
-- With a result store, the caller passes the uploaded object's key and a NULL
-- result; without one, the result itself and a NULL key. The summary columns
-- are copied from the result by the caller; see the 202610070001 migration for
-- what NULL means in each.
WITH prev AS (
    SELECT j.id AS job_id, j.result_s3_key AS previous_s3_key
    FROM analysis_jobs j
    WHERE j.id = sqlc.arg(id)
    FOR UPDATE
)
UPDATE analysis_jobs aj
SET
    status = 'completed',
    result = sqlc.narg(result),
    result_s3_key = sqlc.narg(result_s3_key),
    player0_mistake_index = sqlc.narg(player0_mistake_index),
    player1_mistake_index = sqlc.narg(player1_mistake_index),
    analysis_version = sqlc.arg(analysis_version),
    completed_at = NOW()
FROM prev
WHERE aj.id = prev.job_id AND aj.claimed_by_user_uuid = sqlc.arg(claimed_by_user_uuid)
  AND aj.status IN ('claimed', 'processing')
RETURNING aj.game_id, aj.requested_by_user_uuid,
    EXTRACT(EPOCH FROM (NOW() - aj.claimed_at))::BIGINT * 1000 as duration_ms,
    prev.previous_s3_key;

-- name: GetJobClaim :one
-- Who holds a job, checked before uploading a submitted result so a stale
-- submission doesn't upload an object only to have CompleteJob reject it.
SELECT game_id, status, claimed_by_user_uuid
FROM analysis_jobs
WHERE id = $1;

-- name: FailJob :exec
-- Marks job as failed with error message
UPDATE analysis_jobs
SET
    status = 'failed',
    error_message = $1,
    completed_at = NOW()
WHERE id = $2 AND claimed_by_user_uuid = $3;

-- name: ReclaimStaleJobs :exec
-- Reclaim jobs that haven't sent heartbeat in timeout period
UPDATE analysis_jobs
SET
    status = CASE
        WHEN retry_count >= max_retries THEN 'failed'
        ELSE 'pending'
    END,
    claimed_by_user_uuid = NULL,
    retry_count = retry_count + 1,
    error_message = CASE
        WHEN retry_count >= max_retries THEN 'Max retries - worker timeout'
        ELSE NULL
    END
WHERE status IN ('claimed', 'processing')
  AND heartbeat_at < NOW() - INTERVAL '2 minutes';

-- name: CreateAnalysisJob :one
-- Create a new analysis job
INSERT INTO analysis_jobs (game_id, config_json, priority)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetJobByGameID :one
-- Get most recent job for a game, along with the volunteer who ran it.
-- claimed_at is returned so callers can derive how long the run took
-- (completed_at - claimed_at) and how long it waited (claimed_at - created_at).
SELECT
    aj.id, aj.game_id, aj.status, aj.config_json, aj.result, aj.error_message,
    aj.completed_at, aj.created_at, aj.claimed_at, aj.analysis_version, aj.result_s3_key,
    COALESCE(u.username, '') as analyzed_by_username
FROM analysis_jobs aj
LEFT JOIN users u ON u.uuid = aj.claimed_by_user_uuid
WHERE aj.game_id = $1
ORDER BY aj.created_at DESC
LIMIT 1;

-- name: GetUserJobCount :one
-- Get count of jobs completed by a user
SELECT COUNT(*) as total_jobs
FROM analysis_jobs
WHERE claimed_by_user_uuid = $1 AND completed_at IS NOT NULL;

-- name: CreateUserRequestedJob :one
-- Create a new user-requested analysis job
INSERT INTO analysis_jobs (game_id, config_json, priority, requested_by_user_uuid, request_type)
VALUES ($1, $2, $3, $4, 'user_requested')
RETURNING id;

-- name: RecordUserAnalysisRequest :exec
-- Record that a user requested analysis for a game
INSERT INTO user_analysis_requests (user_uuid, game_id, job_id)
VALUES ($1, $2, $3);

-- name: GetUserRequestCountToday :one
-- Get count of analysis requests by user in last 24 hours
SELECT COUNT(*) as request_count
FROM user_analysis_requests
WHERE user_uuid = $1
  AND requested_at > NOW() - INTERVAL '24 hours';

-- name: CheckExistingUserRequest :one
-- Check if user already requested analysis for this game
SELECT job_id
FROM user_analysis_requests
WHERE user_uuid = $1 AND game_id = $2
LIMIT 1;

-- name: GetQueuePosition :one
-- Get position of a job in the queue (1-indexed)
SELECT COUNT(*) + 1 as position
FROM analysis_jobs aj
WHERE aj.status = 'pending'
  AND (aj.priority > (SELECT priority FROM analysis_jobs target WHERE target.id = $1)
       OR (aj.priority = (SELECT priority FROM analysis_jobs target WHERE target.id = $1)
           AND aj.created_at < (SELECT created_at FROM analysis_jobs target WHERE target.id = $1)));

-- name: GetAdminAnalysisStats :one
-- Get overview stats for admin dashboard
SELECT
    COUNT(*) FILTER (WHERE status = 'completed') as total_completed,
    COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
    COUNT(*) FILTER (WHERE status IN ('claimed', 'processing')) as processing_count
FROM analysis_jobs;

-- name: GetAnalysisLeaderboard :many
-- Get top users who requested the most analyses
SELECT
    u.username,
    COUNT(*) as analysis_count
FROM analysis_jobs aj
JOIN users u ON u.uuid = aj.requested_by_user_uuid
WHERE aj.requested_by_user_uuid IS NOT NULL
GROUP BY u.uuid, u.username
ORDER BY analysis_count DESC
LIMIT $1;

-- name: GetContributorsLeaderboard :many
-- Get top users who contributed the most analyses (i.e. ran the worker)
SELECT
    u.username,
    COUNT(*) as analysis_count
FROM analysis_jobs aj
JOIN users u ON u.uuid = aj.claimed_by_user_uuid
WHERE aj.claimed_by_user_uuid IS NOT NULL
  AND aj.status = 'completed'
GROUP BY u.uuid, u.username
ORDER BY analysis_count DESC
LIMIT $1;

-- name: GetCompletedJobsList :many
-- Get paginated list of completed analysis jobs
SELECT
    aj.id as job_id,
    aj.game_id,
    aj.created_at,
    aj.claimed_at,
    aj.completed_at,
    COALESCE(aj.request_type, 'automatic') as request_type,
    COALESCE(u.username, '') as requested_by_username
FROM analysis_jobs aj
LEFT JOIN users u ON u.uuid = aj.requested_by_user_uuid
WHERE aj.status = 'completed'
ORDER BY aj.completed_at DESC
LIMIT $1 OFFSET $2;

-- name: GetTotalCompletedCount :one
-- Get total count of completed analysis jobs
SELECT COUNT(*) as total
FROM analysis_jobs
WHERE status = 'completed';

-- name: ResetAnalysisJobKeepResult :exec
-- Resets job to pending but keeps result, so league standings keep counting
-- the old analysis until the new one replaces it
UPDATE analysis_jobs
SET status = 'pending',
    error_message = NULL,
    claimed_by_user_uuid = NULL,
    claimed_at = NULL,
    heartbeat_at = NULL,
    completed_at = NULL,
    retry_count = 0
WHERE id = $1;

-- name: ResetAnalysisJobWithPriority :exec
-- Resets job and sets custom priority (for batch requeue)
UPDATE analysis_jobs
SET status = 'pending',
    error_message = NULL,
    claimed_by_user_uuid = NULL,
    claimed_at = NULL,
    heartbeat_at = NULL,
    completed_at = NULL,
    retry_count = 0,
    priority = $2
WHERE id = $1;

-- name: GetAnalyzedGameIds :many
-- Get which of the given game IDs have completed analysis
SELECT game_id
FROM analysis_jobs
WHERE game_id = ANY($1::text[])
  AND status = 'completed';

-- name: BackfillAnalysisSummaryColumns :execrows
-- Copies the summary fields out of the stored result for up to $1 jobs that
-- have a result but no columns yet. protojson drops zero values, so a missing
-- mistakeIndex inside a present summary is 0 and a missing analysisVersion is
-- 0 (v0). Returns the number of jobs updated; 0 means done.
UPDATE analysis_jobs aj
SET player0_mistake_index = CASE WHEN aj.result->'playerSummaries'->0 IS NOT NULL
        THEN COALESCE(aj.result->'playerSummaries'->0->>'mistakeIndex', '0')::DOUBLE PRECISION END,
    player1_mistake_index = CASE WHEN aj.result->'playerSummaries'->1 IS NOT NULL
        THEN COALESCE(aj.result->'playerSummaries'->1->>'mistakeIndex', '0')::DOUBLE PRECISION END,
    analysis_version = COALESCE(aj.result->>'analysisVersion', '0')::INT
WHERE aj.id IN (
    SELECT id FROM analysis_jobs
    WHERE result IS NOT NULL AND analysis_version IS NULL
    LIMIT sqlc.arg(batch_size)::INT
    FOR UPDATE SKIP LOCKED
);

-- name: ListAnalysisResultsToUpload :many
-- Jobs whose result is still only in the result column, in id order after
-- @after, for cmd/backfill-analysis-s3. Zero-turn results have nothing worth
-- an object and are skipped.
SELECT id, game_id, result
FROM analysis_jobs
WHERE result_s3_key IS NULL
  AND result IS NOT NULL
  AND id > sqlc.arg(after)::uuid
  AND jsonb_array_length(COALESCE(result->'turns', '[]'::jsonb)) > 0
ORDER BY id
LIMIT sqlc.arg(batch_size)::INT;

-- name: SetAnalysisResultS3Key :execrows
-- Records an uploaded object for a job that has none yet. 0 rows means a
-- reanalysis stored its own result in the meantime; the caller then deletes
-- the object it uploaded.
UPDATE analysis_jobs
SET result_s3_key = sqlc.arg(result_s3_key)
WHERE id = sqlc.arg(id) AND result_s3_key IS NULL;

-- name: ListAnalysisResultsToClear :many
-- Jobs that still hold a result in the result column but no longer need it:
-- uploaded ones, plus zero-turn ones, which have no object. Requires the
-- summary columns, which the MI queries read once result is gone.
SELECT id, game_id, result, result_s3_key
FROM analysis_jobs
WHERE result IS NOT NULL
  AND analysis_version IS NOT NULL
  AND id > sqlc.arg(after)::uuid
ORDER BY id
LIMIT sqlc.arg(batch_size)::INT;

-- name: ClearAnalysisResult :execrows
-- Drops the result column's copy once the caller has checked the object
-- matches. The key must still be the one checked.
UPDATE analysis_jobs
SET result = NULL
WHERE id = sqlc.arg(id)
  AND result IS NOT NULL
  AND result_s3_key IS NOT DISTINCT FROM sqlc.narg(result_s3_key);
