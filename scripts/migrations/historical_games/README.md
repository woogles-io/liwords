# Historical Games Migration

This directory contains tools for migrating completed games from the main `games` table to the new partitioned `past_games` and `game_players` tables.

## Overview

The migration process moves completed games (games with `game_end_reason != 0`) from the `games` table to:
- `past_games`: Stores game metadata and game documents (JSONB format for efficient querying)
- `game_players`: Stores denormalized player data for fast recent games queries

## Files

- `main.go`: Main migration tool written in Go
- `run_historical_migration.sh`: Shell script wrapper for easy execution
- `README.md`: This documentation

## Usage

### Quick Start

```bash
# Run a dry run first to see what would be migrated
./run_historical_migration.sh

# Run the actual migration
DRY_RUN=false ./run_historical_migration.sh
```

### Environment Variables

- `CONFIG_FILE`: Path to liwords config file (default: uses default config)
- `BATCH_SIZE`: Number of games to process in each batch (default: 100)
- `START_OFFSET`: Starting offset for processing (default: 0)
- `LIMIT`: Maximum number of games to process, 0 = no limit (default: 0)
- `DRY_RUN`: Set to "false" to actually perform migration (default: true)
- `VERBOSE`: Set to "true" for verbose logging (default: false)

### Examples

```bash
# Use a specific config file
CONFIG_FILE=/path/to/config.json DRY_RUN=false ./run_historical_migration.sh

# Migrate first 1000 games only
LIMIT=1000 DRY_RUN=false ./run_historical_migration.sh

# Resume migration from offset 5000
START_OFFSET=5000 DRY_RUN=false ./run_historical_migration.sh

# Use larger batches for faster processing
BATCH_SIZE=500 DRY_RUN=false ./run_historical_migration.sh

# Verbose logging
VERBOSE=true DRY_RUN=false ./run_historical_migration.sh
```

## Safety Features

1. **Dry Run by Default**: The script runs in dry-run mode by default to prevent accidental data migration
2. **Batch Processing**: Processes games in configurable batches to avoid overwhelming the database
3. **Transaction Safety**: Each game migration is wrapped in a transaction
4. **Progress Reporting**: Shows progress every 100 games
5. **Error Handling**: Continues processing even if individual games fail, reporting errors at the end

## Migration Process

For each completed game, the tool:

1. **Reads** the game data from the `games` table
2. **Converts** the protobuf game history to a GameDocument (JSONB format)
3. **Inserts** the game data into `past_games` table
4. **Inserts** player records into `game_players` table (one record per player)
5. **Updates** the `migration_status` field in the original `games` table to mark it as migrated

## Post-Migration Cleanup

The migration tool includes commented code to clear migrated data from the `games` table to save space. Uncomment the cleanup section in `main.go` if you want to remove the original data after migration.

## Monitoring

- Check migration progress in the logs
- Query migration status: `SELECT migration_status, COUNT(*) FROM games GROUP BY migration_status;`
- Verify migrated data: `SELECT COUNT(*) FROM past_games; SELECT COUNT(*) FROM game_players;`

## Rollback

If you need to rollback the migration:

```sql
-- Remove migrated data
DELETE FROM past_games WHERE gid IN (SELECT uuid FROM games WHERE migration_status >= 1);
DELETE FROM game_players WHERE game_uuid IN (SELECT uuid FROM games WHERE migration_status >= 1);

-- Reset migration status
UPDATE games SET migration_status = NULL WHERE migration_status >= 1;
```

## Performance Considerations

- Run during low-traffic periods
- Consider increasing `BATCH_SIZE` for faster processing on powerful systems
- Monitor database performance during migration
- The tool includes small delays between batches to avoid overloading the database