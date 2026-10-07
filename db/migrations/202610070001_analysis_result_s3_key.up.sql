-- Analysis results move to S3. result_s3_key points at the object holding a
-- job's accepted result; each accepted result gets its own key, so a rejected
-- submission can never replace it. NULL means the result (if any) is still in
-- the result column: jobs from before this change, until
-- cmd/backfill-analysis-s3 uploads them, and zero-turn games, which have no
-- object at all.
ALTER TABLE analysis_jobs ADD COLUMN result_s3_key TEXT;

-- Lets cmd/backfill-analysis-s3 page through jobs still to upload.
CREATE INDEX idx_analysis_jobs_result_to_upload ON analysis_jobs (id)
    WHERE result_s3_key IS NULL AND result IS NOT NULL;
