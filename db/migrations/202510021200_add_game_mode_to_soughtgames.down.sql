-- Remove game_mode column and index from soughtgames table
DROP INDEX IF EXISTS idx_soughtgames_game_mode;
ALTER TABLE soughtgames DROP COLUMN IF EXISTS game_mode;
