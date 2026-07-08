#!/usr/bin/env bash
# Monthly Woogles reporting: run the queries in QUERIES once per calendar month
# and email the results (inline summary + full CSVs attached) to Jesse. No git
# commits -- results live in the email and in reporting/_results/.
#
# Scheduling model: a launchd agent (com.jesse.woogles-monthly-report) invokes
# this script hourly whenever the Mac is awake, plus once at load. A stamp file
# records the last month successfully reported; if the stamp matches the current
# month the script exits immediately, so the hourly cadence costs nothing. On
# the 1st-2nd, attempts are further gated to the 18:00-23:59 UTC low-traffic
# window (see below) to keep the heavy queries off the DB's busy hours. This
# replaces the old fire-once-at-9am-on-the-1st StartCalendarInterval design,
# which lost the whole month whenever the machine was off, the tunnel failed, or
# launchd's calendar fired in a stale timezone (all three happened in July 2026;
# see git history of monthly_snapshot.sh, which this script supersedes).
#
# Failure handling: a failed attempt just logs and waits for the next hourly
# tick. If the report still hasn't gone out by the 3rd of the month, ONE alert
# email is sent (per month) so a persistent problem gets noticed without
# hourly spam.
#
# Usage: reporting/scripts/monthly_report.sh   (safe to run manually at any time)
set -uo pipefail

# launchd agents get PATH=/usr/bin:/bin:/usr/sbin:/sbin, which is missing psql
# (Postgres.app) and any Homebrew tools run_query.sh depends on.
export PATH="/Applications/Postgres.app/Contents/Versions/latest/bin:/opt/homebrew/bin:/usr/local/bin:$PATH"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORTING_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RESULTS_DIR="$REPORTING_DIR/_results"
LOG_FILE="$RESULTS_DIR/.monthly_report.log"
STAMP_FILE="$RESULTS_DIR/.monthly_report_sent"      # YYYY-MM of last successful send
ALERT_STAMP="$RESULTS_DIR/.monthly_report_alerted"  # YYYY-MM of last failure alert

# Kill a query attempt that hangs (e.g. scutil --nc start blocking forever, as
# on 2026-07-04). Generous: games_per_month alone takes ~6 min, and the
# all-time MAU query can run considerably longer than that.
QUERY_TIMEOUT_SECS=2700

# Queries to include in the monthly report, relative to the repo root.
QUERIES=(
  "reporting/omgwords/games_per_month.sql"
  "reporting/reporting/mau_reporting.sql"
)

mkdir -p "$RESULTS_DIR"
cd "$REPO_ROOT"

log() {
  echo "$(date '+%Y-%m-%d %H:%M:%S') $1" >> "$LOG_FILE"
}

PERIOD="$(date +%Y-%m)"
MONTH_NAME="$(date '+%B %Y')"
DAY_OF_MONTH=$((10#$(date +%d)))

if [ "$(cat "$STAMP_FILE" 2>/dev/null)" == "$PERIOD" ]; then
  exit 0
fi

# Prefer the low-traffic window for the heavy queries. Per the games-by-hour
# profile (adhoc/games_by_hour_of_day.sql, run Jul 2026; server clock is UTC),
# 18:00-23:59 UTC hours each carry <=3.4% of daily games vs 5.8% at the 13:00
# UTC peak. On the 1st and 2nd of the month only attempt inside that window;
# from the 3rd on, any hourly tick may run so a Mac that was asleep all
# afternoon can't hold the report hostage.
HOUR_UTC=$((10#$(date -u +%H)))
if [ "$DAY_OF_MONTH" -le 2 ] && [ "$HOUR_UTC" -lt 18 ]; then
  exit 0
fi

log "=== Report attempt for $PERIOD ==="

# Marker so we can find CSVs produced by THIS attempt (and ignore stale or
# 0-byte leftovers from earlier failed runs).
ATTEMPT_MARKER="$RESULTS_DIR/.monthly_report_attempt"
touch "$ATTEMPT_MARKER"

# Run a command with a timeout (no GNU timeout binary on this Mac). Kills the
# process and its children if it exceeds QUERY_TIMEOUT_SECS.
run_with_timeout() {
  "$@" >> "$LOG_FILE" 2>&1 &
  local pid=$!
  local waited=0
  while kill -0 "$pid" 2>/dev/null && [ "$waited" -lt "$QUERY_TIMEOUT_SECS" ]; do
    sleep 5
    waited=$((waited + 5))
  done
  if kill -0 "$pid" 2>/dev/null; then
    log "TIMEOUT after ${QUERY_TIMEOUT_SECS}s: $*"
    pkill -P "$pid" 2>/dev/null
    kill "$pid" 2>/dev/null
    wait "$pid" 2>/dev/null
    return 124
  fi
  wait "$pid"
}

ATTACHMENTS=()
FAILED=()

for QUERY_REL in "${QUERIES[@]}"; do
  BASENAME="$(basename "$QUERY_REL" .sql)"
  log "Running $QUERY_REL"

  if ! run_with_timeout "$SCRIPT_DIR/run_query.sh" "$QUERY_REL" --no-open; then
    log "FAILED: $QUERY_REL"
    FAILED+=("$QUERY_REL")
    continue
  fi

  CSV="$(find "$RESULTS_DIR" -maxdepth 1 -name "${BASENAME}_*.csv" -newer "$ATTEMPT_MARKER" -size +0c 2>/dev/null | sort | tail -1)"
  if [ -z "$CSV" ]; then
    log "FAILED: $QUERY_REL produced no (non-empty) result CSV"
    FAILED+=("$QUERY_REL")
    continue
  fi
  ATTACHMENTS+=("$CSV")
done

if [ "${#FAILED[@]}" -eq 0 ] && [ "${#ATTACHMENTS[@]}" -gt 0 ]; then
  BODY_FILE="$RESULTS_DIR/.monthly_report_body.txt"
  {
    echo "Woogles monthly reporting for $MONTH_NAME."
    echo "Generated $(date '+%Y-%m-%d %H:%M %Z'). Full results attached as CSV."
    echo "(This is the plain-text fallback; the HTML version has tables and charts.)"
  } > "$BODY_FILE"

  # HTML body with formatted tables + charts (render_report.py). If rendering
  # fails for any reason the email still goes out, just plain-text.
  HTML_FILE="$RESULTS_DIR/.monthly_report_body.html"
  HTML_ARGS=()
  INLINE_ARGS=()
  if PNGS="$(python3 "$SCRIPT_DIR/render_report.py" \
      --out-html "$HTML_FILE" --out-dir "$RESULTS_DIR" \
      --heading "Woogles monthly reporting for $MONTH_NAME, generated $(date '+%Y-%m-%d %H:%M %Z'). Full history attached as CSV." \
      "${ATTACHMENTS[@]}" 2>> "$LOG_FILE")"; then
    HTML_ARGS=(--html-file "$HTML_FILE")
    while IFS= read -r PNG; do
      [ -n "$PNG" ] && INLINE_ARGS+=(--inline "$PNG")
    done <<< "$PNGS"
  else
    log "WARNING: render_report.py failed; sending plain-text email"
  fi

  ATTACH_ARGS=()
  for CSV in "${ATTACHMENTS[@]}"; do
    ATTACH_ARGS+=(--attach "$CSV")
  done

  if python3 "$SCRIPT_DIR/send_mail.py" \
      --subject "Woogles Monthly Reporting - $MONTH_NAME" \
      --body-file "$BODY_FILE" \
      "${HTML_ARGS[@]}" "${INLINE_ARGS[@]}" \
      "${ATTACH_ARGS[@]}" >> "$LOG_FILE" 2>&1; then
    echo "$PERIOD" > "$STAMP_FILE"
    log "Report for $PERIOD sent (${#ATTACHMENTS[@]} attachment(s))"
  else
    log "FAILED: queries succeeded but sending the report email failed; will retry next tick"
  fi
else
  log "Attempt failed (${#FAILED[@]} query failure(s)); will retry on the next hourly tick"
  if [ "$DAY_OF_MONTH" -ge 3 ] && [ "$(cat "$ALERT_STAMP" 2>/dev/null)" != "$PERIOD" ]; then
    if python3 "$SCRIPT_DIR/send_mail.py" \
        --subject "Woogles monthly reporting FAILED for $PERIOD" \
        --body "The monthly report has not gone out yet this month; attempts keep failing (latest failures: ${FAILED[*]:-none}). Retries continue hourly. Check $LOG_FILE for details." \
        >> "$LOG_FILE" 2>&1; then
      echo "$PERIOD" > "$ALERT_STAMP"
      log "Sent one-time failure alert for $PERIOD"
    else
      log "WARNING: failed to send failure-alert email"
    fi
  fi
fi

log "=== Done ==="
