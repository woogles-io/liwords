-- Add game_mode to game_players so correspondence queries don't need to join games.
ALTER TABLE game_players ADD COLUMN IF NOT EXISTS game_mode SMALLINT NOT NULL DEFAULT 0;

-- Backfill only correspondence games (~296K rows, fast).
UPDATE game_players gp
SET game_mode = 1
WHERE EXISTS (
    SELECT 1 FROM games g
    WHERE g.uuid = gp.game_uuid
      AND (g.game_request->>'game_mode')::int = 1
);

-- Run this manually outside a transaction (cannot use CONCURRENTLY inside one):
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_game_players_player_correspondence
--   ON game_players (player_id, updated_at DESC)
--   WHERE game_mode = 1;

SELECT 'game_players.game_mode added and backfilled; run index creation manually' AS status;
