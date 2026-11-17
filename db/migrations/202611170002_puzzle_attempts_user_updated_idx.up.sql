-- For the player's most recent attempt in a lexicon (GetStartPuzzleId):
-- ORDER BY updated_at DESC LIMIT 1 over idx_puzzle_attempts_user read and
-- sorted every attempt the player had made -- 15k rows and 730 ms for the
-- heaviest player. With updated_at in the index the scan walks back from the
-- newest and stops at the first attempt in the lexicon.
--
-- One statement per file: CREATE INDEX CONCURRENTLY cannot run inside a
-- transaction block, and golang-migrate sends a file as one query.
-- CONCURRENTLY waits out the daily pg_dump (18:00 to ~22:00 UTC) if started
-- during it.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_puzzle_attempts_user_updated
    ON puzzle_attempts (user_id, updated_at DESC);
