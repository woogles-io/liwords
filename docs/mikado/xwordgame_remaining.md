# xwordgame: what is left

Companion to `xwordgame_review.md`, which describes what exists. This one is the
list of what does not, written to be picked up cold. `pr1975_split.md` has the
record of how #1975 landed and the three phases that put xwordgame on the live
path; `remove-game-caches.dot` is the graph of all of it.

**Where things stand (2026-11-02):** the referee and the replay are built and
tested against 113,010 real games, and every step of the #1975 split is merged.
Steps 1 to 6 are deployed: moves are one transaction, writers hold a session
advisory lock, and there is no game cache. The two shadows run on production.
`ongoing_games` (step 7) is written behind a flag that is off, and nothing reads
it. Nothing is served from the new path yet: macondo still referees every move.

## The end state, and the order to get there (2026-11-11)

Where this is going: live games in `ongoing_games` plus `game_turns`, finished
games in S3, stats in ClickHouse, and a `games` table slimmed to metadata,
moved onto RDS in one planned infrastructure downtime. The graph has a node
for each step.

1. **`READ_ONGOING_GAMES` on** (rollout step 6 below), then a release later
   delete the `games` versions of the listing queries and the flag.
2. **Phase 1, the shadow referee**, is the main track. Its gate is met: no
   `shadow-turns-mismatch` or `shadow-load-mismatch` from #2163's deploy on
   2026-11-02 to 2026-11-11, two weekends, with each shadow comparing
   ~12-18k games an hour. Then phases 2 and 3 (`pr1975_split.md`, "After the
   split"). Loading from `game_turns` is phase 2 itself, not a step before
   the referee switch: `entity.Game` embeds macondo's game, and the rebuild
   produces an xwordgame position.
3. **Small, independent PRs alongside it:** `game_turns.event` to bytea
   (before phase 2, which makes every load decode turns), dropping the unused
   `games` indexes, quickdata to joins (`game_players` is in place), and the
   `game_meta_events` table.
4. **Stats to ClickHouse, a separate track that can start any time:** an audit
   of which list stats and game stats to keep, a ClickHouse server, the
   end-of-game writer also writing ClickHouse, then a backfill that
   recomputes every game finished before the writer went on, then the readers
   moved over. The writer goes first so no game finished during the backfill
   is missed. **The backfill reads the `history` bytea, so it must
   run before that column is dropped:** reading it in the database is faster
   and cheaper than fetching ~11M objects from S3.
5. **Drop the live-game columns from `games`** (`timers`, `player_on_turn`,
   `ready_flag`, `started`) once step 1's cleanup is done; `meta_events` once
   its table exists; `last_known_racks` moves to `ongoing_games` with phase 2,
   because the rebuild reads it. `stats` after step 4. **`history` last**,
   after phase 3 and the stats backfill.
6. **Planned infrastructure downtime:** dump the slimmed database and restore
   it onto RDS, whose backups hold no long transaction (today's `pg_dump`
   runs ~4 hours a day and blocks vacuum on every table while it does);
   upgrade NATS, Redis and the servers. `DROP COLUMN` frees no space until
   the table is rewritten, and this restore is that rewrite, so the slimming
   only shows on disk after it. Logical replication into RDS with a short
   switchover is the alternative to a full dump-and-restore window. Fargate
   is earlier, on the zero-downtime track (`zdt_plan.md`).

## Production, as measured

| | |
|---|---|
| schema version | `202610090004` (2026-11-02; #2166's `202611010001` not yet deployed) |
| API | `liwords-api:326` (`master-gh2083`, master `8287124f1`), since 2026-11-02 05:28 UTC; `SHADOW_TURNS`, `SHADOW_TURNS_LOAD`, `DUAL_WRITE_TURNS` on |
| `games` | ~13.3M rows (estimate); 2,092 unfinished, all native, all with both player ids |
| `game_turns` | 30,610 rows over 2,047 games; none belong to a finished game or an archived one |
| `ongoing_games` | does not exist yet; created by `202611020001` |
| `game.Get` | 3.8/s, p50 1.4 ms, p99 55 ms (2026-11-01, no cache) |
| `game.lock.held` | p50 8.3 ms, p99 65 ms; `game.lock.wait` p99 10.6 ms |
| games with real meta events | 2.61% (sampled 2026-09-02), at most 10 per game |

## The life of an `ongoing_games` row

With `WRITE_ONGOING_GAMES` on, each step below is in the same transaction as
the `games` write it goes with:

| When | Code | The row |
|---|---|---|
| A game is created (seek, rematch, tournament or league pairing, bot game) | `Create`, from `InstantiateNewGame` | inserted, `started` false; `ready_flag` copied from `games` |
| A player readies | `SetReady` | its bit ORed into `ready_flag` |
| Both are ready, or the game starts without the ready step | `StartGame` → `Set` | `started`, `timers` |
| Every move, challenge, pass, exchange, meta event | `Set` | `play_state`, `on_turn`, `timers`, `meta_events`, `updated_at` |
| The game ends, any way at all (standard, resign, timeout, adjudication, force-finish, abort, cancel) | `Set` with an end reason | **deleted** |

Players, rules, mode, type, tournament and league are written at insert and
never rewritten. A games row outlives its `ongoing_games` row: it is archived
to S3, its turn rows deleted, and if it was cancelled it is deleted itself
two days later. Annotated games and puzzles never have a row; neither is
created by `Create` or saved by `Set`. Games already live when the flag went
on get theirs from the backfill.

## Rolling out `ongoing_games`

1. **Merge.** The migration runs: an empty table, no lock on `games` (there is
   deliberately no foreign key; the migration says why).
2. **Deploy the code with the flag off.** No change: `Set` writes the row only
   when `WRITE_ONGOING_GAMES` is on.
3. **Turn on `WRITE_ONGOING_GAMES`** in the API task definition. From then on
   a new game gets its row in the transaction that creates it, every save of a
   live game upserts it inside the save transaction, and the save that ends a
   game deletes it. **Behavior change:** a failure here
   fails the move. That is the price of a listing that can never disagree with
   the game it names.
   *Watch:* move error rates (`handle-event` errors, failed saves).
4. **Backfill**, once the API with the flag on is the only one running. Games
   that were live before the flip and have not been saved since -- idle
   correspondence games, and games nobody has moved in -- have no row until
   this runs. It is a one-off command, `cmd/backfill-ongoing-games`, built and
   run by hand like `upload-game-archive`, with the API's database env vars,
   `MACONDO_DATA_PATH` (each game is loaded), and the flag set for it as well:
   it refuses to run without it.

   ```bash
   GOOS=linux GOARCH=amd64 go build -o backfill-ongoing-games ./cmd/backfill-ongoing-games
   # on the server, with the API's DB_* env and MACONDO_DATA_PATH:
   WRITE_ONGOING_GAMES=1 ./backfill-ongoing-games --dry-run   # lists them, writes nothing
   WRITE_ONGOING_GAMES=1 ./backfill-ongoing-games --all
   ```

   Never run it before the flag is on in the API: a game that ends in between
   would keep its row, because the save that ended it did not know to delete
   it. Each game is written under its own game lock, the one every move holds,
   and skipped if it has ended or already has a row, so it is safe to run
   again. It ends with a `done` line giving `wrote`, `skipped`, `failed` and
   `remaining`; `remaining` should be 0. A game whose lock was busy is logged
   as `backfill-failed`, the command exits 1, and running it again picks the
   game up. About 2,000 games, each a lock, a load and one insert: well under
   a minute.
5. **Check it** before anything reads it, with `bin/proddb` (read-only):

   ```sql
   SET statement_timeout = '30s';
   -- rows for games that have ended: must be 0
   SELECT count(*) FROM ongoing_games o JOIN games g ON g.uuid = o.game_uuid
   WHERE g.game_end_reason <> 0;
   -- rows with no games row at all: must be 0
   SELECT count(*) FROM ongoing_games o
   WHERE NOT EXISTS (SELECT 1 FROM games g WHERE g.uuid = o.game_uuid);
   -- live games with no row: must be 0 after the backfill
   SELECT count(*) FROM games g
   WHERE g.game_end_reason = 0 AND g.type IS DISTINCT FROM 1
     AND NOT EXISTS (SELECT 1 FROM ongoing_games o WHERE o.game_uuid = g.uuid);
   -- the denormalized fields agree: must be 0
   SELECT count(*) FROM ongoing_games o JOIN games g ON g.uuid = o.game_uuid
   WHERE o.on_turn IS DISTINCT FROM g.player_on_turn::smallint
      OR o.started IS DISTINCT FROM g.started
      OR o.timers::jsonb IS DISTINCT FROM g.timers::jsonb
      OR o.ready_flag IS DISTINCT FROM COALESCE(g.ready_flag, 0)
      OR o.player0_id <> g.player0_id OR o.player1_id <> g.player1_id
      OR o.tournament_id IS DISTINCT FROM g.tournament_id
      OR o.league_division_id IS DISTINCT FROM g.league_division_id;
   ```

   The third and fourth can be briefly non-zero for a game created or moved
   between the two reads; re-run them before worrying. Run them again a day
   later, and after a busy weekend.

   *Done 2026-11-09:* flag on with `liwords-api:328`, backfill wrote 2,327
   rows (`failed=0 remaining=0`), all four checks 0 then and ten minutes
   later.
6. **Turn on `READ_ONGOING_GAMES`** once the checks have stayed at 0 for a few
   days. The active-game listings -- lobby, tournament and league
   correspondence lists, both adjudicators, the bot unfreeze tool, the
   correspondence count and the moderators' chat review -- then pick their
   games from `ongoing_games` and read the columns from `games` by uuid
   (`pkg/stores/game/listings.go`). Each `...FromOngoing` query returns the
   same columns as its `games` twin; `TestListingsAgreeAcrossTables` holds
   them to the same rows. On 2026-11-09 the old and new queries returned
   identical sets on production (68 real-time, 2,392 correspondence, 2
   tournament, 1 bot on turn). The real-time listing goes from ~122 ms to
   ~3 ms; the full correspondence listing from ~122 ms to ~47 ms, the rest
   being the 2,400 `games` rows it still reads.
   The correspondence adjudicator no longer reads that listing: it asks
   `ListCorrespondenceTimeoutCandidatesFromOngoing` for the games whose
   `ongoing_games` timers say the player on turn may have run out -- RanOut's
   test in SQL, using the increment the timers now carry (#2235) -- and reads
   `games` only for those. On 2026-11-09 that was 0 of 2,389. A row whose
   timers lack `"inc"` is always a candidate, so the filter is exact once
   #2235's migration has run and merely unhelpful before it.
   `TestCorrespondenceTimeoutCandidates` pins it to `RanOut` at the boundary.
   The read flag is ignored unless `WRITE_ONGOING_GAMES` is also on, so
   turning the writer off can never leave listings reading a stale table.
   *Rollback:* unset `READ_ONGOING_GAMES` and redeploy; nothing is written
   differently.
   *Watch:* lobby game counts, correspondence timeouts still firing
   (`adjudicating-after-gameplay-timed-out`), real-time timeouts and
   never-started cancellations.

**Do not turn the flag off and on again** without cleaning up. A game that ends
while the flag is off keeps its row, and a ready while it is off is missing
from the row. If it ever is turned off, delete every row and run the backfill
again once it is back on.

Readiness: the insert copies `games.ready_flag`, and with the flag on `SetReady`
ORs the player's bit into both rows in one transaction (the caller holds the
game lock, as every save does). The rules columns are never rewritten on
conflict; they are fixed when a game is created.

### Then let the shadows run

**Do not serve a reconstructed position until the shadows have been quiet for a
sustained period on real traffic.** Inferring position from an event log without
verifying it is exactly what corrupted 944 games in May 2026. The corpus says
113,009 of 113,010 archived games reconstruct exactly; the shadows are what say
the same of live ones.

Since the cache went (2026-10-31) every load is a rebuild, and that exposed a
macondo replay quirk: `PlayTurn`, which `NewFromHistory` replays through, keeps
the previous play's words after a pass or an exchange, where the live path
clears them. That was all of the 306 `shadow-turns-mismatch` lines seen on
2026-11-01 (16 games, every one a `lastWordsFormed` difference). #2163 refuses
a challenge unless the last event is a tile placement and drops that field from
the comparison when it cannot be challenged. #2163 deployed on 2026-11-02 at 05:28 UTC, and there has not been a mismatch
since: in the first eleven hours, 163,450 load comparisons and 119,332 save
comparisons, none torn, none with too few or too many rows. The bar is
zero `shadow-turns-mismatch` and `shadow-load-mismatch` for a week including a
weekend. Then phase 1 (`pr1975_split.md`, "After the split").

## Shrinking `games`

Once the turns path is proven, the big table stops being where live state lives.
Ordered by what is safest to move.

- **`timers`** -> live only. The historical record already exists and is better:
  every `GameEvent` carries `MillisRemaining`, so per-move timing is in the
  event log. The blob is derived state, rewritten on every move of a 12M-row
  table. The one reader at game end is automod's sit-and-resign check, which
  runs while the game is still live.
- **`history` bytea** -> last to go, not first. It is the fallback the read path
  falls back *to*, and `verifyHistory` deep-compares the turns-assembled history
  against it at archival and refuses to archive on a mismatch. That check is the
  best evidence available that the two agree; keep it until the reconstruction
  has been serving without incident.
- **`ready_flag`** -> live only, eventually. The bug noted here -- `UpdateGame`
  writing a literal `0` on every save, wiping a half-finished ready handshake --
  is fixed: the column is out of that statement, because it is owned by
  `SetReady`, which ORs one bit per player into it inside a single statement.
  The rule that came out of it is worth keeping: a column the database
  accumulates must never appear in a whole-row UPDATE built from in-memory
  state.

### Indexes on `games` that nothing uses

Separate from the column work, and independent of everything else:
`games` carries about **2.5 GB of index the planner has essentially never
chosen.** The statistics are trustworthy -- `pg_stat_database.stats_reset` is
2026-03-03, so the counts below (re-read 2026-11-02) cover eight months.

The cost is not mainly disk. This is the table rewritten on every move, so every
index is maintained on every non-HOT update: it is write amplification on the
hottest path in the application, and it inflates the backup that already takes
~2 hours.

| index | size | scans since 2026-03-03 | verdict |
|---|---|---|---|
| `idx_games_lexicon_created_at` | 682 MB | 10 | unexplained; find the query first |
| `idx_games_player0_filtered` | 542 MB | 2 | **dead** -- see below |
| `idx_games_player1_filtered` | 543 MB | 2 | **dead** -- see below |
| `rematch_req_idx` | 537 MB | 20 | **dead** -- see below |
| `idx_games_deleted_at` | 130 MB | 4 | see below -- probably droppable outright |
| `hastybot_games_index` | 122 MB | 0 | ad-hoc; one user id is hard-coded into it |

None is unique, none is a primary key, and none backs a constraint, so the scan
count is the whole story -- there is no second job any of them is quietly doing.
The handful of scans since September are worth one look at the slow-query log
before dropping, but no query in the code can use them.

**Three are dead because the many-to-many migration moved their queries.**

- `rematch_req_idx` is a hash index on `games.quickdata->>'o'`. `GetRematchStreak`
  now reads `game_players.original_request_id`. Nothing queries the expression
  the index covers.
- `idx_games_player{0,1}_filtered` are `(playerN_id, id DESC) WHERE
  game_end_reason NOT IN (0,5,7)` -- finished games for a player.
  `GetRecentGamesByUserId` now goes through `game_players` as well, and the
  `player0_id`/`player1_id` predicates left in `games.sql` are all for *ongoing*
  correspondence games (`game_end_reason = 0`), which these indexes explicitly
  exclude. They cannot serve the only queries that remain.

That is 1,567 MB explained, and the explanation is the same one each time: the
lookups moved to `game_players` and the old indexes were never dropped.

**`idx_games_deleted_at` indexes 12.7M rows of which zero are soft-deleted.**
If anything still needs the column, a partial index `WHERE deleted_at IS NOT
NULL` would be kilobytes rather than 126 MB. If nothing does, drop it.

**`hastybot_games_index`** has a user id baked into its predicate. That is an index built for one
investigation; it should not outlive it.

Do them one at a time with `DROP INDEX CONCURRENTLY`, and confirm the intent of
the two with non-zero scans before touching those -- 10 scans in six months is a
rare report, not nothing. Dropping is reversible, but rebuilding a 500 MB index
on a live 12M-row table takes minutes, so it is not free.

**Already done:** `idx_games_history_s3_key_pending` was **344 MB holding 28,781
entries** -- it was built when every game matched `history_s3_key IS NULL` and
the backfill then deleted 12.6M entries from it, which plain VACUUM makes
reusable but never returns. Reindexed to **1136 kB** (18 MB again by
2026-11-02). Worth remembering as a
shape: a partial index whose predicate goes from matching everything to matching
almost nothing will not shrink on its own.

### Where meta events should live

**Not in the live table, and not where they are now.** An append-only
`game_meta_events (game_uuid, idx, event, created_at)` keyed on
`(game_uuid, idx)`, never deleted.

The reasoning, since this was the open question:

- **Live-only loses the record**, which is the objection that matters. An abort
  denial is evidence about a player's conduct -- automod reads it for
  `NO_PLAY_DENIED_NUDGE` -- and a moderator may want it years later. Deleting
  the row when the game ends destroys it.
- **Staying on `games` is what we are trying to stop.** It is a mutable jsonb
  blob on the 12M-row table, rewritten whenever a meta event arrives, and 327 MB
  of it is ~97% empty: only 2.61% of games have any meta events at all, and
  never more than 10. As rows, that is roughly 330k rows instead of 12.7M
  mostly-empty values, and no rewrite of the games row.
- **Not in `game_turns`.** They are a different type -- `ipc.GameMetaEvent`, not
  macondo's `GameEvent` -- and `game_turns` feeds `assembleHistory`, which feeds
  the S3 history, GCG export and stats. Worse, comments and puzzles are keyed on
  absolute event index (`pkg/stores/comments/db.go:33`,
  `pkg/stores/puzzles/db.go:167,387`), so inserting anything into that sequence
  re-anchors every comment and moves every puzzle position.

They are events, so they get an event table; they are a different kind of event,
so they get their own. Keep the rows at archival rather than folding them into
the S3 object -- they are small, and a moderator reviewing a player should not
have to fetch from S3 to do it.

## The design changed: state is reconstructed, not stored

An earlier version of this branch stored the position in `ongoing_games.state`
and this document described rolling that out in phases A1–D. That is no longer
the plan, and the phases are gone. Read `game_storage_v2.md` for the design and
the plan file for the ordering; the short version:

A game's position is rebuilt from its event log plus four columns that already
exist and are already written on every save. There is no snapshot, so there is
no second record that can disagree with the first.

| Position field | Source |
|---|---|
| board, scores, bingos, per-player turns, scoreless count | replay `game_turns` |
| racks | `games.last_known_racks` |
| bag | conservation: distribution − board − racks |
| whose turn | derived; `games.player_on_turn` is a cross-check |
| play state | `games.game_end_reason`, stated never inferred |
| rules | `games.game_request` |
| last words formed | recomputed from the board and the last placement |

**No event format change is needed, and that is the load-bearing finding.** The
May 2026 failure was a derivation bug, not missing data: `ReplayHistory`
reconstructs 113,009 of 113,010 production games to macondo's exact position
using only what is stored today. So there are no new event types, no new fields,
and nothing to do to the 10M+ archived games in S3.

The one rule that must never be broken: **`PlayState` is set explicitly from
`game_end_reason`, never left to a proto3 zero value.** That is the entire
incident in one line. `db.go` reads `hist.GetPlayState()` and forces it onto the
loaded game, so a history assembled without it silently reads as `PLAYING`.

## Done

Referee and replay (split step 2, #2028):

- `pkg/xwordgame` and `pkg/xwordbridge` — the referee and the replay, validated
  against 113,010 production games. `StateFromTurns` is the production entry
  point: turns rows plus the `games` columns in, position out, ending *stated*
  from `game_end_reason` rather than left to a proto3 zero value.
- `RulesFor`/`RulesSpec` — one place resolving a game's configuration, used by
  both the corpus test and the load path, so the corpus is evidence about
  production rather than about the test.
- `DivergencesVsReconstruction` — the fields a macondo history rebuild cannot be
  held to, with `PlayStateReliable` required rather than defaulted.
- The stored position is gone entirely: no `state` column, no wire format.
  `State.Digest` remains and hashes fields directly; it is **not** stable across
  builds and must never be persisted.

Correctness on the live path (all unconditional, no flag, all deployed):

- **A move is atomic** (step 4, #2053). Turn rows and the games row commit in
  one transaction, so a reader can never see one without the other.
- **A load is one snapshot** (step 3, #2029). `GetGameWithTurns` returns the
  games row and the event log in a single statement.
- **Writers serialize across app servers** (step 5, #2065). `Cache.LockGame` is
  a session-level advisory lock, held across load-modify-save, with holders
  capped at half the pool.
- **The in-memory game cache is gone** (step 6, #2091), and with it the game
  object's mutex.
- **`UpdateGame` no longer clobbers `ready_flag`** (step 1, #2026).
- **`games.uuid` is unique** (step 1b, #2027).
- **Archival cannot delete turn rows that changed under it**, and runs without
  the game's lock (#2146). A game with fewer rows than events is refused
  (`archive-missing-turns`) rather than archived from the bytea. The pending
  sweep runs hourly (#2162).
- **The adjudicator decides from rows** rather than loading every live game
  every 10 seconds (#2159).
- **A challenge is refused unless the last event is a tile placement** (#2163,
  deployed 2026-11-02). It used to be gated only in the UI.

Not carried over from #1975: automod idempotency (`automod_verdicts`), which
became issue #2036. See `pr1975_split.md`, "Deliberately dropped".

Evidence-gathering, behind flags, serving nothing:

- `-shadow-turns` — after each save, rebuild from the rows just written and diff
  the whole position against the live game. On in production.
- `-shadow-turns-load` — on every load of a live game, rebuild from `game_turns`
  and diff against what is being served. On in production.
- `-write-ongoing-games` — maintain the `ongoing_games` row inside the save
  transaction (step 7). Off; nothing reads the table yet.

## What is left, in order

- [x] **Shadows quiet** for a week including a weekend, counted from #2163's
      deploy on 2026-11-02: zero mismatches through 2026-11-11.
- [ ] **Phases 1 to 3** — xwordgame decides live moves next to macondo, then
      serves, then macondo leaves the live path. They are in
      `pr1975_split.md`, "After the split", and the work items in #2037.
      Serving a position rebuilt from turns is phase 2, not a step before it:
      `DBStore.Get` returns an `entity.Game` embedding macondo's game and
      `StateFromTurns` returns an xwordgame position, so the two switch
      together. Do **not** revive the commented-out `buildHistoryFromTurns`,
      which is kept only as the record of what went wrong. Roll out
      correspondence games **last**: a bad real-time game is over in twenty
      minutes, a bad correspondence game carries a corrupt position for weeks.
- [ ] **When macondo leaves the live path,** only xwordbridge's comparison
      scaffolding goes: `StateFromGame`, `Compare` and the `quirks.go`
      exclusions. `ReplayHistory`, `StateFromTurns`, `DecodeTurn(s)`,
      `MoveFromEvent` and `rules.go` are the turns read path; they move into
      xwordgame or a storage package. And removing macondo from `go.mod`
      depends on moving the `GameHistory` proto into liwords, because stored
      event logs and every S3 archive use macondo's `GameHistory`/`GameEvent`.
- [ ] **`game_turns.event` jsonb → `event_bin bytea`.** ~7× faster to parse, ~4×
      smaller, and **no backfill** -- turn rows are deleted at archival. Write
      both for one release, read `event_bin` when present, then drop `event`.
      `DecodeTurn` already accepts either encoding. Add `cmd/decode-turn` so a
      row can still be read during an incident.
- [ ] **Move the listings onto `ongoing_games`** -- behind
      `READ_ONGOING_GAMES` (rollout step 6). Once it has been on for a release,
      delete the `games` versions of the queries and the flag, and drop
      `idx_games_game_end_reason` if nothing else uses it.
      The older `ListOngoing*` queries in `ongoing_games.sql` still have no
      callers; delete them then too.
- [x] **Correspondence adjudication without reading `games`** -- the
      increment is in the timers (#2235) and the adjudicator pre-filters in
      SQL behind `READ_ONGOING_GAMES` (rollout step 6).
- [ ] **Time-bank updates take no game lock.** `AddTimeBank*` (league admin)
      update the timers in SQL; a move saved at the same moment writes the
      timers it loaded and drops the added time. Rare and admin-only; route
      them through the lock (or retry) if it ever matters.
- [ ] **Delete the `Cache` wrapper.** `pkg/stores/game/cache.go` is a
      pass-through to `DBStore` now, apart from the per-server `activeGames`
      listing TTL.
- [ ] **Shrink `games`** -- see above. Quickdata goes with the `game_players`
      joins.
- [ ] **Re-enable the OTel meter provider** (`cmd/liwords-api/otel.go`,
      `TODO(metrics-leak)`) with delta temporality, deny-by-default views and
      OTLP export, then watch `go.runtime.heap_alloc_bytes` for a full release
      before widening the allowlist. Until then spans are the only measurement.

## The archive contract

`archiveContractViolations` (`s3.go`) runs on every archival and logs
`archive-contract-violation` for anything a later reconstruction will need and
the object does not carry. It logs rather than blocks: refusing an archival
would leave turn rows behind and an orphaned S3 object, which is worse than a
log line for something that has never happened.

`verifyHistory` cannot cover this. It compares the turns-assembled history
against the bytea one, so it is blind to a field that *both* sides stop
carrying -- which is exactly the failure being guarded against.

What is required, and why each one, measured across 6,031 finished games:

| field | read by | if it goes missing |
|---|---|---|
| `PlayState` | `applyStoredEnding` | a finished game reconstructs as in progress -- the May 2026 bug |
| `FinalScores` | `applyStoredEnding` | the second witness that a game ended; `PlayState` becomes a single point of failure |
| `LastKnownRacks` | `assignFinalRacks` | racks cannot be restored, so the bag is wrong too |
| end-rack `Rack` | `sameRack` | not the position -- the *check* that would catch a wrong one |

`Winner` is deliberately not checked: nothing in `pkg/xwordbridge` reads it. It
matters because it is not derivable for adjudicated and forfeited games, but
losing it costs the outcome, not the position.

Aborted and cancelled games are exempt from `FinalScores` -- they never produced
a score. All 21 exceptions in the corpus are aborts.

## Guard rails to keep

- `verifyHistory` (`s3.go:198-214`) deep-compares the turns-assembled history
  against the bytea history at archival and refuses to archive on a mismatch.
  That is already a two-record cross-check; keep it until the bytea column goes
  away, and treat any `archive-verify-mismatch` as a stop signal.
- Keep `history` bytea written throughout. It is the fallback and the reference.

## Watch: WordSmog word validation

macondo v0.13.5 changed how WordSmog decides validity. It no longer asks the
gaddag for an anagram; it loads an *alpha dawg* — a word graph of alphagrams,
the `.kad` files in `data/lexica/gaddag` — and validates and generates cross
sets from that.

`xwordgame.Lexicon` still asks for `HasAnagram`, which a plain `kwg.Lexicon`
answers by walking the ordinary graph. The two agreed on all 226 verdicts the
wordsmog differential produced against v0.13.5, so nothing is known to be wrong.
But they are now different structures answering the same question, and agreement
on 226 random tile arrangements is not agreement in general.

- [ ] for WordSmog games, hand `Rules.Lexicon` macondo's alphadawg-backed
      lexicon rather than a plain KWG, so both engines consult the same
      structure. The interface already allows it; only the caller changes.
- [ ] widen the wordsmog differential, which currently lands almost no accepted
      plays because random tiles are rarely anagrams of anything.

## Rules-era drift

Three classes of old game were played under rules that no longer exist. All
three load correctly, because the recorded outcome is read rather than
recomputed; they are recorded here so nobody mistakes them for bugs.

- Six scoreless turns counting a **zero-point tile placement** as scoreless.
  macondo today resets the counter on any placement (`game.go:576-580`, "no
  international rule counts a score of 0 as a scoreless turn if it's from tiles
  being played on the board"). 3 games in the corpus, reported as
  `EndRackGameNotOver`; e.g. `TzeCLa8q` (2021-10-26), which our path loads as
  `GAME_OVER [61 29]` — matching its own recorded final scores — where macondo
  loads it as `PLAYING` with player 1 on turn.
- **One-letter opening plays**, accepted by Woogles in early 2021.
- **Since-delisted words**, e.g. a 2021 ECWL game containing `SEZ`.

`Rules.TrustRecordedPlays` is what waives re-deciding these. It must never be
set for live play. `XWORDGAME_GAME=<id> go test ./pkg/xwordbridge/ -run
TestInspectGame -v` prints one game through every path, which is the tool for
answering "would this game load correctly".

## Improvements worth taking while nearby

- [ ] have macondo emit an event for a triple-challenge ending. It currently
      writes nothing at all, so the log cannot be told from a game in progress.
      Cosmetic now that the stored fields are read, but it would make the log
      self-describing for GCG export and third-party replay.
- [ ] fuse `crossWordScore` and `formedCrossWord`, which walk the same geometry
      twice when `ScoreWords` is called.
- [ ] macondo's `PlayTurn` (v0.13.5) differs from its live `playMove` in two
      ways that matter to the shadows: it increments per-player turn counts
      only in the exchange branch, and it sets the last play's words only for
      tile placements, so after a pass or an exchange a rebuilt game still
      holds the previous play's words. The shadows exclude both, and #2163
      makes the stale words unreachable. Fixing `PlayTurn` upstream would let
      the comparison tighten; it stops mattering in phase 3.

## Unrelated bugs found along the way

Not part of this work; recorded so they are not lost.

- [ ] **League standings have no durable protection against double-counting.**
      *Automod had the identical hole and has been fixed; copy that shape.*
      `automod_verdicts` records the verdict per (player, game) before anything
      is applied, with the effects skipped when the row already existed, and
      GOOD verdicts are recorded too because a good game decrements. Standings
      want the same: a row per counted game, and the increment conditional on
      inserting it.
      `UpdateStandingsIncremental` (`pkg/league/standings.go:509`) applies
      deltas -- `+1` win, `+1` loss, `+spread` -- so running it twice for one
      game gives a player two wins. The guard against that is
      `entity.Game.LeagueStandingsProcessed`, an in-memory bool set at
      `pkg/league/standings_updater.go:128` and checked at `:96`.

      It is never persisted, and since the game cache was removed every load
      returns a fresh object with the flag `false`. (League games are
      correspondence games, which were never cached, so for them it was always
      so.) The guard is inert.

      What actually prevents double-counting today is the callers of
      `performEndgameDuties`, all of which check persisted state:
      `AdjudicateGame`/`ForfeitGame`/`AbortGame` check
      `GameEndReason != NONE`, `TimedOut` checks `Playing() == GAME_OVER`, and
      the move path rejects moves on finished games. `performEndgameDuties`
      itself has no guard (`pkg/gameplay/end.go:79-87`) -- it sets the end
      reason if unset and runs everything unconditionally. So the protection is
      a property of who calls it rather than of the operation, and any new path
      that reaches it twice double-counts silently.

      Fix: make it idempotent in the database. A row per counted game with a
      unique constraint on the game id, with the increment conditional on
      inserting that row, inside the same transaction. Then delete
      `LeagueStandingsProcessed` rather than leave something that reads as
      protection and is not.

      Still open on 2026-11-02.

- [x] `pkg/league/force_finish_games.go` never called `ArchiveAndCleanup`, so
      league-adjudicated games never reached S3 and their `game_turns` rows
      leaked. It goes through `gameplay.AdjudicateGame` now, and the hourly
      pending-archival sweep picked up the ones left behind. On 2026-11-02 no
      finished game has turn rows.
- [x] 3 games were archived but still had turn rows. None do on 2026-11-02.
- [ ] macondo cannot open games containing since-delisted words. Nine in a
      113,010 sample.

## Much later

- [ ] move the annotator off `pkg/cwgame` and onto xwordgame. `FivePointMode`
      and `LegacyFivePointMode` exist so its scoring can be reproduced exactly.
- [ ] retire `pkg/cwgame`.

## Known limits, not to be mistaken for todos

- Replaying an *archived* game cannot recompute an end-of-game rack adjustment
  when the triggering move changed the rack — the tiles drawn in a final
  exchange are revealed only by the end-rack event that arrives after the
  penalty is due. 10 games in 113,010, reported as `EndRackRackUnknown`. The
  recorded value is read instead, and `EndRackArithmeticWrong` fails the test if
  we ever knew the rack and still got the number wrong. It is zero. Live play
  has no such gap.
- Mid-game replay has partly-guessed opponent racks between their moves. macondo
  has the identical limitation.
- One 2020 game's archived record contradicts itself: the history says `PLAYING`
  and the games row says it ended on a triple challenge. No replayer can
  reconcile that. `game_end_reason` is now read, so the games row wins.
