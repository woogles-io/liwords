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
-- Marks job as completed and returns processing duration
UPDATE analysis_jobs
SET
    status = 'completed',
    result_proto = $1,
    completed_at = NOW()
WHERE id = $2 AND claimed_by_user_uuid = $3 AND status IN ('claimed', 'processing')
RETURNING EXTRACT(EPOCH FROM (NOW() - claimed_at))::BIGINT * 1000 as duration_ms;

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
SELECT id, status, result_proto, error_message, completed_at, created_at
FROM analysis_jobs
WHERE game_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetUserJobCount :one
-- Get count of jobs completed by a user
SELECT COUNT(*) as total_jobs
FROM analysis_jobs
WHERE claimed_by_user_uuid = $1 AND completed_at IS NOT NULL;
