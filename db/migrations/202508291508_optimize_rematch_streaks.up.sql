BEGIN;

-- Add original_request_id column to game_players table for efficient rematch streak queries
ALTER TABLE game_players ADD COLUMN original_request_id text;

-- Create index for efficient lookups by original_request_id
CREATE INDEX idx_game_players_orig_req ON game_players(original_request_id);

-- Drop the old rematch indexes since we'll query game_players instead of past_games/games
-- XXX: Drop these later, once we've completed the full migration.
-- DROP INDEX IF EXISTS idx_past_games_rematch_req_idx;
-- DROP INDEX IF EXISTS rematch_req_idx;

COMMIT;