-- Analysis results live in the result store (S3, see result_s3_key), and the
-- summary columns hold everything queried. Nothing reads or writes result any
-- more. Dropping it also drops idx_analysis_jobs_result_to_upload, which
-- referenced it.
ALTER TABLE analysis_jobs DROP COLUMN result;
