# League Tester

A testing tool for the league functionality in Liwords. This tool allows you to create test users, leagues, register players, simulate games, and inspect league state.

## Prerequisites

- Running local database (PostgreSQL)
- Running local Redis instance
- Proper configuration in environment variables or config files

## Commands

### 1. Create Test Users

Creates fake test users for league testing.

```bash
go run cmd/league-tester create-users --count 20 --output test_users.json
```

**Options:**
- `--count`: Number of users to create (default: 20)
- `--output`: Output JSON file with user UUIDs (default: test_users.json)

**Output:** Creates a JSON file with user information:
```json
{
  "users": [
    {
      "uuid": "...",
      "username": "league_test_user_01",
      "email": "league_test_user_01@example.com"
    },
    ...
  ]
}
```

### 2. Create Test League

Creates a test league with configurable settings.

```bash
go run cmd/league-tester create-league \
  --name "Test League" \
  --slug "test-league" \
  --division-size 15 \
  --output test_league.json
```

**Options:**
- `--name`: League name (default: "Test League")
- `--slug`: League slug (default: "test-league")
- `--division-size`: Ideal division size - target number of players per division (default: 15)
- `--output`: Output JSON file (default: test_league.json)

**Output:** Creates a JSON file with league information:
```json
{
  "league_uuid": "...",
  "league_slug": "test-league",
  "season_uuid": "...",
  "season_number": 1
}
```

### 3. Register Users

Registers all test users for the current season of a league.

```bash
go run cmd/league-tester register-users \
  --league test-league \
  --users-file test_users.json
```

**Options:**
- `--league`: League slug or UUID (required)
- `--users-file`: JSON file with user UUIDs (default: test_users.json)

### 4. Start Season

**Note:** Use the existing maintenance command instead:

```bash
go run cmd/maintenance league-season-starter
```

This will:
- Find all SCHEDULED seasons that are past their start date
- Transition them to ACTIVE
- Create all round-robin games for the season

### 5. Simulate Games

Simulates game completions with random but realistic results. This automatically updates league standings.

```bash
go run cmd/league-tester simulate-games \
  --season <season-uuid> \
  --all \
  --seed 12345
```

**Options:**
- `--season`: Season UUID (required)
- `--all`: Simulate all games at once (default: true)
- `--rounds`: Number of rounds to simulate instead of all (default: 0 = all)
- `--seed`: Random seed for reproducibility (default: 0 = random)

**How it works:**
1. Loads each incomplete game
2. Generates random scores (300-450 range)
3. Sets winner/loser and game end reason
4. Saves game to database
5. Updates league standings via `league.UpdateGameStandings()`

### 6. Inspect League

Displays current state of a league including all seasons, divisions, standings, and game completion status.

```bash
go run cmd/league-tester inspect --league test-league
```

**Options:**
- `--league`: League slug or UUID (required)

**Output example:**
```
================================================================================
LEAGUE: Test League (test-league)
================================================================================

Season 1 - ACTIVE
  UUID: a1b2c3d4-...
  Start: 2025-11-01
  End: 2025-12-01

  Division 1 (8 players)
    Rank Player                        W    L    D Spread  Games
    ────────────────────────────────────────────────────────────
       1 league_test_user_01           5    2    0   +250      7
       2 league_test_user_03           4    3    0   +120      7
       ...
    Games: 28/28 completed
```

## Example Workflow

Here's a complete workflow for testing league functionality:

```bash
# 1. Create 20 test users
go run cmd/league-tester create-users --count 20

# 2. Create a test league
go run cmd/league-tester create-league --slug test-league

# 3. Register all users
go run cmd/league-tester register-users --league test-league

# 4. You can manually place users into divisions, or use the placement manager
# For testing, you could manually create divisions and assign players

# 5. Start the season (creates all games)
go run cmd/maintenance league-season-starter

# 6. Get the season UUID from the league JSON file or database
cat test_league.json

# 7. Simulate all games with a specific seed for reproducibility
go run cmd/league-tester simulate-games \
  --season <season-uuid-from-above> \
  --seed 12345

# 8. Inspect the results
go run cmd/league-tester inspect --league test-league

# 9. Close the season (calculates final standings, promotion/relegation)
go run cmd/maintenance league-season-closer

# 10. Inspect again to see promotion/relegation results
go run cmd/league-tester inspect --league test-league

# 11. Run another season
go run cmd/maintenance league-season-starter
go run cmd/league-tester simulate-games --season <new-season-uuid>
go run cmd/maintenance league-season-closer
```

## Testing Different Scenarios

### Test Promotion/Relegation

1. Create a league with 16+ users to have 2+ divisions
2. Run a complete season
3. Check standings to see who gets promoted/relegated
4. Start next season and verify players moved correctly

### Test Different Division Sizes

```bash
go run cmd/league-tester create-league \
  --slug small-divisions \
  --division-size 8

go run cmd/league-tester create-league \
  --slug large-divisions \
  --division-size 20
```

### Test Reproducibility

Use the same seed to get identical results:

```bash
go run cmd/league-tester simulate-games --season <uuid> --seed 12345
# Run again with same seed to verify identical results
go run cmd/league-tester simulate-games --season <uuid> --seed 12345
```

## Troubleshooting

**"Failed to connect to database"**
- Check that your PostgreSQL is running
- Verify `DB_CONN_DSN` environment variable or config

**"League not found"**
- Verify the league slug is correct
- Use UUID instead of slug if needed

**"Failed to get games"**
- Ensure the season has been started (games created)
- Check that divisions exist with players assigned

## Notes

- Test users are created with password "testpassword123"
- All test users start with 1500 rating
- Game simulation uses realistic score ranges (250-450)
- Standings are automatically updated when games complete
- The tool uses the same league logic as production code
