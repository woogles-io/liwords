-- ============================================================================
-- REMOVE GAMES DIVISION-UNFINISHED INDEX - MANUAL EXECUTION REQUIRED
-- ============================================================================
--
-- To rollback, run this manually:
--
-- DROP INDEX CONCURRENTLY IF EXISTS idx_games_division_unfinished;
--
-- Note: DROP INDEX CONCURRENTLY also cannot run inside a transaction block.

-- No-op statement to make migration valid
SELECT 'games division-unfinished index removal documented - manual execution required' AS rollback_status;
