# League Integration Tests

End-to-end integration tests for league functionality with time manipulation.

## Quick Start

```bash
# 1. Start integration test environment
make -f Makefile.integration-test integration-test-up

# 2. Set up league with users
make -f Makefile.integration-test integration-test-setup

# 3. Browse to http://localhost:18080/league/integration-test
#    (You can now poke around manually)

# 4. Run force-finish scenario
make -f Makefile.integration-test integration-test-force-finish

# 5. Clean up when done
make -f Makefile.integration-test integration-test-down
```

## Full Automated Test

Run everything with one command (with a pause for manual inspection):

```bash
make -f Makefile.integration-test integration-test-full
```

## What Gets Created

### Users
- 30 test users (`testuser1` through `testuser30`) with passwords `testpass1` through `testpass30`
- Varied ratings (1200-2000) for realistic division splitting
- Saved to `scripts/integration-test/test_users.json`

### League
- Name: "Integration Test League"
- Slug: `integration-test`
- Season length: 30 days
- Time control: 8 hour increment, 72 hour time bank
- Lexicon: CSW24
- Division size: 15 players
- Saved to `scripts/integration-test/test_league.json`

### Season
- Season 1 bootstrapped and ready
- All users registered
- Divisions prepared (2 divisions of 15 players each)

## Scenarios

### Force Finish Season

Tests the league game adjudication when season ends with unfinished games.

**What it does:**
1. Starts Season 1 (creates all games)
2. Simulates 5 games (leaves rest unfinished)
3. Fast-forwards time to season end using `LEAGUE_NOW`
4. Runs hourly maintenance task
5. Verifies:
   - Games were adjudicated (end_reason = ADJUDICATED)
   - `game_players` rows were created
   - League standings were updated
   - Games show up in player game lists

**Run it:**
```bash
make -f Makefile.integration-test integration-test-force-finish
```

**Expected outcome:**
- All unfinished games get `ADJUDICATED` end reason
- Winner determined by current score
- Games appear in player game lists (14/14 games)
- Standings reflect all completed games

## Environment Details

The integration test environment runs in complete isolation:

- **Frontend:** http://localhost:18080
- **API:** http://localhost:18001
- **Database:** localhost:25432 (postgres/pass)
- **Traefik Dashboard:** http://localhost:18888

All services use the `integration-test-net` Docker network and won't interfere with your dev environment.

## Time Manipulation

Set the `LEAGUE_NOW` environment variable to fake the current time:

```bash
# In maintenance task
LEAGUE_NOW="2025-01-15T12:00:00Z" go run cmd/maintenance/*.go hourly-league-runner

# Or set in docker-compose
docker-compose -f docker-compose.integration-test.yml up -d \
  -e LEAGUE_NOW="2025-01-15T12:00:00Z"
```

This allows testing time-dependent workflows without waiting.

## Manual Testing Workflow

1. Start environment and set up league (see Quick Start)
2. Log in as a test user:
   - Username: `testuser1` (or any testuser1-30)
   - Password: `testpass1` (matches username number)
3. Browse to http://localhost:18080/league/integration-test
4. Click around, view standings, games, etc.
5. Run scenarios to advance state
6. Verify UI updates correctly

## Creating New Scenarios

Add new scenario scripts to `scripts/integration-test/scenarios/`:

```bash
#!/bin/bash
# Scenario: Your Scenario Name
# Description of what this tests

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Database config (reuse this boilerplate)
export DB_HOST=localhost
export DB_PORT=25432
export DB_NAME=liwords_integration_test
export DB_USER=postgres
export DB_PASSWORD=pass
export DB_SSL_MODE=disable

# Your test logic here...
cd "$PROJECT_ROOT"
go run cmd/league-tester/*.go <command> ...
```

Then add a Makefile target:

```makefile
integration-test-your-scenario:
	@./scripts/integration-test/scenarios/your-scenario.sh
```

## Troubleshooting

**Services won't start:**
```bash
# Check logs
make -f Makefile.integration-test integration-test-logs

# Try clean restart
make -f Makefile.integration-test integration-test-clean integration-test-up
```

**Database connection errors:**
```bash
# Verify DB is up
PGPASSWORD=pass psql -h localhost -p 25432 -U postgres -d liwords_integration_test -c '\l'
```

**Frontend not loading:**
```bash
# Frontend takes a while to compile on first start
# Check frontend logs:
docker-compose -f docker-compose.integration-test.yml logs frontend
```

## Tips

- Use `integration-test-inspect` frequently to check state
- The environment stays running - you can run multiple scenarios without restarting
- Use `integration-test-psql` to inspect database directly
- Test users all have the same password pattern: `testpass<N>` where N is the user number
- You can modify `NUM_USERS`, `DIVISION_SIZE` etc. when running setup
