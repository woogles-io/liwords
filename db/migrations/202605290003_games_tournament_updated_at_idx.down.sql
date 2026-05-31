-- Restore old single-column index if needed (run manually):
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_games_tournament_id ON games (tournament_id);
-- DROP INDEX CONCURRENTLY IF EXISTS idx_games_tournament_updated_at;

SELECT 'tournament index rollback: run CREATE/DROP INDEX CONCURRENTLY manually' AS status;
