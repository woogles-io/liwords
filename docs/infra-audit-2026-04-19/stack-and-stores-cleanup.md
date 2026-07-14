# Stack simplification and stores/cache pattern cleanup

**Date:** 2026-04-19
**Base commit:** `f3ab03aafd860aa93d934bc60687581f7784bf06`
**Postgres version on prod:** 14.6
**Scope:** Data-store roles (Postgres / Redis / NATS), transaction pattern, cache layer, JSONB vs protobuf, licensing
**Goal:** Reduce the number of places that could surprise a new developer. Consolidate where possible without sacrificing capability.

**Start here:** `index.md` — index, topic, reading order.

**Related specs:**
- `deploy-safety.md` — operational safety; cache retirement and advisory locks introduced here feed that work
- `games-storage-redesign.md` — operational wrapper around `docs/mikado/game_storage_v2.md`; unit-of-work (Q3) aligns with v2's short focused write APIs
- `deep-dive.md` — detailed Q&A reasoning; §27 documents relationship to v2

---

> **Reader signpost:** The AGPL section (Q5) was sharpened during review: external consumers of the current `.proto` files (e.g. omgbot using `prost_build`) are in a live license conflict today, not a hypothetical future one. The "Priority" list reflects this — Q5 is elevated if such consumers exist. Chat-move from Redis to Postgres (added to Q1 during review) is a separate migration from the `RedisConfigStore` retirement.

## Questions this spec answers

1. Can we run on Postgres alone and drop Redis + NATS?
2. Should we keep the LRU cache layer or retire it?
3. What is the right transaction scope for cross-entity operations?
4. Protobuf bytea vs JSONB: what goes where?
5. Are we safe under AGPL for `.proto` consumers?

---

## Q1. Postgres-only vs current stack

### Current roles

| Store | Purpose | Evidence |
|-------|---------|----------|
| Postgres (pgx) | authoritative state: users, games, tournaments, puzzles, comments, leagues, sessions | `pkg/stores/*/db.go`, `pkg/stores/models/*.sql.go` |
| Redis (redigo) | presence, chat recency, config cache, connection locks | `pkg/stores/redis/*.go` |
| NATS (nats.go) | pub/sub fan-out with queue groups (`pbworkers`, `requestworkers`) | `pkg/bus/bus.go:111,116` |

### Honest assessment

**Postgres-only is not practical** for this workload. Specific reasons:

- **Presence** is a write-heavy, ephemeral KV. Every heartbeat, every connect, every disconnect. On Postgres this turns into dead-tuple churn on a hot row, WAL volume, and autovacuum pressure for data that is meaningless 60 seconds later. Redis `SET EX 60` is what this workload wants. Replacing it with Postgres degrades both systems.
- **Pub/sub fan-out with queue groups.** Postgres `LISTEN/NOTIFY` is per-connection, has no queue groups, no persistence, no replay, and does not scale past roughly 100 listeners. NATS provides queue groups (already used at `bus.go:111`), ack-based delivery via JetStream, and request/reply. Moving to Postgres loses these capabilities with no equivalent.
- **Sub-ms real-time broadcast.** WebSocket fan-out needs a latency budget in single-digit milliseconds. NATS is purpose-built; PG LISTEN/NOTIFY is a blunter tool even when it works.

### What we can consolidate

- **Retire `RedisConfigStore`** (`pkg/stores/config`). Config reads are cold, configuration rarely changes, and using Redis here only fans out to more code paths. Move config to a Postgres table with a small in-process cache (refresh every N seconds) or even just environment variables. One less dependency for new developers to reason about.
- **Drop in-process game locks in favor of Postgres advisory locks** (covered by P3 in the deploy-safety spec). Eliminates the `gameLocks map` in `pkg/stores/game/cache.go:92-391`. One less state-management surface.
- **Move chat storage from Redis to Postgres.** See "Chat storage migration" section below.

### Chat storage migration

Chat currently lives in Redis (`pkg/stores/redis/redis_chat.go`, lua script at `pkg/stores/redis/add_chat.lua`). Redis gives microsecond append/read but:

- Not durable beyond best-effort AOF. A Redis crash or forced flush loses chat history.
- Not queryable by SQL. Moderation tools (flag user, find all messages by X containing Y, cross-channel search) require full scans.
- Not covered by the Postgres backup pipeline. Chat archive ops separate.
- Schema evolution is awkward (Lua + sorted-set conventions).

#### Proposed schema

```sql
CREATE TABLE chat_messages (
    id bigint NOT NULL,
    channel_id text NOT NULL,              -- "chat.game.<uuid>", "chat.tournament.<uuid>", etc.
    user_id int NOT NULL,
    message text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    edited_at timestamptz,
    deleted_at timestamptz,
    PRIMARY KEY (channel_id, id)
) PARTITION BY HASH (channel_id);

-- 16 to 32 hash partitions spread write concurrency across file extents.
-- Sub-partition by created_at monthly if archival matters.

CREATE INDEX ON chat_messages (channel_id, created_at DESC)
    WHERE deleted_at IS NULL;              -- tail reads
CREATE INDEX ON chat_messages (user_id, created_at DESC);   -- moderation
CREATE INDEX ON chat_messages USING gin (to_tsvector('english', message))
    WHERE deleted_at IS NULL;              -- full-text search
```

Realtime fan-out stays on NATS:

- Write path: INSERT into `chat_messages` → publish to `chat.<channel>` on NATS.
- Subscribe path: sockets subscribe to `chat.<channel>` as today.
- Tail read on join: SELECT last N messages from Postgres (indexed, sub-ms to low-ms).

#### Latency considerations

- Redis append: ~100µs. Postgres INSERT: ~1ms. At chat peak load (order 10-50 msgs/sec across channels during tournaments), this is not a throughput concern, and users do not notice a 1ms delta on a chat message.
- Redis tail read: ~200µs. Postgres tail read with index on `(channel_id, created_at DESC)`: 1-3ms. Acceptable for "fetch last 50" on channel join.

#### Migration phases

1. Add `chat_messages` table and partitions. Deploy.
2. Dual-write: app writes to both Redis and Postgres. Deploy.
3. Reads switch to Postgres (behind feature flag, gradual rollout).
4. Stop writing Redis. Deploy.
5. Remove `pkg/stores/redis/redis_chat.go` and related code.

Zero downtime. Weeks.

#### Redis footprint after migration

With chat gone and `RedisConfigStore` retired, Redis usage shrinks to presence + per-user ephemera. Still worth keeping Redis for those (presence heartbeats are the wrong shape for Postgres). Result: fewer places in the code talk to Redis, cleaner role boundary.

### Target

Postgres + Redis + NATS stays as the stack. The wins are: fewer places in the codebase talking to Redis (just presence and chat recency), clear role boundaries, and new developers needing to understand only Postgres + one pub/sub system + one ephemeral KV.

### Backup / ops implications

Per the concern that more stores mean more backup complexity: Redis can run with AOF for durability and is trivially snapshotted; NATS file store is small and replicated via NATS cluster. Neither competes with Postgres for backup attention. The operational simplification from dropping Redis config store and in-process locks is real; the operational cost of Redis + NATS as ephemeral-data systems is low.

---

## Q2. LRU cache layer

### Current state

Two caches:
- `pkg/stores/game/cache.go:59` `CacheCap = 400` games, per-process LRU
- `pkg/stores/tournament/cache.go:51` `CacheCap = 50` tournaments, per-process LRU

Both write-through (DB first, then cache). Correspondence games bypass the cache entirely (`cache.go:172`). The `stores.go:27,33` comments explicitly call these out as debt: "we need to get rid of this cache."

### When the cache helps

- Hot read-loop on the same game within one request.
- Repeated `Get` calls from the same handler on the same game.
- Tournament data accessed across multiple bus events for the same tournament.

### When the cache hurts

- Peak active-games > 400: LRU thrashes, hit rate drops, adds lock contention without benefit.
- Multi-instance deploys: cache is per-process, and writes on instance A leave stale cache on instance B (deploy-safety issue I3).
- Write-through semantics are broken by the lack of tx awareness (writes go to DB, then the cache is updated regardless of whether the caller's business-level tx commits or rolls back — though today very little code uses transactions anyway).

### Recommendation

**Retire both caches once the games-storage redesign lands.** After the redesign:
- `games` row is skinny (no `history bytea`, no large JSONB). DB lookup is sub-ms.
- `game_moves` fold is O(moves) regardless of caching.
- Read bottleneck moves from "fetching the row" to "folding the moves."

If the move-fold cost becomes a hotspot, the right cache is a small **per-request in-memory fold cache** (tied to request context, not a long-lived LRU), or a Redis-backed shared cache for the folded state if we want cross-instance coherence.

**Interim:** until the storage redesign is complete, bump `CacheCap` to 4x peak active games (memory budget allows, per the comment at `cache.go:51-58` the RAM is modest). Add hit/miss metrics via `expvar` so we have data for the retirement decision.

---

## Q3. Transaction scope for cross-entity operations

### Current state

Transactions are scoped inside individual `DBStore` methods. Each method does its own `dbPool.BeginTx`, as seen at `pkg/stores/game/db.go:396,553`, `pkg/stores/session/db.go:36`, `pkg/stores/mod/db.go:29`, and many others.

The `pkg/gameplay/*` business-logic layer has **zero** `BeginTx` calls. A typical "make a move" flow does:

1. `stores.GameStore.Get(ctx, id)` — opens tx internally, closes, returns entity
2. Compute new state in memory
3. `stores.GameStore.Set(ctx, entity)` — opens tx internally, closes
4. `stores.UserStore.SetRatings(...)` if rated game — opens tx internally, closes
5. `leagueUpdater.Update(...)` if league game — opens tx internally, closes

Five separate commits for one logical business operation. Any crash between commits leaves inconsistent state.

### Target: unit-of-work pattern

Business layer opens one transaction, passes a `*models.Queries` bound to that tx into all sub-operations.

sqlc generates `Queries.WithTx(tx)` automatically. The missing piece is:
1. A repository-like wrapper that accepts a `*Queries` instead of opening its own tx.
2. Business-layer functions that open the tx and call into those wrappers.

```go
// in pkg/stores/stores.go
func (s *Stores) InTx(ctx context.Context, fn func(q *models.Queries) error) error {
    tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
    if err != nil { return err }
    defer tx.Rollback(ctx)  // no-op if committed
    if err := fn(s.Queries.WithTx(tx)); err != nil { return err }
    return tx.Commit(ctx)
}

// in pkg/gameplay/game.go
func HandleMove(ctx context.Context, s *stores.Stores, moveReq *pb.ClientGameplayEvent) error {
    return s.InTx(ctx, func(q *models.Queries) error {
        // acquire advisory lock on game (spec deploy-safety P3)
        if err := q.AdvisoryLockGame(ctx, moveReq.GameId); err != nil { return err }

        game, err := q.GetGameForUpdate(ctx, moveReq.GameId)
        if err != nil { return err }
        // compute new state
        if err := q.InsertGameMove(ctx, ...); err != nil { return err }           // append to game_moves
        if err := q.UpdateGameSummary(ctx, ...); err != nil { return err }        // update games scalar columns
        if gameEnded {
            // insert one row per player into game_players (pre-existing table, denormalized outcome)
            if err := q.InsertGamePlayer(ctx, player0Row); err != nil { return err }
            if err := q.InsertGamePlayer(ctx, player1Row); err != nil { return err }
        }
        if rated {
            if err := q.UpdateUserRating(ctx, ...); err != nil { return err }
        }
        if league {
            if err := q.UpdateLeagueStanding(ctx, ...); err != nil { return err }
        }
        return nil
    })
}
```

All writes commit atomically or none at all.

### Migration cost

- Every `DBStore.<Method>` that opens its own tx becomes `<Method>(q *models.Queries, ...)` and no longer begins a tx.
- Call sites either pass in a tx-bound `*Queries` (inside `InTx`) or use `s.Queries` directly (auto-commit mode).
- Large mechanical change, bounded scope. Can be done per-domain (game, user, tournament, etc.) in separate PRs.
- `pkg/stores/common/db.go` helpers already take `pgx.Tx` so they are mostly ready.

### Benefits

- Crash safety: business op either fully applies or not at all.
- Deadlock-free concurrent modifiers: advisory lock inside the same tx serializes by entity.
- Simpler idempotency: the "check-before-write" pattern in `gameplay.TimedOut:753` becomes reliable rather than best-effort.
- Clean path to retire the write-through cache: when writes are tx-scoped, cache invalidation can be a post-commit hook.

---

## Q4. Protobuf bytea vs JSONB vs columns

### Decision matrix

| Access pattern | Encoding |
|----------------|----------|
| Indexed / filtered by field | **Column** |
| Structured, queryable subfields, shape evolves | **JSONB** |
| Opaque blob, read-as-whole, never queried by content | **bytea proto** |
| Structured, queryable, ops needs readability | **JSONB** |

Protobuf wins on wire size and type safety. JSONB wins on ops inspection, SQL queryability, and standard tooling. Columns win when access is repeated and well-known.

### Applied to liwords

The games-storage-redesign spec covers the schema choices in detail. Summary of where each encoding lands:

- **`games` scalar columns** (ids, timers, flags): promoted from JSONB. Queryable, indexable, HOT-friendly.
- **`games.quickdata`**: residual JSONB after promoting `tournament_id`, `original_request_id`.
- **`games.stats`, `games.meta_events`**: JSONB. Small, open-ended, end-of-game.
- **`game_moves.event`**: JSONB. Queryable for word search, rack analysis, score distribution.
- **`game_moves` flat columns** (`word`, `rack`, `score`, `position`, `player_idx`): promoted for indexed query.
- **No bytea in final schema.** Protobuf remains the RPC and NATS wire format; DB stores JSONB via `protojson.Marshal`.

### Cost acceptance

JSONB is 2-4x larger than protobuf bytea on disk. PG14 lz4 TOAST narrows this. Total disk cost increase: estimated 30-60% during dual-write, neutral to reduced long-term after `history bytea` is dropped.

### Non-lossy roundtrip

Standard Go `json.Unmarshal` into `interface{}` makes all numbers `float64`, which is lossy for int64 > 2^53. Rules for liwords JSONB code:

- Always scan into typed struct fields. Never into `map[string]any` for production data paths.
- For rare untyped scans, use `decoder.UseNumber()` and convert via `json.Number.Int64()`.
- For proto-shaped JSONB: use `protojson.Unmarshal` into the proto struct. Handles int64 correctly.
- Add a lint check (go vet custom analyzer or golangci rule) that flags JSON decoding into `interface{}` or `map[string]any` outside of test files.

---

## Q5. AGPL and `.proto` consumers

### Context

Liwords is AGPL-3.0. `.proto` files in `rpc/api/proto/` define the RPC surface. External clients (Rust bots, iOS apps, third-party tools) may generate Go, Swift, Rust, TypeScript bindings from these files.

**Current status:** no exemption exists. Under a strict reading of AGPL-3.0, any project that consumes these `.proto` files (for example `omgbot`, which uses `prost_build` to generate Rust from the liwords schema) produces a derivative work of AGPL source and inherits AGPL obligations on its entire output. If such downstream projects are licensed under anything other than AGPL (MIT, BSD, Apache, proprietary), there is a latent license conflict today.

This is not hypothetical future-proofing; it should be treated as a live issue the moment any non-AGPL external client uses these schemas.

### Recommendation

**Dual-license `.proto` files under a permissive license.** Typical choice: Apache-2.0.

Implementation:

1. Add `rpc/api/proto/LICENSE` containing the full Apache-2.0 text.
2. Add an SPDX header to each `.proto`:
   ```
   // SPDX-License-Identifier: Apache-2.0
   // This schema is licensed under Apache-2.0 to permit external clients
   // to generate code without AGPL obligations. The liwords application
   // implementing these RPCs remains AGPL-3.0.
   ```
3. Update top-level `LICENSE` or add `COPYING` note:
   ```
   The liwords application source code is licensed under AGPL-3.0.
   RPC schema files under rpc/api/proto/ are dual-licensed and may also
   be used under the Apache-2.0 license, see rpc/api/proto/LICENSE.
   ```
4. Document in `README.md`: external clients may implement these RPCs under Apache-2.0.

### Caveats

- **Not legal advice.** This pattern is common (gRPC itself, many AGPL services with client SDKs) but licensing decisions require human counsel sign-off.
- CLA / contributor agreements should cover the dual license if contributors outside the project add to `.proto` files.
- **Past contributors have default AGPL copyright on their additions.** Relicensing `.proto` files to Apache-2.0 requires either explicit relicense consent from each contributor or rewriting the affected files with fresh attribution. Run `git log --follow --pretty=format:'%an %ae' rpc/api/proto/` for the contributor list; it is likely short.
- Until dual-licensing lands, any external non-AGPL consumer of these schemas is in a grey zone. Treat as blocking for any external client project that expects a permissive license.

### Adjacent concern: macondo `.proto` is GPL (not AGPL)

Liwords' AGPL dual-license is necessary but **not sufficient** for omgbot-style external consumers. Two license axes stack:

- liwords `.proto` under AGPL (addressed by the dual-license above)
- macondo `.proto` under **GPL** (separate project, owned by a different maintainer)

liwords embeds macondo types in its wire surface today (`macondopb.GameHistory` in `games.history` bytea and in select RPC responses). External clients that parse `GameHistory` currently carry both AGPL and GPL exposure.

The authoritative games-storage plan (`docs/mikado/game_storage_v2.md`) solves this as a side effect:

> Where does `GameHistory` proto live? — `liwords/api/proto/ipc/`. Macondo becomes a pure library with plain Go structs; liwords owns the wire types.

Post-v2, `GameHistory` lives under `rpc/api/proto/` and is covered by the Apache-2.0 dual-license proposed in this Q5. Macondo retreats to library-only (plain Go structs), and external consumers don't touch its `.proto` at all.

Interim options (until v2 lands):
- Ask macondo maintainer to add SPDX permissive headers to `macondo/gen/api/proto/macondo/*.proto` — same pattern as liwords fix.
- Or accept GPL exposure for any omgbot code path that touches macondopb.

Full fix for omgbot = (liwords AGPL dual-license) + (v2 wire-type ownership transfer). Neither alone is enough. Deep-dive §24 covers this in more detail.

---

## Priority

Rough ordering:

1. **AGPL `.proto` exemption (Q5):** low effort, unblocks external clients. Add LICENSE file, SPDX headers, README note. Needs legal sign-off. **Elevated to urgent if external non-AGPL consumers already exist (e.g. omgbot).**
2. **Config store move from Redis to Postgres (Q1):** small, removes one dependency, low risk.
3. **Unit-of-work pattern (Q3):** medium-large, mechanical. Unblocks atomic multi-entity operations and pairs with games-storage-redesign. Can be incremental (per-domain).
4. **Advisory locks (deploy-safety P3, cross-referenced from Q1):** small, done alongside unit-of-work.
5. **Chat storage Redis to Postgres (Q1):** medium, dual-write migration. Pairs with the broader push to make Postgres authoritative for durable data.
6. **Cache retirement (Q2):** defer until games-storage-redesign lands; interim is to bump cap + add metrics.
7. **JSONB vs protobuf decisions (Q4):** subsumed by games-storage-redesign spec.

---

## Open questions for review

- Config store move: Postgres table or env-vars-only? Depends on how much admin UI expects mutable runtime config.
- Unit-of-work pattern: start with games domain (highest-value) or leaves domain (lowest-risk)? Recommend games, since it dovetails with games-storage-redesign and advisory locks.
- Drop JSONB `interface{}` / `map[string]any` lint: opt-in gradually or fail CI on new additions? Recommend the latter; legacy usages grandfathered.
