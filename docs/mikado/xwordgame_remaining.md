# xwordgame: what is left

Companion to `xwordgame_review.md`, which describes what exists. This one is the
list of what does not, written to be picked up cold.

**Where things stand:** the referee and the replay are built and tested against
113,010 real games. Two shadow modes are wired into the live path, both behind
flags that default off; with them off, no behaviour changes at all. Nothing is
served from the new path yet.

## Deploying this branch

**Measured against production on 2026-09-02**, because most of the decisions
below depend on facts rather than intentions:

| | |
|---|---|
| prod schema version | `202607300001` |
| `ongoing_games` | **does not exist in production**; the migration has never run |
| `game_turns` | live since 2026-05-06; 30,861 rows over 1,876 games right now |
| `games` | 12,696,909 rows |
| games with real meta events | **2.61%** (sampled), at most 10 per game |
| `games.meta_events` total | 327 MB, of which ~97% is an empty `{"events":null}` |

### Already done, and worth knowing why

`202608040001_ongoing_games` had never been applied anywhere but local test
databases -- production is on `202607300001` -- so the columns that held a game
position were removed from the migration rather than created and later dropped.
`play_state` and `on_turn` stayed: they are not snapshot mirrors, they are what
lets the active-game listings filter a small table instead of scanning 12M rows.
Check any staging database before assuming the same is still true there.

### Deploy order

Each stage is separately revertible, and nothing below serves a reconstructed
position to a player. Do not collapse stages 3 and 4.

1. **Migrations + automod + the ready_flag fix.** Three migrations run:
   `ongoing_games` (now without any position in it), `automod_verdicts`, and the
   `notoriousgames` unique index (a no-op -- production already has it, applied
   by hand). With them, the automod split, filing notoriety after the save
   rather than before, and taking `ready_flag` out of `UpdateGame`. All
   independent of the rest and worth having on their own: between them they
   close a hole that fired six times in five years, four of them three weeks
   ago, and a bug that made a player ready up twice.
   *Watch:* `automod-already-applied` at DEBUG -- a steady trickle means
   something is still judging games twice and the guard is now catching it.

2. **Shadows on, serving nothing.** `-shadow-turns` and `-shadow-turns-load`.
   Both read-only, both goroutines, neither can change what a player sees. Turn
   them on early: they cost nothing and every day they run is evidence.
   *Watch:* `shadow-load-mismatch` and `shadow-turns-mismatch` at ERROR (each
   line names every disagreeing field), `shadow-load-torn`, and
   `shadow-load-count-mismatch`.

3. **The write path: one transaction per move.** Turn rows and the games row now
   commit together. **Behaviour change:** a `game_turns` write failure now fails
   the move, where it used to be logged and swallowed. That is correct -- the
   alternative is a permanently inconsistent log -- but it is player-visible.
   *Watch:* move error rates, and `dual-write-turns-error` disappearing in
   favour of failed moves rather than in addition to them.

4. **The lock.** `Cache.LockGame` becomes a session-level advisory lock. This is
   the riskiest single change on the branch: it puts a database round trip on
   the hot path of every move, pins a pooled connection for the duration, and
   introduces a failure mode that did not exist (`ErrGameLockBusy` after 10s).
   Deploy it alone.
   *Watch:* `ErrGameLockBusy`, `game-unlock-failed` (never thinned -- a lock
   that leaks blocks that game for the life of the process), pool saturation,
   and move latency. Concurrent holders are capped at half the pool; with
   `MaxConns = 25` that is 12.

5. **Remove the cache.** Loads go from ~1.5/sec to ~12/sec.
   *Watch:* p99 load latency against the current 2.5 ms, database CPU, and the
   shadow rates from stage 2 -- which get much better coverage once every load
   is a real load.

6. **`-write-ongoing-games`,** once stage 1's migration has created the table.
   The row is written inside the save transaction, so a failure fails the move;
   that is the price of the row never disagreeing with the game it names.
   Nothing reads it, so this is purely so the table can be checked before the
   listings are moved onto it.
   *Watch:* move error rates again, and `SELECT count(*) FROM ongoing_games`
   against the number of games `games` says are unfinished. They should track.

### Then stop, and let it run

**Do not serve a reconstructed position until the shadows have been quiet for a
sustained period on real traffic.** Inferring position from an event log without
verifying it is exactly what corrupted 944 games in May 2026. The corpus says
113,009 of 113,010 archived games reconstruct exactly; the shadows are what say
the same of live ones, and they are cheap enough to leave on for as long as it
takes.

A useful bar before enabling the read path: zero `shadow-load-mismatch` across
at least one full week including a weekend, and `shadow-load-torn` at a rate
that is explainable rather than merely low.

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

Separate from the column work, and independent of everything else on the
branch: `games` carries about **2.4 GB of index the planner has essentially
never chosen.** The statistics are trustworthy -- `pg_stat_database.stats_reset`
is 2026-03-03 and the server has not restarted since, so these counts cover 184
uninterrupted days.

The cost is not mainly disk. This is the table rewritten on every move, so every
index is maintained on every non-HOT update: it is write amplification on the
hottest path in the application, and it inflates the backup that already takes
~2 hours.

| index | size | scans in 184d | verdict |
|---|---|---|---|
| `rematch_req_idx` | 534 MB | 0 | **dead** -- see below |
| `idx_games_player0_filtered` | 517 MB | 0 | **dead** -- see below |
| `idx_games_player1_filtered` | 516 MB | 0 | **dead** -- see below |
| `idx_games_lexicon_created_at` | 562 MB | 10 | unexplained; find the query first |
| `idx_games_deleted_at` | 126 MB | 3 | see below -- probably droppable outright |
| `hastybot_games_index` | 118 MB | 0 | ad-hoc; `player_id = 230` is hard-coded into it |

None is unique, none is a primary key, and none backs a constraint, so the scan
count is the whole story -- there is no second job any of them is quietly doing.

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

**`hastybot_games_index`** has a user id baked into its predicate
(`player0_id = 230 OR player1_id = 230`). That is an index built for one
investigation; it should not outlive it.

Do them one at a time with `DROP INDEX CONCURRENTLY`, and confirm the intent of
the two with non-zero scans before touching those -- 10 scans in six months is a
rare report, not nothing. Dropping is reversible, but rebuilding a 500 MB index
on a live 12M-row table takes minutes, so it is not free.

**Already done:** `idx_games_history_s3_key_pending` was **344 MB holding 28,781
entries** -- it was built when every game matched `history_s3_key IS NULL` and
the backfill then deleted 12.6M entries from it, which plain VACUUM makes
reusable but never returns. Reindexed to **1136 kB**. Worth remembering as a
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

Referee and replay:

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

Correctness on the live path (all unconditional, no flag):

- **A move is atomic.** Turn rows and the games row commit in one transaction,
  so a reader can never see one without the other.
- **A load is one snapshot.** `GetGameWithTurns` returns the games row and the
  event log in a single statement.
- **Writers serialize across app servers.** `Cache.LockGame` is a session-level
  advisory lock, held across load-modify-save, with holders capped at half the
  pool.
- **The in-memory game cache is gone.**
- **Automod is idempotent and correctly ordered** — `automod_verdicts`, and
  notoriety filed after the save rather than before.
- **`UpdateGame` no longer clobbers `ready_flag`.**
- **Archival cannot delete turn rows that changed under it.**

Evidence-gathering, behind flags, serving nothing:

- `-shadow-turns` — after each save, rebuild from the rows just written and diff
  the whole position against the live game.
- `-shadow-turns-load` — on every load of a live game, rebuild from `game_turns`
  and diff against what is being served. Reports the AppendTurns/Set torn read
  as a torn read.
- `-write-ongoing-games` — maintain the `ongoing_games` row inside the save
  transaction. Nothing reads the table yet.

## What is left, in order

The deploy sequence and the stop point are at the top of this document. What
follows is the work after that.

- [ ] **Serve from turns.** Only once the shadows have been quiet -- see the bar
      above. Enable the branch in `DBStore.Get` (marked with a comment saying
      not yet), using `xwordbridge.StateFromTurns`; do **not** revive the
      commented-out `buildHistoryFromTurns`, which is kept only as the record of
      what went wrong. Keep the bytea fallback. Roll out correspondence games
      **last**: a bad real-time game is over in twenty minutes, a bad
      correspondence game carries a corrupt position for weeks.
- [ ] **`game_turns.event` jsonb → `event_bin bytea`.** ~7× faster to parse, ~4×
      smaller, and **no backfill** -- turn rows are deleted at archival. Write
      both for one release, read `event_bin` when present, then drop `event`.
      `DecodeTurn` already accepts either encoding. Add `cmd/decode-turn` so a
      row can still be read during an incident.
- [ ] **Move the listings onto `ongoing_games`.** The table has a writer and no
      reader; the listing queries have no Go callers. Populate it in production
      first, check it against the `games` filters, then switch the readers.
- [ ] **Shrink `games`** -- see below.
- [ ] **Re-enable the OTel meter provider** (`cmd/liwords-api/otel.go:63-70`)
      with delta temporality, deny-by-default views and OTLP export, then watch
      `go.runtime.heap_alloc_bytes` for a full release before widening the
      allowlist. Cumulative temporality retaining unbounded `http.route` values
      is the leading hypothesis for the leak, not a reproduction.

## Blockers

- [x] **Cutover contract, as a check not prose. Done.**
      `archiveContractViolations` (`s3.go`) runs on every archival and logs
      `archive-contract-violation` for anything a later reconstruction will
      need and the object does not carry. It logs rather than blocks: refusing
      an archival would leave turn rows behind and an orphaned S3 object, which
      is worse than a log line for something that has never happened.

      Note that `verifyHistory` cannot cover this. It compares the
      turns-assembled history against the bytea one, so it is blind to a field
      that *both* sides stop carrying -- which is exactly the failure being
      guarded against.

      What is required, and why each one, measured across 6,031 finished games:

      | field | read by | if it goes missing |
      |---|---|---|
      | `PlayState` | `applyStoredEnding` | a finished game reconstructs as in progress -- the May 2026 bug |
      | `FinalScores` | `applyStoredEnding` | the second witness that a game ended; `PlayState` becomes a single point of failure |
      | `LastKnownRacks` | `assignFinalRacks` | racks cannot be restored, so the bag is wrong too |
      | end-rack `Rack` | `sameRack` | not the position -- the *check* that would catch a wrong one |

      `Winner` is deliberately not checked: nothing in `pkg/xwordbridge` reads
      it. It matters because it is not derivable for adjudicated and forfeited
      games, but losing it costs the outcome, not the position.

      Aborted and cancelled games are exempt from `FinalScores` -- they never
      produced a score. All 21 exceptions in the corpus are aborts.
- [ ] **`games.uuid` unique index — checked, migration written, not yet applied
      to production.** Zero duplicates and zero NULLs across all 12,692,837
      rows, so nothing needs cleaning up first. The migration
      (`202609030001_games_uuid_unique`) is idempotent and covers fresh
      databases; production wants these two by hand, because building a 705 MB
      index inline would lock out writers on a 12M-row table:

          CREATE UNIQUE INDEX CONCURRENTLY idx_games_uuid_unique ON games (uuid);
          DROP INDEX CONCURRENTLY idx_games_uuid;

      The drop is not optional housekeeping: `idx_games_uuid` is a plain btree
      on the same column, so leaving it means 1.4 GB of index doing one job.
      The partial index on `uuid` used by the pending-archival scan is a
      different thing and stays.

      Note the name is deliberately not `idx_games_uuid`: that name is taken by
      the non-unique index from the initial migration, so reusing it with
      `IF NOT EXISTS` would skip on a fresh database and leave it without the
      constraint.

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
- [ ] macondo's `PlayTurn` increments per-player turn counts only in the
      exchange branch, so a reconstructed game reports a turn count equal to its
      number of exchanges. `DivergencesVsReconstruction` excludes the field;
      fixing it upstream would let the comparison tighten.

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

      It is never persisted, and for league games it is never even *reused*:
      league season games are created with `GameMode_CORRESPONDENCE`
      (`pkg/league/season_start.go:331`), and correspondence games are
      deliberately not cached (`pkg/stores/game/cache.go:218`), so every load
      returns a fresh object with the flag `false`. The guard is inert for
      exactly the games it was written for.

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

      Not a cache-removal blocker: these games already live in the no-cache
      world, so removing the cache changes nothing here.

- [ ] `pkg/league/force_finish_games.go` never calls `ArchiveAndCleanup`, so
      league-adjudicated games never reach S3 and their `game_turns` rows leak.
      27 of 71 adjudicated games. Forfeits are 93,021/93,021 clean, because
      `ForfeitGame` goes through `performEndgameDuties` and this does not.
- [ ] 3 games are archived but still have turn rows, which `CommitArchival`
      should make impossible — probably a backfill using `SetHistoryS3Key`.
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
