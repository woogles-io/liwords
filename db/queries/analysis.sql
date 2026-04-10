-- name: ClaimNextJob :one
-- Claims the next available job atomically using FOR UPDATE SKIP LOCKED
UPDATE analysis_jobs
SET
    status = 'claimed',
    claimed_by_user_uuid = $1,
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
-- Marks job as completed and returns game_id and processing duration
UPDATE analysis_jobs
SET
    status = 'completed',
    result = $1,
    completed_at = NOW()
WHERE id = $2 AND claimed_by_user_uuid = $3 AND status IN ('claimed', 'processing')
RETURNING game_id, requested_by_user_uuid, EXTRACT(EPOCH FROM (NOW() - claimed_at))::BIGINT * 1000 as duration_ms;

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
-- Get most recent job for a game
SELECT id, game_id, status, config_json, result, error_message, completed_at, created_at
FROM analysis_jobs
WHERE game_id = $1
ORDER BY created_at DESC
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

-- name: GetAnalysisJobWithDetails :one
-- Get full details of an analysis job
SELECT
    id,
    game_id,
    status,
    requested_by_user_uuid,
    request_type,
    result,
    error_message,
    created_at,
    completed_at,
    priority
FROM analysis_jobs
WHERE id = $1;

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

-- name: GetJobByID :one
SELECT id, game_id, status, result
FROM analysis_jobs
WHERE id = $1;

-- name: ResetAnalysisJobKeepResult :exec
-- Resets job to pending but keeps result for JIT MI subtraction
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

-- name: GetVerticalOpenerJobs :many
-- Find completed jobs where the first turn was a vertical opening move
-- (column-first coordinates like A1, B3, etc. indicate vertical plays)
SELECT id, game_id
FROM analysis_jobs
WHERE status = 'completed'
  AND result->'turns'->0->>'playedMove' ~ '^[A-O][0-9]';
