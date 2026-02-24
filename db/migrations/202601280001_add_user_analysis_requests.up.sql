BEGIN;

-- Add field to track who requested the analysis (game player)
ALTER TABLE analysis_jobs ADD COLUMN requested_by_user_uuid TEXT;
ALTER TABLE analysis_jobs ADD COLUMN request_type TEXT DEFAULT 'automatic';
-- request_type: 'automatic' (league games), 'user_requested' (manual request)

-- Index for finding user's requested analyses
CREATE INDEX idx_analysis_jobs_requested_by ON analysis_jobs(requested_by_user_uuid)
WHERE requested_by_user_uuid IS NOT NULL;

-- Table for tracking daily analysis request limits (5 per day per user)
CREATE TABLE user_analysis_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_uuid TEXT NOT NULL,
    game_id TEXT NOT NULL,
    job_id UUID REFERENCES analysis_jobs(id) ON DELETE CASCADE,
    requested_at TIMESTAMPTZ DEFAULT NOW(),

    -- Prevent duplicate requests for same game
    UNIQUE(user_uuid, game_id)
);

-- Index for counting requests per day
CREATE INDEX idx_user_analysis_requests_daily
ON user_analysis_requests(user_uuid, requested_at);

COMMIT;
