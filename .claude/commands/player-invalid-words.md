---
description: Look up a player's invalid word attempts (VOID challenge rejections) from CloudWatch liwords-api logs
argument-hint: <username> [days=90]
allowed-tools: [Bash]
---

# Player Invalid Words Lookup

Search CloudWatch logs for words a Woogles player tried to play that were rejected in VOID-challenge games.

## Arguments

The user invoked this with: $ARGUMENTS

Parse the arguments as:
- First argument: the username to search for (required)
- Second argument: number of days to look back (optional, default: 90)

## Instructions

1. Parse the username and days from `$ARGUMENTS`. If no days value is given, use 90.

2. Look up the player's UUID from the production database:

```bash
bin/proddb -c "SELECT uuid FROM users WHERE username ILIKE 'USERNAME';"
```

If no user is found, report that the username doesn't exist and stop.

3. Start a CloudWatch Logs Insights query using the UUID (substitute UUID and DAYS).

   **Primary query** — rich logs (emitted after the logging improvement was deployed):
   ```bash
   AWS_PROFILE=woogles-prod aws logs start-query \
     --log-group-name /ecs/liwords-api \
     --start-time $(date -d 'DAYS days ago' +%s) \
     --end-time $(date +%s) \
     --query-string 'fields @timestamp, @message | filter @message like "UUID" and @message like "invalid-word-play" | sort @timestamp asc | limit 10000' \
     --output text
   ```

   **Fallback query** — old-format logs (from before the improvement, no game ID):
   ```bash
   AWS_PROFILE=woogles-prod aws logs start-query \
     --log-group-name /ecs/liwords-api \
     --start-time $(date -d 'DAYS days ago' +%s) \
     --end-time $(date +%s) \
     --query-string 'fields @timestamp, @message | filter @message like "UUID" and @message like "process-message-publish-error" and @message like "invalid words" | sort @timestamp asc | limit 10000' \
     --output text
   ```

   Run **both** queries. Each returns a query ID.

4. Wait a few seconds, then fetch the results for each query ID:

```bash
sleep 5 && AWS_PROFILE=woogles-prod aws logs get-query-results \
  --query-id QUERY_ID \
  --output json
```

If the status is not `Complete`, wait a few more seconds and retry.

5. Parse the results:

   **Rich logs** (`invalid-word-play`): Each log entry is JSON like:
   ```json
   {"level":"info","gameID":"AbCxYz123","userID":"3eBxxpLe84ikdRDu5nn896","words":["CRANE","AR","NE"],"error":"the play contained invalid words: AR, NE","time":"...","message":"invalid-word-play"}
   ```
   Fields: `gameID`, `userID`, `words` (all formed words — main word + cross-words), `error` (which of those words were invalid).

   **Old-format logs** (`process-message-publish-error`): Each entry has only the `error` field containing `"the play contained invalid words: WORD1, WORD2"`. No game ID, no valid words. Label these with "(legacy log — no game ID)".

6. Report:
   - Total count of rejected plays (rich + legacy combined)
   - Results grouped by game ID (for rich logs), then chronologically within each game. For each rejected play show:
     - Timestamp
     - All words formed: list `words` with invalid ones flagged (cross-reference with the `error` field)
     - Game ID (or "(legacy — no game ID)")
   - Legacy-log entries listed separately if any, with a note that they predate the richer logging.
   - Any patterns worth noting: words probed multiple times, two-letter fishing, common root being inflected, etc.
