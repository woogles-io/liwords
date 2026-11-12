-- Rebuilding takes minutes on the live table; see 202509091458 for the
-- original. Run by hand if needed:
-- CREATE INDEX CONCURRENTLY idx_games_player0_filtered ON games (player0_id, id DESC)
--   WHERE game_end_reason NOT IN (0, 5, 7);
SELECT 1;
