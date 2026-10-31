# Zero-downtime deploys and more than one API server

Status: audit, first draft (2026-10-31). Nothing here is implemented yet.

## Goal

Deploy the API without an outage: start the new task, move traffic to it, stop
the old one, and lose no moves or data on the way. After that, run two API
servers all the time.

Today every API deploy is a short outage. The service runs one task with
minimum healthy 0% and maximum 100%, so ECS stops the old task before it
starts the new one, and moves sent in the gap fail.

The game store is ready for this: #2091 removed the game cache, and games are
serialized by a Postgres advisory lock (`pkg/stores/game/lock.go`) rather than
an in-process mutex. Everything below is what is still in the way.

## Summary

| # | Area | Problem | Severity | Size |
|---|---|---|---|---|
| 1 | Deploy infrastructure | A second API task cannot be placed | blocker | infra |
| 2 | Tournament store | Per-server cache and in-process mutex; two servers overwrite each other | blocker, correctness | large |
| 3 | Event-stream API | Queue-subscribed, so each server sees only some events | correctness | small |
| 4 | Broadcast feed cache | Only the server that polls refreshes; others stale up to 30 min | staleness | small |
| 5 | Shutdown | Moves in flight are killed; NATS keeps delivering during shutdown | lost moves on every deploy | medium |
| 6 | Version skew | Old and new code run side by side during a deploy | process | ongoing |
| 7 | Background jobs | Run once per server | duplicate work, not correctness | medium |
| 8 | Socket server | Already overlaps on deploy; clients reconnect | acceptable | none now |

## 1. Deploy infrastructure

What is there now (read from ECS and the load balancer, 2026-10-31):

- `liwords-api` task def: bridge networking, container port 8001 mapped to a
  **fixed host port 8001**. Two API tasks cannot share an instance.
- The task reserves 1,741 MB. The cluster is three t3.small instances with
  1.4-1.7 GB free each, so a second API task does not fit anywhere.
- `woogles-api` service: minimum healthy 0%, maximum 100%.
- Target group: deregistration delay **5s**, health check `/ping` every 5s.
- No `stopTimeout` on the container, so ECS sends SIGKILL 30s after SIGTERM.
- For comparison, `woogles-socket` already runs minimum 100% / maximum 200%
  with a 30s deregistration delay, so socket deploys overlap today.

What is needed:

- **Room for two tasks.** Fargate does this directly: each task gets its own
  network interface (awsvpc), so there is no host port to collide on and no
  instance to size. The alternative on EC2 is dynamic host ports plus a bigger
  instance or a fourth one.
- **Overlapping deploys:** minimum healthy 100%, maximum 200%.
- **Deregistration delay at least as long as the drain** in section 5, and a
  `stopTimeout` above it (ECS allows up to 120s). Fargate sends the same
  SIGTERM, waits `stopTimeout`, then SIGKILL, so section 5 applies unchanged.

## 2. Tournament store

`pkg/stores/tournament/cache.go` keeps up to 50 tournaments in an LRU, per
server, and hands every caller the same `*entity.Tournament`. Writers
serialize on that object's own mutex: 34 `Lock`/`RLock` calls in
`pkg/tournament/tournament.go` and `preview.go`. This is the pattern the game
cache had before #2091.

With two servers each has its own copy and its own mutex. A result recorded
on server A is invisible to server B's cached copy, and B's next save writes
its stale copy over A's. That loses results and can corrupt pairings. It is
the most serious item here.

The fix is the one the games got:

1. A Postgres advisory lock per tournament, held across load-modify-save.
2. Load from the database inside the lock; drop the LRU and the entity mutex.
3. Read-only paths (viewing a tournament) load without the lock.

What it costs, measured so far:

- Size: of the 146 tournaments updated in the last 30 days, divisions average
  8.8 KB stored (compressed). The largest is 90 KB stored, 232 KB as JSON.
- Decoding divisions (`dbObjToEntity`): 2.5ms for the largest, 0.3ms for the
  median, on a laptop. Probably about twice that on a t3.small.
- Load frequency: unknown, because a cache hit records no span. From
  00:00 to 04:00 UTC on 2026-10-31 there were 652 `select tournaments` spans
  (cache misses, slug lookups and listings) and 29 `update tournaments`.

**Step 0 is a span on `Cache.Get` with a hit/miss attribute**, deployed through
a busy tournament weekend, so the number of loads per second the database
would take is measured rather than guessed. If it turns out large, the middle
ground is to keep the cache but check a version column on every `Get` and
reload when it moved, with writes still under the advisory lock.

The listing paths already avoid decoding full divisions (`listingColumns`), so
they are unaffected.

## 3. Event-stream API subscriptions

`pkg/bus/bus.go` subscribes to `user.>` and `game.>` with a NATS queue group
(`pbworkers`). Those topics feed the event-stream API (`/api/eventstream/`,
`/api/game/eventstream/`), whose clients are connected to one particular
server. With a queue group, NATS delivers each message to only one server, so
a stream client connected to the other server misses those events.

Fix: plain subscriptions for those two topics, so every server receives every
event. `ipc.pb.>`, `ipc.request.>` and `bot.publish_event.>` must stay queue
subscriptions, because each must be handled exactly once.

Open question: who uses the event-stream API today.

## 4. Broadcast feed cache

The broadcast poller (`pkg/broadcasts/poller.go`) runs every 30s on every
server. Whether a broadcast is due is decided by `last_polled_at` in the
database, which is shared, so only one server polls each feed. That server
refreshes its in-memory feed cache and drops its RPC response caches. The
other server does neither: its feed cache lives 30 minutes (`feedCacheTTL`),
so its viewers can see scores up to 30 minutes old.

Fix, either:

- each server refreshes its own feed cache from the NATS update that
  `publishBroadcastUpdate` already sends with the divisions in it; or
- poll on every server, keeping the feed cache per server, and use
  `last_polled_at` only for display.

The first avoids hitting the external feed once per server.

## 5. Shutdown and draining

What happens on SIGTERM now (`cmd/liwords-api/main.go`):

1. `srv.Shutdown` stops accepting HTTP requests and waits up to 30s for those
   running.
2. Archive uploads still running are waited for (#2146).
3. `pubsubCancel()` stops the bus loop; `main` returns.

Moves do not arrive over HTTP. They come from the socket server over NATS, and
the bus handles each message in its own goroutine (`pkg/bus/bus.go`, the
`ipc.pb.>` case). Nothing tracks those goroutines, so a move being handled when
the process exits is killed. The bus also stays subscribed through steps 1
and 2, so new moves keep arriving while the server is shutting down.

Since #2090 the move transaction is atomic, so a killed move is either fully
saved or not saved at all. What is lost is everything after the save:
sending the move to the players, and at game end the end-of-game work
(ratings, standings, lobby, recently ended list). The archive is recovered by
the hourly sweep; the rest is not.

Needed, in order, within `stopTimeout`:

1. Fail `/ping` so the load balancer stops sending HTTP traffic.
2. `natsconn.Drain()`: unsubscribe from the queue groups so new moves go to the
   other server, then deliver what was already received.
3. Wait for the per-message goroutines (a `sync.WaitGroup`, as
   `HistoryArchiver.Wait` does for uploads).
4. `srv.Shutdown`, then wait for archive uploads.
5. Exit.

The socket server sends requests (`ipc.request.registerRealm` and others) with
a timeout; during a drain they go to the remaining server.

## 6. Old and new versions side by side

During an overlapping deploy both versions run for a minute or more. Rules
this imposes from then on:

- **Migrations** run when the new task starts (`migrate.Up` in
  `cmd/liwords-api/main.go`). golang-migrate takes a database lock, so two
  starting tasks do not both run them. But the old code keeps running against
  the new schema, so a migration must not break it: add columns and tables
  first, and drop or rename only in a later release once nothing reads them.
- **NATS messages and protos** must be readable by both versions: add fields,
  don't repurpose them.
- **Flags** that change what is written need the same care, since both
  versions write.

This belongs in the contributor docs once the first overlapping deploy ships.

## 7. Background jobs

What runs on every API server today, and what happens with two:

| Job | Interval | With two servers |
|---|---|---|
| Real-time adjudicator (`bus.adjudicateGames`) | 10s | Safe: acts only under the game lock and re-checks. Twice the query load; after #2159 that is one 114ms query per server per tick. |
| Correspondence adjudicator | 60s | Same. |
| Seek expiry | 10 min | Safe: one SQL delete. |
| Channel monitor | 5s | Per-server logging. Fine. |
| Broadcast poller | 30s | See section 4. |
| Analysis job reclaim | 30s | Safe: one SQL update. |
| VDO stream poller | ~60s, jittered | Duplicate calls to VDO.Ninja; deactivation is idempotent. |
| Client record pruner | daily | Safe (says so in its comment). |
| Shadow summary | hourly | Per-server counters. Fine. |

None of these is a correctness problem with two servers once section 4 is
fixed. A separate worker service (the same binary in a worker mode, its own
ECS service, one task) would remove the duplicate work and give these jobs one
owner. A gap while the worker redeploys only delays a timeout by a few
seconds. It is worth doing, but after sections 1-5.

The scheduled maintenance tasks (`aws/cfn/maintenance-tasks.yaml`) already run
as their own ECS task and are not affected.

## 8. Socket server

- It subscribes to NATS without queue groups, so every socket server receives
  every message and delivers it to its own clients. Presence is in Redis.
  Running two is already supported in principle.
- Deploys already overlap (100% / 200%).
- On SIGTERM it calls `srv.Shutdown`, which does not close WebSocket
  connections; they drop when the process exits. The frontend reconnects
  every second, indefinitely (`liwords-ui/src/socket/socket.ts`).

Nothing needed now. Later: close connections with a "going away" frame and
some jitter, so a deploy doesn't reconnect every client in the same second.

## Order of work

1. **Measure tournaments** (section 2, step 0): the `Cache.Get` span.
2. **Small fixes, each safe on one server:**
   - event-stream subscriptions (section 3);
   - broadcast feed refresh (section 4);
   - draining: health check, NATS drain, tracked handlers, `stopTimeout`
     (section 5). This one helps immediately: deploys stop killing moves.
3. **Tournaments without the cache**, with a lock per tournament (section 2).
4. **Infrastructure:** Fargate (or dynamic ports), overlapping deploys, load
   balancer timings (section 1).
5. **Two API servers on staging:** bots playing real-time games and a
   tournament round while deploying, then killing a server mid-game.
6. **Production:** overlapping deploys first, then two servers.
7. **Worker service** for the background jobs (section 7).

The version-skew rules (section 6) apply from step 4 on.

## Not covered yet

- The other in-memory structures in the API (`pkg/bus/event_api.go`
  subscriber maps, OBS stream handlers) were looked at only for whether they
  hold per-connection state, which they do. That is fine as long as each
  connection stays on one server. (The analysis `MemoryResultStore` is
  test-only.)
- Redis usage (presence, chat) was not audited beyond noting it is shared.
- Long-lived HTTP streams (event-stream API, OBS) and how they behave during
  a drain: they keep the old task alive until the deregistration delay ends.
