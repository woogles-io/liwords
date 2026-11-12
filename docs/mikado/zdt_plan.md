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
| 2a | COP pairing | Runs for seconds under the tournament lock | correctness with two servers | medium |
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

What a deploy looks like now (ECS events and API logs, 2026-11-11, times
UTC):

| Time | |
|---|---|
| 05:22:30 | Old task deregistered from the load balancer. HTTP gets 503s from here. |
| 05:22:59 | Old task gets SIGTERM and exits within a millisecond. Moves over NATS fail from here. |
| 05:23:11 | New task started. |
| 05:23:18 | New task listening, and subscribed to NATS: moves work again. |
| 05:23:22 | New task registered with the load balancer. |
| ~05:23:32 | Earliest it can pass 2 health checks 5s apart. HTTP works again. |

So HTTP is down for over a minute, and moves for about 20s: the old task keeps
handling NATS for the 30s between deregistration and SIGTERM, and the new one
subscribes before it passes health checks. Draining (section 5) adds the time
in-flight work takes, normally milliseconds, to the gap between the tasks;
only overlapping deploys remove the gap.

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

**Status (2026-11-10): done**, as step 1d of `tournament_store_rewrite.md`.
The cache and the entity mutex are gone. Every change goes through
`LockForChange` (`pkg/stores/tournament/lock.go`): an in-process lock on the
tournament, then a transaction holding `pg_advisory_xact_lock` on it, the
load, the change, the save and the commit. Socket messages go out after the
commit. Reads load from the database each time. Measured before: full loads
2-15 ms, saves 8-33 ms, on the ten largest production tournaments.

### 2a. Pairing rounds with COP

**Problem.** COP pairs a round by simulating the rest of the tournament. At
the UI defaults (100,000 division sims, 10,000 control-loss sims) that takes
2.5-3.5 s on 16 cores mid-event and 4.5-7 s with two cores, which is what
production has. It runs inside the change that makes the round pairable, so
the tournament lock is held the whole time:

- every other change to the tournament, in every division, waits for it;
- with two servers, a change waiting on the other server's lock gives up
  after the 10 s `lock_timeout` and the result is lost;
- the last game of a round ends under its game lock, so that game's final
  save waits for COP too;
- a server stopped mid-COP (every deploy) loses the change in flight.

COP is reached from `ClassicDivision.PairRound` → `pairRoundWithCOP`, which is
called from `SubmitResult`, the director's Pair Round, `SetRoundControls`,
`SetSingleRoundControls`, `AddPlayers`, `RemovePlayers` and
`PairLaggedRoundAfter`. The COP lambda (`cmd/coprunner`,
`PairService.HandlePairRequest`) never touches tournaments.

**Rejected: retry around the callers (#2251, closed).** Apply the change
with COP disabled; on a miss, abandon it, run COP unlocked, and apply again,
using an answer only for an identical request. It is correct, but covers only
the callers that are wrapped, keeps the finishing game waiting, and loses the
change if the server stops mid-COP.

**Plan: a durable pairing job (transactional outbox).**

1. **Mark, don't pair.** When the division would pair a COP round,
   `pairRoundWithCOP` records the round as pending instead of calling COP.
   Every caller goes through it, so one change covers them all. Other pairing
   methods are cheap and stay inline.
2. **Write the job in the same transaction as the change.** A `pair_round`
   row (tournament, division, round, `run_at = now`) in `tournament_jobs`,
   saved with the change that made the round pairable. The result and the
   promise to pair it commit together or not at all, so a job survives a
   deploy or a crash. The lock goes back to being held for milliseconds.
   After the commit, the server publishes a wake message (below).
3. **The job runner pairs.** It claims the job with a lease, loads the
   tournament without the lock, builds the `PairRequest` and runs COP, in
   process or on the lambda. Then it takes `LockForChange`, builds the request
   again, and:
   - if it is equal, applies the pairings, marks the job done, commits, and
     publishes the pairings;
   - if not (a director amended a score meanwhile), runs COP again. Nothing
     waits on it, so there is no fallback to computing under the lock.
   A job that keeps failing is reported to the directors through the
   existing auto-pairing failure path (`TakeAutoPairingFailures`).
4. **Pending state in the UI:** "Pairing round N…" until the pairings arrive.

**The job runner (`pkg/jobs`).** `pair_round` is its first kind; tournament
automation (scheduled round starts, ready deadlines and so on) adds more kinds
to the same table and runner later.

```sql
tournament_jobs (
  id            bigserial primary key,
  tournament_id uuid,
  kind          text not null,              -- pair_round, ...
  division      text not null default '',
  round         int  not null default -1,
  run_at        timestamptz not null,
  lease_until   timestamptz,
  done_at       timestamptz,
  attempts      int not null default 0,
  last_error    text,
  unique (tournament_id, kind, division, round)
);
```

- **No fixed polling.** One goroutine sleeps until the next job is due
  (`min(coalesce(lease_until, run_at))` over jobs not done), or until a wake
  message arrives, and at most 5 minutes as a safety net. A `pair_round` job
  is due at once, so with the wake message COP starts milliseconds after the
  result commits. Idle cost: one indexed query every 5 minutes.
- **Wake messages.** Whoever enqueues a job publishes
  `tournament.jobs.wake` on NATS after the commit. Runners subscribe with a
  plain subscription, not a queue group, so every runner hears it. A lost
  wake delays a job by at most the safety-net sleep.
- **Claims and leases.** Due jobs are claimed with `UPDATE … FOR UPDATE SKIP
  LOCKED`, setting `lease_until = now + 30 s` (COP takes about 5 s). The
  claim commits before the job runs: no transaction stays open across COP.
  Each job goes to exactly one runner; if a runner dies mid-job its lease
  expires and another takes it.
- **Idempotent handlers.** Every handler re-checks the state under
  `LockForChange`; pairing a round that is already paired is a no-op. So a
  duplicate run or a retry after a lost lease is harmless.
- **Shutdown.** On drain (section 5) the runner stops claiming. A job in
  flight finishes or its lease expires.

**Where it runs.**

1. **Until the worker service exists:** in every API server. The runner is
   built for several at once, so this is safe on one server and on two.
2. **Then:** only in the worker service (section 7), behind a mode flag. API
   servers only enqueue and wake. COP's CPU moves off the API servers with
   it, or to the lambda. The worker then needs overlapping deploys, or
   pairing waits out its restart.

"Round complete, next round not yet paired" already exists: MANUAL rounds,
and lagged rounds whose follow-on pairing failed. `autoStartRounds` logs and
carries on when the next round can't open, so the round start just waits for
the job.

Before COP moves to the lambda, `HandlePairRequest` needs auth and a cap on
sims; today anyone can call it.

Later, the same outbox could make bus publishes survive a crash between
commit and publish. Not needed now: clients refetch when they reconnect.

**Status (2026-11-11): planned.** Needed before two servers share the lock
(step 5 below); with one server, COP under the lock blocks other changes for
a few seconds, as it always has. It runs in the API servers first because the
worker service comes later.

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

**Status (2026-11-11): done** (`queueGroup` in `pkg/bus/bus.go`). With one
server nothing changes: it already received every message.

### NATS subscriptions with more than one of each service (audit, 2026-11-12)

A subscription is either in a queue group (each message to one subscriber of
the group) or plain (each message to every subscriber). The queue groups on
`ipc.pb.>` and `ipc.request.>` date from 2020, when those were the only
topics; `user.>` and `game.>` were added to the same loop in 2022 for the
event-stream API and joined `pbworkers` by default, not by choice.

API (`liwords-api`):

| Topic | Message types | Handler touches | Delivery |
|---|---|---|---|
| `ipc.pb.>` | seek request, chat, decline seek, game meta event, seek accepted, gameplay event, timed out, ready for game, ready for tournament game, `initRealmInfo`, `leaveSite`, `leaveTab`, `pongReceived` | Postgres (under the game or tournament lock), Redis (presence, chat, config), NATS publishes | queue `pbworkers`: once |
| `ipc.request.>` | `registerRealm`, `getFollowers`, `getFollows` | reads, one reply | queue `requestworkers`: once |
| `bot.publish_event.>` | bot moves | same path as a player's move | queue `pbworkers`: once |
| `user.>`, `game.>` | everything the API sends to sockets | this server's event-stream clients | plain: every server |
| `channel.anno*`, broadcast and user-slot subjects | "changed, reload" signals | this server's OBS overlay clients | plain: every server |

"Once" is right only if no handler changes something kept in one server's
memory, which the other server would then never see. Checked: games and
tournaments aren't cached (#2091, #2242); presence, chat and the
games-enabled flag are in Redis; seeks, users and sessions in Postgres; the
bus has no package-level state. The one exception is the game store's
5-second cache of the active-games list (`pkg/stores/game/cache.go`): a server
clears it when a game starts or ends there, so the other's copy can lag the
lobby by up to 5s. Acceptable.

Publishes from outside the bus: analysis results to `user.<uuid>` (sockets
and event streams), `sessionend.<uuid>` (every socket server), annotation and
broadcast change signals (every API server's OBS subscriptions). No message is
meant for one API server only.

Socket server (`liwords-socket`): plain subscriptions on all its topics
(`services/socketsrv/pkg/hub/pubsub.go`), so every socket server gets every
message and delivers it to its own clients; `connid.<cid>` reaches only the
server holding that connection. What it sends goes to the API's queue groups.
Its followers and follows caches (5-minute TTL, never invalidated) are stale
the same way with one server or two.

Bot (`macondo-bot`, in the macondo repo): one queue subscription,
`bot.commands` in `bot_queue` (`bot/bot.go`), so each move request goes to
one bot. It sends the move with a NATS request to `bot.publish_event.<gid>`
and retries until the API acknowledges, so a lost acknowledgement can deliver
a move twice; the second is rejected as not on turn. The service already
deploys with overlap (100% / 200%). On SIGTERM it exits without draining, so
a move being computed is lost: a real-time bot game then ends when the bot's
clock runs out, and a correspondence game waits for an admin's
`UnfreezeBot`. Worth a drain like the API's, in the macondo repo.

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

**Status (2026-11-12): done in code** (`pkg/bus/drain.go`,
`cmd/liwords-api/shutdown.go`). On SIGTERM `/ping` returns 503, then:

1. HTTP shutdown, capped at 5s. Streaming routes (event-stream API, OBS
   overlays) end 2s after it starts, so an open stream can't hold it. HTTP
   goes first because requests send events through the bus.
2. `Bus.Drain`: drain each subscription (not the connection, which handlers
   still publish on), stop adjudication and seek expiry, then wait until the
   loop has taken everything buffered, every handler goroutine is done, and
   every event is published. The loop checks those conditions itself,
   between iterations (`waitInLoop`), since it twice holds a message that is
   in no buffer and not yet counted.
3. The archive wait, then a NATS flush and close.

It has 25s (`GracefulShutdownTimeout`), plus 2s each for the NATS flush and
the span export, under ECS's default 30s `stopTimeout`. Each shutdown is an
`api.shutdown` trace in Jaeger with a span per step; `bus.drain` has an event
per step. Spans carry `host.name`, so the messages a drain waited for are
that host's spans inside the drain's window. Still to do with the
infrastructure (section 1): set `stopTimeout` and a deregistration delay to
match.

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
owner. It is worth doing, but after sections 1-5.

Background work then comes in two kinds:

- **Tickers** (the table above) are safe on every server and move to the
  worker only to stop running twice.
- **Durable jobs** must run once and survive deploys: rows in
  `tournament_jobs`, run by the job runner of section 2a. The first kind is
  COP pairing (`pair_round`); tournament automation adds more
  (`docs/automated_tournaments_plan.md` §3.2). The runner starts out in the
  API servers and moves to the worker with the tickers.

The worker must use overlapping deploys (min 100% / max 200%, like the socket
service), not stop-then-start. For tickers a 30-60 s gap would only delay a
timeout, but durable jobs would wait it out: a 15 s "next round" countdown
would take a minute, and a round's COP pairings would wait for the restart.
Two workers running briefly during a deploy is safe, because jobs are claimed
with `SKIP LOCKED` leases.

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
   Then COP pairing as a job outside the lock (section 2a).
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
