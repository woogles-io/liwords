# Infrastructure deep-dive Q&A

**Date:** 2026-04-19
**Base commit:** `f3ab03aafd860aa93d934bc60687581f7784bf06` (master)
**Postgres version on prod:** 14.6 on 2-core EC2
**Start here:** `index.md` — index, topic, reading order. Read that first if you are new to this audit.

**Scope:** Reference document capturing the detailed reasoning behind the three actionable specs:
- `deploy-safety.md`
- `games-storage-redesign.md`
- `stack-and-stores-cleanup.md`

This document preserves the question-and-answer form of the investigation for future readers who need the "why" behind the decisions. The three spec documents are the actionable outputs; this document is the supporting analysis.

> **Reader signpost:** This document evolved across a long conversation. Some early sections are refined or corrected by later ones. Quick map:
> - §5 (Protobuf vs JSONB) → refined by §10 (UPDATE-rewrites-column is not JSONB-specific) and §21 (drop all bytea).
> - §13 (Promote JSONB to columns with dual-write) → **superseded for most cases** by PG 18 virtual generated columns; see §26.
> - §15 (Partitioning after the fact) mentions "logical replication to new cluster" → cutover mechanics are in §25 (pgBouncer, not DNS).
> - §19 explicitly recommends a **two-table** pattern (active `games` + completed `past_games` partitioned monthly on `ended_at` UTC), not a single LIST-partitioned `games`. Aligns with PR #1503 and existing `game_players` precedent. Do not default to monthly time-range partitioning on a single table.
> - §26 (PG version upgrade path) is the definitive recommendation: **target PG 18.3+ directly, skip 17**. Earlier tables / sections citing "target 17" should be read as superseded.
> - §22 (LRU cache) recommends measure-first, retire-later. Not "delete immediately."

---

## Table of contents

1. [Why redeploying the backend causes downtime](#1-why-redeploying-the-backend-causes-downtime)
2. [In-memory game, tournament, and other caches across restarts](#2-in-memory-game-tournament-and-other-caches-across-restarts)
3. [What is needed for 100% safe multi-instance and rolling deploys](#3-what-is-needed-for-100-safe-multi-instance-and-rolling-deploys)
4. [DB stores + caches pattern: transactions and multi-table joins](#4-db-stores--caches-pattern-transactions-and-multi-table-joins)
5. [Protobuf binary vs JSONB in the database](#5-protobuf-binary-vs-jsonb-in-the-database)
6. [Ever-growing games table and hours-long backups](#6-ever-growing-games-table-and-hours-long-backups)
7. [Storing moves on the same row (UPDATE) vs INSERT-per-move](#7-storing-moves-on-the-same-row-update-vs-insert-per-move)
8. [Postgres-only vs Redis + NATS + possibly ClickHouse](#8-postgres-only-vs-redis--nats--possibly-clickhouse)
9. [Non-lossy JSONB in Go; is protobuf actually safer?](#9-non-lossy-jsonb-in-go-is-protobuf-actually-safer)
10. [UPDATE rewrites whole column: also true for protobuf bytea?](#10-update-rewrites-whole-column-also-true-for-protobuf-bytea)
11. [`games.history` queried for words: inappropriate as protobuf?](#11-gameshistory-queried-for-words-inappropriate-as-protobuf)
12. [`.proto` file licensing under AGPL](#12-proto-file-licensing-under-agpl)
13. [Promote JSONB fields to columns without downtime](#13-promote-jsonb-fields-to-columns-without-downtime)
14. [Queryable game metadata even when game content is cold-storaged](#14-queryable-game-metadata-even-when-game-content-is-cold-storaged)
15. [Partitioning an existing table after the fact](#15-partitioning-an-existing-table-after-the-fact)
16. [TOAST compression, AWS RDS support, and 2-core prod](#16-toast-compression-aws-rds-support-and-2-core-prod)
17. [Autovacuum and downtime](#17-autovacuum-and-downtime)
18. [ELI5 HOT (heap-only tuple)](#18-eli5-hot-heap-only-tuple)
19. [Partitioning active vs completed games (games can span months)](#19-partitioning-active-vs-completed-games-games-can-span-months)
20. [Read-as-replay cost with per-move rows](#20-read-as-replay-cost-with-per-move-rows)
21. [Drop all bytea in favor of JSONB?](#21-drop-all-bytea-in-favor-of-jsonb)
22. [Is the LRU cache paying off?](#22-is-the-lru-cache-paying-off)
23. [Chat storage: Redis to Postgres](#23-chat-storage-redis-to-postgres)
24. [AGPL and downstream projects (e.g. omgbot)](#24-agpl-and-downstream-projects-eg-omgbot)
25. [DNS flip TTL lag and pgBouncer alternative](#25-dns-flip-ttl-lag-and-pgbouncer-alternative)
26. [PG version upgrade path: 14 to 16, 17, 18](#26-pg-version-upgrade-path-14-to-16-17-18)

---

## 1. Why redeploying the backend causes downtime

Root causes observed in the current code at the base commit:

### Single task / no rolling deploy
If ECS runs one task per service, the sequence stop-old-then-start-new leaves a gap. Safe rolling deploy requires `minimumHealthyPercent = 100`, `maximumPercent = 200`, and at least two tasks per service, with ALB health checks passing before old tasks drain.

### WebSocket connections severed
`cmd/socketsrv/main.go:82` wires `srv.Shutdown(ctx)` with a 30s timeout. `http.Server.Shutdown` stops accepting new requests and waits for active handlers to return, but WebSocket connections are hijacked out of the HTTP lifecycle and do not return. At the 30s mark, the process is SIGKILL'd and every connected client receives a TCP reset. All clients reconnect simultaneously (thundering herd), and any in-flight outgoing message is lost.

### ALB deregistration delay vs graceful window
ALB default `deregistration_delay.timeout_seconds` is 300s. The app process has a 30s graceful shutdown (`GracefulShutdownTimeout = 30 * time.Second` at `cmd/liwords-api/main.go:92`). During the 270s gap, ALB continues to route a small number of new connections to the dead task's IP, returning 502s.

### New task not healthy before old stops
Cold start includes DB pool init, NATS connect, and the schema version check (when migrations are gated off). If the new task's readiness lags past the ALB health-check grace, there is a window with zero healthy targets.

### In-memory NATS subscriptions lost
`pkg/bus/bus.go:106-123` sets up NATS subscriptions per instance. During a restart, events published to those subjects between the old instance's unsubscribe and the new instance's subscribe are dropped (core NATS pub/sub, not JetStream). Clients observe stale state until they refetch.

### Inline migrations
`cmd/liwords-api/main.go:159-181` runs migrations when `cfg.RunMigrations = true`. In production this is set to false (separate migration task). The comment at line 158 confirms: "In production, migrations are run via a separate ECS task before deployment." This is the right structure; no issue in production, but watch for drift in staging configs.

These issues compound during a two-instance rolling deploy: each task restart hits the same issues, and clients see intermittent failures for the duration of the deployment.

---

## 2. In-memory game, tournament, and other caches across restarts

The backend has several in-memory state layers. Most survive restarts; some raise different cross-instance concerns.

### Game LRU cache

`pkg/stores/game/cache.go:59` declares `CacheCap = 400` games. The comments at `:51-58` show the maintainer's sizing math: roughly 300MB for 400 game slots. `Cache.Set` and `Cache.Create` write the DB first and then the cache, so a crash never leaves phantom data in the new process. On restart the cache is cold; the first reads hit the DB directly. **No data loss**, brief DB-read spike as warm traffic refills the cache.

### Tournament LRU cache

`pkg/stores/tournament/cache.go:51` declares `CacheCap = 50`. Same write-through pattern, same cold-start behavior. **No data loss**.

### Active games list cache

5-second TTL (`cache.go:122`). Trivial during deploys.

### Game timers

`pkg/entity/game.go:26-32` stores `TimeStarted` and `TimeRemaining` in a persisted JSONB `Timers` struct. `GameTimer.Now()` at `:272` uses wall-clock (`time.Now().UnixMilli()`). After restart, clocks are reconstructed exactly from persisted DB state. **No drift, no loss.**

### Correspondence game locks

`pkg/stores/game/cache.go:92` declares `gameLocks map[string]*gameLock` protected by an in-process `sync.Mutex`. These are process-local locks for serializing access to correspondence game mutations. On restart the map is empty; the new process rebuilds locks on demand. **Not a restart issue; however it is a multi-instance correctness issue** (see section 3, issue I2).

### NATS subscriptions

Per-instance. On graceful shutdown via `pubsubCancel()` at `cmd/liwords-api/main.go:523`, the subscriptions are torn down. Events fired during the blackout window are dropped. **Not a data loss in durable storage, but visible as stale UI until client refetches.**

### Presence / session

Lives in Redis (`pkg/stores/redis/redis_presence.go`), not in-memory. Survives restarts and is cross-instance coherent by virtue of Redis being shared.

### Chat recency

Lives in Redis (`pkg/stores/redis/redis_chat.go`). Same story as presence. (See section 23 for proposal to move to Postgres.)

### Summary

Caches do not cause restart data loss because of the write-through pattern. The real issues show up when two instances run concurrently: process-local locks lose their meaning, process-local LRU caches diverge, and process-local NATS subscriptions compete.

---

## 3. What is needed for 100% safe multi-instance and rolling deploys

The deploy-safety spec (`deploy-safety.md`) enumerates eight issues and eight prioritized fixes. Summary here; full detail in that spec.

### Issues

**I1. Background tickers assume singleton.** Real-time adjudicator (`pkg/bus/bus.go:384`), correspondence adjudicator (`:392`), seek expirer (`:400`), broadcast poller (`pkg/broadcasts/poller.go:23`), VDO webhook poller (`pkg/vdowebhook/service.go:312`), and analysis reclaim worker (`pkg/analysis/service.go:405`) all run per-instance. Two instances means double adjudication, duplicate polls, race on `TimedOut` (`pkg/gameplay/game.go:737`).

**I2. Correspondence game lock is process-local.** `pkg/stores/game/cache.go:92` is in-process only. Two instances can mutate the same game concurrently; the idempotency guard `Playing() == GAME_OVER` (at `game.go:753`) is best-effort and allows stale-read overwrite.

**I3. LRU caches are not coherent across instances.** Instance A writes, its LRU is fresh; instance B's LRU is stale. No invalidation channel exists.

**I4. Events use core NATS (fire-and-forget).** Queue-group subscriptions are already safe for inbound (section 8); the outbound fan-out lacks replay. Blackout windows drop events.

**I5. WebSockets not drained on shutdown.** See section 1.

**I6. Shutdown timeout not aligned with ALB deregistration.** See section 1.

**I7. `liwords-worker` role is not explicit.** All tickers live in `liwords-api`. Fixing I1 by Redis-lock leader election duplicates scheduling logic; splitting to a worker service is cleaner.

**I8. `gameEventChan` / per-instance channel writes.** `pkg/bus/bus.go:82` declares in-process channels; consumers must always republish to NATS for cross-instance fan-out. Needs verification that no in-proc-only consumer exists.

### Fixes

**P1. Schema version guard on API startup.** Fail fast if the DB schema is older than the build expects. XS effort.

**P2. Split a `liwords-worker` service.** New binary running all background tickers; desiredCount=1 ECS service; `liwords-api` loses the tickers. Medium effort, high risk reduction. Fixes I1 cleanly without leader-election code.

**P3. Postgres advisory lock for game mutations.** Replace the in-process `gameLocks` map with `pg_advisory_xact_lock(hashtextextended(game_id, 0))` inside a DB transaction. Delete `gameLocks`, `gameLocksMu`, `cleanupExpiredLocks`, `StopCleanup`. Fixes I2. Small effort.

**P4. Cross-instance LRU cache invalidation via NATS.** Publish `cache.invalidate.game.<id>` (non-queue-group) on every `Cache.Set` / `Cache.Create` / `Cache.Unload`. Each instance subscribes and evicts. Fixes I3.

**P5. WebSocket drain on shutdown.** Add `Hub.StopAccepting()` and `Hub.DrainAndClose(ctx, 1012, "service restart")` methods. On SIGTERM: stop new upgrades, send close frames to all clients, wait up to 20s for unregister, then `srv.Shutdown`. Frontend handles code 1012 as immediate reconnect with no backoff. Fixes I5.

**P6. Align graceful shutdown with ALB deregistration.** Bump `GracefulShutdownTimeout` to 90s. Add 15s pre-drain sleep on SIGTERM before `srv.Shutdown`. Set ALB `deregistration_delay.timeout_seconds = 70`. Set ECS `minimumHealthyPercent = 100`, `maximumPercent = 200`, `healthCheckGracePeriodSeconds = 90`, `desiredCount >= 2`. Fixes I6.

**P7. Verify `gameEventChan` publish path.** Trace all writes and confirm every consumer eventually republishes to NATS. Rewrite any in-proc-only path. XS effort, unknown risk reduction until traced.

**P8. Migrate events to NATS JetStream.** Convert `game.>`, `tournament.>`, `user.>` subjects to JetStream with `AckExplicit` and `DeliverLastPerSubject`. Clients reconnect → consumer replays missed events. Large effort, optional.

### Post-fix topology

| Service | desiredCount | ALB | Contains |
|---------|--------------|-----|----------|
| `liwords-api` | ≥ 2 | yes | HTTP / ConnectRPC, NATS queue-group subs, LRU caches with invalidation |
| `socketsrv` | ≥ 2 | yes (sticky not needed) | WebSocket fan-out, presence in Redis |
| `liwords-worker` | 1 | no | adjudicators, seek expirer, broadcast/VDO pollers, analysis reclaim |
| `db-migration-task` | one-shot | no | runs pre-deploy, blocks next step |

Deploy order: migration task → worker → api + socketsrv (parallel rolling).

---

## 4. DB stores + caches pattern: transactions and multi-table joins

### Current state

`pkg/stores/stores.go:25-50` declares the store registry. Line 27 and 33 contain maintainer notes: "we need to get rid of this cache" and "this cache too". Line 45: "We probably are going to be moving everything to a single queries thingy." The direction is toward sqlc-generated `*models.Queries`.

### Transaction scoping problem

Every `DBStore.<Method>` opens its own `BeginTx`:

- `pkg/stores/game/db.go:396, 553`
- `pkg/stores/session/db.go:36, 64, 91, 110, 131`
- `pkg/stores/mod/db.go:29, 74`
- `pkg/stores/stats/db.go:33, 56`
- `pkg/omgwords/stores/db.go:43, 92, 122, 147`

The `pkg/gameplay/*` business-logic layer has **zero** `BeginTx` calls. A typical "make a move" flow:

1. `stores.GameStore.Get(ctx, id)` — opens tx, closes, returns entity
2. Compute new state in memory
3. `stores.GameStore.Set(ctx, entity)` — opens tx, closes
4. `stores.UserStore.SetRatings(...)` if rated — opens tx, closes
5. `leagueUpdater.Update(...)` if league game — opens tx, closes

Five separate commits for one logical operation. Any crash or rollback mid-sequence leaves inconsistent state.

### Multi-table joins

`pkg/stores/common/db.go:112-196` has hand-rolled helpers taking `pgx.Tx`: `GetUserDBIDFromUUID`, `GetUserBy`, `GetGameInfo`, etc. These compose inside a transaction, but higher-level `DBStore.Get` methods each open their own tx and cannot be composed across entities without nesting.

`common.GetUserBy` with `IncludeProfile=true` (`:312, :346`) does two separate `QueryRow` calls on the same tx. Could be a single LEFT JOIN.

### Two SQL systems coexist

- sqlc `*models.Queries` (generated, composable via `WithTx`)
- Hand-rolled `common.go` helpers

Migration path per the maintainer comment is toward sqlc-only. Retiring `common.go` is mostly mechanical.

### Cache does not participate in transactions

`Cache.Set` → `backing.Set` (its own tx) → `c.cache.Add`. Caller has no tx handle, cannot roll back. Since business-layer code does not open outer transactions anyway, this has not caused bugs in practice. It will become a problem the moment business-layer transactions are introduced.

### Concurrency defense today

`TimedOut` at `pkg/gameplay/game.go:753` checks `Playing() == GAME_OVER` as an idempotency guard. This catches late arrivals, but the read-then-check-then-write window allows concurrent mutators to interleave.

### Target: unit-of-work pattern

Add `Stores.InTx(ctx, func(q *models.Queries) error { ... })`. Business layer opens one transaction, passes a tx-bound `*Queries` into all sub-operations. Combined with advisory locks (P3), this gives atomic cross-entity mutation with entity-level serialization.

Detail in the stack-and-stores-cleanup spec.

### Hot-queryable fields in JSONB

- `games.quickdata`: holds `tournament_id`, `original_request_id`, `player_info`. Queried via `->>`. The hash index at `:137` (`rematch_req_idx ON games USING hash ((quickdata ->> 'o'::text))`) confirms hot access. **These should be proper columns.**
- `games.timers`: small fixed shape (`TimeRemaining`, `TimeOfLastUpdate`, `TimeStarted`, `MaxOvertime`). Rewritten every move. Should be four columns, not JSONB.
- `profiles.ratings`, `profiles.stats`: legitimate use of JSONB — variant key → rating struct map. But the hot-path `jsonb_set` updates on every rated game are costly.

---

## 5. Protobuf binary vs JSONB in the database

> **Signpost:** The "UPDATE rewrites whole column" framing here is refined in §10 — that cost is not JSONB-specific; bytea pays the same price. Section §21 gives the final recommendation: drop all bytea from the DB in favor of JSONB + columns.

### Tradeoff matrix

| Encoding | Disk size | Queryable in SQL | Type safety | Ops readability |
|----------|-----------|------------------|-------------|-----------------|
| bytea (proto) | smallest | no | yes | no |
| JSONB | 2-4x bigger (less with lz4 TOAST) | yes (path ops, GIN) | loose | yes |
| proper columns | smallest for scalars | yes (native, indexable) | yes | yes |
| JSONB with GIN | bigger + index overhead | yes (fast) | loose | yes |

Decision matrix (access pattern → encoding):
- Indexed or filtered → column
- Open-ended map whose keys evolve → JSONB
- Opaque write-once blob, never queried by content → bytea proto
- Structured and searchable, ops needs readability → JSONB (possibly with column extraction of hot fields)

Protobuf wins on wire size and type safety. JSONB wins on ops inspection, SQL queryability, and standard tooling. Columns win when access is repeated and well-known.

### Applied to liwords today (current state)

- `games.history` = binary proto bytea. Written whole every move. Queried by game load. **Inappropriate** because word-level search requires decoding every row in Go (see section 11).
- `games.request`, `games.game_request` = mixed bytea and JSON-proto. Inconsistent encoding.
- `games.quickdata` = JSONB, contains hot-queryable fields.
- `games.timers` = JSONB with small fixed shape.
- `games.stats`, `games.meta_events` = JSONB.
- `profiles.ratings`, `profiles.stats` = JSONB with `jsonb_set` hot-path updates.

### Target after storage redesign

- No bytea anywhere in DB. Protobuf remains the RPC and NATS wire format; DB encodes via `protojson.Marshal`.
- `games` table has hot fields promoted to columns (`tournament_id`, `lexicon`, `variant`, timer fields).
- `games.stats`, `games.meta_events` stay JSONB (open-ended).
- New `game_moves` table has one row per move with both promoted columns (`word`, `rack`, `score`, `position`, `player_idx`) and full `event jsonb` for replay.
- Disk cost: ~2x current proto size, narrowed by lz4 TOAST. Net: roughly neutral long-term after `history bytea` is dropped.

### Why keep protobuf on the wire but not in DB

- Proto gives compact serialization over the network (smaller RPC payloads).
- Proto gives schema-evolution guarantees (reserved field numbers) that are well-suited to multi-client version drift.
- JSONB in DB gives SQL queryability and ops inspection.
- `protojson.Marshal` bridges the two; the proto file remains the schema of record.

---

## 6. Ever-growing games table and hours-long backups

### Why backups are slow

- `pg_dump` is single-threaded per table. Compression is CPU-bound. On a 2-core EC2 box, one core is saturated for the whole dump.
- The `games` table holds every game ever played. Proportional to all-time activity, not current activity.
- `history bytea` grows every move within a single row. TOAST chunks accumulate.
- UPDATE-heavy workload creates dead tuples faster than autovacuum reclaims. Dump reads dead space as well as live.
- No partitioning means a full table scan per dump.

### Fixes in order of ROI

**F1. Physical incremental backups.** Replace `pg_dump` with pgBackRest or WAL-G (or `pg_basebackup --incremental` on PG 17+). Full basebackup once, WAL archive continuous. Backup window drops from hours to minutes. No CPU-bound per-table serial compression. No schema changes needed.

**F2. Two-table pattern: keep active `games` unpartitioned; put completed games in `past_games` partitioned monthly on `ended_at` UTC.** Old partitions immutable → back up once, skip forever. Detach and archive past retention. See section 19.

**F3. Repack existing bloat.** `pg_repack` online reclaims dead space without locks. Run after any migration touching many rows.

**F4. TOAST compression tuning.** `ALTER TABLE games ALTER COLUMN history SET COMPRESSION lz4` (PG 14+). Faster decompress than pglz default, slightly better ratio. Applies to new writes; existing rows unchanged until rewritten.

**F5. Per-hot-table autovacuum.**
```sql
ALTER TABLE games SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_vacuum_cost_limit = 2000,
    autovacuum_analyze_scale_factor = 0.02
);
```
More aggressive reclaim on the hot write path.

**F6. Cold-storage archive to S3.** Older partitions → blob columns move to S3; summary row stays in DB with `s3_key` pointer. Live-table size drops 10-100x. Keep summary columns in DB forever so queries like "user X's earliest game" never touch cold storage (see section 14).

F1 alone is the highest ROI; it does not require any app or schema change and can land in a week.

---

## 7. Storing moves on the same row (UPDATE) vs INSERT-per-move

### Current write pattern

`UpdateGame` at `db/queries/games.sql:117-138` rewrites ~20 columns per move, including `history bytea`, `timers jsonb`, `stats jsonb`, `quickdata jsonb`, `meta_events jsonb`. Every move is a full-row rewrite.

### Cost of UPDATE-per-move on a wide row

- **WAL per move** = full new tuple + FPI (full-page image) on the first touch after a checkpoint. A 20KB game row updated 50 times generates roughly 1MB of WAL per game. Replication and backups pay this cost.
- **Dead tuples.** Each UPDATE leaves one. Table bloats 2-10x until autovacuum catches up.
- **HOT updates rarely fire** (see section 18). The `history` bytea grows each move → row no longer fits in the same page → new page → HOT breaks. Indexed columns like `tournament_id` or `player_on_turn` changing also break HOT. Result: every update touches every index.
- **Read overhead.** Fetching `timers` alone still loads the row including `history` unless explicit column projection is used. TOAST partially mitigates (history stored out-of-line) but not fully.
- **No audit trail.** Game state at move N is only reachable by parsing the full `history` blob.
- **Concurrency window.** Read-modify-write on the same row is a serialization point. Currently defended by in-process mutex (`pkg/stores/game/cache.go:92`), but that only works per-instance.

### Proposed: INSERT-per-move (event-sourced)

```sql
CREATE TABLE game_moves (
    game_id bigint NOT NULL,
    move_idx int NOT NULL,
    move_type text NOT NULL,
    player_idx smallint NOT NULL,
    word text,
    rack text,
    position text,
    score int,
    cumulative_score int,
    time_remaining_ms int,
    event jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (game_id, move_idx)
) PARTITION BY RANGE (created_at);

CREATE INDEX ON game_moves (word) WHERE word IS NOT NULL;
CREATE INDEX ON game_moves (rack) WHERE rack IS NOT NULL;
```

The `games` table shrinks to a summary:
```sql
games (
    id, uuid, player0_id, player1_id, tournament_id, league_id,
    started, ended, winner_idx, player_on_turn,
    time_remaining_p0, time_remaining_p1, time_of_last_update,
    created_at, updated_at
)
```

### Properties of the new layout

- `game_moves` is append-only: zero dead tuples on the hot path, HOT is irrelevant, autovacuum does almost nothing there.
- `games` row stays small: HOT can fire on updates to scalar fields. Fillfactor 70 leaves in-page room.
- WAL per move drops from "full game row with history" to "one move event". Roughly 10-100x less WAL.
- Full audit trail for free.
- Move-level queries become native SQL (section 11).

### Read-as-replay concern

Loading a full game state requires fetching N moves and folding them. For games with fewer than 2k moves (essentially all liwords games), this is ~400KB of data + a microsecond-scale in-memory fold. Comparable to the current `proto.Unmarshal` cost of the whole history blob. Not a regression. See section 20.

### Migration

Trigger-based dual-write + backfill + atomic swap, detailed in the storage-redesign spec.

---

## 8. Postgres-only vs Redis + NATS + possibly ClickHouse

### Current roles

| Store | Role |
|-------|------|
| Postgres | authoritative OLTP state |
| Redis | presence, chat recency, config cache, some locks |
| NATS | pub/sub fan-out with queue groups (`pbworkers`, `requestworkers`) |
| No ClickHouse | — |

### Why Postgres-only is not practical

- **Presence.** Write-heavy ephemeral KV. Every heartbeat on Postgres is a row update with dead tuples, WAL, and autovacuum pressure for data that is meaningless in 60 seconds. Redis `SET EX 60` is the right shape.
- **Pub/sub with queue groups.** Postgres `LISTEN/NOTIFY` is per-connection, has no queue groups, no persistence, no replay, and does not scale past ~100 listeners. NATS queue groups (`pkg/bus/bus.go:111, 116`) distribute inbound messages across instances cleanly. No PG equivalent.
- **Sub-ms real-time broadcast.** WebSocket fan-out needs single-digit-ms latency. NATS delivers; PG LISTEN/NOTIFY is a coarser tool even when it works.

### What can be consolidated

- **Retire `RedisConfigStore`** (`pkg/stores/config`): config is cold, Postgres table + small in-process cache works.
- **Drop in-process game locks** in favor of PG advisory locks (P3 in deploy-safety spec).
- **Move chat storage** from Redis to Postgres (section 23).

After these changes, Redis shrinks to presence + small ephemera. Still worth keeping for that role. NATS stays for pub/sub fan-out. PG + Redis + NATS is the sustainable target.

### ClickHouse

Not needed yet. Move-level analytics ("find games where word X was played") fits into `game_moves` with partition pruning and GIN indexes in Postgres for years. ClickHouse becomes worth it only when move-level queries are a product feature at scale — 10M+ games with complex OLAP — and even then as a sink, not an authoritative store. Defer.

### Simplification ceiling

"PG + NATS" (drop Redis) is the most aggressive realistic simplification. Trades presence durability for ops simplicity. Not worth it today given presence shape is wrong for PG. Stay with three.

---

## 9. Non-lossy JSONB in Go; is protobuf actually safer?

### Where JSONB decodes lose data

Default `encoding/json.Unmarshal` into `interface{}` turns all numbers into `float64`. Values above 2^53 lose precision. `pgx.Scan` into `map[string]any` for JSONB columns hits this.

### Non-lossy JSONB patterns in Go

- **Scan into a typed struct**, not `interface{}`. Field types preserve int64, bigint, etc.
- For unavoidable untyped scans: construct a `json.Decoder` and call `UseNumber()`, then convert `json.Number` to `int64` explicitly.
- For proto-shaped JSONB: use `protojson.Unmarshal` into the proto struct. int64 handled correctly.

### Is protobuf actually safer?

For int64 precision: marginally. Both protobuf-with-typed-decoding and JSONB-with-typed-struct are exact. Lossiness only appears when decoding into generic maps.

Protobuf's real wins:
- Unknown-field preservation on round-trip (safer schema evolution).
- Smaller wire size (2-10x).
- No "42 vs \"42\"" string-vs-number ambiguity.
- Explicit versioning rules (reserved field numbers).

For liwords data specifically: Unix-ms timestamps, int scores. Neither crosses 2^53 in practice. Both formats are safe with disciplined decoding. The argument for moving away from bytea protobuf in the DB is ops visibility and SQL queryability, not precision.

### Recommended discipline for the liwords codebase

- Never scan JSONB into `interface{}` or `map[string]any` in production paths.
- Use typed Go structs matching the proto message shape.
- Consider a lint rule that flags `interface{}` / `map[string]any` JSON decoding outside tests.

---

## 10. UPDATE rewrites whole column: also true for protobuf bytea?

> **Signpost:** This section refines §5. The "UPDATE rewrites whole column" cost is not JSONB-specific; bytea pays the same price. This decouples the "WAL churn" problem from the "queryability" problem.

Yes, same for bytea. Postgres MVCC never edits a column in place; every UPDATE writes a new tuple version.

Clarification that corrects an earlier framing:
- The cost of "UPDATE rewrites the whole row" is **not JSONB-specific**. bytea, text, and all varlena columns follow the same rule.
- TOAST out-of-line storage saves you for **unchanged** large columns: the new main-heap tuple carries the same TOAST pointer, so TOAST chunks are not rewritten. This applies equally to bytea and JSONB.
- For `games` today: `history bytea` changes every move, so its TOAST chunks do churn every move regardless of encoding.

The "JSONB vs bytea" axis is therefore orthogonal to the "UPDATE churn" axis. They are separate issues:

| Issue | Cause | Fix |
|-------|-------|-----|
| Main-heap bloat | UPDATE on wide row with changing columns | shrink row (move `history` to a separate table) |
| TOAST churn | `history` column changes every move | append-only `game_moves` table |
| Query opacity | protobuf in SQL is unreadable | JSONB or proper columns for queryable fields |
| Disk size | protobuf is 2-10x smaller than JSONB | keep proto only for cold archival if size matters |

---

## 11. `games.history` queried for words: inappropriate as protobuf?

### Current pain

A query like "find games where the word QUIXOTIC was played" requires:
1. Scan all rows of `games`.
2. For each, fetch `history bytea`.
3. `proto.Unmarshal` in Go (`pkg/stores/game/db.go:165`).
4. Iterate events, filter by word.

O(all games × decode cost). Unworkable at scale. The use case exists and is important (moderation, analysis, content features).

### Right model

`game_moves` table with:
- One row per move.
- Promoted columns: `word`, `rack`, `score`, `position`, `player_idx`, `move_type`.
- `event jsonb` with the full event for replay.
- Indexes on `word`, `rack`, `(game_id, move_idx)`.

Query becomes `SELECT game_id FROM game_moves WHERE word = 'QUIXOTIC'`. Milliseconds.

`history bytea` in `games` is obsolete once `game_moves` is populated. Drop in Phase G of the migration.

---

## 12. `.proto` file licensing under AGPL

See section 24 for current status and implications. Short version: the `.proto` files at `rpc/api/proto/` are AGPL-3.0 by default under the repo license. External consumers (Rust bots via `prost_build`, iOS clients, third-party tools) produce derivative works that inherit AGPL obligations. Proposed fix: dual-license under Apache-2.0 with SPDX headers and a separate LICENSE file in `rpc/api/proto/`. Full detail in the stack-and-stores-cleanup spec. Legal sign-off required.

---

## 13. Promote JSONB fields to columns without downtime

> **Signpost:** The dual-write pattern below was the plan on PG 14.6. **On PG 18.3+ (the current upgrade target, see §26), virtual generated columns collapse this whole pattern into a single DDL with zero backfill and no dual-write.** The dual-write pattern below is retained as reference for pre-upgrade migrations or PG-version-agnostic scenarios; for new work post-upgrade, prefer virtual generated columns.

### Why ALTER TABLE ADD COLUMN GENERATED STORED is bad

`ALTER TABLE games ADD COLUMN tournament_id TEXT GENERATED ALWAYS AS (quickdata->>'tournament_id') STORED` rewrites every row to compute the stored value → AccessExclusiveLock for the duration → downtime proportional to table size.

### Zero-downtime pattern

1. `ALTER TABLE games ADD COLUMN tournament_id TEXT NULL` — metadata-only, instant.
2. App deploy: writes go to both `quickdata.tournament_id` and the new column.
3. Background backfill in 10k-row chunks: `UPDATE games SET tournament_id = quickdata->>'tournament_id' WHERE id BETWEEN $1 AND $2 AND tournament_id IS NULL`. Rate-limited to stay within autovacuum budget.
4. `CREATE INDEX CONCURRENTLY idx_games_tournament_id ON games (tournament_id)`. No exclusive lock.
5. Switch reads to the column. Deploy.
6. Stop writing the JSONB field. Deploy.
7. Optional cleanup: remove the field from `quickdata` much later.

Timeline: 2-4 weeks of deploys. Zero downtime at every step.

### Alternative on PG 18+

Virtual generated columns (non-stored) may allow:
```sql
ALTER TABLE games ADD COLUMN tournament_id text
    GENERATED ALWAYS AS (quickdata->>'tournament_id') VIRTUAL;
CREATE INDEX CONCURRENTLY idx_games_tournament_id ON games (tournament_id);
```
No storage rewrite, no backfill, no dual-write. Seconds of work. Verify against actual PG 18 release notes before relying on this.

---

## 14. Queryable game metadata even when game content is cold-storaged

Archive the heavy blobs, never the summary. Target shape has four tables (see §19):

- **`games`** (active-only, unpartitioned): `uuid, player0_id, player1_id, tournament_id, league_id, started, created_at`, ephemeral clock fields. Small, indexed, never archived.
- **`past_games`** (completed-only, monthly partitioned, **new**): `uuid, player0_id, player1_id, tournament_id, league_id, game_type, winner_idx, loser_idx, game_end_reason, created_at, ended_at, stats, quickdata, tournament_data`. Summary row kept forever; heavy JSONB fields archive-eligible.
- **`past_game_players`** (completed-only, monthly partitioned, **already exists as `game_players`**, ~20M rows): `(game_uuid, player_id)` PK, plus score, won, opponent_id, game_type, created_at, league_season_id. Right table for player-scoped queries.
- **`game_moves`** (append-only, monthly partitioned, **new**): one row per move, `event jsonb` + flat columns (word, rack, score, etc.).

Queries that touch only summary / per-player tables never require rehydrating blob storage. For "find user X's earliest game", use `past_game_players`:

```sql
SELECT game_uuid, created_at FROM past_game_players
WHERE player_id = $1
ORDER BY created_at LIMIT 1;
-- hits idx_game_players_player_created directly
```

Faster than scanning `games` with `player0_id = $1 OR player1_id = $1` because `past_game_players` has a single `(player_id, created_at DESC)` btree. No S3, no blob read, no index union.

For cold archival: `past_games` partitions older than retention have their JSONB columns (stats, quickdata, tournament_data) moved to S3 via Parquet dumps (PR 1503's `PHASE2_S3_ARCHIVAL.md` design). Metadata columns (uuid, players, tournament, winner, etc.) stay in the DB partition indefinitely so queries like "user X's earliest game" never touch S3.

---

## 15. Partitioning an existing table after the fact

> **Signpost:** Cutover mechanics mentioned here (logical replication to new cluster) are detailed in §25 — do **not** use DNS flip; use pgBouncer upstream swap.

Doable. Not trivial.

### Approach 1: new partitioned parent + trigger-based dual-write

1. Create `games_new` as a partitioned table.
2. Create partitions for current + next periods.
3. Create a trigger on old `games` that mirrors every INSERT / UPDATE / DELETE into `games_new`. The trigger runs in the same transaction as the app write, closing any race.
4. Chunked backfill of old rows into `games_new` in 10k-row batches.
5. Verify row counts and checksums.
6. Atomic swap during low-traffic window:
   ```sql
   BEGIN;
   ALTER TABLE games RENAME TO games_old;
   ALTER TABLE games_new RENAME TO games;
   DROP TRIGGER dual_write_games ON games_old;
   COMMIT;
   ```
   AccessExclusiveLock held for seconds.
7. Drop `games_old` after the retention window.

This is the preferred approach. It closes race windows without requiring app-level transaction awareness, because the trigger runs in the same DB transaction as the app's write.

### Approach 2: logical replication to a new cluster

1. Stand up new cluster with partitioned schema.
2. Logical replication streams all changes from old to new.
3. Cutover via pgBouncer upstream swap (section 25).

More infrastructure, but cleanest separation. Useful if you also want a PG version upgrade at the same time.

### Do not rely on DNS for cutover

DNS TTL lag can stretch from 30s to hours depending on resolver behavior. See section 25 for pgBouncer as the indirection layer.

### Partition key design for liwords

See section 19. Short answer: **two tables** — active `games` (unpartitioned, small, caching target) + completed `past_games` (partitioned by `ended_at` monthly in UTC). Not RANGE on `created_at` in a single table, because games can span months and active/completed have different columns.

---

## 16. TOAST compression, AWS RDS support, and 2-core prod

### Changeable after the fact

- Per-column: `ALTER TABLE games ALTER COLUMN history SET COMPRESSION lz4`. Instant metadata change. New writes use lz4. Existing TOAST chunks untouched until rewritten.
- Per-table default: `ALTER TABLE games SET (toast_compression = 'lz4')`.
- To recompress existing chunks: `VACUUM FULL` (locking) or `pg_repack` (online).

Requires PG 14+. Confirmed available on 14.6.

### AWS RDS support

Supports lz4 TOAST on PG 14+. Regular database user can `ALTER TABLE ... SET COMPRESSION` without `rds_superuser`.

### AWS EC2 self-managed

All options available; lz4 is stdlib in PG 14+.

### 2-core prod implications

- `pg_dump` is single-threaded and CPU-bound on compression; burns one of two cores. Moving to physical backups (pgBackRest / WAL-G / `pg_basebackup --incremental`) removes this hot-spot; WAL streaming is not CPU-bound like pg_dump.
- lz4 TOAST decompress is 3-5x faster than pglz default. Net CPU win on the 2-core box.
- Autovacuum on 2 cores can starve queries; tune `autovacuum_vacuum_cost_delay` and `autovacuum_max_workers` to avoid oversubscription.

---

## 17. Autovacuum and downtime

Autovacuum does not cause downtime. It takes `ShareUpdateExclusiveLock`, which blocks DDL (`ALTER TABLE`, non-concurrent `CREATE INDEX`) but **not** SELECT / INSERT / UPDATE / DELETE.

CPU and I/O impact on 2 cores is noticeable. Tune:
- `autovacuum_vacuum_cost_delay = 2ms` (more throttled)
- `autovacuum_max_workers = 2` (don't oversubscribe)
- Per-hot-table: lower `autovacuum_vacuum_scale_factor` for more frequent, smaller passes.

Different from `VACUUM FULL`, which takes `AccessExclusiveLock` and blocks everything. Do not autorun `VACUUM FULL`. Use `pg_repack` for online space reclamation.

---

## 18. ELI5 HOT (heap-only tuple)

Normal UPDATE in Postgres:
- MVCC: never overwrite in place; write a new row version, mark old dead.
- Every index pointing at the old row must be updated to point at the new row.
- Old row waits for autovacuum to reclaim.

HOT trick:
- If the new row fits in the **same page** AND no **indexed** column changes, Postgres chains old → new inside the page.
- Indexes still point at the old row's position; readers follow the chain.
- **No index writes.** WAL is ~3x smaller. Vacuum cleanup is cheaper.

Requirements:
1. Free space in the page (tune `fillfactor` lower than 100 to leave headroom; 70-80 is typical for hot tables).
2. No change to any **indexed** column.

Why the current `games` table breaks HOT:
- `history` bytea grows every move → row no longer fits in the same page → new page → HOT fails.
- Indexed `tournament_id`, `player_on_turn` change → HOT fails.

Why the proposed skinny `games` (active-only) row would fire HOT:
- Small row (no bytea, no large JSONB) fits in page with fillfactor=70.
- Only scalar columns change on per-move UPDATE.
- Result: in-page chain, no index writes, autovacuum barely needs to work.

---

## 19. Active vs completed games: two-table pattern

> **Signpost:** This section is the **final table-shape decision** for liwords. If any earlier text (including earlier drafts of this section) recommended "LIST partition on `ended`, sub-partitioned by `ended_at`" as a single-table scheme, treat that as superseded. The final shape is **two tables**: `games` (active-only, unpartitioned) + `past_games` (completed-only, partitioned by `ended_at` monthly in UTC). This aligns with PR #1503 and with the existing `game_players` precedent.

### Why RANGE on `created_at` is wrong

A correspondence game created in March might end in May. Partitioning `games` by `created_at` month would leave active games in older partitions, defeating hot/cold separation.

### Why two tables beats a LIST-partitioned `games`

Two genuine reasons the table shape should split, not just the partition:

1. **Different columns per lifecycle.** Active games need ephemeral state: `player_on_turn`, `time_remaining_p0`, `time_remaining_p1`, `time_of_last_update`. Completed games don't. A LIST-partitioned `games` table forces completed partitions to carry those columns (waste) or NULL them (bookkeeping). Separate `past_games` with a different schema fits the data cleanly.
2. **Truly append-only completed side.** In a LIST-partitioned scheme, `UPDATE games SET ended = true WHERE uuid = $1` is a partition-key UPDATE — Postgres executes it as DELETE-from-active + INSERT-into-completed. That still hits the completed partition with write activity, violating the "immutable" property we want for backups and cold storage. Separate `past_games` receives only explicit INSERTs from app code, never partition migration.

And one pragmatic reason:

3. **`game_players` already uses this pattern.** Populated only at game end. NOT NULL outcome columns. ~20M rows, indexed for historical queries. `past_games` follows the same convention for full-game summaries. Consistency beats novelty.

### Schema

```sql
-- Active only (after migration). Unpartitioned. Caching target.
CREATE TABLE games (
    id bigint PK,
    uuid char(24) UNIQUE NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    player0_id int NOT NULL, player1_id int NOT NULL,
    tournament_id uuid, league_id uuid, season_id uuid, league_division_id uuid,
    game_type int NOT NULL,
    started boolean NOT NULL DEFAULT false,
    player_on_turn smallint,
    time_remaining_p0 int, time_remaining_p1 int, time_of_last_update bigint,
    time_started bigint, max_overtime int,
    lexicon text, variant text, rating_mode smallint, challenge_rule smallint, bot_level smallint,
    quickdata jsonb, meta_events jsonb,
    ready_flag bigint, deleted_at timestamptz
) WITH (fillfactor = 70);

-- Completed only. Partitioned. Append-only.
CREATE TABLE past_games (
    uuid char(24) NOT NULL,
    created_at timestamptz NOT NULL,
    ended_at timestamptz NOT NULL,
    player0_id int NOT NULL, player1_id int NOT NULL,
    tournament_id uuid, league_id uuid, season_id uuid, league_division_id uuid,
    game_type int NOT NULL,
    winner_idx smallint, loser_idx smallint, game_end_reason int NOT NULL,
    lexicon text, variant text, rating_mode smallint,
    stats jsonb, quickdata jsonb, tournament_data jsonb,
    PRIMARY KEY (uuid, ended_at)
) PARTITION BY RANGE (ended_at);

CREATE TABLE past_games_2026_04 PARTITION OF past_games
    FOR VALUES FROM (TIMESTAMPTZ '2026-04-01 00:00:00+00')
              TO   (TIMESTAMPTZ '2026-07-01 00:00:00+00')
    WITH (fillfactor = 100);
```

### Partition boundaries: explicit UTC

`timestamptz` literals without an offset are parsed in the session's `TIMEZONE`. On non-UTC sessions, DST transitions can shift partition boundaries by an hour. Always write boundaries as `TIMESTAMPTZ 'YYYY-MM-DD 00:00:00+00'` or run DDL with `SET LOCAL TIMEZONE = 'UTC'`.

### Game end: INSERT + DELETE in one transaction

```sql
BEGIN;
  SELECT pg_advisory_xact_lock(hashtextextended(uuid, 0));
  INSERT INTO past_games (uuid, created_at, ended_at, ..., stats, quickdata) VALUES (...);
  INSERT INTO past_game_players (...) VALUES (...), (...);   -- two rows, one per player
  DELETE FROM games WHERE uuid = $1;
COMMIT;
```

### Properties

- `games`: always small (hundreds to low thousands of rows), hot writes, HOT fires, low bloat.
- `past_games_YYYY_MM`: append-only (game-end INSERT is the only write), never updated, fillfactor 100 packs maximum density, backup-friendly because immutable.
- Game spanning weeks lives in `games` the whole time. One INSERT+DELETE at end moves summary to `past_games`.
- `game_moves` rows insert during active play and stay in their `created_at` partition forever. Long correspondence games span partitions but per-game lookup (PK `(game_uuid, move_idx, created_at)`) is still fast.

### Read routing

- Active game: `SELECT ... FROM games WHERE uuid = $1`. If not found → try completed.
- Completed game: need partition key. Either route via `game_players` (which has `(player_id, created_at DESC)` covering most real queries), or a small `game_uuid_to_ended_at` lookup table.
- Never scan all partitions to find a single game by uuid.

### Caveats

- Foreign keys: `game_players` (already exists), `game_moves` (new) reference `game_uuid` as `char(24)`. No hard FKs — partitioned cross-table FKs are awkward. App + trigger consistency.
- Retention: `ALTER TABLE past_games DETACH PARTITION past_games_2024_01` plus archive job. Simple.

---

## 20. Read-as-replay cost with per-move rows

For loading a full game state:
- Query: `SELECT event FROM game_moves WHERE game_uuid = $1 ORDER BY move_idx`.
- Index on `(game_uuid, move_idx)` within each partition. Long correspondence games may span 1-2 partitions.
- Liwords game move counts: < 2k in all realistic cases.
- ~200 bytes per move event (JSONB, lz4 TOAST) × 2000 = ~400KB.
- One SSD read per partition touched + microsecond-scale in-memory fold.

Comparable to the current `history bytea` unmarshal cost, which is also O(moves) for decoding.

Login edge cases: loading a full state for each of many active games is not a normal flow; UI shows a list from `games` (active) or `past_games` (completed summary) and loads full state on click. List queries hit skinny indexed tables.

---

## 21. Drop all bytea in favor of JSONB?

> **Signpost:** This is the **final encoding decision** for liwords. Supersedes any earlier "keep some bytea for size" framing. Zero bytea in the target DB schema; protobuf stays as the wire format only.

Yes, with caveats.

### Pros of dropping bytea

- `psql` readability: `SELECT event FROM game_moves WHERE ...` shows human-readable JSON.
- SQL queryability of subfields.
- `pg_dump` output is text, grep-able, diff-able.
- Logical replication / CDC tools handle JSONB natively.
- Ops can inspect without Go tooling.

### Cost

- JSONB is 2-4x larger than protobuf bytea on disk.
- Encode/decode is slower by microseconds per event (irrelevant at per-move scale).
- PG 14 lz4 TOAST narrows the gap to ~2x.

### Recommendation

- `game_moves.event` → JSONB (queryable by `event->>'word'`, etc., even without column promotion).
- `games.history` → removed entirely. `game_moves` is the source.
- `games.timers` → four columns, not JSONB (hot path, fixed shape).
- `games.quickdata` hot fields → promoted to columns.
- `games.stats`, `profiles.ratings`, `profiles.stats` → JSONB (open-ended, admin-inspectable).
- Zero bytea in the final schema. Protobuf stays as the wire format for RPC and NATS.

---

## 22. Is the LRU cache paying off?

> **Signpost:** Recommendation is **measure first, retire later**. Not "delete immediately." Actual retirement is gated on the games-storage redesign landing (§7, §19) so that direct DB fetches on a skinny row are fast enough to make the cache unnecessary.

Current cap: `CacheCap = 400` games (`pkg/stores/game/cache.go:59`). RAM estimate per comment at `:51-58`: roughly 300MB. Tournament cap is 50.

### When the cache helps

- Repeated reads of the same game within a request or tight loop.
- Tournament data accessed across multiple bus events.

### When it hurts

- Peak active games > cap: LRU thrashes, hit rate drops, lock contention on `c.Lock()` for evictions grows.
- Multi-instance deploys: cache is per-process; writes on one instance leave stale cache on others (issue I3 in deploy-safety spec).
- Correspondence games bypass the cache entirely (`cache.go:172`), so the cache only helps real-time games.

### Measurement first

Add `expvar` hit/miss counters. Count active real-time games. If hit rate is below ~70% at peak, the cache is mostly overhead.

### Post-storage-redesign

After `games` row becomes skinny (no `history` bytea, no large JSONB), a direct DB fetch is sub-ms. The cache's marginal benefit shrinks. The maintainer comments at `pkg/stores/stores.go:27, 33` ("we need to get rid of this cache") point in this direction. Retire the cache; rely on DB + partition pruning + HOT updates for performance.

Interim (until redesign lands): bump `CacheCap` to 4x peak active real-time games, add metrics. RAM budget allows up to a few thousand entries without stress.

---

## 23. Chat storage: Redis to Postgres

### Why move

- Redis chat is non-durable beyond AOF best-effort.
- Not queryable by SQL: moderation tools (user-level search, keyword search, cross-channel) require full scans or external indexing.
- Not covered by the Postgres backup pipeline.
- Schema evolution is awkward (Lua scripts + sorted-set conventions at `pkg/stores/redis/redis_chat.go`, `pkg/stores/redis/add_chat.lua`).

### Design

```sql
CREATE TABLE chat_messages (
    id bigint NOT NULL,
    channel_id text NOT NULL,
    user_id int NOT NULL,
    message text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    edited_at timestamptz,
    deleted_at timestamptz,
    PRIMARY KEY (channel_id, id)
) PARTITION BY HASH (channel_id);

CREATE INDEX ON chat_messages (channel_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX ON chat_messages (user_id, created_at DESC);
CREATE INDEX ON chat_messages USING gin (to_tsvector('english', message))
    WHERE deleted_at IS NULL;
```

16-32 hash partitions spread write concurrency. Sub-partition by `created_at` monthly if archival is desired.

### Fan-out

NATS remains the delivery path. Write = INSERT + `nats.Publish("chat.<channel>", ...)`. Subscribers receive from NATS as today. Postgres is the durable store; NATS is the broadcast bus.

### Latency

- Redis append: ~100µs. Postgres INSERT: ~1ms. At peak chat load (tens of messages per second across channels), not a throughput concern, and users do not notice 1ms on a chat message.
- Redis tail read: ~200µs. Postgres tail read with `(channel_id, created_at DESC)` index: 1-3ms. Acceptable for "fetch last 50 on channel join".

### Migration

Dual-write → reads cut over → stop writing Redis → remove Redis chat code. Zero downtime. Weeks.

After this plus `RedisConfigStore` removal, Redis usage is just presence + small ephemera. Role boundary is cleaner.

---

## 24. AGPL and downstream projects (e.g. omgbot)

> **Signpost:** This is a **live license conflict today**, not a hypothetical future concern. Any non-AGPL external consumer of the current `.proto` files is in a grey zone. The stack-and-stores-cleanup spec Q5 contains the actionable dual-license plan.

### Current state

Liwords is licensed AGPL-3.0. `.proto` files at `rpc/api/proto/` inherit the repo license. External projects that consume these `.proto` files and generate bindings (for example `omgbot` using `prost_build` to generate Rust) produce derivative works of AGPL source. Under a strict reading of the license, the entire downstream binary inherits AGPL obligations.

If a downstream project is licensed MIT, BSD, Apache, or proprietary, there is a latent license conflict today. This is not hypothetical; it is a live issue the moment any such project exists.

### Proposed fix

Dual-license the `.proto` files under a permissive license (Apache-2.0 is the common choice).

1. Add `rpc/api/proto/LICENSE` containing the Apache-2.0 text.
2. Add SPDX header to each `.proto`:
   ```
   // SPDX-License-Identifier: Apache-2.0
   // This schema is licensed under Apache-2.0 to permit external clients
   // to generate code without AGPL obligations. The liwords application
   // implementing these RPCs remains AGPL-3.0.
   ```
3. Update top-level `LICENSE` or add `COPYING`:
   ```
   The liwords application source code is licensed under AGPL-3.0.
   RPC schema files under rpc/api/proto/ are dual-licensed and may also
   be used under the Apache-2.0 license, see rpc/api/proto/LICENSE.
   ```
4. Update `README.md`: external clients may implement these RPCs under Apache-2.0.

### Caveats

- **Not legal advice.** Pattern is common (gRPC itself, many AGPL services with client SDKs), but licensing decisions require counsel sign-off.
- Past contributors to `.proto` files have default AGPL copyright on their additions. Relicensing requires either explicit consent from each contributor (recommend `git log --follow --pretty=format:'%an %ae' rpc/api/proto/` for the list) or rewriting affected files with fresh attribution.
- Priority: **urgent** if external non-AGPL consumers already exist.

---

## 25. DNS flip TTL lag and pgBouncer alternative

> **Signpost:** This is the **final cutover mechanism decision**. Any earlier reference to "DNS flip" in this document or the sibling specs should be read as superseded by pgBouncer upstream swap.

DNS TTL is cached per resolver. TTL values as low as 30 seconds routinely take longer to propagate because some resolvers ignore TTL. Users remain on the old endpoint until their resolver refreshes.

### Better cutover mechanisms for app-tier

- ALB weighted target group shifting: 0% → 100% in seconds, no DNS change.
- ALB listener DefaultAction swap: instant.

### Better cutover mechanism for database

**pgBouncer** (or PgCat, Odyssey, HAProxy with Postgres protocol) between app and Postgres:

- App connects to pgBouncer, not directly to PG.
- During a migration: both old and new PG clusters run; logical replication keeps new in sync.
- Cutover: edit pgBouncer `databases.ini` upstream, reload (`pgbouncer -R` or SIGHUP). New client connections immediately hit the new cluster. Existing connections drain on idle timeout or server lifetime (default 1 hour, tunable).
- No DNS change. No TTL lag. Predictable cutover in seconds.

Side benefit: pgBouncer multiplexes many app-side connections into fewer PG backends. On a 2-core box, this prevents connection count from overwhelming CPU. Worth running even without cluster cutover as a motivator.

Recommendation: introduce pgBouncer before any cluster migration. Use it as the indirection layer for every subsequent change (version upgrade, partitioning swap, instance resize).

---

## 26. PG version upgrade path: 14 to 16, 17, 18

> **Signpost:** This section is the **final PG upgrade recommendation**. If any earlier text (e.g. "target PG 17 as a first step" in early drafts) is still visible, treat this section as superseding it. Current recommendation: **target PG 18.3+ directly, skip 17**. Rationale: PG 18.3 released 2026-02-26 is past the early-release gate; virtual generated columns in PG 18 collapse the JSONB-to-column migration pattern in §13.

Prod is 14.6 on a 2-core EC2. Each major version adds features that matter to liwords.

### Release dates (confirmed)

- PG 16: released 2023-09-14
- PG 17: released 2024-09-26
- PG 18.0: released 2025-09-25
- PG 18.3: released 2026-02-26 (out-of-cycle CVE + regression fixes; includes CVE-2026-2006 `substring()` fix and CVE-2026-2007 `pg_trgm` fix)

PG 18 is past the initial .1/.2 maturity gate; 18.3 is the recommended current minor.

### Features by version

| Feature | PG 14 | PG 15 | PG 16 | PG 17 | PG 18 |
|---------|-------|-------|-------|-------|-------|
| lz4 TOAST compression | yes | yes | yes | yes | yes |
| `SET COMPRESSION lz4` per column | yes | yes | yes | yes | yes |
| Partition key UPDATE row move | yes (since 11) | yes | yes | yes | yes |
| `CREATE INDEX CONCURRENTLY` | yes | yes | yes | yes | yes |
| Generated STORED columns | yes (since 12) | yes | yes | yes | yes |
| MERGE statement | no | yes | yes | yes | yes |
| Logical replication from standby | no | no | yes | yes | yes |
| `pg_stat_io` view | no | no | yes | yes | yes |
| Parallel vacuum for indexes | partial | partial | improved | improved | improved |
| Incremental basebackup in `pg_basebackup` | no | no | no | yes | yes |
| Streaming I/O infrastructure | no | no | no | yes | yes |
| VACUUM memory 2-3x improvement | no | no | no | yes | yes |
| JSON_TABLE | no | no | no | yes | yes |
| `MERGE WHEN NOT MATCHED BY SOURCE` | no | no | no | yes | yes |
| **Async I/O subsystem** (seq scans, bitmap heap scans, vacuum) | no | no | no | no | **yes** |
| **Virtual generated columns** (now default, `STORED` opt-in) | no | no | no | no | **yes** |
| `uuidv7()` (time-sortable UUID) | no | no | no | no | yes |
| OAuth authentication method | no | no | no | no | yes |
| Temporal constraints (`WITHOUT OVERLAPS`, `PERIOD` FKs) | no | no | no | no | yes |
| OLD / NEW in `RETURNING` clauses | no | no | no | no | yes |
| Skip-scan on multi-column B-tree indexes | no | no | no | no | yes |
| pg_upgrade preserves optimizer stats | no | no | no | no | yes |
| Data checksums enabled by default | no | no | no | no | yes |
| `MIN`/`MAX` on arrays and composite types | no | no | no | no | yes |

### What each step buys for liwords

**14 → 16:** MERGE for sync/backfill patterns; `pg_stat_io` observability (valuable on 2-core); logical replication from standby for zero-downtime migrations; parallel vacuum improvements.

**16 → 17:** For a 2-core, RAM-constrained box the highlight is **VACUUM memory reduced 2-3x**. Autovacuum on `games` gets dramatically faster. Also: incremental basebackup in stock tooling (simpler than WAL-G), streaming I/O for faster seq scans and backups, JSON_TABLE for SQL-level JSONB projections.

**17 → 18:** Three features collapse work elsewhere in this audit:
1. **Virtual generated columns** are the default in PG 18. The entire "promote JSONB to column" dual-write migration pattern (section 13) collapses into one DDL:
   ```sql
   ALTER TABLE games ADD COLUMN tournament_id text
       GENERATED ALWAYS AS (quickdata->>'tournament_id') VIRTUAL;
   CREATE INDEX CONCURRENTLY idx_games_tournament_id ON games (tournament_id);
   ```
   Zero storage rewrite. Zero backfill. No dual-write phase. Seconds of work per promoted field.
2. **Async I/O** on Linux improves seq scans, bitmap heap scans, and vacuum. On a 2-core I/O-bound box this is a direct CPU-idle improvement.
3. **`uuidv7()`** — time-sortable UUIDs. If `game_moves` keys include game-scoped UUIDs, this improves b-tree locality.

Also useful: **OLD/NEW in RETURNING** for audit triggers; **skip-scan indexes** reduce the number of composite indexes needed; **pg_upgrade preserves optimizer stats** so no post-upgrade ANALYZE runs required.

Breaking changes to watch in 18:
- `VACUUM`/`ANALYZE` now process inheritance children by default (use `ONLY` for old behavior). Affects any maintenance SQL that assumes non-recursive.
- `AFTER` triggers execute as the role active at queue time, not at execution time.
- `COPY FROM` no longer treats `\.` as EOF in CSV. Check any import tooling.
- MD5 password auth deprecated with warnings; plan SCRAM migration.

### Recommended path (updated)

**Target PG 18.3+ directly, skipping the intermediate stop at 17.**

Rationale:
- 18.3 released 2026-02-26 with CVE fixes. Past the early-release maturity gate.
- Virtual generated columns eliminate the need for the dual-write JSONB-to-column pattern, which was the largest mechanical chunk of the storage-redesign spec's Phase E-F work.
- Async I/O compounds with lz4 TOAST and VACUUM memory improvements for the 2-core box.
- Skipping 17 saves one upgrade cycle (one downtime/cutover event, one ops verification pass).

Alignment with storage-redesign spec:
- **Phase A prerequisite: upgrade 14.6 → 18.3.**
- Dual-write patterns for JSONB-to-column promotion become DDL-only. The spec's Phase E can drop most of its column-promotion steps and use virtual columns + `CREATE INDEX CONCURRENTLY` instead.
- The `game_moves` table and hot/cold partitioning remain as designed (those require real schema work, not virtual columns).

Driver/tool compat:
- `pgx` v5 supports PG 18. No upgrade needed.
- `sqlc` 1.27+ supports PG 18. Check current pinned version.
- `golang-migrate` works unchanged.
- AWS RDS currently offers PG 18 as of the latest AWS release notes; self-managed EC2 has all options.

### Upgrade mechanics (updated)

Given the target is two major versions ahead of prod (14 → 18):

- **`pg_upgrade` in place**: supported across 14 → 18 in one hop (pg_upgrade can skip intermediate majors). Minutes of downtime. Lowest operational complexity.
- **Logical replication to new cluster**: zero downtime. More ops work. Cutover via pgBouncer (section 25), not DNS. Recommended if any downtime is unacceptable.
- **pg_dumpall + restore**: slowest, do not use for multi-hundred-GB production data.

For liwords (short downtime unacceptable per user's constraint): logical replication to a new PG 18 cluster + pgBouncer upstream swap.

### Driver and tool compatibility

- `pgx` v5 supports PG 11+. No driver change required.
- `sqlc` 1.25+ supports PG 17. Upgrade as needed.
- `golang-migrate` Postgres driver works across versions.
- AWS RDS and self-managed EC2 both support these versions; AWS RDS major-version upgrades use `pg_upgrade` under the hood.

### Upgrade mechanics

- **`pg_upgrade` in place**: minutes of downtime. Fast, lowest operational complexity.
- **Logical replication to new cluster**: zero downtime. More ops work. Cutover via pgBouncer (section 25), not DNS.

For liwords with "even short downtime is unacceptable": logical replication + pgBouncer cutover is the right path.

---

## Cross-references

| Topic | Spec |
|-------|------|
| Deploy safety (P1-P8) | `deploy-safety.md` |
| Games table redesign, backups, partitioning, migration | `games-storage-redesign.md` |
| Stack simplification, chat move, `.proto` licensing, unit-of-work | `stack-and-stores-cleanup.md` |
| PG version upgrade | section 26 above; also referenced from storage-redesign Phase A |

## Code references

| Topic | Location |
|-------|----------|
| API graceful shutdown | `cmd/liwords-api/main.go:92, 513` |
| socketsrv graceful shutdown | `cmd/socketsrv/main.go:81-92` |
| NATS queue groups | `pkg/bus/bus.go:111, 116` |
| Bus tickers | `pkg/bus/bus.go:138-149, 384-422` |
| Adjudication | `pkg/bus/gameplay.go:523`, `pkg/gameplay/game.go:737` |
| Correspondence in-process lock | `pkg/stores/game/cache.go:92-391` |
| Game LRU cache | `pkg/stores/game/cache.go:59` |
| Tournament LRU cache | `pkg/stores/tournament/cache.go:51` |
| Broadcast poller | `pkg/broadcasts/poller.go:23` |
| VDO webhook poller | `pkg/vdowebhook/service.go:312` |
| Analysis reclaim worker | `pkg/analysis/service.go:405` |
| Inline migration guard | `cmd/liwords-api/main.go:159-181` |
| Stores registry + maintainer notes | `pkg/stores/stores.go:25-50` |
| History unmarshal | `pkg/stores/game/db.go:158-175` |
| UpdateGame SQL | `db/queries/games.sql:117-138` |
| Common tx helpers | `pkg/stores/common/db.go` |
| Games schema | `db/migrations/202203290423_initial.up.sql:95-137` |
| Redis presence | `pkg/stores/redis/redis_presence.go` |
| Redis chat | `pkg/stores/redis/redis_chat.go`, `pkg/stores/redis/add_chat.lua` |
| Migration task infra | `aws/cfn/db-migration-task.yaml` |

---

## 27. Relationship to `game_storage_v2.md` mikado plan

> **Signpost:** This section is the **final scope resolution** for the games-storage workstream. The sibling `games-storage-redesign.md` document has been narrowed to an operational addendum around `docs/mikado/game_storage_v2.md`. Sections 5, 7, 11, 14, 19, 21, 22 of this deep-dive still hold for their specific questions, but the schema-shape recommendations in those sections are superseded by v2 where they conflict. Specifically:
> - §19's two-table (`games` + `past_games`) recommendation is superseded by v2's single-`games` + ephemeral-`game_turns` shape.
> - §21's "permanent `game_moves` with ML arrays" is superseded by v2's ephemeral `game_turns` + S3 archive.
> - §14's `game_metadata` adoption is superseded by v2 keeping metadata on the forever `games` row.
> - §11's word-search via `game_moves` moves to a separate ClickHouse migration per v2's out-of-scope list.

### What `game_storage_v2.md` says (source of truth)

On `origin/feat/game-turns-dual-write` branch, commit `59e41770` dated 2026-04-19. Authored by César Del Solar. In-progress implementation already touches `pkg/stores/game/db.go`, `pkg/gameplay/game.go`, adds `game_turns` migration, new sqlc queries, `DUAL_WRITE_TURNS` and `SHADOW_TURNS` feature flags.

Key design choices:

| Question | v2 answer |
|----------|-----------|
| Runtime game state | Native Go structs in `pkg/game/`, not proto |
| Per-move persistence | `game_turns` — one proto-marshaled `ipc.GameEvent` per row |
| `game_turns` lifetime | Ephemeral; DELETEd after GameHistory assembled and S3 upload confirmed |
| Live-state serialization | None during play; replay events on node wake-up |
| Coordination | `pg_advisory_xact_lock(hashtext(game_uuid))` (matches deploy-safety P3) |
| Schema shape | One `games` table (forever) + ephemeral `game_turns` + S3 archive |
| S3 archive format | Gzipped protojson (`.json.gz`) |
| `games` heavy columns | `history`, `stats`, `quickdata`, `meta_events` cleared after backfill; `history_s3_key` added |
| Write API | `AppendTurn`, `UpdateTimers`, `AppendMetaEvent`, `SetReady`, `EndGame` — short focused txs |
| `pkg/cwgame/*` | Retired after annotator migration |
| macondo runtime dep | Dropped (only used for lexicon data) |
| Partitioning | None; `games` stays small, `game_turns` stays small and ephemeral |

Priority in v2: multi-node + cache removal first, S3 archival second, GameDocument deprecation third, macondo-dep removal fourth.

Explicitly out of scope in v2 (separate mikado branches): puzzles macondo removal, memento proto-rename, **ClickHouse stats migration**, tournament-store GORM removal.

### What this audit still contributes after v2

This audit's remaining, non-overlapping contributions:

- **Deploy safety P1-P7 in `deploy-safety.md`** — v2 references advisory locks and multi-node but does not enumerate the full fix list. WebSocket drain (P5), ALB deregistration alignment (P6), JetStream event replay (P8), worker service split (P2), schema version guard (P1) are audit-specific additions.
- **Stack + stores cleanup in `stack-and-stores-cleanup.md`** — chat Redis→PG, `RedisConfigStore` retire, AGPL `.proto` dual-license, unit-of-work pattern. None overlap v2.
- **Operational infrastructure in `games-storage-redesign.md`** — PG 14.6→18.3 upgrade, pgBackRest / WAL-G / `pg_basebackup --incremental`, pgBouncer cutover, TOAST lz4 tuning, autovacuum per-hot-table tuning. v2 doesn't detail these, though v2's context implicitly motivates them.
- **Machine-letter storage insight (§11, §21)** — preserved for whenever the word-search ClickHouse migration is planned (out of scope for v2).

### Reading order if landing both

1. This audit's `deploy-safety.md` (P1-P7) — can land independently.
2. This audit's operational work in `games-storage-redesign.md` (PG upgrade, physical backups, pgBouncer).
3. v2 plan execution per `docs/mikado/game_storage_v2.md`.
4. Stack-cleanup items (`stack-and-stores-cleanup.md`) as capacity allows.

Items 1 and 2 are prerequisites that make v2's execution smoother (backup window shrinks so v2's backfill isn't held hostage to a 2h backup; pgBouncer lets PG upgrade land without downtime; advisory locks are already specified).
