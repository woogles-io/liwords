BEGIN;

-- Drop the partial index
DROP INDEX IF EXISTS idx_game_players_player_league_season;

-- Drop the foreign key constraint
ALTER TABLE game_players DROP CONSTRAINT IF EXISTS fk_game_players_league_season;

-- Drop the league_season_id column
ALTER TABLE game_players DROP COLUMN IF EXISTS league_season_id;

COMMIT;
