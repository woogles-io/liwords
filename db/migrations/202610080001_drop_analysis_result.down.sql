-- The data is gone for good; this only restores the (empty) column.
ALTER TABLE analysis_jobs ADD COLUMN result JSONB;
