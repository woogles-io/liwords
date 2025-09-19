# Game Players Migration

This script migrates existing game data from the `games` table to the new `game_players` table.

## Purpose

The `game_players` table is designed to improve query performance for operations like `GetRecentGames` by:
- Creating a many-to-many relationship between games and players
- Denormalizing frequently accessed data (scores, opponent info)
- Enabling efficient indexing on player_id + created_at

## What it does

1. Processes all completed games (excludes ongoing games with `game_end_reason = 0` and cancelled games with `game_end_reason = 7`)
2. **Uses quickdata PlayerInfo ordering**: Extracts player order from `quickdata->'PlayerInfo'` to ensure correct turn order
3. Resolves player UUIDs from PlayerInfo to actual user IDs
4. For each game, creates two rows in `game_players` with proper `player_index` (0 = first player, 1 = second player)
5. Extracts scores from the `quickdata->>'finalScores'` JSON field
6. Determines win/loss status from `winner_idx` (which refers to PlayerInfo indices)
7. Includes rematch tracking via `original_request_id` from `quickdata->>'o'`

## Important Note on Player Ordering

Historical games (before ~March 2025) had inconsistent `player0_id`/`player1_id` ordering that didn't always reflect who went first/second. This migration uses the `quickdata->'PlayerInfo'` array which contains the correct turn order:
- PlayerInfo[0] = first player (gets player_index = 0)
- PlayerInfo[1] = second player (gets player_index = 1)

## Usage

```bash
# Run with default batch size (500)
go run scripts/migrations/game_players/main.go

# Run with custom batch size
go run scripts/migrations/game_players/main.go 1000
```

## Safety

- Uses `ON CONFLICT (game_uuid, player_id) DO NOTHING` to handle re-runs safely
- Processes data in batches with transactions
- Provides progress logging with ETA
- Can be interrupted and resumed safely
- Skips games without proper PlayerInfo data with warnings

## Expected Data

For a database with ~9.4M games, this will create ~18.8M rows in `game_players` (2 per game).