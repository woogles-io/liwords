DROP INDEX IF EXISTS idx_broadcast_games_completed_at;

ALTER TABLE broadcast_games
    DROP COLUMN IF EXISTS completed_at,
    DROP COLUMN IF EXISTS stats_computed_at,
    DROP COLUMN IF EXISTS stats;
