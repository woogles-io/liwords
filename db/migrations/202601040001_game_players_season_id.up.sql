BEGIN;

-- Add league_season_id column to game_players table to enable efficient player+season queries
-- NULL for non-league games
ALTER TABLE game_players ADD COLUMN league_season_id uuid;

-- Add foreign key constraint to ensure referential integrity
-- References league_seasons(uuid), not (id), since games.season_id is also UUID
ALTER TABLE game_players ADD CONSTRAINT fk_game_players_league_season
  FOREIGN KEY (league_season_id) REFERENCES league_seasons(uuid);

-- Backfill league_season_id ONLY for league games (much faster than updating all 20M rows)
-- This updates only games where season_id IS NOT NULL
UPDATE game_players gp
SET league_season_id = g.season_id
FROM games g
WHERE gp.game_uuid = g.uuid
  AND g.season_id IS NOT NULL;

-- Create PARTIAL index for fast player+season lookups on league games only
-- This keeps the index small and efficient (only indexes league games, not all 20M rows)
CREATE INDEX idx_game_players_player_league_season
  ON game_players(player_id, league_season_id, created_at DESC)
  WHERE league_season_id IS NOT NULL;

COMMIT;
