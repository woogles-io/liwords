-- Remove league metadata from games table
DROP INDEX IF EXISTS idx_games_league_division_id;
DROP INDEX IF EXISTS idx_games_season_id;
DROP INDEX IF EXISTS idx_games_league_id;

ALTER TABLE games DROP COLUMN IF EXISTS league_division_id;
ALTER TABLE games DROP COLUMN IF EXISTS season_id;
ALTER TABLE games DROP COLUMN IF EXISTS league_id;

-- Drop indexes
DROP INDEX IF EXISTS idx_leagues_slug;
DROP INDEX IF EXISTS idx_league_standings_user_id;
DROP INDEX IF EXISTS idx_league_standings_division_id;
DROP INDEX IF EXISTS idx_league_registrations_division_id;
DROP INDEX IF EXISTS idx_league_registrations_season_id;
DROP INDEX IF EXISTS idx_league_registrations_user_id;
DROP INDEX IF EXISTS idx_league_divisions_season_id;
DROP INDEX IF EXISTS idx_league_seasons_status;
DROP INDEX IF EXISTS idx_league_seasons_league_id;

-- Drop tables (in reverse order due to foreign keys)
DROP TABLE IF EXISTS league_standings;
DROP TABLE IF EXISTS league_registrations;
DROP TABLE IF EXISTS league_divisions;
DROP TABLE IF EXISTS league_seasons;
DROP TABLE IF EXISTS leagues;
