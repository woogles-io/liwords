#!/bin/bash

# Discord webhook notification script
# Usage:
#   Basic message: ./notify-discord.sh "Your message here"
#   With embeds:   DISCORD_EMBEDS='[{...}]' ./notify-discord.sh
#   Custom message with embeds: DISCORD_EMBEDS='[{...}]' ./notify-discord.sh "Optional message"

set -euo pipefail

# Check if DISCORD_WEBHOOK is set
if [ -z "${DISCORD_WEBHOOK:-}" ]; then
    echo "Warning: DISCORD_WEBHOOK environment variable is not set. Skipping notification." >&2
    exit 0
fi

# Get the message from command line argument or use empty string
MESSAGE="${1:-}"

# Replace GitHub Actions variables if we're in GitHub Actions context
if [ -n "${GITHUB_ACTIONS:-}" ]; then
    # Common GitHub variables that might be used in messages
    MESSAGE="${MESSAGE//\{\{GITHUB_REF_NAME\}\}/${GITHUB_REF_NAME:-}}"
    MESSAGE="${MESSAGE//\{\{GITHUB_REPOSITORY\}\}/${GITHUB_REPOSITORY:-}}"
    MESSAGE="${MESSAGE//\{\{GITHUB_RUN_NUMBER\}\}/${GITHUB_RUN_NUMBER:-}}"
    MESSAGE="${MESSAGE//\{\{GITHUB_SHA\}\}/${GITHUB_SHA:-}}"
    MESSAGE="${MESSAGE//\{\{GITHUB_ACTOR\}\}/${GITHUB_ACTOR:-}}"
    MESSAGE="${MESSAGE//\{\{GITHUB_EVENT_NAME\}\}/${GITHUB_EVENT_NAME:-}}"
fi

# Build the JSON payload
if [ -n "${DISCORD_EMBEDS:-}" ]; then
    # If embeds are provided, use them
    if [ -n "$MESSAGE" ]; then
        # Both message and embeds
        PAYLOAD=$(cat <<EOF
{
  "content": "$MESSAGE",
  "embeds": $DISCORD_EMBEDS
}
EOF
        )
    else
        # Only embeds
        PAYLOAD=$(cat <<EOF
{
  "embeds": $DISCORD_EMBEDS
}
EOF
        )
    fi
else
    # Simple message only
    if [ -z "$MESSAGE" ]; then
        echo "Error: No message provided and no DISCORD_EMBEDS set" >&2
        exit 1
    fi
    PAYLOAD=$(cat <<EOF
{
  "content": "$MESSAGE"
}
EOF
    )
fi

# Send the webhook request
HTTP_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Content-Type: application/json" \
    -X POST \
    -d "$PAYLOAD" \
    "$DISCORD_WEBHOOK")

if [ "$HTTP_RESPONSE" -eq 204 ]; then
    echo "Discord notification sent successfully"
elif [ "$HTTP_RESPONSE" -eq 200 ]; then
    echo "Discord notification sent successfully"
else
    echo "Warning: Discord notification may have failed (HTTP $HTTP_RESPONSE)" >&2
    # Don't exit with error to maintain the same behavior as continue-on-error: true
    exit 0
fi