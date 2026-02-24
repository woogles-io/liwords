BEGIN;

DROP TABLE IF EXISTS user_analysis_requests;
DROP INDEX IF EXISTS idx_analysis_jobs_requested_by;
ALTER TABLE analysis_jobs DROP COLUMN IF EXISTS request_type;
ALTER TABLE analysis_jobs DROP COLUMN IF EXISTS requested_by_user_uuid;

COMMIT;
