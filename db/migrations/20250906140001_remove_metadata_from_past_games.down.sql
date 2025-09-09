-- Restore game_request and tournament_data columns to past_games
-- Note: This will lose data if already migrated to game_metadata
ALTER TABLE past_games ADD COLUMN IF NOT EXISTS game_request BYTEA;
ALTER TABLE past_games ADD COLUMN IF NOT EXISTS tournament_data JSONB;