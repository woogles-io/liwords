#!/bin/bash

# Historical Games Migration Script
# This script migrates completed games from the games table to past_games and game_players tables

set -e

# Configuration - use config file approach like other migrations
CONFIG_FILE="${CONFIG_FILE:-}"
BATCH_SIZE="${BATCH_SIZE:-100}"
START_OFFSET="${START_OFFSET:-0}"
LIMIT="${LIMIT:-0}"
DRY_RUN="${DRY_RUN:-true}"
VERBOSE="${VERBOSE:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting Historical Games Migration${NC}"
echo "Config File: ${CONFIG_FILE:-default config}"
echo "Batch Size: $BATCH_SIZE"
echo "Start Offset: $START_OFFSET"
echo "Limit: $LIMIT"
echo "Dry Run: $DRY_RUN"
echo "Verbose: $VERBOSE"
echo ""

# Build the migration tool
echo -e "${YELLOW}Building migration tool...${NC}"
cd "$(dirname "$0")"
go build -o historical_games main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to build migration tool${NC}"
    exit 1
fi

# Prepare arguments
ARGS=""
if [ -n "$CONFIG_FILE" ]; then
    ARGS="$ARGS -config=\"$CONFIG_FILE\""
fi
ARGS="$ARGS -batch=$BATCH_SIZE"
ARGS="$ARGS -offset=$START_OFFSET"
ARGS="$ARGS -limit=$LIMIT"
ARGS="$ARGS -dry-run=$DRY_RUN"
ARGS="$ARGS -verbose=$VERBOSE"

# Run the migration
echo -e "${YELLOW}Running migration...${NC}"
eval "./historical_games $ARGS"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Migration completed successfully!${NC}"
else
    echo -e "${RED}Migration failed!${NC}"
    exit 1
fi

# Clean up
rm -f historical_games

echo -e "${GREEN}Done!${NC}"