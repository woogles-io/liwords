-- Analysis job queue table
CREATE TABLE analysis_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id TEXT NOT NULL,

    -- Job state
    status TEXT NOT NULL DEFAULT 'pending',
    -- Status: pending, claimed, processing, completed, failed

    -- Worker tracking (user who claimed the job)
    claimed_by_user_uuid TEXT,
    claimed_at TIMESTAMPTZ,
    heartbeat_at TIMESTAMPTZ,

    -- Configuration (stored as JSON)
    config_json JSONB NOT NULL,

    -- Results (protobuf bytes)
    result_proto BYTEA,

    -- Error tracking
    error_message TEXT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,

    -- Priority (higher = more urgent)
    priority INT DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    CONSTRAINT valid_status CHECK (status IN ('pending', 'claimed', 'processing', 'completed', 'failed'))
);

-- Index for claiming jobs (pending jobs by priority)
CREATE INDEX idx_analysis_jobs_pending
ON analysis_jobs(priority DESC, created_at ASC)
WHERE status = 'pending';

-- Index for heartbeat timeout detection
CREATE INDEX idx_analysis_jobs_stale
ON analysis_jobs(heartbeat_at)
WHERE status IN ('claimed', 'processing');

-- Index for looking up job by game
CREATE INDEX idx_analysis_jobs_game_id ON analysis_jobs(game_id);

-- Index for tracking user's jobs
CREATE INDEX idx_analysis_jobs_user ON analysis_jobs(claimed_by_user_uuid);
