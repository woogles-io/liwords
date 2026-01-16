#!/bin/bash
# Integration Test Setup Script
# Creates users, league, and bootstraps a season ready for testing

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Configuration
NUM_USERS=${NUM_USERS:-30}
LEAGUE_NAME=${LEAGUE_NAME:-"Integration Test League"}
LEAGUE_SLUG=${LEAGUE_SLUG:-"integration-test"}
DIVISION_SIZE=${DIVISION_SIZE:-15}
USERS_FILE="$SCRIPT_DIR/test_users.json"
LEAGUE_FILE="$SCRIPT_DIR/test_league.json"

# Database connection for integration test environment
export DB_HOST=localhost
export DB_PORT=25432
export DB_NAME=liwords_integration_test
export DB_USER=postgres
export DB_PASSWORD=pass
export DB_SSL_MODE=disable
export REDIS_URL="redis://localhost:16379"  # Adjust if different

echo "========================================="
echo "Integration Test League Setup"
echo "========================================="
echo "Users: $NUM_USERS"
echo "League: $LEAGUE_NAME ($LEAGUE_SLUG)"
echo "Division size: $DIVISION_SIZE"
echo "========================================="
echo

# Wait for database to be ready
echo "⏳ Waiting for database to be ready..."
until PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c '\q' 2>/dev/null; do
  echo "   Database not ready, waiting..."
  sleep 2
done
echo "✅ Database is ready"
echo

# Step 1: Create test users with varied ratings
echo "📝 Step 1: Creating $NUM_USERS test users..."
cd "$PROJECT_ROOT"
go run cmd/league-tester/*.go create-rated-users \
  --count $NUM_USERS \
  --output "$USERS_FILE"
echo "✅ Created $NUM_USERS users (saved to $USERS_FILE)"
echo

# Step 2: Create test league
echo "🏆 Step 2: Creating league '$LEAGUE_NAME'..."
go run cmd/league-tester/*.go create-league \
  --name "$LEAGUE_NAME" \
  --slug "$LEAGUE_SLUG" \
  --division-size $DIVISION_SIZE \
  --output "$LEAGUE_FILE"
echo "✅ Created league (saved to $LEAGUE_FILE)"
echo

# Step 3: Register all users for season 1
echo "📋 Step 3: Registering users for Season 1..."
go run cmd/league-tester/*.go register-users \
  --league "$LEAGUE_SLUG" \
  --season 1 \
  --users-file "$USERS_FILE"
echo "✅ Registered $NUM_USERS users for Season 1"
echo

# Step 4: Prepare divisions
echo "🎯 Step 4: Preparing divisions for Season 1..."
go run cmd/league-tester/*.go prepare-divisions \
  --league "$LEAGUE_SLUG" \
  --season 1
echo "✅ Divisions prepared"
echo

echo "========================================="
echo "✅ Setup Complete!"
echo "========================================="
echo
echo "League is ready for testing:"
echo "  - League slug: $LEAGUE_SLUG"
echo "  - Season: 1 (status: REGISTRATION_OPEN with divisions prepared)"
echo "  - Users: $NUM_USERS registered"
echo
echo "Next steps:"
echo "  1. Browse to http://localhost:18080/league/$LEAGUE_SLUG"
echo "  2. Run scenarios (see scripts/integration-test/scenarios/)"
echo
echo "Useful commands:"
echo "  - Inspect state:    make integration-test-inspect"
echo "  - Start season:     make integration-test-start-season"
echo "  - Force finish:     make integration-test-force-finish"
echo
