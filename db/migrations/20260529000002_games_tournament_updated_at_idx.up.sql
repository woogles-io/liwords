-- Replace the single-column tournament_id index with a composite (tournament_id, updated_at DESC)
-- so that GetRecentTourneyGames can stop at LIMIT without fetching and sorting all rows.
--
-- Run both statements manually outside a transaction:
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_games_tournament_updated_at
--   ON games (tournament_id, updated_at DESC)
--   WHERE tournament_id IS NOT NULL;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_games_tournament_id;

SELECT 'tournament index upgrade: run CREATE/DROP INDEX CONCURRENTLY manually' AS status;
