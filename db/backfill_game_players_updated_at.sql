-- Backfill game_players.updated_at for rows where it's NULL.
-- Sets it to games.updated_at (the game's last modification time).
-- This may take a while on large tables; consider running during low traffic.
UPDATE game_players gp
SET updated_at = g.updated_at
FROM games g
WHERE gp.game_uuid = g.uuid
  AND gp.updated_at IS NULL;
