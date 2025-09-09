  ---
  Phase 1: Foundation (No Behavior Changes)

  PR 1.1: Complete GORM → sqlc Migration

  Goal: Finish moving to sqlc without changing functionality
  - Complete all remaining GORM functions to sqlc
  - Keep exact same table structure
  - No new tables, no schema changes
  - Dependencies: None
  - Enables: All future changes
  - Safe to deploy: Yes, zero behavior change

  PR 1.2: Consolidate request columns

  Goal: Single source of truth for game requests
  - Migrate request (bytea) data to game_request (JSONB)
  - Update all code to use game_request only
  - Drop request column
  - Dependencies: PR 1.1
  - Enables: Cleaner data model
  - Safe to deploy: Yes, backwards compatible

  ---
  Phase 2: Add New Tables (Write-Only)

  PR 2.1: Create new tables

  Goal: Set up new table structure
  - Create past_games partitioned table
  - Create game_players table
  - Create initial partitions for next 6 months
  - No code changes yet - just schema
  - Dependencies: PR 1.1
  - Enables: PR 2.2, PR 3.1
  - Safe to deploy: Yes, unused tables

  PR 2.2: Dual-write to new tables

  Goal: Start populating new tables without reading from them
  - When game ends, write to BOTH:
    - games table (as before)
    - past_games + game_players (new)
  - Add migration_status column to games
  - Include original_request_id in game_players
  - Dependencies: PR 2.1
  - Enables: PR 2.3, PR 3.1
  - Safe to deploy: Yes, write-only

  PR 2.3: Backfill historical data

  Goal: Populate new tables with existing games
  - Script to migrate completed games to past_games + game_players
  - Run in batches with progress tracking
  - Set migration_status = 1 on migrated games
  - Keep all data in games table (no clearing yet)
  - Dependencies: PR 2.2
  - Enables: PR 3.1 (can't remove quickdata until game_players is populated)
  - Safe to deploy: Yes, additive only

  ---
  Phase 3: Optimize Current Structure

  PR 3.1: Replace Quickdata with JOINs

  Goal: Reduce storage by 400 bytes per game
  - Add final_scores INT[2] column to games table
  - Migrate scores from quickdata to new column
  - Update rematch queries to use game_players.original_request_id
  - Update game list queries to JOIN with users table for player names
  - Drop quickdata column
  - Dependencies: PR 2.3 (game_players must be populated!)
  - Enables: Smaller games table
  - Safe to deploy: Yes, after game_players is fully populated

  PR 3.2: Add performance indexes

  Goal: Optimize for new query patterns
  - Add index on game_players(original_request_id) for rematches
  - Add index on games(created_at, game_end_reason) for analytics
  - Add index on games((game_request->>'lexicon')) for lexicon queries
  - Dependencies: PR 3.1
  - Enables: Better performance
  - Safe to deploy: Yes, just indexes

  ---
  Phase 4: Start Reading from New Tables

  PR 4.1: Add dual-read with feature flag

  Goal: Safely switch to new tables
  - Add USE_PAST_GAMES_TABLE environment variable
  - Implement dual-mode queries
  - Default to false (keep using games table)
  - Dependencies: PR 2.3 (tables must be populated)
  - Enables: PR 4.2
  - Safe to deploy: Yes, flag defaults to old behavior

  PR 4.2: Enable new tables in staging

  Goal: Test new query paths
  - Set USE_PAST_GAMES_TABLE=true in staging
  - Monitor performance and errors
  - Dependencies: PR 4.1
  - Enables: PR 4.3
  - Safe to deploy: Staging only

  PR 4.3: Production rollout

  Goal: Switch production to new tables
  - Gradual rollout with monitoring
  - Dependencies: PR 4.2
  - Enables: PR 5.1
  - Safe to deploy: Yes, with rollback plan

  ---
  Phase 5: Optimize Storage

  PR 5.1: Clear large fields from games table

  Goal: Reduce games table size
  - Update migration to clear history, stats, meta_events, timers
  - Set migration_status = 2 for cleared games
  - Games table becomes pure metadata
  - Dependencies: PR 4.3 (must be reading from new tables)
  - Enables: PR 5.2
  - Safe to deploy: Yes, after new tables are primary

  PR 5.2: Implement S3 archival (Future)

  Goal: Move old partitions to S3
  - Archive partitions > 6 months old
  - Implement retrieval for archived games
  - Dependencies: PR 5.1
  - Enables: Unlimited storage scaling
  - Safe to deploy: Yes, with fallback