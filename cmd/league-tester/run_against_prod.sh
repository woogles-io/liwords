#!/bin/bash
# Helper script to run league-tester against production database
# Prerequisites:
#   1. Connect to prod VPC via WireGuard
#   2. Set your readonly database credentials below

# Production database configuration
export DB_HOST="your-prod-db-host"      # e.g., "10.x.x.x" or "prod-db.internal"
export DB_PORT="5432"                   # Default PostgreSQL port
export DB_NAME="liwords"                # Production database name
export DB_USER="readonly_user"          # Your readonly database user
export DB_PASSWORD="your-readonly-password"
export DB_SSL_MODE="require"            # Use 'require' for production

# Redis configuration (optional, only needed if tests require it)
# export REDIS_URL="redis://prod-redis:6379/0"

# Run the test-dp-rebalance command
# Usage: ./run_against_prod.sh <league-slug> <season-number>

if [ $# -lt 2 ]; then
    echo "Usage: $0 <league-slug> <season-number>"
    echo "Example: $0 cll-season-11 11"
    exit 1
fi

LEAGUE="$1"
SEASON="$2"

echo "========================================="
echo "Testing DP Rebalance Against Production"
echo "========================================="
echo "League: $LEAGUE"
echo "Season: $SEASON"
echo "DB Host: $DB_HOST"
echo "DB User: $DB_USER"
echo "========================================="
echo ""

# Run the command
go run . test-dp-rebalance --league "$LEAGUE" --season "$SEASON"
