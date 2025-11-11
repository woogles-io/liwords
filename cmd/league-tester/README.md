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

Registers all test users for a specific season of a league. The season must have REGISTRATION_OPEN status.

```bash
go run cmd/league-tester register-users \
  --league test-league \
  --season 1 \
  --users-file test_users.json
```

**Options:**
- `--league`: League slug or UUID (required)
- `--season`: Season number to register for (required)
- `--users-file`: JSON file with user UUIDs (default: test_users.json)

**Note:** This command will fail if the season status is not REGISTRATION_OPEN. Use `open-registration` command or the frontend to open registration first.

### 4. Season Lifecycle Commands

These commands manage the season lifecycle and mirror what the maintenance tasks do in production.

#### 4a. Open Registration

Opens registration for a specific season by changing its status from SCHEDULED to REGISTRATION_OPEN.

```bash
go run cmd/league-tester open-registration --league test-league --season 1
```

**Options:**
- `--league`: League slug or UUID (required)
- `--season`: Season number to open registration for (required)

**What it does:**
- Changes the specified SCHEDULED season to REGISTRATION_OPEN status
- Allows players to register for that season

#### 4b. Prepare Divisions

Closes registration and creates divisions for a season (Day 21 at 7:45 AM in production).

```bash
go run cmd/league-tester prepare-divisions \
  --league test-league \
  --season 1
```

**Options:**
- `--league`: League slug or UUID (required)
- `--season`: Season number to prepare (required)

**What it does:**
- Categorizes players as NEW (rookies) or RETURNING
- Creates rookie divisions if ≥10 new players
- Rebalances regular divisions for returning players
- Changes season status from REGISTRATION_OPEN to SCHEDULED

**For Season 1:** All players are considered rookies, so this will create rookie divisions.

#### 4c. Start Season

Starts a scheduled season and creates all games (Day 21 at 8:00 AM in production).

```bash
go run cmd/league-tester start-season \
  --league test-league \
  --season 1
```

**Options:**
- `--league`: League slug or UUID (required)
- `--season`: Season number to start (required)

**What it does:**
- Changes season status from SCHEDULED to ACTIVE
- Creates all round-robin games for all divisions
- Sets the season as the current season

#### 4d. Close Season

Closes the current active season (Day 20 in production).

```bash
go run cmd/league-tester close-season --league test-league
```

**Options:**
- `--league`: League slug or UUID (required)

**What it does:**
- Force-finishes any unfinished games
- Marks season outcomes (PROMOTED/RELEGATED/STAYED)
- Changes season status to COMPLETED

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

### Season 1 (Bootstrapped)

#### Option A: Bootstrap with REGISTRATION_OPEN (default)

```bash
# 1. Create 30+ test users
go run cmd/league-tester create-users --count 32

# 2. Create a test league (bootstraps Season 1 with REGISTRATION_OPEN status)
go run cmd/league-tester create-league --slug test-league

# 3. Register all users for season 1
go run cmd/league-tester register-users --league test-league --season 1

# 4. Prepare divisions (creates rookie divisions since this is Season 1)
go run cmd/league-tester prepare-divisions --league test-league --season 1

# 5. Start the season (creates all games)
go run cmd/league-tester start-season --league test-league --season 1

# 6. Simulate all games
go run cmd/league-tester simulate-games \
  --season <season-uuid> \
  --seed 12345

# 7. Inspect the results
go run cmd/league-tester inspect --league test-league

# 8. Close the season
go run cmd/league-tester close-season --league test-league
```

#### Option B: If Season 1 was created as SCHEDULED

If you have a Season 1 that's already SCHEDULED (not REGISTRATION_OPEN), you can open registration:

```bash
# After prepare-divisions closes registration and season becomes SCHEDULED
# You can reopen registration for that season:
go run cmd/league-tester open-registration --league test-league --season 1

# Then continue with registration, prepare, start flow
```

### Season 2+ (Normal Flow)

**Note**: For Season 2+, you would typically use the maintenance task `league-registration-opener` on Day 15 to create the next season with REGISTRATION_OPEN status. For manual testing with league-tester, you can create a new season and then open registration:

```bash
# If you need to manually open registration for Season 2 after it's been created as SCHEDULED:
go run cmd/league-tester open-registration --league test-league --season 2

# 2. Players register via frontend or register-users command
go run cmd/league-tester register-users --league test-league --season 2

# 3. Close registration and prepare divisions (Day 21 at 7:45 AM)
go run cmd/league-tester prepare-divisions --league test-league --season 2

# 4. Start the season (Day 21 at 8:00 AM)
go run cmd/league-tester start-season --league test-league --season 2

# 5. Simulate games
go run cmd/league-tester simulate-games --season <season-uuid> --seed 12345

# 6. Close season (Day 20 of next month)
go run cmd/league-tester close-season --league test-league
```

### Using Maintenance Commands

The maintenance commands can also be used in production:

```bash
# Day 15: Open registration
go run cmd/maintenance league-registration-opener

# Day 20: Close current season
go run cmd/maintenance league-season-closer

# Day 21 at 7:45 AM: Prepare divisions
go run cmd/maintenance league-division-preparer

# Day 21 at 8:00 AM: Start season
go run cmd/maintenance league-season-starter
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
