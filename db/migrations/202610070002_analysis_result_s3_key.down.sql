DROP INDEX IF EXISTS idx_analysis_jobs_result_to_upload;
ALTER TABLE analysis_jobs DROP COLUMN IF EXISTS result_s3_key;
