#!/usr/bin/env bash
# Run a .sql file from reporting/ against the reporting DB (reachable over the
# Wireguard tunnel), print results in the terminal, save them to CSV, and
# open the CSV. Replaces the PgAdmin GUI workflow. Brings up the "Woogles"
# WireGuard VPN service automatically if it isn't already connected.
#
# Usage:
#   reporting/run_query.sh reporting/omgwords/games_per_month.sql
#   reporting/run_query.sh reporting/omgwords/games_per_month.sql --no-open
#
# Connection details live in reporting/.env (gitignored). The DB password
# is read from ~/.pgpass, not from this script.
set -euo pipefail

VPN_SERVICE="Woogles"

ensure_vpn_connected() {
  if nc -z -w3 "$REPORTING_DB_HOST" "$REPORTING_DB_PORT" 2>/dev/null; then
    return
  fi

  echo "$VPN_SERVICE tunnel not reachable, connecting..." >&2
  scutil --nc start "$VPN_SERVICE" >/dev/null 2>&1 || {
    echo "Failed to start VPN service '$VPN_SERVICE'. Is it configured in scutil --nc list?" >&2
    exit 1
  }

  for _ in $(seq 1 15); do
    if nc -z -w3 "$REPORTING_DB_HOST" "$REPORTING_DB_PORT" 2>/dev/null; then
      echo "$VPN_SERVICE connected." >&2
      return
    fi
    sleep 1
  done

  echo "Timed out waiting for $VPN_SERVICE tunnel / DB to become reachable." >&2
  exit 1
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"
LOCK_DIR="$SCRIPT_DIR/_results/.run_query.lock"

acquire_lock() {
  if mkdir "$LOCK_DIR" 2>/dev/null; then
    echo "$$" > "$LOCK_DIR/pid"
    echo "$1" > "$LOCK_DIR/query"
    trap 'rm -rf "$LOCK_DIR"' EXIT
    return
  fi

  local held_pid held_query
  held_pid="$(cat "$LOCK_DIR/pid" 2>/dev/null || true)"
  held_query="$(cat "$LOCK_DIR/query" 2>/dev/null || echo "unknown")"

  if [ -n "$held_pid" ] && kill -0 "$held_pid" 2>/dev/null; then
    echo "Another run_query.sh is already running (PID $held_pid, query: $held_query)." >&2
    echo "Wait for it to finish, or if it's stuck: kill $held_pid && rm -rf $LOCK_DIR" >&2
    exit 1
  fi

  echo "Found a stale lock (owning process no longer running), clearing it." >&2
  rm -rf "$LOCK_DIR"
  mkdir "$LOCK_DIR"
  echo "$$" > "$LOCK_DIR/pid"
  echo "$1" > "$LOCK_DIR/query"
  trap 'rm -rf "$LOCK_DIR"' EXIT
}

if [ ! -f "$ENV_FILE" ]; then
  echo "Missing $ENV_FILE. Create it with REPORTING_DB_HOST/PORT/NAME/USER." >&2
  exit 1
fi
# shellcheck disable=SC1090
source "$ENV_FILE"

: "${REPORTING_DB_HOST:?REPORTING_DB_HOST not set in $ENV_FILE}"
: "${REPORTING_DB_PORT:?REPORTING_DB_PORT not set in $ENV_FILE}"
: "${REPORTING_DB_NAME:?REPORTING_DB_NAME not set in $ENV_FILE}"
: "${REPORTING_DB_USER:?REPORTING_DB_USER not set in $ENV_FILE}"

if [ $# -lt 1 ]; then
  echo "Usage: $0 <path-to-query.sql> [--no-open]" >&2
  exit 1
fi

QUERY_FILE="$1"
if [ ! -f "$QUERY_FILE" ] && [ -f "$SCRIPT_DIR/$QUERY_FILE" ]; then
  QUERY_FILE="$SCRIPT_DIR/$QUERY_FILE"
fi
if [ ! -f "$QUERY_FILE" ]; then
  echo "Query file not found: $1" >&2
  exit 1
fi

NO_OPEN=0
if [ "${2:-}" == "--no-open" ]; then
  NO_OPEN=1
fi

RESULTS_DIR="$SCRIPT_DIR/_results"
mkdir -p "$RESULTS_DIR"

acquire_lock "$QUERY_FILE"

BASENAME="$(basename "$QUERY_FILE" .sql)"
TS="$(date +%Y%m%d_%H%M%S)"
CSV_PATH="$RESULTS_DIR/${BASENAME}_${TS}.csv"

CONN="host=$REPORTING_DB_HOST port=$REPORTING_DB_PORT dbname=$REPORTING_DB_NAME user=$REPORTING_DB_USER sslmode=prefer"

ensure_vpn_connected

# Record activity so the idle-check LaunchAgent (com.jesse.woogles-tunnel-idle-check)
# knows a query just ran and keeps the tunnel up for the idle window. That agent
# -- not this script -- is responsible for disconnecting the tunnel once it's been
# sitting idle, so there is no fragile background process to spawn here.
touch "$RESULTS_DIR/.last_activity"

echo "Running $QUERY_FILE against $REPORTING_DB_USER@$REPORTING_DB_HOST/$REPORTING_DB_NAME ..." >&2

psql "$CONN" --csv -f "$QUERY_FILE" > "$CSV_PATH"

ROW_COUNT=$(($(wc -l < "$CSV_PATH") - 1))
echo "Saved $ROW_COUNT row(s) to $CSV_PATH" >&2
echo >&2

if command -v column >/dev/null 2>&1; then
  column -s, -t < "$CSV_PATH"
else
  cat "$CSV_PATH"
fi

if [ "$NO_OPEN" -eq 0 ] && command -v open >/dev/null 2>&1; then
  open "$CSV_PATH"
fi
