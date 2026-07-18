---
description: Look up a player's IP addresses from CloudWatch liwords-socket logs and geolocate them
argument-hint: <username> [days=3]
allowed-tools: [Bash]
---

# Player IP Lookup

Look up IP addresses for a Woogles player from CloudWatch logs and geolocate each unique IP.

## Arguments

The user invoked this with: $ARGUMENTS

Parse the arguments as:
- First argument: the username to search for (required)
- Second argument: number of days to look back (optional, default: 3)

## Instructions

1. Parse the username and days from `$ARGUMENTS`. If no days value is given, use 3.

2. Run this command to fetch logs (substitute USERNAME and DAYS):

```bash
AWS_PROFILE=woogles-prod aws logs filter-log-events \
  --log-group-name "/ecs/liwords-socket" \
  --filter-pattern '{ $.username = "USERNAME" }' \
  --start-time $(date -d 'DAYS days ago' +%s000) \
  --query 'events[*].message' \
  --output json
```

3. From the results, extract all values of the `ips` field. The format is `"client_ip, aws_ip"` — the **first** IP is the client's IP; the second is AWS infrastructure (ignore it).

4. Collect all unique client IPs across all log entries.

5. For each unique client IP, run:

```bash
curl -s https://ipinfo.io/IP_HERE/json
```

6. Report:
   - The unique client IP(s) found with geolocation details (city, region, country, org/ISP, hostname)
   - A session summary table: date/time, client IP, duration (first to last pong in the session), connection ID
   - Note if the IP looks like a VPN, proxy, or datacenter vs. residential ISP

7. For each unique client IP, check whether any other registered players used it, over the **same DAYS window** (never unbounded — filtering by IP scans every log event, not just this user's, so it is much slower than the username filter and must stay time-boxed):

```bash
AWS_PROFILE=woogles-prod aws logs filter-log-events \
  --log-group-name "/ecs/liwords-socket" \
  --filter-pattern '{ $.ips = "IP_HERE*" }' \
  --start-time $(date -d 'DAYS days ago' +%s000) \
  --query 'events[*].message' \
  --output json
```

   Extract the `username` field from each matching event and collect the unique set. Exclude the original username and any `anon-*` entries (anonymous/pre-login sessions — expected noise, not a match) before reporting. If other registered usernames remain, call them out explicitly as a possible shared-IP/multi-account signal. If this step would need a longer lookback than DAYS to be useful, ask the user before widening it rather than scanning further unprompted.
