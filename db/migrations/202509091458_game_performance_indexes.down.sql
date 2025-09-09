-- ============================================================================  
-- REMOVE GAME PERFORMANCE INDEXES - MANUAL EXECUTION REQUIRED
-- ============================================================================
--
-- To rollback the performance indexes, run these commands manually:

-- 1. Remove user game lookup index - player 0
-- DROP INDEX CONCURRENTLY IF EXISTS idx_games_player0_filtered;

-- 2. Remove user game lookup index - player 1  
-- DROP INDEX CONCURRENTLY IF EXISTS idx_games_player1_filtered;

-- 3. Remove tournament game lookup index
-- DROP INDEX CONCURRENTLY IF EXISTS idx_games_tournament_filtered;

-- Note: DROP INDEX CONCURRENTLY also cannot run inside a transaction block
-- Removal time: Much faster than creation (typically seconds to minutes)
-- Downtime during removal: ZERO (thanks to CONCURRENTLY)

-- No-op statement to make migration valid
SELECT 'Game performance indexes removal documented - manual execution required' AS rollback_status;