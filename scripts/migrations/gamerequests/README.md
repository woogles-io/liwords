# Game Request Migration Script

This script migrates game request data from the old `request` column (bytea with protobuf data) to the new `game_request` column (jsonb).

## Purpose

As part of the migration from GORM to sqlc and from protobuf to JSON storage, this script:

1. Reads protobuf data from the `games.request` column (bytea)
2. Converts it to JSON format 
3. Stores it in the `games.game_request` column (jsonb)

## Usage

```bash
# Basic usage with default batch size (1000)
go run main.go

# With custom batch size
go run main.go 500
```

The script uses the standard config loading from environment variables, same as other migration scripts.

## Safety Features

- **Batched processing**: Processes games in configurable batches to avoid long-running transactions
- **Transactional**: Each batch is processed in its own transaction for consistency
- **Resume-friendly**: Only migrates games where `game_request IS NULL`, so it can be safely rerun
- **Error handling**: Skips individual games that fail to parse, logs warnings, continues processing
- **Progress tracking**: Logs progress throughout the migration

## Prerequisites

- The `game_request` column must already exist in the `games` table
- Dual-write should be enabled in the application (Phase 1 complete)

## Migration Phases

This script is for **Phase 2** of the game request migration:

- **Phase 1** (✅ Complete): Dual-write to both columns
- **Phase 2** (This script): Backfill existing data  
- **Phase 3** (Future): Switch reads to use jsonb column
- **Phase 4** (Future): Remove bytea column

## Example Output

```
2025-09-09T23:45:00-04:00 INF Starting game request migration from bytea to jsonb
2025-09-09T23:45:00-04:00 INF Found rows to migrate total_rows=50000
2025-09-09T23:45:01-04:00 INF Migrated batch batch_size=1000 processed=1000 total=50000
2025-09-09T23:45:02-04:00 INF Migrated batch batch_size=1000 processed=2000 total=50000
...
2025-09-09T23:45:50-04:00 INF Migration completed total_migrated=50000
2025-09-09T23:45:50-04:00 INF Migration completed successfully
```

## Rollback

If needed, you can clear the migrated data:

```sql
-- Clear jsonb data (will be repopulated by dual-write)
UPDATE games SET game_request = NULL WHERE game_request IS NOT NULL;
```

## Monitoring

Monitor the migration with:

```sql
-- Check progress
SELECT 
  COUNT(*) as total_games,
  COUNT(request) as has_bytea,
  COUNT(game_request) as has_jsonb,
  COUNT(CASE WHEN game_request::text != '{}' THEN 1 END) as has_actual_jsonb_data,
  COUNT(CASE WHEN request IS NOT NULL AND (game_request IS NULL OR game_request::text = '{}') THEN 1 END) as needs_migration
FROM games;
```