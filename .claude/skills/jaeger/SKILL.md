---
name: jaeger
description: Query production Jaeger traces for liwords — latency per operation or endpoint, slow queries and N+1 loops, request rates, cache hit rates, which code paths call an operation, a span tag's top values (e.g. per tournament or game), or one trace as a tree. Talks to the OpenSearch span store on woogles-jaeger over ssh, never Jaeger's bulk trace API, which gets OOM-killed. Use when the user asks how a deploy, endpoint, query or span is doing in prod, what's slow, or to check traces.
---

# Querying prod Jaeger

Prod Jaeger stores spans in OpenSearch on the `woogles-jaeger` host
(`ssh woogles-jaeger`). Query OpenSearch there, on `localhost:9200`, and let it
aggregate.

**Don't pull traces in bulk through Jaeger's HTTP API** (`jaeger.woogles.io/api/traces`).
The Jaeger container has a 512 MiB limit and big searches kill it: `limit=20000`
did on 2026-10-24, and it has OOM'd dozens of times. Single-trace links in the UI
(`http://jaeger.woogles.io/trace/<id>`) are fine to hand to the user.

## The script

`.claude/skills/jaeger/scripts/jaeger_es.py` covers the usual questions. Run it
from the repo root; it needs only ssh access to `woogles-jaeger`.

| Question | Command |
|---|---|
| What's slow / busy overall? | `jaeger_es.py ops --since 1h --sort total` (or `--sort p99`, `--prefix tournament_service`) |
| How is a deploy doing? | `ops` with `--from <deploy time>`; compare with the same window before it |
| Requests per second over time | `jaeger_es.py rate gameplay.HandleEvent --since 6h --interval 5m` |
| Cache hit rate | `jaeger_es.py tag cache.Get cache.hit --where otel.scope.name=tournament-cache` |
| Busiest tournaments, games… | `jaeger_es.py tag cache.Get tournament.id --where otel.scope.name=tournament-cache` |
| What calls this? | `jaeger_es.py parents cache.Get --where otel.scope.name=tournament-cache` |
| Slowest instances, with links | `jaeger_es.py slow gameplay.HandleEvent --since 1h --show game.id` |
| Where did one request spend its time? | `jaeger_es.py trace <traceID>` |
| N+1 loops, repeated queries, uninstrumented time | `jaeger_es.py fetch --out <scratchpad>/traces --since 6h`, then `scripts/jaeger-audit.py --cache-dir <scratchpad>/traces` |

Common options: `--since 30m|1h|2d` (default 1h) or `--from/--to` (ISO 8601 or
epoch seconds), `--where key=value` (repeatable tag filter; `--where key` alone
means the tag exists), `--top N`, `--service` (default `liwords`).

For anything the script doesn't cover, write the query yourself (below), and
consider adding a subcommand.

## What's in the index

- One index per UTC day, `jaeger-span-YYYY-MM-DD`. A cron at 2 AM UTC deletes old
  ones, so **only about the last day exists**. Measurements that need a busy
  weekend have to be taken during or right after it.
- About 100 million spans and 5 GB a day. Always filter on `startTimeMillis`, and
  use `size: 0` with aggregations. Fetch documents only in small numbers.
- Fields:
  - `operationName` (keyword)
  - `duration`, in **µs**
  - `startTime`, in µs
  - `startTimeMillis`, a date in ms: filter on this one
  - `traceID`, `spanID`
  - `process.serviceName`: `liwords` for the API. The socket server sends no spans.
- `tags` is **nested** `{key, type, value}`, and `value` is a keyword, even for
  booleans (`"true"`). Filter with a nested query, and aggregate with `nested` →
  `filter` on `tags.key` → `terms` on `tags.value`, then `reverse_nested` to get
  back to the span's fields (such as `duration`).
- The parent is `references[0].spanID` (`references` is nested too). A span with
  no references is a trace root. For "who calls X", collect X's parent span IDs,
  then aggregate `operationName` over `{"terms": {"spanID": [...]}}`.
- `cache.Get` is used by more than one cache. `otel.scope.name` tells them apart:
  `tournament-cache`, `game-cache`.
- Operation names over 256 characters aren't indexed. Many sqlc query spans are
  named `query -- name: X :one` followed by the SQL, so a long one can't be found
  by name. Use the calling RPC span, or `trace` to see queries inside a request.

Useful operations: RPCs (`tournament_service.TournamentService/GetTournament`),
`gameplay.HandleEvent` (a move, from NATS), `nats.publish.*` (the NATS handler's
root span), `game.lock.wait` / `game.lock.held`, `game.Get`, `cache.Get`,
`pool.acquire`, `middleware.*` (one per HTTP request each; `middleware.hlog`
roughly counts HTTP requests).

## Raw queries

```bash
ssh -o LogLevel=ERROR woogles-jaeger \
  "curl -s -H 'Content-Type: application/json' 'localhost:9200/jaeger-span-*/_search' -d @-" <<'JSON'
{"size": 0, "timeout": "30s",
 "query": {"bool": {"filter": [
   {"range": {"startTimeMillis": {"gte": "now-1h"}}},
   {"term": {"operationName": "cache.Get"}},
   {"nested": {"path": "tags", "query": {"bool": {"must": [
     {"term": {"tags.key": "otel.scope.name"}}, {"term": {"tags.value": "tournament-cache"}}]}}}}]}},
 "aggs": {"p": {"percentiles": {"field": "duration", "percents": [50, 90, 99]}}}}
JSON
```

## Privacy

`pgx.query.parameters` holds query arguments, including session IDs and email
addresses. Never print it or put it in a report. Use it only to compare
arguments, as `jaeger-audit.py` does for duplicates. The script's `--show`
refuses it.
