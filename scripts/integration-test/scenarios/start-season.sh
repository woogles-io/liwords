#!/bin/bash
# Scenario: Start Season
# Simply starts the season and creates games for manual testing

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
echo "Start Season Scenario"
echo "========================================="
echo "League: $LEAGUE_SLUG"
echo "Season: $SEASON"
echo "========================================="
echo

# Start the season (creates games)
echo "🎮 Starting Season $SEASON..."
cd "$PROJECT_ROOT"
go run cmd/league-tester/*.go start-season \
  --league "$LEAGUE_SLUG" \
  --season $SEASON
echo "✅ Season started, games created"
echo

# Inspect current state
echo "📊 Current state:"
go run cmd/league-tester/*.go inspect \
  --league "$LEAGUE_SLUG"
echo

echo "========================================="
echo "✅ Season Started!"
echo "========================================="
echo
echo "What's available now:"
echo "  - All pairings created (14 games per player)"
echo "  - Games are ready to be played"
echo "  - Browse to http://localhost:18080/league/$LEAGUE_SLUG"
echo
echo "Next steps (pick one):"
echo "  1. Manual testing:"
echo "      - Log in as testuser1 (password: testpass1)"
echo "      - View your games, play if you want (with bot profile)"
echo "      - Click around the league interface"
echo
echo "  2. Simulate some games:"
echo "      DB_HOST=localhost DB_PORT=25432 DB_NAME=liwords_integration_test \\"
echo "      DB_USER=postgres DB_PASSWORD=pass DB_SSL_MODE=disable \\"
echo "      go run cmd/league-tester/*.go simulate-games --league $LEAGUE_SLUG --season $SEASON --count 10"
echo
echo "  3. Force-finish the season:"
echo "      make -f Makefile.integration-test integration-test-force-finish"
echo
