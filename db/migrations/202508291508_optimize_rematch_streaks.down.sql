BEGIN;

-- Recreate the original rematch indexes
-- CREATE INDEX idx_past_games_rematch_req_idx
--     ON public.past_games USING hash (((quickdata ->> 'o'::text)));
-- CREATE INDEX rematch_req_idx ON public.games USING hash (((quickdata ->> 'o'::text)));

-- Drop the game_players index
DROP INDEX IF EXISTS idx_game_players_orig_req;

-- Remove the original_request_id column from game_players
ALTER TABLE game_players DROP COLUMN IF EXISTS original_request_id;

COMMIT;