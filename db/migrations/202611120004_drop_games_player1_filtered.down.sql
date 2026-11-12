-- Run by hand if needed:
-- CREATE INDEX CONCURRENTLY idx_games_player1_filtered ON games (player1_id, id DESC)
--   WHERE game_end_reason NOT IN (0, 5, 7);
SELECT 1;
