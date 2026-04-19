# Multi-instance and rolling-deploy safety audit

**Date:** 2026-04-19
**Base commit:** `f3ab03aafd860aa93d934bc60687581f7784bf06` (master, merge of #1804). All `file:line` references below are valid at this revision.
**Postgres version on prod:** 14.6 (all fixes confirmed compatible).
**Scope:** `liwords-api`, `socketsrv`, NATS/Redis/Postgres integration
**Goal:** Zero-downtime rolling deploys with multiple concurrent instances of each service. No data races, no dropped events, no mid-flight corruption.
**Start here:** `index.md` — index, topic, reading order.

**Related specs:**
- `games-storage-redesign.md` — backup + schema + partitioning (prerequisite for some deploy fixes to stay effective at scale)
- `stack-and-stores-cleanup.md` — PG+Redis+NATS roles, unit-of-work pattern, cache retirement
- `deep-dive.md` — detailed Q&A reasoning behind all three specs, including PG version upgrade path

---

## TL;DR

> **Reader signpost:** Some claims in earlier sections of this spec were revised after review. See [Corrections / clarifications](#corrections--clarifications-added-after-initial-review) below before acting on specifics. In particular: the `jsonb_set` bug claim was retracted, and the "UPDATE rewrites whole column" point is not JSONB-specific.

The codebase already handles the **inbound NATS path** safely across instances (queue groups), but several background components assume a singleton deployment:

- Per-instance tickers (adjudicator, correspondence adjudicator, seek expirer, broadcast poller, VDO poller, analysis reclaim worker) will multiply work and race when run on N > 1 instances.
- Correspondence game serialization uses an in-process `sync.Mutex` map, which does not prevent concurrent mutation across instances.
- Per-instance LRU caches for games and tournaments are write-through to the DB, but caches on other instances are not invalidated after a write, so reads may be stale.
- Core NATS pub/sub (not JetStream) means events published during a deploy blackout are dropped rather than replayed.
- WebSocket connections are not explicitly closed on shutdown; `http.Server.Shutdown` cannot drain hijacked connections, so clients see hard disconnects at the 30s SIGKILL.
- The ALB deregistration delay and the in-process graceful shutdown timeout are not coordinated.

None of these are blockers for a single-instance deployment, which is why they have been latent. Rolling deploys with two or more instances will expose all of them.

Estimated lift to reach 100% safe: **medium**. No architectural rewrites; most items are contained, localized changes. The largest item (NATS JetStream migration) is optional and can be deferred.

---

## Prior art in the repo

- **PR #1634** (`origin/maintenance-overlay` branch, commit `fffd47d7e` by César Del Solar, 2025-12-09, OPEN, not merged): "add a maintenance overlay and pause real-time games when deploying". Contains a backend pause-games-during-deploy path plus frontend overlay modal. PR body notes: "Need to expose `/ping` in Cloudfront to the front end prior to deploying this!" — blocked on CloudFront config, not code. This is a **workaround** — during a deploy, all real-time games are paused so users do not notice disconnects. It works but is a band-aid: it still requires user-visible pause, and correspondence games, tournament flows, and background jobs are still at risk.
- **PR #1503** (`origin/partitioned-games` branch, marked `[obsolete, but using as a reference]`): game-table-refactor work split into smaller PRs per Mikado method. Not directly about deploy safety, but notable because its `scripts/migrations/historical_games/` backfill tool and `pkg/stores/game/migration.go` scaffolding are relevant reference for the storage-redesign work that dovetails with deploy-safety P3 (advisory locks).
- The P1-P7 fixes in this spec are the proper alternative to PR 1634's workaround: instead of pausing user activity during deploy, make the deploy itself safe. If P1-P7 land, PR 1634 can be abandoned. If time pressure forces a partial fix, consider merging PR 1634 as an interim measure (after resolving CloudFront `/ping` exposure) while the full spec work proceeds.

---

## Corrections / clarifications (added after initial review)

- **`jsonb_set` intermediate-path gotcha:** earlier discussion speculated that this pattern caused live bugs in the codebase. Review of `pkg/stores/common/db.go:199,204` shows defensive `NULLIF(... , 'null') IS NULL` checks before `jsonb_set`. **No live bug evidence.** Retracted.
- **"UPDATE rewrites whole column" is not JSONB-specific.** Both bytea and JSONB columns are fully rewritten by Postgres on any UPDATE to the row. TOAST out-of-line storage does reuse unchanged chunks via pointer, but the main-heap tuple is always a new version (MVCC). The `history bytea` column in `games` changes every move, so its TOAST chunks churn every move, independent of encoding choice. Bloat mitigation comes from row-shrinking (moving `history` out of `games`), not from switching encoding.
- **Postgres version** on prod is 14.6 (confirmed). All advisory-lock, `ALTER COLUMN SET COMPRESSION lz4`, partition-key UPDATE, and `CREATE INDEX CONCURRENTLY` features used in the proposed fixes are available on 14.6. No need to block on a major upgrade.

---

## Current state

### Services in this repo

| Binary | Entry | Purpose |
|--------|-------|---------|
| `liwords-api` | `cmd/liwords-api/main.go` | HTTP + ConnectRPC, NATS subscriber, background tickers |
| `socketsrv` | `cmd/socketsrv/main.go` | WebSocket hub, NATS subscriber for fan-out |
| `db-migration-task` | `aws/cfn/db-migration-task.yaml` | One-shot, pre-deploy |
| `maintenance-tasks` | `aws/cfn/maintenance-tasks.yaml` | Ad-hoc admin tasks |

### What is already safe across instances

- **Inbound NATS messages.** `pkg/bus/bus.go:111,116` uses `ChanQueueSubscribe` with queue groups `pbworkers` and `requestworkers`, so each published message is processed by exactly one subscriber instance.
- **Presence / session state.** Lives in Redis via `pkg/stores/redis/redis_presence.go`; it survives restarts and is shared across instances.
- **Migrations.** Gated on `cfg.RunMigrations` (`pkg/config/config.go:88`), default `false`. Production runs migrations via the dedicated ECS task; API startup does not race to apply them.
- **Game clocks.** Wall-clock based (`GameTimer.Now()` in `pkg/entity/game.go`), with `TimeStarted` and `TimeOfLastUpdate` persisted. Timers reconstruct correctly after restart.
- **Write-through caches for single-instance correctness.** `Cache.Set`/`Create` write the DB first, then the LRU (`pkg/stores/game/cache.go:223`, `pkg/stores/tournament/cache.go`). A crash after the DB write leaves no phantom data in the cache of a newly-started process.

### What is not safe across instances

Each issue is detailed in its own section below.

---

## Issues

### I1. Background tickers assume singleton

**Evidence:**
- Real-time adjudicator: `pkg/bus/bus.go:384` (`adjudicator.C`) calls `adjudicateGames(ctx, false)`
- Correspondence adjudicator: `pkg/bus/bus.go:392` (`correspondenceAdjudicator.C`)
- Seek expirer: `pkg/bus/bus.go:400` calls `SoughtGameStore.ExpireOld`
- Broadcast poller: `pkg/broadcasts/poller.go:23` 30s ticker, fetches and publishes per broadcast
- VDO webhook poller: `pkg/vdowebhook/service.go:312` jittered poll over active monitoring streams
- Analysis reclaim worker: `pkg/analysis/service.go:405` 30s ticker, `ReclaimStaleJobs`

**What breaks at N=2:**
- Both instances run `adjudicateGames` in the same tick window. Both see the same active game, both call `gameplay.TimedOut`. `TimedOut` (`pkg/gameplay/game.go:737`) uses `stores.GameStore.LockGame` (per-instance mutex, see I2) and then checks `Playing() == GAME_OVER` for idempotency. The idempotency check is best-effort: if instance B loaded the game before instance A committed, B's cached view is stale, B proceeds and attempts a write that conflicts with A's committed state.
- Broadcast poller double-publishes to NATS per tick; `UpdateBroadcastLastPolled` double-writes without CAS.
- VDO poller double-hits external API per active stream.
- Seek expirer is idempotent at the SQL level (`UPDATE ... WHERE expires_at < now()`), so duplicated calls are wasteful but not incorrect.

**Severity:** High. Correctness risk for adjudication and broadcast polling.

### I2. Correspondence game lock is process-local

**Evidence:** `pkg/stores/game/cache.go:92` defines `gameLocks map[string]*gameLock` protected by an in-process `sync.Mutex`. Used by `LockGame`/`UnlockGame` and called from `gameplay.TimedOut` (`pkg/gameplay/game.go:744`) and `bus.adjudicateGames` (`pkg/bus/gameplay.go:575`).

**What breaks at N=2:** Two instances each acquire their own in-memory lock for the same `gameID`. They then both read, mutate, and write the same game row. Postgres serializes the two UPDATEs, but the "read-decide-write" window allows stale decisions (e.g. both instances compute `setTimedOut` for the same player, second write overwrites first with an inconsistent state).

**Severity:** High. Silent data corruption on correspondence games under concurrent access.

### I3. LRU caches are not coherent across instances

**Evidence:**
- `pkg/stores/game/cache.go:59` `CacheCap = 400`, per-process LRU
- `pkg/stores/tournament/cache.go:51` `CacheCap = 50`, per-process LRU
- Neither publishes invalidations.

**What breaks at N=2:** Instance A mutates and updates its LRU. Instance B has a stale copy in its LRU, serves stale reads for up to the entry's LRU lifetime. Correspondence games already bypass the cache (`cache.go:172`) so they are unaffected. Real-time games read stale metadata (scores, game state) across instances.

**Severity:** Medium. User-visible staleness; no corruption because mutations go through the DB.

### I4. Events use core NATS (fire-and-forget)

**Evidence:** `bus.go:116` subscribes via `ChanQueueSubscribe`; publishes via `natsconn.Publish` throughout. No JetStream usage.

**What breaks during deploy:** Instance A is being shut down. A client disconnects from A's socketsrv. Events fire during the blackout. Client reconnects to B. Without replay, the client missed events and sees stale state until the next full fetch.

**Severity:** Medium. Already a latent issue on single-instance deploys; worse during rolling deploys because the blackout window widens.

### I5. WebSocket connections are not drained on shutdown

**Evidence:** `cmd/socketsrv/main.go:86` calls `srv.Shutdown(ctx)` with a 30s timeout. `http.Server.Shutdown` stops accepting new requests and waits for active handlers to return, but WebSocket connections have been hijacked out of the HTTP lifecycle, so they do not return. At the 30s mark, the process is SIGKILL'd and every client gets a TCP reset.

**What breaks during deploy:** Every client reconnects at the same instant (thundering herd), and any in-flight outgoing message is lost.

**Severity:** Medium-high (UX). Always happens on any restart, not just multi-instance.

### I6. Shutdown timeout is not aligned with ALB deregistration

**Evidence:** `cmd/liwords-api/main.go:92` `GracefulShutdownTimeout = 30 * time.Second`. ALB default `deregistration_delay.timeout_seconds` is 300s.

**What breaks during deploy:** ECS sends SIGTERM, the process stops accepting new connections after 30s and dies. The ALB is still in the deregistration-draining state for another ~270s and continues to route a small number of new connections to the IP of the dead task, returning 502s.

**Severity:** Medium. Manifests as transient 5xx during every deploy.

### I7. `liwords-worker` role is not explicit

**Evidence:** All tickers live inside `liwords-api` (bus) or services invoked from `main.go`. There is no dedicated "worker" service with `desiredCount=1`.

**Consequences:** Fixing I1 by adding Redis-lock leader election is possible, but it duplicates scheduling logic and adds an ongoing failure mode. A cleaner alternative is to split a dedicated worker service.

**Severity:** Structural; not a bug on its own, but it amplifies the cost of fixing I1.

### I8. `gameEventChan` / per-instance channel writes

**Evidence:** `pkg/bus/bus.go:82` declares `gameEventChan`, `tournamentEventChan`, `genericEventChan` as in-process channels. Various places write into them (e.g. `bus/gameplay.go:593`).

**What to verify:** That every consumer of these channels ultimately republishes to NATS (so that any instance's socketsrv can fan out to the right clients), rather than relying on a same-process socketsrv. If any path is "in-proc only," a two-instance deploy loses events destined for clients on the other instance.

**Severity:** Unknown until verified. Potentially high.

---

## Fix plan

Priorities ordered by risk reduction per unit of effort.

### P1. Move migrations invariant explicit in startup

**Effort:** XS. **Risk reduction:** Low (mostly hardening; prod already uses the task).

**Change:** In `cmd/liwords-api/main.go`, after the existing `RunMigrations` block, add a schema version check:

```go
// Fail fast if schema is older than this build expects.
m, err := migrate.New(cfg.DBMigrationsPath, cfg.DBConnUri)
if err != nil { panic(err) }
v, dirty, err := m.Version()
if err != nil || dirty || v < cfg.MinSchemaVersion {
    log.Fatal().Uint("have", v).Uint("need", cfg.MinSchemaVersion).Msg("schema too old; run migration task first")
}
m.Close()
```

Add `MinSchemaVersion` to `pkg/config/config.go`. Set it at build time from the source migrations directory.

**After:** Rolling out a new binary before migrations have completed fails fast with a clear error, rather than panicking deep in a handler.

### P2. Split a `liwords-worker` service (fixes I1, lays groundwork for I7)

**Effort:** M. **Risk reduction:** High.

**Change:**
- New binary `cmd/liwords-worker/main.go`. It imports the same config and stores, constructs the bus (without queue-group subscriptions; worker does not process inbound NATS), and starts:
  - Real-time adjudicator ticker
  - Correspondence adjudicator ticker
  - Seek expirer ticker
  - Broadcast poller
  - VDO webhook poller
  - Analysis reclaim worker
- Move the ticker cases out of `bus.ProcessMessages` into the worker.
- `liwords-api` keeps the NATS queue-group subscribers (inbound request/publish handling) and loses the tickers.
- Deploy as a separate ECS service with `desiredCount=1`, no ALB, no health-check grace concerns (internal only).
- New `aws/cfn/` template similar to `maintenance-tasks.yaml` but long-running.

**After:** Tickers run exactly once per cluster. Adjudication, polling, and reclamation are singleton. No leader-election code needed.

**Tradeoff:** The worker is a single point of failure for background tasks. If it dies, no new adjudications run until ECS restarts it. Acceptable because (a) tickers recover on restart, (b) clock-based timeouts are deterministic from DB state, and (c) `desiredCount=1` with ECS managed restart typically restarts in under a minute.

### P3. Postgres advisory lock for correspondence game mutation (fixes I2)

**Effort:** S. **Risk reduction:** High.

**Change:**

In `pkg/stores/game/cache.go`, replace the in-process `gameLocks` map with a backing store call that acquires a Postgres transaction-scoped advisory lock:

```go
func (c *Cache) LockGame(ctx context.Context, gameID string) error {
    return c.backing.AdvisoryLockGame(ctx, gameID)
}
```

Implementation in `pkg/stores/game/db.go`:

```go
func (s *DBStore) AdvisoryLockGame(ctx context.Context, gameID string) error {
    _, err := s.db.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", gameID)
    return err
}
```

The lock is tied to the transaction, so the caller must wrap the mutate-write pair in a `BEGIN...COMMIT`. Refactor `gameplay.TimedOut`, `gameplay.AbortGame`, and any other correspondence-path mutations to run in a single transaction.

Delete the `gameLocks` map, `gameLocksMu`, `cleanupExpiredLocks`, `StopCleanup` from `cache.go:92-391`. Remove the `stores.GameStore.StopCleanup()` call in `cmd/liwords-api/main.go:524`.

**After:** Any instance (API or worker) that tries to mutate a correspondence game acquires a DB-level exclusive lock. Safe under any number of concurrent instances.

### P4. Cross-instance LRU cache invalidation via NATS (fixes I3)

**Effort:** S. **Risk reduction:** Medium.

**Change:**

1. Define a new NATS topic `cache.invalidate.game.<id>` and `cache.invalidate.tournament.<id>`.
2. In `Cache.Set`, `Cache.Create`, `Cache.Unload` (for both game and tournament caches), publish a fire-and-forget invalidation message after the DB write succeeds.
3. In `NewCache`, take a `*nats.Conn` and subscribe (non-queue-group, because every instance must hear every invalidation) to the topic. On receive, `c.cache.Remove(id)`.

```go
// after successful DB write in setOrCreate:
if c.nc != nil {
    c.nc.Publish("cache.invalidate.game."+gameID, nil)
}

// in NewCache:
c.nc.Subscribe("cache.invalidate.game.*", func(m *nats.Msg) {
    id := strings.TrimPrefix(m.Subject, "cache.invalidate.game.")
    c.cache.Remove(id)
})
```

**After:** Instance B sees an invalidation within a few ms of instance A's write and refreshes from the DB on next read. Stale-read window bounded by NATS delivery latency.

**Known limitation:** The 5-second `activeGames` TTL is independent and remains as-is. The staleness window for "list of all active games" stays at 5s per instance, which is acceptable for lobby refresh.

### P5. WebSocket drain on shutdown (fixes I5)

**Effort:** S. **Risk reduction:** Medium-high (UX).

**Change:** In `services/socketsrv/pkg/hub/hub.go`, add two methods on `Hub`:

```go
func (h *Hub) StopAccepting() { ... } // used by /ws handler to reject new upgrades
func (h *Hub) DrainAndClose(ctx context.Context, code int, reason string) {
    // snapshot clients, send Close frame to each, wait up to ctx deadline for unregisters
}
```

In `cmd/socketsrv/main.go`, replace the signal handler with:

```go
case <-sig:
    h.StopAccepting()
    drainCtx, drainCancel := context.WithTimeout(context.Background(), 20*time.Second)
    h.DrainAndClose(drainCtx, 1012, "service restart")
    drainCancel()
    shutCtx, shutCancel := context.WithTimeout(context.Background(), GracefulShutdownTimeout)
    srv.Shutdown(shutCtx)
    shutCancel()
    close(idleConnsClosed)
```

Frontend (liwords-ui): on WebSocket close code `1012`, reconnect immediately with no backoff. The user's session is already in Redis presence, so the new socketsrv instance restores context transparently.

**After:** No client sees a TCP reset during deploy. Clients receive a graceful close frame, reconnect to a healthy instance, and resume. Thundering herd is smoothed because close frames are emitted as the drain proceeds, not all at once at SIGKILL.

### P6. Align graceful shutdown with ALB deregistration (fixes I6)

**Effort:** XS. **Risk reduction:** Medium.

**Change:**
- `cmd/liwords-api/main.go`: `GracefulShutdownTimeout = 90 * time.Second`.
- Add a pre-drain sleep before `srv.Shutdown`:
  ```go
  case <-sig:
      log.Info().Msg("got quit signal, waiting for LB deregistration...")
      time.Sleep(15 * time.Second)
      ctx, shutdownCancel := context.WithTimeout(context.Background(), GracefulShutdownTimeout)
      srv.Shutdown(ctx)
      ...
  ```
- ALB target group settings (update the CloudFormation template that defines the service — not in this repo today, so this change lives in the infra repo):
  - `deregistration_delay.timeout_seconds = 70` (less than 90s graceful, greater than 15s pre-drain)
  - `healthcheck_interval_seconds = 10`, `healthy_threshold_count = 2`, `unhealthy_threshold_count = 2`
  - `slow_start.duration_seconds = 60` so new targets ramp up
- ECS service settings:
  - `deploymentConfiguration.minimumHealthyPercent = 100`, `maximumPercent = 200`
  - `deploymentCircuitBreaker = { enable: true, rollback: true }`
  - `healthCheckGracePeriodSeconds = 90`
  - `desiredCount = 2` at minimum for both `liwords-api` and `socketsrv`

**After:** The 15-second pre-drain lets the ALB notice the task is unhealthy and stop routing new connections before the process starts rejecting. In-flight requests have up to 90s to complete. Connection drain finishes before the next task is killed in a rolling deploy.

### P7. Verify `gameEventChan` publish path (fixes I8)

**Effort:** XS. **Risk reduction:** Depends on findings.

**Change:** Trace every write to `bus.gameEventChan`, `bus.tournamentEventChan`, `bus.genericEventChan`. Confirm the consumer (the goroutine draining the channel) always publishes to NATS. If any code path is "in-proc socketsrv notification only," rewrite it to go through NATS.

Grep targets:
```
pkg/bus/bus.go: gameEventChan, tournamentEventChan, genericEventChan
```

**After:** No event is delivered only via in-proc channel. All fan-out goes through NATS, and any socketsrv instance can deliver to its local WebSocket clients.

### P8. Migrate game/tournament events to NATS JetStream (fixes I4)

**Effort:** L. **Risk reduction:** High for correctness during network blips and deploys. Optional — not required for "safe rolling deploy" but necessary for "no lost events, ever."

**Change:**
- Enable JetStream on the NATS cluster.
- Define streams for the event topics we care about preserving: `game.>`, `tournament.>`, `user.>`.
- Replace `natsconn.Publish` for those subjects with `js.Publish`.
- socketsrv subscribers become JetStream pull/push consumers with `AckExplicit` and `DeliverLastPerSubject`.
- Clients reconnecting after a gap receive missed events until caught up.

**After:** Deploy blackouts and network hiccups do not lose events. Clients transparently catch up on reconnect.

**Tradeoff:** Operational complexity for NATS, per-message ack overhead. Worth it if we regularly see client-reported state drift.

---

## What happens after all fixes land

### Deployment topology

| Service | `desiredCount` | Behind ALB? | Notes |
|---------|----------------|-------------|-------|
| `liwords-api` | ≥ 2 | Yes | Stateless, handles HTTP + ConnectRPC + NATS queue-group subs |
| `socketsrv` | ≥ 2 | Yes (sticky not required) | WebSocket fan-out, presence in Redis |
| `liwords-worker` | 1 | No | All tickers: adjudicator, pollers, reclaim |
| `db-migration-task` | one-shot | No | Runs pre-deploy, blocks on success |

### Deploy order

1. CI builds and tags images.
2. `db-migration-task` runs. Deploy blocked until it exits 0.
3. `liwords-worker` deploys (old task stops, new task starts; brief gap in tickers, self-recovering on restart).
4. `liwords-api` and `socketsrv` roll in parallel, `minimumHealthyPercent=100`, `maximumPercent=200`. Old tasks drain per P5/P6.

### Runtime behavior

- **New client request during deploy:** ALB routes to healthy task (old task marked unhealthy after 15s pre-drain). New task accepts after `healthCheckGracePeriodSeconds=90`.
- **In-flight HTTP request during deploy:** Completes up to 90s. ALB drain window is 70s. The 20s margin absorbs DB latency spikes.
- **WebSocket client during deploy:** Receives close code `1012`, reconnects to a healthy socketsrv instance, resumes via Redis presence.
- **Game mutation during deploy:** Wrapped in a transaction with `pg_advisory_xact_lock` on game ID. Only one instance holds the lock at a time. Mutation succeeds or retries.
- **Adjudicator runs during deploy:** Worker is briefly down during its own roll; next tick picks up whichever games timed out in the interim. No duplicate runs because only one worker instance exists.
- **Cache read during deploy:** API reads from local LRU. If another instance invalidated via NATS, entry is gone, fetches DB. Acceptable staleness: < NATS delivery latency.

### Failure modes and mitigations

- **Worker crashes and ECS restarts slowly:** Adjudication pauses. Clients do not see active-game corruption (game state is DB-authoritative); only the timeout detection is delayed. Mitigation: alarm on `liwords-worker` task health.
- **NATS cluster split:** Queue-group subs and cache invalidations may be temporarily partitioned. Postgres advisory locks still protect correctness. Cache coherence recovers on NATS reconnect.
- **Postgres failover:** Advisory locks are lost on connection drop. In-flight transactions roll back. Caller retries. Safe by construction.

---

## Deferred / out of scope

- **Event sourcing for game moves.** Would make every move replayable and every mid-deploy failure recoverable. Large rewrite. Not required for safe deploys.
- **Replacing in-process LRU with Redis cache.** Simpler invariant (one source of cache truth), higher latency per op. Evaluate after P4 if coherence remains a pain point.
- **JetStream migration (P8).** Optional; do if event-loss reports appear.
- **Sticky WebSocket sessions at ALB.** Not needed: presence is in Redis, any socketsrv can serve any user.

---

## Verification checklist

After each fix:

- [ ] `go test ./...` passes
- [ ] `gofmt -l <changed>` clean
- [ ] `go build ./cmd/liwords-api ./cmd/socketsrv ./cmd/liwords-worker` succeeds
- [ ] Manual rolling deploy test in staging:
  - [ ] Tail logs on both instances during deploy
  - [ ] Confirm no panic, no 5xx burst
  - [ ] Confirm no duplicate adjudication log lines
  - [ ] Confirm WebSocket client receives code 1012, reconnects, sees continuity
  - [ ] Confirm correspondence game mutation during deploy succeeds (use a scripted test)
- [ ] Load test to verify cache invalidation coherence: write on instance A, read from instance B, observe freshness within 100ms

---

## Code references

| Topic | File:line |
|-------|-----------|
| API graceful shutdown | `cmd/liwords-api/main.go:92`, `:513` |
| socketsrv graceful shutdown | `cmd/socketsrv/main.go:81` |
| NATS queue groups | `pkg/bus/bus.go:111`, `:116` |
| Bus tickers | `pkg/bus/bus.go:138-149`, `:384-422` |
| Adjudication logic | `pkg/bus/gameplay.go:523`, `pkg/gameplay/game.go:737` |
| Correspondence lock (in-proc) | `pkg/stores/game/cache.go:92-391` |
| Game LRU cache | `pkg/stores/game/cache.go:59` |
| Tournament LRU cache | `pkg/stores/tournament/cache.go:51` |
| Broadcast poller | `pkg/broadcasts/poller.go:23` |
| VDO webhook poller | `pkg/vdowebhook/service.go:312` |
| Analysis reclaim worker | `pkg/analysis/service.go:405` |
| Inline migration guard | `cmd/liwords-api/main.go:159` |
| Redis presence | `pkg/stores/redis/redis_presence.go` |
| Migration task infra | `aws/cfn/db-migration-task.yaml` |
