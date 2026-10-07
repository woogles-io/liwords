-- Copy the few fields we query out of analysis_jobs.result into columns, so
-- standings and game lists stop reading the full result (~38 kB, TOASTed)
-- and the result itself can later move to S3.
--
-- NULL mistake index: no summary for that player (not analyzed, or a
-- zero-turn game). A perfect game is 0, not NULL.
-- NULL analysis_version: result not yet copied over (see
-- cmd/backfill-analysis-columns); a v0 result is 0.
ALTER TABLE analysis_jobs
    ADD COLUMN player0_mistake_index DOUBLE PRECISION,
    ADD COLUMN player1_mistake_index DOUBLE PRECISION,
    ADD COLUMN analysis_version INT;
