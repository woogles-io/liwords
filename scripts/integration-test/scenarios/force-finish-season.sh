#!/bin/bash
# Scenario: Force Finish Season
# Tests league game adjudication when season ends with unfinished games

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

LEAGUE_SLUG=${LEAGUE_SLUG:-"integration-test"}
SEASON=${SEASON:-1}

# Database connection
export DB_HOST=localhost
export DB_PORT=25432
export DB_NAME=liwords_integration_test
export DB_USER=postgres
export DB_PASSWORD=pass
export DB_SSL_MODE=disable
export REDIS_URL="redis://localhost:16379"

echo "========================================="
echo "Force Finish Season Scenario"
echo "========================================="
echo "League: $LEAGUE_SLUG"
echo "Season: $SEASON"
echo "========================================="
echo

# Step 1: Start the season (creates games)
echo "🎮 Step 1: Starting Season $SEASON..."
cd "$PROJECT_ROOT"
go run cmd/league-tester/*.go start-season \
  --league "$LEAGUE_SLUG" \
  --season $SEASON
echo "✅ Season started, games created"
echo

# Step 2: Simulate some (but not all) games
echo "🎲 Step 2: Simulating some games (leaving some unfinished)..."
go run cmd/league-tester/*.go simulate-games \
  --league "$LEAGUE_SLUG" \
  --season $SEASON \
  --count 5
echo "✅ Simulated 5 games (some remain unfinished)"
echo

# Step 3: Check current state
echo "📊 Step 3: Inspecting current state..."
go run cmd/league-tester/*.go inspect \
  --league "$LEAGUE_SLUG"
echo

# Step 4: Get season dates to calculate fast-forward time
echo "📅 Step 4: Calculating season end time..."
SEASON_END=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c \
  "SELECT end_date FROM seasons WHERE season_number = $SEASON ORDER BY created_at DESC LIMIT 1;" | xargs)

if [ -z "$SEASON_END" ]; then
  echo "❌ Error: Could not find season end date"
  exit 1
fi

echo "   Season ends at: $SEASON_END"
# Add 1 hour past season end to ensure cleanup runs
FORCE_FINISH_TIME=$(date -d "$SEASON_END + 1 hour" -Iseconds 2>/dev/null || \
                    date -v+1H -j -f "%Y-%m-%d %H:%M:%S%z" "$SEASON_END" "+%Y-%m-%dT%H:%M:%S%:z" 2>/dev/null || \
                    echo "$SEASON_END")
echo "   Will fast-forward to: $FORCE_FINISH_TIME"
echo

# Step 5: Fast-forward time and run maintenance task
echo "⏩ Step 5: Fast-forwarding time to end of season..."
echo "   Setting LEAGUE_NOW=$FORCE_FINISH_TIME"

# Run the maintenance task with LEAGUE_NOW set
LEAGUE_NOW="$FORCE_FINISH_TIME" go run cmd/maintenance/*.go hourly-league-runner
echo "✅ Ran hourly league runner with fast-forwarded time"
echo

# Step 6: Verify outcomes
echo "🔍 Step 6: Verifying force-finished games..."

# Count force-finished games
ADJUDICATED_COUNT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c \
  "SELECT COUNT(*) FROM games WHERE game_end_reason = 9;" | xargs)

echo "   Games with ADJUDICATED end reason: $ADJUDICATED_COUNT"

# Check game_players rows were created
GAME_PLAYERS_COUNT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c \
  "SELECT COUNT(*) FROM game_players gp
   JOIN games g ON g.uuid = gp.game_uuid
   WHERE g.game_end_reason = 9;" | xargs)

echo "   game_players rows for adjudicated games: $GAME_PLAYERS_COUNT"

# Check standings were updated
STANDINGS_COUNT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c \
  "SELECT COUNT(*) FROM league_standings WHERE season_id IN (
     SELECT uuid FROM seasons WHERE season_number = $SEASON
   );" | xargs)

echo "   Standings rows: $STANDINGS_COUNT"
echo

# Step 7: Final inspection
echo "📊 Step 7: Final state inspection..."
go run cmd/league-tester/*.go inspect \
  --league "$LEAGUE_SLUG"
echo

echo "========================================="
echo "✅ Force Finish Scenario Complete!"
echo "========================================="
echo
echo "Summary:"
echo "  - Adjudicated games: $ADJUDICATED_COUNT"
echo "  - game_players rows: $GAME_PLAYERS_COUNT"
echo "  - Standings entries: $STANDINGS_COUNT"
echo
echo "Verification:"
echo "  1. Browse to http://localhost:18080/league/$LEAGUE_SLUG"
echo "  2. Click on any player - verify all 14 games show up"
echo "  3. Check that adjudicated games show proper end reason"
echo "  4. Verify standings show correct game counts"
echo

if [ "$GAME_PLAYERS_COUNT" -gt 0 ] && [ "$ADJUDICATED_COUNT" -gt 0 ]; then
  echo "✅ SUCCESS: Games were properly adjudicated and game_players rows created"
else
  echo "❌ FAILURE: Something went wrong with force-finish process"
  exit 1
fi
