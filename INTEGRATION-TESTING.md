# League Integration Testing

Complete end-to-end integration test setup for league functionality with time manipulation.

## 🚀 Quick Start (3 commands)

```bash
# 1. Start everything
make -f Makefile.integration-test integration-test-up

# 2. Set up league with 30 users
make -f Makefile.integration-test integration-test-setup

# 3. Start the season and browse around
make -f Makefile.integration-test integration-test-start-season
```

**Now browse to:** http://localhost:18080/league/integration-test

**Log in as:**
- Username: `testuser1` (or any testuser1-30)
- Password: `testpass1` (number matches username)

## 🧪 Test the Force-Finish Bug Fix

```bash
# After setup above, run the force-finish scenario
make -f Makefile.integration-test integration-test-force-finish
```

This will:
1. Start a season with all games
2. Simulate 5 games (leaving some unfinished)
3. Fast-forward time to season end using `LEAGUE_NOW`
4. Run the maintenance task that force-finishes games
5. Verify:
   - ✅ Games were adjudicated
   - ✅ `game_players` rows were created
   - ✅ Games show up in player game lists (14/14 instead of 13/14)
   - ✅ Standings were updated

## 📋 Full Command Reference

```bash
# See all commands
make -f Makefile.integration-test help

# Common workflows
make -f Makefile.integration-test integration-test-up          # Start
make -f Makefile.integration-test integration-test-setup       # Setup league
make -f Makefile.integration-test integration-test-inspect     # Check state
make -f Makefile.integration-test integration-test-start-season # Start games
make -f Makefile.integration-test integration-test-force-finish # Test scenario
make -f Makefile.integration-test integration-test-down        # Stop
make -f Makefile.integration-test integration-test-clean       # Clean everything
```

## 🎯 What Gets Created

**Users:**
- 30 test users with varied ratings (1200-2000)
- Pattern: `testuser1` / `testpass1`, `testuser2` / `testpass2`, etc.

**League:**
- Name: "Integration Test League"
- Slug: `integration-test`
- 2 divisions of 15 players each
- 30-day seasons
- 8h increment, 72h time bank

**Access Points:**
- Frontend: http://localhost:18080
- API: http://localhost:18001
- Database: `localhost:25432` (postgres/pass)
- Traefik: http://localhost:18888

## 🔧 Manual Testing Workflow

1. **Start & Setup:**
   ```bash
   make -f Makefile.integration-test integration-test-up
   make -f Makefile.integration-test integration-test-setup
   make -f Makefile.integration-test integration-test-start-season
   ```

2. **Browse & Explore:**
   - Go to http://localhost:18080
   - Log in as `testuser1` / `testpass1`
   - Click "Leagues" → "Integration Test League"
   - View standings, pairings, your games

3. **Simulate Games (optional):**
   ```bash
   # Simulate 10 random game completions
   DB_HOST=localhost DB_PORT=25432 DB_NAME=liwords_integration_test \
   DB_USER=postgres DB_PASSWORD=pass DB_SSL_MODE=disable \
   go run cmd/league-tester/*.go simulate-games \
     --league integration-test --season 1 --count 10
   ```

4. **Force-Finish & Verify:**
   ```bash
   make -f Makefile.integration-test integration-test-force-finish
   ```
   Then refresh the UI and verify:
   - Games with "Adjudicated" end reason
   - All 14 games appear in player's game list
   - Standings show 14/14 games

5. **Clean up:**
   ```bash
   make -f Makefile.integration-test integration-test-down
   # or for fresh start:
   make -f Makefile.integration-test integration-test-clean
   ```

## ⏰ Time Manipulation

The integration test environment supports time manipulation via `LEAGUE_NOW`:

```bash
# Set fake time in docker-compose
LEAGUE_NOW="2025-02-01T00:00:00Z" \
  docker-compose -f docker-compose.integration-test.yml up -d

# Or in maintenance task
LEAGUE_NOW="2025-02-01T00:00:00Z" \
  go run cmd/maintenance/*.go hourly-league-runner
```

This allows testing time-dependent workflows (season start, end, reminders, etc.) without waiting.

## 🐛 Troubleshooting

**Services won't start:**
```bash
make -f Makefile.integration-test integration-test-logs
# or
docker-compose -f docker-compose.integration-test.yml ps
```

**Clean slate:**
```bash
make -f Makefile.integration-test integration-test-clean integration-test-up
```

**Database connection:**
```bash
# Connect directly
make -f Makefile.integration-test integration-test-psql

# Or manually
PGPASSWORD=pass psql -h localhost -p 25432 -U postgres -d liwords_integration_test
```

**Check league state:**
```bash
make -f Makefile.integration-test integration-test-inspect
```

## 📁 Project Structure

```
scripts/integration-test/
├── README.md                      # Detailed documentation
├── setup-league.sh                # Creates users, league, registers
├── test_users.json                # Generated user data
├── test_league.json               # Generated league data
└── scenarios/
    ├── start-season.sh            # Just start season (for manual testing)
    └── force-finish-season.sh     # Full force-finish test scenario

docker-compose.integration-test.yml  # Isolated test environment
Makefile.integration-test            # Convenient make targets
```

## 💡 Tips

- The environment stays running between commands - no need to restart
- Use `integration-test-inspect` frequently to check state
- Test users all follow the pattern `testuser<N>` / `testpass<N>`
- You can run multiple scenarios without restarting
- Check `scripts/integration-test/README.md` for more details

## 🎓 Creating New Scenarios

See `scripts/integration-test/scenarios/force-finish-season.sh` as a template.

Basic pattern:
```bash
#!/bin/bash
set -e

# Database config (copy this boilerplate)
export DB_HOST=localhost DB_PORT=25432
export DB_NAME=liwords_integration_test
export DB_USER=postgres DB_PASSWORD=pass DB_SSL_MODE=disable

# Your scenario logic
cd /path/to/liwords
go run cmd/league-tester/*.go <command> <args>
go run cmd/maintenance/*.go <task>

# Verify outcomes
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT \
  -U $DB_USER -d $DB_NAME -c "SELECT ..."
```

## 🚨 Known Limitations

- Frontend npm install takes ~1-2 minutes on first start
- Time manipulation only affects code that checks `LEAGUE_NOW` env var
- Bot profile is optional (use `--profile with-bot` to enable)
- Services share code volume with dev environment (changes affect both)

## 📚 Further Reading

- Full documentation: `scripts/integration-test/README.md`
- League tester: `cmd/league-tester/` (see `--help` for each command)
- Maintenance tasks: `cmd/maintenance/`
