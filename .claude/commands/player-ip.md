---
description: Look up a player's IP addresses and client identifiers (DB first, CloudWatch logs as fallback), find other accounts that share them, and geolocate the IPs
argument-hint: <username> [days=3]
allowed-tools: [Bash]
---

# Player IP Lookup

Look up where a Woogles player connects from (IP addresses and first-party
client identifiers), find other accounts sharing them, and geolocate each IP.

**The database is the primary source.** CloudWatch logs are a fallback, used only
when the DB can't answer the question (see "When to fall back to logs").

## Arguments

The user invoked this with: $ARGUMENTS

Parse the arguments as:
- First argument: the username to look up (required)
- Second argument: number of days to look back (optional, default: 3). This only
  bounds **log** searches; DB lookups use everything the DB retains.

## Data available in the DB

- `users.registration_ip`, `users.registration_client_id` — where the account was
  created from. Written once; NULL for accounts created before this was recorded.
- `user_clients (user_id, ip, client_id, first_seen, last_seen)` — one row per
  distinct (ip, client_id) the account has been seen from while logged in
  (login, and every socket connect / page load). `last_seen` is refreshed at most
  hourly. Rows not seen for 180 days are pruned.
  - `client_id` is a random ID stored in a long-lived first-party cookie, so it
    identifies a **browser**, independent of IP. An empty `client_id` means the
    request came from a browser that didn't have the cookie yet (its first
    request); the next request from that browser records the real ID.

Run queries with `bin/proddb -c "..."` (or `-Atc` for unaligned output) from the
repo root.

## Instructions

1. **Resolve the account** case-insensitively (usernames are case-sensitive in
   logs, so get the exact spelling here):

```sql
SELECT u.id, u.uuid, u.username, u.email, u.created_at, u.verified,
       host(u.registration_ip) AS registration_ip, u.registration_client_id,
       (SELECT count(*) FROM games g WHERE g.player0_id = u.id OR g.player1_id = u.id) AS games,
       (SELECT string_agg(ua.action_type::text || '@' || ua.start_time::date, ',')
          FROM user_actions ua WHERE ua.user_id = u.id AND ua.removed_time IS NULL) AS active_actions
FROM users u WHERE u.username ILIKE 'USERNAME';
```

   If nothing matches, try `ILIKE '%USERNAME%'` and confirm with the user.

2. **Where the account connects from:**

```sql
SELECT host(uc.ip) AS ip, uc.client_id, uc.first_seen, uc.last_seen
FROM user_clients uc JOIN users u ON u.id = uc.user_id
WHERE u.username = 'EXACT_USERNAME'
ORDER BY uc.last_seen DESC;
```

3. **Other accounts sharing an IP or client ID** — across both registration and
   seen-from records:

```sql
WITH me AS (SELECT id, registration_ip, registration_client_id FROM users WHERE username = 'EXACT_USERNAME'),
     my_ips AS (SELECT ip FROM user_clients WHERE user_id = (SELECT id FROM me)
                UNION SELECT registration_ip FROM me WHERE registration_ip IS NOT NULL),
     my_cids AS (SELECT client_id FROM user_clients WHERE user_id = (SELECT id FROM me) AND client_id <> ''
                 UNION SELECT registration_client_id FROM me WHERE registration_client_id IS NOT NULL),
     hits AS (
       SELECT uc.user_id, 'client_id' AS via, uc.client_id AS value, uc.last_seen FROM user_clients uc
        WHERE uc.client_id IN (SELECT client_id FROM my_cids)
       UNION ALL
       SELECT u.id, 'registration_client_id', u.registration_client_id, u.created_at FROM users u
        WHERE u.registration_client_id IN (SELECT client_id FROM my_cids)
       UNION ALL
       SELECT uc.user_id, 'ip', host(uc.ip), uc.last_seen FROM user_clients uc
        WHERE uc.ip IN (SELECT ip FROM my_ips)
       UNION ALL
       SELECT u.id, 'registration_ip', host(u.registration_ip), u.created_at FROM users u
        WHERE u.registration_ip IN (SELECT ip FROM my_ips))
SELECT u.username, u.created_at, h.via, h.value, max(h.last_seen) AS last_seen,
       (SELECT string_agg(ua.action_type::text, ',') FROM user_actions ua
         WHERE ua.user_id = u.id AND ua.removed_time IS NULL) AS active_actions
FROM hits h JOIN users u ON u.id = h.user_id
WHERE h.user_id <> (SELECT id FROM me)
GROUP BY u.username, u.created_at, h.via, h.value, u.id
ORDER BY h.via, last_seen DESC;
```

4. **Geolocate** each distinct IP:

```bash
curl -s https://ipinfo.io/IP_HERE/json
```

5. **Report:**
   - The account summary from step 1 (created, games, active mod actions).
   - Registration IP / client ID, if recorded.
   - A table of IPs with geolocation (city, region, country, org/ISP, hostname),
     which client IDs used each, and first/last seen. Note whether each IP looks
     like residential ISP, mobile carrier, VPN/proxy, or datacenter.
   - Other accounts found in step 3, grouped by how they matched.

## Interpreting matches

- **Shared `client_id`** is a strong signal: the same browser profile was used
  for both accounts. (Exceptions: a shared/family computer, or a public machine.)
- **Shared IP** is weaker. Mobile carriers (CGNAT), universities, offices, VPN exit
  nodes and public wifi put many unrelated people behind one address. A
  residential ISP address shared by two accounts is more meaningful than a mobile
  or VPN one.
- **Absence of a match is not proof.** Clearing site data or using a private
  window yields a new `client_id`; a VPN or different network yields a new IP.
- Say "connected from the same browser/IP", not "is the same person".

## When to fall back to logs

Use the CloudWatch socket logs only if the DB can't answer:
- the account has **no `user_clients` rows** (nothing recorded since this data
  started being collected, or no logged-in activity within the retention window);
- you need **per-session detail** (exact connection times and durations), which
  the hourly `last_seen` doesn't give;
- you need **logged-out activity** — the DB only records logged-in accounts.
  Anonymous (`anon-*`) visitors on an IP or client ID are only in the logs.

Log searches are slow (they scan every event in the window), so always bound them
by DAYS, and ask before widening the window.

Socket log lines (`got-pong`, every 10th pong) carry `username`, `ips`
(`"client_ip, cdn_ip"` — the **first** IP is the client), `clientID`, and `connID`.
Filter by username (use the exact spelling from step 1), IP, or client ID:

```bash
AWS_PROFILE=woogles-prod aws logs filter-log-events \
  --log-group-name "/ecs/liwords-socket" \
  --filter-pattern '{ $.username = "EXACT_USERNAME" }' \
  --start-time $(date -d 'DAYS days ago' +%s000) \
  --query 'events[*].message' \
  --output json
```

Replace the filter with `'{ $.ips = "IP_HERE*" }'` or
`'{ $.clientID = "CLIENT_ID" }'` to find every session — including `anon-*`
ones — on an IP or browser. When reporting other usernames from logs, exclude the
target and treat `anon-*` sessions as logged-out activity (worth describing, not a
match by themselves). Log retention is 14 days.
