-- ============================================================================
-- GAME PERFORMANCE INDEXES - MANUAL EXECUTION REQUIRED
-- ============================================================================
--
-- This migration documents the performance indexes that need to be created
-- manually using CREATE INDEX CONCURRENTLY (which cannot run in transactions).
--
-- MANUAL EXECUTION INSTRUCTIONS:
-- Connect to your PostgreSQL database and run these commands individually:

-- 1. User game lookup optimization - player 0 index
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_games_player0_filtered
-- ON public.games (player0_id, id)
-- WHERE game_end_reason NOT IN (0, 5, 7);

-- 2. User game lookup optimization - player 1 index
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_games_player1_filtered
-- ON public.games (player1_id, id)
-- WHERE game_end_reason NOT IN (0, 5, 7);

-- 3. Tournament game lookup optimization
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_games_tournament_filtered
-- ON public.games (tournament_id, updated_at)
-- WHERE game_end_reason NOT IN (0, 5, 7) AND tournament_id IS NOT NULL;

-- Expected performance improvement: 100x+ for high-volume players (bots with 50k+ games)
-- Index creation time: 5-15 minutes per index on 10M+ game table
-- Downtime during creation: ZERO (thanks to CONCURRENTLY)

-- No-op statement to make migration valid
SELECT 'Game performance indexes documented - manual execution required' AS migration_status;