#!/usr/bin/env bash
# One-shot idle check for the "Woogles" WireGuard tunnel. Run periodically by a
# launchd LaunchAgent (com.jesse.woogles-tunnel-idle-check, StartInterval 300s).
# Each invocation checks once and exits -- it is NOT a long-lived loop, so there
# is nothing for a terminating shell / Claude Code job to orphan (the old
# nohup-based idle_watchdog.sh died exactly that way, leaving the tunnel open).
#
# Disconnects the tunnel only when ALL of these hold:
#   1. the tunnel is currently connected, AND
#   2. no ESTABLISHED TCP connection is routed over the tunnel interface (this
#      tunnel is shared across projects, so ANY connection over it counts, not
#      just the reporting DB), AND
#   3. run_query.sh has recorded no activity within IDLE_SECONDS.
# (2)+(3) mean a live psql / VSCode session (any project) or a recent query keeps
# the tunnel up -- it is only reaped when genuinely sitting open with nothing
# using it.
#
# Usage:
#   reporting/tunnel_idle_check.sh            # check and act
#   reporting/tunnel_idle_check.sh --dry-run  # report the decision, change nothing
set -uo pipefail

VPN_SERVICE="Woogles"
IDLE_SECONDS=1800

DRY_RUN=0
if [ "${1:-}" == "--dry-run" ]; then
  DRY_RUN=1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/_results"
ENV_FILE="$SCRIPT_DIR/.env"
ACTIVITY_FILE="$RESULTS_DIR/.last_activity"
LOG_FILE="$RESULTS_DIR/.tunnel_idle_check.log"

log() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') $1" >> "$LOG_FILE"
}

# In dry-run, narrate to stdout as well as the log so a manual test is legible.
say() {
  log "$1"
  if [ "$DRY_RUN" -eq 1 ]; then
    echo "$1"
  fi
}

# --- 1. Is the tunnel even up? ------------------------------------------------
# Capture the status line into a variable and compare as a string. Do NOT pipe
# scutil into `head -1 | grep -q` inside an `if`: head closes the pipe early,
# scutil dies with SIGPIPE (141), and `set -o pipefail` then reports the whole
# pipeline as failed even when grep matched -- which would misread a live tunnel
# as down. `sed -n 1p` reads all input, so scutil never gets SIGPIPE.
STATUS_LINE="$(scutil --nc status "$VPN_SERVICE" 2>/dev/null | sed -n '1p')"
if [ "$STATUS_LINE" != "Connected" ]; then
  say "tunnel not connected (status: ${STATUS_LINE:-unknown}) -- nothing to do"
  exit 0
fi

# --- 2. Any live connection over the tunnel (ANY project)? --------------------
# This tunnel is shared: besides the reporting DB (10.0.0.76) it also routes the
# 10.19.88.0/24 subnet used by other projects. So "in use" must mean "any
# established connection routed over the tunnel interface", not just the
# reporting DB -- otherwise a busy other-project session could be cut off.
#
# The reporting DB host from .env is a stable, known IP on the tunnel; use it
# only to discover which interface the tunnel currently owns (the utun index can
# change across reconnects), then match every established connection against it.
DB_HOST=""
if [ -f "$ENV_FILE" ]; then
  # shellcheck disable=SC1090
  source "$ENV_FILE" 2>/dev/null || true
  DB_HOST="${REPORTING_DB_HOST:-}"
fi

route_iface() { route -n get "$1" 2>/dev/null | awk '/interface:/{print $2; exit}'; }

if [ -n "$DB_HOST" ]; then
  TUNNEL_IF="$(route_iface "$DB_HOST")"
  if [ -n "$TUNNEL_IF" ]; then
    # Foreign IPs of all established IPv4 TCP connections (strip the .port suffix).
    while read -r fip; do
      [ -z "$fip" ] && continue
      if [ "$(route_iface "$fip")" == "$TUNNEL_IF" ]; then
        say "established connection to $fip routes over tunnel ($TUNNEL_IF) -- keeping tunnel up"
        exit 0
      fi
    done < <(netstat -an 2>/dev/null \
      | awk '$1=="tcp4" && $NF=="ESTABLISHED" {print $5}' \
      | sed -E 's/\.[0-9]+$//' | sort -u)
  fi
fi

# --- 3. Recent run_query.sh activity? -----------------------------------------
if [ -f "$ACTIVITY_FILE" ]; then
  last="$(stat -f %m "$ACTIVITY_FILE" 2>/dev/null || stat -c %Y "$ACTIVITY_FILE" 2>/dev/null || echo 0)"
  now="$(date +%s)"
  elapsed=$(( now - last ))
  if [ "$elapsed" -lt "$IDLE_SECONDS" ]; then
    say "last activity ${elapsed}s ago (< ${IDLE_SECONDS}s) -- keeping tunnel up"
    exit 0
  fi
fi

# --- Idle: connected, no DB connection, no recent activity --> disconnect -----
if [ "$DRY_RUN" -eq 1 ]; then
  say "WOULD disconnect $VPN_SERVICE (idle: connected, no DB connection, no recent activity)"
  exit 0
fi

if scutil --nc stop "$VPN_SERVICE" >/dev/null 2>&1; then
  log "disconnected $VPN_SERVICE after >= ${IDLE_SECONDS}s idle"
else
  log "WARNING: scutil --nc stop $VPN_SERVICE failed"
fi
