ALTER TABLE analysis_jobs
    DROP COLUMN IF EXISTS player0_mistake_index,
    DROP COLUMN IF EXISTS player1_mistake_index,
    DROP COLUMN IF EXISTS analysis_version;
