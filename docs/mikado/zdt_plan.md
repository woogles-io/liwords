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
| 4 | Broadcast feeds | Only the server that polls refreshes; others stale up to 30 min | staleness | small |
| 5 | Shutdown | Moves in flight are killed; NATS keeps delivering during shutdown | lost moves on every deploy | medium |
| 6 | Version skew | Old and new code run side by side during a deploy | process | ongoing |
| 7 | Background jobs | Loops run once per server; maintenance runs in a separate, partly wired container | duplicate work; league game events dropped | medium |
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

**Plan: a pairing job, with COP on the lambda and no lock held while
pairing.**

1. **Mark, don't pair.** When the division would pair a COP round,
   `pairRoundWithCOP` records the round as pending instead of calling COP.
   Every caller goes through it, so one change covers them all. Other pairing
   methods are cheap and stay inline.
2. **Write the job in the same transaction as the change.** A `pair_round`
   row (tournament, division, round) in `jobs` (section 7), saved with the
   change that made the round pairable, and already claimed by the server
   writing it (`lease_until = now + 60 s`). The result and the promise to
   pair it commit together or not at all, so the job survives a deploy or a
   crash. The lock goes back to being held for milliseconds.
3. **The same server pairs, right after the commit,** in a goroutine the
   drain (section 5) waits for:
   1. It builds the `PairRequest` from the state it just committed, and
      invokes the COP lambda. No lock is held; the server only waits on the
      call, and uses no CPU for it.
   2. It takes `LockForChange`, loads the division and builds the request
      again:
      - equal: applies the pairings, marks the job done, commits, and
        publishes the pairings;
      - the round is already paired (a director paired it by hand): marks
        the job done;
      - otherwise (a director amended a score meanwhile): back to step 1
        with the new request, up to 3 times.
   3. On a failure (lambda error or timeout) it records `attempts` and
      `last_error`, sets `run_at` with backoff and clears the lease. After
      several failures the directors are told through the existing
      auto-pairing failure path (`TakeAutoPairingFailures`).
4. **The job runner is the backstop.** It claims `pair_round` jobs that are
   due and unclaimed, or whose lease expired because a server stopped or
   died mid-call, and runs the same steps. COP is a pure function of the
   request, so running it twice is harmless.
5. **Pending state in the UI:** "Pairing round N…" until the pairings arrive.

There is no wake message and no polling on the normal path: the server that
made the round pairable starts pairing at once. The runner only picks up
what a server dropped, so its polling interval bounds recovery, not normal
latency.

**COP on the lambda.** COP needs a lot of CPU for a few seconds, a few times
per tournament round. On the API servers it competes with moves for two
cores; on an always-on worker we would pay for cores that sit idle. The
lambda bills per millisecond, and divisions that finish at the same time
each get their own CPUs.

What exists: `services/cop-lambda`, deployed by hand with SAM as
`COPPairFunction`. Its only caller is the API's public
`PairService.HandlePairRequest` (the function has no URL and no resource
policy), which TSH and `cmd/coprunner` call without an account. Today it has
1,024 MB (about 0.6 vCPU) and a 20 s timeout. Over the last 60 days it ran
104 times, averaging 4-6 s, at most 10.8 s.

**Two functions from the one template.** The pairing job needs more CPU
than TSH's calls do, and the public RPC has no auth:

- `COPPairFunction` stays as it is, for `HandlePairRequest`. TSH keeps
  working without accounts, and what an anonymous caller can spend stays
  what it is today.
- A second function, at higher memory, is invoked only by the pairing job,
  directly through the AWS SDK with the API's IAM role. Nothing outside
  AWS can call it.

Its memory sets its CPU: Lambda gives 1 vCPU per 1,769 MB, up to 6 vCPU at
10,240 MB. Measure the UI defaults (100,000 / 10,000 sims) at 4, 6 and 10 GB
and take the knee. At 10 GB a 5 s pairing is 50 GB-s, under a tenth of a
cent. Timeout 60 s.

**Auth for `HandlePairRequest`, later and only if it is abused.**
- Woogles API keys (`X-Api-Key`) already exist, and any account can create
  one. TSH would send one from its config.
- To get there: log whether each call carries a key, tell TSH users, then
  require it.
- A cap on sims and a limit on calls in flight are the cheaper first steps.
- None of this is needed for the pairing job.

**Other changes:**

- **Deploy both functions from CI with the API.** They carry their own copy
  of COP; deployed by hand, a COP fix reaches the API's tests but not the
  pairings.
- **Logging:** keep logging each request, so pairings can be checked after
  the fact, as one line of JSON instead of indented. The log group keeps 14
  days (the last call, on 2026-10-18, has already expired). Raise the
  retention if we want to look further back.
- **Dev and tests:** with no function name configured, `cop.COPPair` runs in
  process, through the same steps. The job calls COP through an interface,
  so tests use a fake.
- **Permissions:** the API invokes the functions with its instance role.
  `HandlePairRequest` already does, so the permission is likely in place for
  the first function. Add the second, and check both on Fargate, where the
  task role replaces the instance role.

A Go lambda cold-starts in a few hundred milliseconds, small next to COP.

**Tests** (against the test database, with a fake COP client that can be
held on a channel, and leases short enough to expire within a test):

1. **Mark, don't pair:** the last result of a round before a COP round
   leaves the round pending with no pairings, and a `pair_round` row. A
   change that fails after the row is written leaves neither.
2. **No lock during COP:** with the fake held, another change to the same
   tournament commits.
3. **Normal path:** the fake answers; the pairings are applied and
   published, and the job is done.
4. **Changed meanwhile:** a score amended while the fake is held throws the
   first answer away; COP runs again on the new request, and those
   pairings are applied.
5. **Paired by hand meanwhile:** the job finishes without touching the
   director's pairings.
6. **Server dies mid-call:** the first runner claims the job and stops
   without finishing. A second runner on the same database doesn't take it
   while the lease holds, takes it once it expires, and applies the
   pairings.
7. **One claim each:** with several runners polling at once, every due job
   is claimed by exactly one.
8. **Failures:** a failing COP raises `attempts` and pushes `run_at` back;
   after the last attempt the directors get the failure through
   `TakeAutoPairingFailures`.
9. **Drain:** a draining runner claims nothing new, and the job in flight
   finishes.
10. **Old runner, new kind:** a runner without a handler for a kind leaves
    its rows alone.

"Round complete, next round not yet paired" already exists: MANUAL rounds,
and lagged rounds whose follow-on pairing failed. `autoStartRounds` logs and
carries on when the next round can't open, so the round start just waits for
the job.

Later, the same outbox could make bus publishes survive a crash between
commit and publish. Not needed now: clients refetch when they reconnect.

**Status (2026-11-13): planned.** Revised from the 2026-11-11 plan: COP runs
on the lambda, the server that commits the change starts the pairing, and
there is no NATS wake message. Needed before two servers share the lock
(step 6 of the order of work); with one server, COP under the lock blocks
other changes for a few seconds, as it always has.

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

## 4. Broadcast feeds

Before: the broadcast poller (`pkg/broadcasts/poller.go`) ran every 30s on
every server, and `last_polled_at` decided whether a feed was due, without
claiming it. The server that polled kept the parsed feed in memory for 30
minutes; another server kept serving its own copy, up to 30 minutes old.
Nothing was stored, so on a cache miss a viewer's request fetched the
director's feed itself, and a broadcast whose feed URL went away showed no
pairings or standings.

**Status (2026-11-12): done in code** (`pkg/broadcasts/feed_store.go`):

- The poller claims each due broadcast (`ClaimBroadcastForPolling`, `FOR
  UPDATE SKIP LOCKED` on `next_poll_at`), so each feed is fetched once per
  interval whatever the number of servers. It becomes a schedule in
  `pkg/jobs` (section 7) as is.
- It stores the feed, gzipped, in `broadcast_feeds` under a hash of its
  content, and writes it only when the hash changes. Failures are recorded
  there too and shown in the director panel; the stored feed stays.
- Request paths never fetch. Each server keeps parsed feeds under their hash
  and checks the hash in the database before using one. Ended broadcasts are
  served from their last stored feed.
- `BROADCAST_UPDATED` goes out only when the feed changed.
- The short response caches are keyed by the feed hash and
  `broadcasts.games_version` (bumped on claims, slots and finished
  annotations), so they are correct on every server.

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

Background work runs two ways today, and neither runs once.

**Loops inside every API server:**

| Job | Interval | With two servers |
|---|---|---|
| Real-time adjudicator (`bus.adjudicateGames`) | 10s | Safe: acts only under the game lock and re-checks. Twice the query load; after #2159 that is one 114ms query per server per tick. |
| Correspondence adjudicator | 60s | Same. |
| Analysis job reclaim | 30s | Safe: one SQL update. |
| Broadcast poller | 30s | Safe: each feed is claimed by one server (section 4). |
| VDO stream poller | ~60s, jittered | Duplicate calls to VDO.Ninja; deactivation is idempotent. |
| Seek expiry | 10 min | Safe: one SQL delete. |
| Client record pruner | daily | Safe (says so in its comment). |
| Channel monitor | 5s | Per-server logging. Stays per server. |
| Shadow summary | hourly | Per-server counters. Stays per server. |

**Maintenance tasks** (`cmd/maintenance`): a separate container, started by
two EventBridge rules (`aws/cfn/maintenance-tasks.yaml`).

| Rule | Tasks | Run time, last 7 days |
|---|---|---|
| Hourly at :01 UTC | blog RSS (calls the API with the admin key), sub badges, league hourly runner, archive sweep | usually 3 s, up to 7 min |
| Daily at 05:00 UTC | Patreon token refresh, cancelled games, monitoring streams, expired sessions, unverified users, league unstarted-game reminder | about 30 s, once 4 min |
| By hand | resend season-started emails | |

Two problems with the maintenance container:

- **It builds its own partial copy of the API's wiring.** The league hourly
  runner starts league games with an event channel that a goroutine drains
  (`cmd/maintenance/league_tasks.go`), so the events for those games never
  reach a socket.
- **Its long runs are email loops.** The 7-minute runs send 215
  registration-open emails, 2 s apart, with nothing recording which went
  out. That is fine in a container nobody stops; anything that restarts on
  deploys would lose the rest of the loop.

Coming: COP pairing (`pair_round`, section 2a) and tournament automation
(scheduled round starts, ready deadlines;
`docs/automated_tournaments_plan.md` §3.2).

### One mechanism: `pkg/jobs`

One table and one runner for everything above except the two per-server
monitors.

```sql
jobs (
  id           bigserial primary key,
  kind         text not null,             -- adjudicate_realtime, pair_round, ...
  key          text not null default '',  -- '' for schedules; tournament/division/round for pair_round
  schedule     text,                      -- '@every 10s', '@hourly :01', '@daily 05:00'; null for one-off jobs
  run_at       timestamptz not null,
  lease_until  timestamptz,
  done_at      timestamptz,               -- one-off jobs only
  attempts     int not null default 0,
  last_run_at  timestamptz,
  last_ms      int,
  last_error   text,
  unique (kind, key)
);
```

- **Schedules** are one row per periodic task, upserted by the runner at
  startup. After each run, `run_at` moves to the next time the schedule
  gives.
- **One-off jobs** are `pair_round`, email batches, manual runs and,
  later, tournament automation. `done_at` is set when they finish.
- **Claims:** due rows are claimed with `UPDATE … FOR UPDATE SKIP LOCKED`,
  setting `lease_until`, the pattern the broadcast poller uses. The claim
  commits before the job runs. A runner only claims the kinds it has
  handlers for, so an old server in a deploy leaves a new kind alone.
- **No fixed polling.** The runner sleeps until the earliest `run_at`, at
  most 30 s. The real-time adjudicator, every 10 s, is the most frequent
  thing it wakes for. `pair_round` doesn't wait for it (section 2a).
- **Concurrency:** each job runs in its own goroutine, up to a cap, so a
  long email batch doesn't delay adjudication. A job can't overlap itself:
  its row is claimed.
- **Leases are renewed while a job runs.** A runner that dies loses its
  leases when they expire (30 s, 60 s for `pair_round`), and another
  runner takes the jobs.
- **Every handler must be safe to run twice.** Most already are:
  adjudication and pairing re-check under their locks, and the cleanups are
  single statements. League emails need rework: one row per recipient
  batch, marked as sent.
- **Panics are recovered per job** and recorded as the job's error. The API
  has no panic-recovery middleware, so an unrecovered panic in a job would
  take the whole process down.
- **Drain:** on SIGTERM the runner stops claiming and gives running jobs the
  drain's time (section 5). A job still running after that is picked up
  again when its lease expires.
- **Visibility:** each run is a `job.run` span tagged with its kind, and the
  row keeps the last run, duration and error. Alert when the oldest due job
  is more than a minute late.
- **Manual runs** (resend season-started emails) insert a one-off row.
  `LEAGUE_NOW` testing runs against a local API.
- **Tests:** schedules compute their next run (`@every`, `@hourly :01`,
  `@daily 05:00`, in UTC); a schedule runs again after its run; a job that
  panics is recorded and the runner carries on; claims, leases, expiry and
  drain are covered by the `pair_round` tests (section 2a).

### Where the runner runs

**Recommended: inside the API servers, behind a flag (`JOBS_RUNNER`, on by
default).** With COP on the lambda (section 2a), nothing left needs much CPU.
Running in the API is the same wiring, so jobs send events like any request
(the league games' dropped events are fixed by this alone). It costs nothing
extra and doesn't wait on the infrastructure in section 1. With two API
servers both run it; claims make each job run once.

**Later, if needed: a worker service.** The same image with only the runner
(`JOBS_RUNNER` on there, off in the API servers), its own ECS service, one
task, overlapping deploys. A deploy change, not a code change. Worth it if
jobs start to slow requests. It costs about $9/month at 0.25 vCPU and 0.5 GB
on Fargate, and Fargate bills the reserved size, not wake-ups.

To check before moving the maintenance tasks: the maintenance task role
(`liwordsMaintenanceTaskRole`) grants SES for email. The API's role needs the
same.

### Order

1. **`pkg/jobs` and the `jobs` table**, running in the API servers, with the
   API's loops as schedules. The loops go away.
2. **Maintenance tasks as schedules.** Check one run of each, then disable
   the EventBridge rules. `cmd/maintenance` is kept only to insert manual
   runs.
3. **COP `pair_round` on the lambda** (section 2a).
4. **League emails in resumable batches.**

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

1. **Measure tournaments** (section 2, step 0): the `Cache.Get` span. Done.
2. **Small fixes, each safe on one server:** event-stream subscriptions
   (section 3), broadcast feeds (section 4), draining (section 5). Done;
   `stopTimeout` waits for step 5.
3. **Tournaments without the cache**, with a lock per tournament (section
   2). Done.
4. **Jobs and COP off the lock** (sections 7 and 2a): `pkg/jobs` in the API
   servers with the API's loops, then the maintenance tasks, then COP
   `pair_round` on the lambda, then resumable league emails.
5. **Infrastructure:** Fargate (or dynamic ports), overlapping deploys, load
   balancer timings, `stopTimeout` (section 1).
6. **Two API servers on staging:** bots playing real-time games and a
   tournament round while deploying, then killing a server mid-game.
7. **Production:** overlapping deploys first, then two servers.

A separate worker service is no longer a step: the runner moves out of the
API servers only if jobs start to slow requests (section 7).

The version-skew rules (section 6) apply from step 5 on.

## Not covered yet

- The other in-memory structures in the API (`pkg/bus/event_api.go`
  subscriber maps, OBS stream handlers) were looked at only for whether they
  hold per-connection state, which they do. That is fine as long as each
  connection stays on one server. (The analysis `MemoryResultStore` is
  test-only.)
- Redis usage (presence, chat) was not audited beyond noting it is shared.
- Long-lived HTTP streams (event-stream API, OBS) and how they behave during
  a drain: they keep the old task alive until the deregistration delay ends.
