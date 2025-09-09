-- Drop game_metadata table
DROP INDEX IF EXISTS idx_game_metadata_tournament;
DROP INDEX IF EXISTS idx_game_metadata_created_at;
DROP TABLE IF EXISTS game_metadata;