#!/bin/bash
# Backfill game_players.updated_at to match games.updated_at.
# Cursors by gp.updated_at to start from affected rows (post-deploy).
# Only updates rows where updated_at is NULL or mismatched.
# Saves cursor to file for auto-resume after crash/^C.
# Usage: ./backfill_game_players_updated_at.sh [connection_string] [start_timestamp]
# Example: ./backfill_game_players_updated_at.sh "dbname=liwords" "2026-01-25 00:00:00+00"

CONNSTR="${1:-dbname=liwords}"
START="${2:-2026-01-25 00:00:00+00}"
BATCH=5000
CURSOR_FILE="$(dirname "$0")/backfill_game_players_cursor"
TOTAL=0
SKIPPED=0

# Resume from saved cursor if it exists
if [ -f "$CURSOR_FILE" ]; then
  CURSOR=$(cat "$CURSOR_FILE")
  echo "Resuming from cursor: $CURSOR"
else
  CURSOR="$START"
  echo "Starting from: $CURSOR"
fi

while true; do
  # Get the last updated_at in this batch
  LAST=$(psql "$CONNSTR" -t -A -c "
    SELECT gp.updated_at FROM game_players gp
    WHERE gp.updated_at >= '$CURSOR'
    ORDER BY gp.updated_at
    LIMIT $BATCH
  " | tail -1)

  if [ -z "$LAST" ]; then
    echo "Done. Updated: $TOTAL rows, Skipped: $SKIPPED batches"
    rm -f "$CURSOR_FILE"
    break
  fi

  # Only update rows where updated_at doesn't match games.updated_at
  COUNT=$(psql "$CONNSTR" -t -A -c "
    UPDATE game_players gp SET updated_at = g.updated_at
    FROM games g
    WHERE gp.game_uuid = g.uuid
      AND gp.updated_at >= '$CURSOR'
      AND gp.updated_at <= '$LAST'
      AND gp.updated_at != g.updated_at
  " | grep -oP '\d+')

  CURSOR="$LAST"
  echo "$CURSOR" > "$CURSOR_FILE"

  if [ "$COUNT" = "0" ] || [ -z "$COUNT" ]; then
    SKIPPED=$((SKIPPED + 1))
  else
    TOTAL=$((TOTAL + COUNT))
    echo "Updated $COUNT rows (total: $TOTAL, cursor: $CURSOR)"
  fi

  sleep 0.1
done
