# Splitting PR #1975

PR #1975 (`xwordgame-referee-state`) is ~15k lines across 86 files. It mixes
inert new packages, flag-gated evidence gathering, and several unconditional
changes to the move path. Shipped as one PR, the deploy order in
`xwordgame_remaining.md` cannot be followed: the transaction, the lock and the
cache removal have no flags, so they would all go live at once.

This document is the plan for landing it as a stack of PRs, one deployable and
revertible stage each, and the record of how each step was checked. It also
holds what comes after the split: putting xwordgame on the live path, in
three phases (see "After the split").

**Reference:** branch `xwordgame-referee-state` at `08da29c33` (master merged
in, migrations renumbered). Until the split is finished, that branch is the
source of truth and must not change. If it has to, redo the ledger below.

## The invariant: nothing is lost

The stack is linear: each split branch starts from the one before it. At every
step:

1. Every file the step touches is in one of three states, checked by
   `ledger.sh` (below): **= master**, **= reference**, or **intermediate**.
   An intermediate file is one whose content is neither, because only some of
   its hunks have landed. Every intermediate is listed in this document with
   the reason, and each one has a step that turns it into **= reference**.
2. It builds, vets, and the affected packages' tests pass against a real
   Postgres.
3. Generated code (`pkg/stores/models`) comes from `go generate ./...`, never
   hand-edited. The generator image now has a newer sqlc than the reference
   was built with, so regenerated files carry a newer `// sqlc vX` header;
   that is fine. Generated files a step does not change are reverted, so each
   PR's diff is only its own. The ledger ignores the header line.

After the last step, the tree of 7 with 1b merged in equals the reference
tree, except for migration filenames (see below), this document, and the
files listed under "Deliberate deviations". That final diff is what
proves nothing was dropped.

## Migration numbering

golang-migrate only applies versions above the database's current one, and
skips anything numbered lower **without an error**. So each migration's
number is assigned when its PR is opened, in deploy order, and re-checked
when it merges. If anything with a higher number has merged in the meantime,
renumber it.

The reference branch numbers `ongoing_games` lowest (`202610090001`), but it
ships last. Kept as-is, it would be skipped.

Also: PR #2023 (`202610080001_drop_analysis_result`) must be deployed before
any of these, or it will be skipped.

## The stack

| # | Branch | What | Behaviour change | Flags |
|---|---|---|---|---|
| 1 | `referee-split-1-automod` | `ready_flag` out of `UpdateGame`; `notoriousgames` unique index migration (already in production) | yes, small: fixes the ready-up bug | none |
| 1b | `referee-split-1b-games-uuid` | `games.uuid` unique index migration | none **if** the manual step ran first | none |
| 2 | `referee-split-2-referee` | `pkg/xwordgame`, `pkg/xwordbridge` | none; nothing calls them | none |
| 3 | `referee-split-3-shadows` | rewritten `-shadow-turns`, new `-shadow-turns-load`, `GetGameWithTurns`, archive-contract log, `cmd/profile-game-load` | none with flags off | `SHADOW_TURNS`, `SHADOW_TURNS_LOAD` |
| 4 | `referee-split-4-atomic-save` | `StageTurns`; turn rows and the games row in one transaction; `CommitArchival` count check | **yes**: a failed `game_turns` write fails the move | none |
| 5 | `referee-split-5-session-lock` | session advisory lock replaces the in-process mutex map | **yes**: DB round trip per move, holds a pool connection, `ErrGameLockBusy` | none |
| 6 | `referee-split-6-no-cache` | remove the game LRU cache; automod `Classify` refactor and its tests (they replace a test that needs the cache) | **yes**: every load hits the DB | none |
| 7 | `referee-split-7-ongoing-games` | `ongoing_games` migration, queries, writer; mikado docs | none with the flag off | `WRITE_ONGOING_GAMES` |

Step 1 was cut down on 2026-10-10. It used to carry #1975's automod changes;
see "Deliberately dropped" below for what went where and why. The branch
name still says `automod` because renaming it would detach PR #2026.

1b branches from master, not from 1, so the manual step never blocks the rest
of the stack. It can merge whenever that step is done (renumbering first if
anything higher has merged). Everything else is a linear stack: 2 on 1, 3 on
2, and so on.

### File map

Whole files move to the reference version in exactly one step. Shared files
are split by hunk; the step that finishes a file is in **bold**.

| File | Steps |
|---|---|
| `.gitignore` | **1** |
| `pkg/mod/automod.go` | **6**, without the verdict guard (deviation) |
| `pkg/mod/classify_test.go`, `pkg/mod/automod_accumulation_test.go` | **6** |
| `pkg/mod/automod_test.go` (deleted in reference) | **deleted in 6** |
| `db/migrations/*_notoriousgames_unique.*` | **1**, comment rewritten (deviation) |
| `db/migrations/*_automod_verdicts.*`, `db/queries/mod.sql`, `pkg/stores/mod/db.go`, `pkg/stores/models/mod.sql.go`, `pkg/mod/automod_apply_test.go` | never: dropped, see below |
| `db/migrations/*_games_uuid_unique.*` | **1b** |
| `db/queries/games.sql` | 1 (`ready_flag`), **3** (`GetGameWithTurns`) |
| `pkg/gameplay/end.go` | 4 (`StageTurns`); the automod-after-save hunks are dropped; **5** (`AdjudicateGame` and `ForfeitGame` lock, then load: deviation) |
| `pkg/stores/game/db.go` | 1 (`ReadyFlag` param), 3 (shadows, `gameFromRow`, `GetHistory` comment), 4 (`StageTurns`/`AppendTurns`, transaction in `Set`, `CommitArchival`), 5 (`lockSlots`), **7** (`syncOngoingGame` in `Set`) |
| `go.mod` (xxhash direct) | **2** |
| `pkg/xwordgame/*`, `pkg/xwordbridge/*` | **2** |
| `docs/mikado/referee_gap_audit.md`, `docs/mikado/xwordgame_review.md` | **2** |
| `pkg/config/config.go` | 3 (`ShadowTurnsLoad`), **7** (`WriteOngoingGames`) |
| `pkg/stores/game/s3.go` | 3 (contract check), **4** (`CommitArchival` count) |
| `pkg/stores/game/shadowload_test.go`, `archivecontract_test.go` | 3 (shadowload is intermediate), **4** |
| `cmd/profile-game-load/main.go` | **3** |
| `pkg/gameplay/game.go` | 3 (shadow call site, intermediate), 4 (`StageTurns`, shadow call removed), **5** (`LockGame`) |
| `pkg/entity/game.go` | **4** |
| `db/queries/game_turns.sql`, `pkg/stores/game/tx.go`, `atomicity_test.go` | **4** |
| `pkg/stores/game/cache.go` | 4 (`StageTurns`, `CommitArchival`, interface), 5 (`LockGame`), **6** (LRU removed) |
| `db/queries/game_lock.sql`, `pkg/stores/game/lock.go`, `lock_test.go` | **5** |
| `pkg/bus/gameplay.go`, `pkg/gameplay/meta_events.go` | **5**; `bus/gameplay.go` also reloads before the never-started abort (deviation) |
| `pkg/league/force_finish_games.go`, `pkg/league/service.go`, `pkg/gameplay/end_lock_test.go` | **5** (deviation; the reference never touches them) |
| `cmd/liwords-api/main.go` (gameCacheSize expvar) | **6** |
| `pkg/gameplay/common_test.go`, `game_playing_test.go`, `meta_events_test.go` | **6** |
| `db/migrations/*_ongoing_games.*`, `db/queries/ongoing_games.sql`, `pkg/stores/game/ongoing.go`, `ongoing_test.go` | **7** |
| `docs/mikado/xwordgame_remaining.md`, `remove-game-caches.dot`, `.svg` | **7** |
| `pkg/stores/models/*` | regenerated at every step |

### Intermediate code that exists only in the split

Code that is in neither master nor the reference branch. Each item has a step
that removes it.

- **Step 3: `SpawnShadowCompare` is still called from `PlayMove`**, where
  master calls it, after the immediate `AppendTurns` and skipping finished
  games. `pkg/gameplay/game.go` is untouched in step 3. Only the function body
  is the reference's; its doc comment (db.go hunk 7, "called by Set") waits for
  step 4, where the call moves into `Set` after the transaction commits.
- **Step 3: `shadowload_test.go` writes turns with `AppendTurns`** instead of
  `StageTurns` + `Set`, two lines, each marked `Split step 3`. Replaced with
  the reference version in 4.
- **Step 4: `Set`'s comment on serialization stops at "callers hold the
  game's lock across load, move and save".** The reference names the session
  lock and `lock.go`, which arrive in step 5; the rest of that comment comes
  back with them. (Done in step 5.) `tx.go` is the reference file verbatim, so its header
  already describes step 5's session lock.
- **Step 4: `Cache.setOrCreate` drops the game from the LRU when a save
  fails.** Step 4 stages turn rows on the game object and `Set` writes them.
  If a commit succeeds but its acknowledgement is lost, the cached object
  still has the rows staged, and every later save collides with them on
  `game_turns_pkey`, so the game can't be moved until it leaves the cache.
  Dropping it makes the next request load what actually committed.
  `cache_failedsave_test.go` fakes the lost acknowledgement; without the line,
  the second and third saves fail with the duplicate key. Step 6 deletes the
  LRU and the line with it. The test stays, because it checks the same thing
  without a cache.
- **Step 5: the game LRU is still there, beside the session lock.** The
  reference removes both the in-process lock map and the LRU in one commit;
  step 5 removes only the lock map and points `Cache.LockGame` at the
  database lock. The cache's own comment still describes caching. Step 6
  removes the LRU.
- **Step 3: `pkg/config/config.go` has `ShadowTurnsLoad` but not
  `WriteOngoingGames`.** The reference adds both in the same hunks; the second
  lands with `ongoing_games` in 7.

### Coverage the reference branch dropped

The replacement automod tests (step 6) test `Classify` and the notoriety
accumulation directly. Two things the deleted end-to-end test checked are not
covered by them:

- casual (unrated) games never reach automod. The gate is in
  `pkg/gameplay/end.go`.
- `performEndgameDuties` calls automod at all.

Before 6 merges, add a `pkg/gameplay` test for both, or accept the gap
explicitly.

## Deliberate deviations

Everything here differs from the reference on purpose. Each is its own commit,
each file is listed in `ledger.sh` (as `DEVIATION` or `DROPPED`), and together
they explain the whole final diff against the reference.

### Deliberately dropped: automod idempotency (issue #2036)

#1975 carried automod changes that have nothing to do with the game-storage
redesign. They came in because one test, `pkg/mod` `TestNotoriety`, blocked
the cache removal: it forced a player to be on turn and relied on every store
call sharing one cached `*entity.Game`. Rewriting it led to `automod_verdicts`,
a table recording a verdict for every (player, game) automod judges, GOOD
verdicts included, so automod could not apply twice. That was a reaction to
2026-08-07, when the end-of-game sequence ran twice for several games and
automod filed the same game twice against a player.

Dropped, because:

- **It guards the wrong layer.** If the end-of-game sequence runs twice,
  ratings, stats, tournament results and league standings run twice too. A
  guard table per side effect does not scale. The real fix is to make the
  end of a game exactly-once, with a transactional outbox: issue #2036.
- **The part that matters is already covered.** Production has had a unique
  `(player_id, game_id)` index on `notoriousgames` since it was built by
  hand, and `updateNotoriety` files that row before it touches the score, so
  a second run cannot penalise twice. It can only forgive twice for a good
  game.
- **It is expensive for what is left.** About 75k rows a week, roughly 85% of
  them GOOD verdicts for players already at zero notoriety, which a second
  run could not change anyway.

What went where:

| #1975 change | Now |
|---|---|
| `notoriousgames` unique index migration | step 1; comment rewritten, since it referred to `automod_verdicts` |
| `ready_flag` out of `UpdateGame` | step 1 (unrelated bug, small) |
| `.gitignore` additions | step 1 |
| `automod_verdicts` table, queries, `RecordAutomodVerdict`, `automod_apply_test.go` | dropped (#2036) |
| automod moved after the save (`end.go` hunks 3, 4) | dropped (#2036) |
| `Classify` split out of `Automod`, `classify_test.go`, `automod_accumulation_test.go` | step 6, without the verdict guard. That is what replaces `TestNotoriety`, which stops working without the cache |
| `pkg/mod/automod_test.go` deleted | step 6 |
| `models.go` `AutomodVerdict` struct | never generated |

### How step 6 keeps automod correct without the cache and without the table

1. **`TestNotoriety` is replaced, not fixed.** Bring #1975's `Classify`
   refactor and its two test files, minus `RecordAutomodVerdict` (the
   `NotorietyStore` method and the guard at the top of `updateNotoriety`).
   Neither test file uses the guard. Then delete `automod_test.go`.
2. **Ending a game twice stays prevented, by the lock.** With the cache gone,
   the protection is the game lock held from load to save (step 5): a second
   `TimedOut` waits for the first, reloads, and sees the game ended. That
   only holds for entry points that take `LockGame` *before* loading. Three do
   not; step 5 fixes them (see "End-of-game entry points lock, then load"):
   - `bus.adjudicateGames`' abort branch loads the game, then locks;
   - `league` `ForfeitGame` callers (`pkg/league/service.go`) and
     `force_finish_games.go` load the game and take only the in-memory
     `g.Lock()`, which protects nothing once each load is its own copy.
   The simple fix is one helper that takes `LockGame`, loads the game inside
   the lock, and returns early if `game_end_reason` is already set, used by
   all of them.
3. **Backstop:** the `notoriousgames` unique index from step 1 still stops a
   double penalty if something gets through.

### `SHADOW_TURNS` false mismatches on `turns[]` (step 3)

Found by running the `pkg/gameplay` suite with `DUAL_WRITE_TURNS`,
`SHADOW_TURNS` and `SHADOW_TURNS_LOAD` on, against master, step 3 and the
reference. The reference produced `from_turns.turns[0]=1 live.turns[0]=0`
in `TestDoubleChallengeGoodWord` and `TestQuickdata`.

Cause, confirmed in macondo v0.13.5: live play (`playMove`,
`game/game.go:582` and `:644`) increments a player's turn count for
placements and passes, while replay (`PlayTurn`, `:1144`, which
`NewFromHistory` uses) increments it only for exchanges. Any game macondo
rebuilt from history undercounts. That is every correspondence game now, and
every game after step 6. `SpawnShadowCompare` diffed with `CompareStates`,
which excludes nothing, on the assumption that it was comparing against a
game played live. Nothing in liwords reads the count (`TurnsForNick` has no
callers).

Fix: `shadowCompareTurns` drops `turns[*]` and compares everything else.
It does not reuse `DivergencesVsReconstruction`, which also waives fields
live play does maintain. `SHADOW_TURNS_LOAD` already excluded the field.
New test `shadowturns_test.go` plays moves live, reloads the game from
history, and checks the shadow is quiet. Its precondition asserts the counts
really differ, so it fails loudly once macondo is fixed and the exclusion
can go. Mutation-checked: without the exclusion it fails with exactly the
reference's `turns[0]`/`turns[1]` mismatch and no other field.

Deviation from the reference: the exclusion and one comment in `db.go`, plus
the new test file. These carry through to the top of the stack.

### Shadow summary counts (after step 3)

Added after step 3 shipped, in its own PR. A shadow comparison that agrees
logs only at debug level, which production does not emit, so a week with no
mismatch lines could not be told apart from a week in which no comparison
ran. Each shadow now counts its outcomes (ok, mismatch, torn, more or fewer
rows than events, no turn rows, skipped, errored), and the API logs one
`shadow-summary` line per enabled shadow per hour, at info level, with the
counts for that hour.

The same PR fixes a false `shadow-turns-mismatch: more rows than events`.
`SHADOW_TURNS` snapshots the game on the caller's goroutine and reads
`game_turns` in its own, so the next move can commit its rows in between.
Every one of the 13 such lines in production in the week to 2026-10-11 was a
real-time game against a bot, exactly one row over. The reference has the
same race, and step 4 makes it more frequent by running the shadow on every
`Set`, game creation included. Turn rows are only inserted while a game is
live, so the shadow now compares the first `wantEventCount` rows, the ones
its snapshot was taken against. Extra rows the history really lacks are left
to `SHADOW_TURNS_LOAD`, which reads the games row and the turns in one
statement. Test: `TestShadowTurnsComparesSnapshotWhenGameMovedOn`, which fails
without the fix.

Deviation from the reference: `pkg/stores/game/shadowsummary.go` and its test
(new), one counter line at each outcome and the moved-on prefix in `db.go`,
a count assertion and the moved-on test in `shadowturns_test.go`, and the
`RunShadowSummary` goroutine in `cmd/liwords-api/main.go`. When steps 4 and 6
bring in the reference's `db.go` and `main.go` hunks, these lines are carried
over by hand.

### End-of-game entry points lock, then load (step 5)

The reference moves every `LockGame` call to the session lock but leaves three
places that load the game first and lock it (or only its in-memory mutex)
afterwards:

- season-end force-finish (`ForceFinishUnfinishedGames`) loads each game and
  calls `AdjudicateGame`, which takes only `g.Lock()`;
- the league's forfeit of a removed player's games (`pkg/league/service.go`)
  does the same with `ForfeitGame`;
- `bus.adjudicateGames` loads a never-started game, then locks it to abort.

A correspondence game is a fresh copy on every load, so a copy read before the
lock can be out of date by the time the lock is held, and ending the game from
it saves over whatever happened in between. Once step 6 removes the cache,
real-time games are the same.

`AdjudicateGame` and `ForfeitGame` now take a game id instead of a loaded game
and go through one helper, `withLockedGame`: take the lock, load inside it,
return early if the game has already ended. Each had one caller and no tests,
so a caller can no longer hand them a stale copy. The abort branch loads the
game again inside the lock and aborts only if it is still unstarted and not
ended. `RepairForceFinishedGames` is left alone: it only touches games that
have already ended, which nothing else writes.

Test: `TestAdjudicateLoadsTheGameInsideTheLock` holds the lock, starts an
adjudication, plays a pass under the lock, then releases it. The adjudication
must wait, and the finished game must keep the pass. With the load moved above
the lock, the finished game has no moves at all. It also checks that ending a
finished game again does nothing.

Deviation from the reference: `pkg/gameplay/end.go`,
`pkg/league/force_finish_games.go`, `pkg/league/service.go`,
`pkg/bus/gameplay.go` and the new `pkg/gameplay/end_lock_test.go`.

### Fixes from reviewing xwordgame against macondo (step 2)

On 2026-10-10 xwordgame was compared, area by area, against macondo v0.13.5,
whose game rules are unchanged in current macondo. The rules match for every
path liwords uses. Each finding below was confirmed in the source. Fixed in
step 2, each as its own commit with a regression test that fails without it:

- **A refused exchange lost the tiles.** With an exchange limit below seven
  (FISE, FILE), naming more tiles than the bag holds passed `ValidateMove`,
  and `ApplyExchange` took the tiles off the rack before the draw failed.
  `ValidateMove` now refuses it up front, as macondo does
  (`apply.go`, `apply_test.go`).
- **A successful triple challenge counted a scoreless turn in replay.**
  `ApplyReturnedPhony` could fire the six-scoreless-turns ending and deduct
  both racks from a game that never had them deducted. It now ends the game
  with the challenger winning and counts nothing, as `AdjudicateChallenge`
  and macondo do (`challenge_outcome.go`, `challenge_test.go`). This is in
  the shadow path. Not re-checked against the production corpus, which needs
  an export.
- **Docs that were wrong:** when macondo clears the challengeable words (its
  live path does, its reconstruction doesn't), how `TurnNum` moves, and what
  the "67k challenges, 0 divergences" figure proves (the replay applies
  recorded verdicts; it never decides one) (`state.go`, `quirks.go`,
  `xwordgame_review.md`).

Everything that only matters once xwordgame runs live games is in #2037. That
includes the wordsmog lexicon file (`.kwg` vs macondo's `.kad`) and
`RulesFor` ignoring a failed lexicon load. An earlier note here said those
touched the shadows; they don't, because replay never checks words.

### For step 7: the mikado graph and `xwordgame_remaining.md`

They say "delete xwordbridge" when macondo leaves the live path. Only the
scaffolding goes: `StateFromGame`, `Compare` and the `quirks.go` exclusions.
`ReplayHistory`, `StateFromTurns`, `DecodeTurn(s)`, `MoveFromEvent` and
`rules.go` are the future turns read path, and have to move into xwordgame or
a liwords storage package. Also, removing macondo from `go.mod` has to depend
on `MoveGameHistoryProto`, because stored event logs and every S3 archive use
macondo's `GameHistory`/`GameEvent` protos. And `ReadPathFromTurns` ("serve
a position rebuilt from game_turns") is drawn as a separate, earlier step than
`MigrateNativeGamesToPkgGame` (put the referee on the live path). It can't be:
`DBStore.Get` returns an `entity.Game` embedding macondo's game, and
`StateFromTurns` returns an xwordgame position. They are one step, phase 2
below; merge the nodes and replace the "what is left" list with the phases.
Correct all of this when those docs land in step 7 (a deviation; add them to
the ledger's list then).

## After the split: putting xwordgame on the live path

The point of all this is for xwordgame, not macondo, to referee live games,
with the position rebuilt from `game_turns` on every load. #1975 planned that
destination but not the route: it treated serving from turns as a step before
the referee switch (impossible while `entity.Game` embeds macondo's game),
called the remaining work "wiring it in", and had no stage where xwordgame
decides real moves before it serves them. These phases fix that. Start phase 1
once step 3's shadows have been quiet in production.

Today, 16 non-test files reach into macondo's game through `entity.Game`
(about 63 explicit call sites, mostly `History`, `Playing`, `PlayerOnTurn`,
plus promoted methods called directly), centred on `pkg/gameplay` and the
store's load and save. The work items are tracked in #2037, by phase.

### Phase 1: run xwordgame alongside macondo on live moves

Flag: `SHADOW_REFEREE`, default off. Serves nothing.

- macondo still decides and serves every move. In the background, xwordgame
  applies the same client move to the position rebuilt from turns, and the
  two results are compared: the new position, and the events each would
  write.
- This is where the live wiring gets built, exercised on real traffic before
  it serves anything:
  - parsing and scoring client moves;
  - turning `ApplyResult` into macondo-style events;
  - the position before the last play, for challenges (from the replay, with
    the racks corrected);
  - returned tiles for challenge results;
  - clock effects (the timeout fix, overtime penalty, winner after penalties).
- It covers what synthetic tests can't: real challenges under every rule,
  timeouts, exchanges and endings, decided by both engines. That closes the
  evidence gap in #2037 (challenge decisions never compared against macondo).
- Needs: step 3 (shadows, `GetGameWithTurns`). Doesn't need steps 4-6,
  because it serves nothing.
- Watch: a `referee-shadow-mismatch` log line naming every disagreeing field
  and event. Bar to move on: zero for a week including a weekend, across real
  challenges of each rule in use.

### Phase 2: xwordgame serves

Flag: `XWORDGAME_REFEREE`, default off, turned on per game mode: real-time
first, correspondence last (a bad real-time game is over in twenty minutes, a
bad correspondence game carries a corrupt position for weeks).

- Loading means `StateFromTurns`. xwordgame applies moves; phase 1's emitters
  write the events.
- Keep writing the `history` bytea throughout the rollout, assembled from the
  same events. That makes rollback instant: flip the flag off and macondo
  reloads from the bytea. Because both engines read and write the same event
  format, a game can switch engines between moves.
- Bots: liwords validates bot moves through the referee too, so they go
  through xwordgame the same way.
- Needs: step 4 (turn rows and games row in one transaction), steps 5 and 6
  (lock held across load-modify-save, no cache: every move rebuilds from
  turns, so the lock is what serializes moves).
- Watch: move error rates, `verifyHistory` / `archive-verify-mismatch` at
  archival, phase 1's shadow left on for flag-off games.

### Phase 3: take macondo off the live path

- `entity.Game` stops embedding macondo's game and holds the xwordgame
  position plus the history proto.
- Stop writing the `history` bytea (the last step of shrinking `games`; see
  `xwordgame_remaining.md`).
- Delete the bridge scaffolding (`StateFromGame`, `Compare`, `quirks.go`) and
  move its permanent half (`ReplayHistory`, `StateFromTurns`, `DecodeTurn(s)`,
  `MoveFromEvent`, `rules.go`) into xwordgame or a liwords storage package.
- Copy the `GameHistory`/`GameEvent` protos into liwords before removing
  macondo from `go.mod` (the graph's `MoveGameHistoryProto`).
- Separate, later: the annotator moves off `pkg/cwgame` onto xwordgame
  (`FivePointMode` and `LegacyFivePointMode` exist for it).

## Manual steps

- **Before 1b deploys:** production does not have the unique index (checked
  2026-10-09: `idx_games_uuid` exists, `idx_games_uuid_unique` does not). Run
  by hand, outside a transaction:

      CREATE UNIQUE INDEX CONCURRENTLY idx_games_uuid_unique ON games (uuid);
      DROP INDEX CONCURRENTLY idx_games_uuid;

  Otherwise the migration builds a 705 MB index inside a transaction and blocks
  writes to `games` while it runs.
- **Before 3's flags are turned on:** confirm `DUAL_WRITE_TURNS` is on in the
  task definition. Without it `game_turns` is empty and the shadows skip
  everything.

## Deploy and watch

Per step, from `xwordgame_remaining.md`:

1. *Watch* that both players still have to ready up only once. The
   `notoriousgames` migration is a no-op in production.
3. Turn on `SHADOW_TURNS` and `SHADOW_TURNS_LOAD`. *Watch*
   `shadow-turns-mismatch`, `shadow-load-mismatch` (ERROR, each names every
   disagreeing field), `shadow-load-torn`, `shadow-load-count-mismatch`,
   `archive-contract-violation`. Before step 4, torn reads are expected: that
   is the race step 4 fixes, and the count is the measurement of it. The
   hourly `shadow-summary` lines say how many comparisons each shadow made;
   "quiet" means mismatch 0 with ok well above 0.
4. *Watch* move error rates; `dual-write-turns-error` should turn into failed
   moves, not appear alongside them. `shadow-load-torn` should go to zero.
   Before deploying, check that no live game has more turn rows than history
   events: its next move would collide with an existing `turn_idx` and fail
   every time. Checked 2026-10-12: 2,567 live games, 2,530 exact, 37 with no
   rows, none ahead.
5. *Watch* `ErrGameLockBusy` (as `handle-event-lock-failed`,
   `timed-out-lock-failed`, `cant-lock-for-bot-move`, `cant-lock-for-abort`),
   `game-unlock-failed`, pool saturation, move latency. Season-end
   force-finish now takes the lock per game, so watch the first rollover
   after the deploy.
6. *Watch* p99 load latency, DB CPU.
7. Turn on `WRITE_ONGOING_GAMES`. *Watch* move errors, and
   `SELECT count(*) FROM ongoing_games` against unfinished games in `games`.

## ledger.sh

Run from the repo root on a split branch, with `08da29c33` (the reference) as
the default second argument. For every file that differs between
master and the reference, it prints whether this branch has the master
version, the reference version, or something else.

```bash
#!/usr/bin/env bash
# usage: ledger.sh [master-ref] [reference-ref]
# For every file that differs between master and the reference, say whether
# HEAD has the master version, the reference version, or neither.
# Migrations are matched by content, because they are renumbered.
# Generated models are compared ignoring the sqlc version header line.
m=${1:-origin/master}; r=${2:-08da29c33}

# Files the split changes on purpose, so they will never equal the reference.
# Each is explained under "Deliberate deviations" in docs/mikado/pr1975_split.md.
deviations="
pkg/stores/game/shadowturns_test.go
pkg/stores/game/shadowsummary.go
pkg/stores/game/shadowsummary_test.go
pkg/stores/game/cache_failedsave_test.go
cmd/liwords-api/main.go
pkg/stores/game/db.go
pkg/gameplay/end.go
pkg/gameplay/end_lock_test.go
pkg/bus/gameplay.go
pkg/league/force_finish_games.go
pkg/league/service.go
pkg/mod/automod.go
pkg/stores/models/models.go
db/migrations/202610090003_notoriousgames_unique.up.sql
pkg/xwordgame/apply.go
pkg/xwordgame/apply_test.go
pkg/xwordgame/challenge_outcome.go
pkg/xwordgame/challenge_test.go
pkg/xwordgame/state.go
pkg/xwordbridge/quirks.go
docs/mikado/xwordgame_review.md
"
# Reference changes that will not land at all (issue #2036). They must stay
# at master.
dropped="
db/migrations/202610090002_automod_verdicts.up.sql
db/migrations/202610090002_automod_verdicts.down.sql
db/queries/mod.sql
pkg/stores/mod/db.go
pkg/stores/models/mod.sql.go
pkg/mod/automod_apply_test.go
"
listed() { grep -qxF "$1" <<<"$2"; }
blob() { git rev-parse -q --verify "$1:$2" 2>/dev/null || echo none; }

git diff --name-only "$m" "$r" | while read -r f; do
  here=$(blob HEAD "$f"); mm=$(blob "$m" "$f"); rr=$(blob "$r" "$f")
  if [[ $f == db/migrations/* && $rr != none ]]; then
    hit=$(git ls-tree HEAD db/migrations/ | awk -v b="$rr" '$3==b{print $4}')
    [[ -n $hit ]] && { printf '%-22s %s  (as %s)\n' "= reference" "$f" "${hit##*/}"; continue; }
  fi
  if [[ $here == "$rr" ]]; then s="= reference"
  elif [[ $f == pkg/stores/models/* && $here != none ]] && diff -q \
       <(git show "HEAD:$f" | grep -v '^//   sqlc v') \
       <(git show "$r:$f" 2>/dev/null | grep -v '^//   sqlc v') >/dev/null; then
    s="= reference"   # differs only in the sqlc version header
  elif [[ $here == "$mm" ]]; then s="= master"
  else s="INTERMEDIATE"; fi
  if listed "$f" "$dropped"; then
    [[ $s == "= master" ]] && s="DROPPED" || s="DROPPED BUT PRESENT!"
  elif listed "$f" "$deviations" && [[ $s != "= reference" ]]; then
    s="DEVIATION ($s)"
  fi
  printf '%-22s %s\n' "$s" "$f"
done
# Anything HEAD changed that the reference never touched is a leak, unless it
# is a listed deviation.
git diff --name-only "$m" HEAD | while read -r f; do
  [[ $f == docs/mikado/pr1975_split.md ]] && continue
  if git diff --quiet "$m" "$r" -- "$f"; then
    if listed "$f" "$deviations"; then printf '%-22s %s\n' "DEVIATION (added)" "$f"
    elif [[ $f == db/migrations/* ]]; then
      # renumbered migration whose content matches the reference: fine
      b=$(blob HEAD "$f")
      git ls-tree -r "$r" db/migrations/ | awk -v b="$b" '$3==b{f=1} END{exit !f}' || printf '%-22s %s\n' "NOT IN REF" "$f"
    else printf '%-22s %s\n' "NOT IN REF" "$f"; fi
  fi
done
```

Migration files are matched by content rather than name, since their numbers
change (see above). `NOT IN REF` means a split branch changed a file the
reference never touched, which should never happen. `DEVIATION` marks a
file listed in the script's `deviations` variable, and `DROPPED` a reference
change listed in `dropped` that correctly stayed at master; both are explained
under "Deliberate deviations". `DROPPED BUT PRESENT!` is an error.

## hunks.py

How partial files are built: every line comes verbatim from the reference, by
applying chosen hunks of the master..reference diff with `git apply`. `list`
shows what each hunk contains. Hunks used so far, numbered as `list` prints
them against `08da29c33`:

| File | Step 1 | Step 3 | Still to land |
|---|---|---|---|
| `db/queries/games.sql` | 2 | 1 | none |
| `pkg/gameplay/end.go` | | | 1, 2 (step 4); 3, 4 dropped |
| `pkg/stores/game/db.go` | 13 | 2, 4, 5, 6, 8, 9, 10, 11, 17 | 7, 12, 14, 15, 16 (4); 1, 3 (5); 14's `syncOngoingGame` part and 12's `writeOngoing` (7) |
| `pkg/stores/game/s3.go` | | 1, 3, 5 | 2, 4 (step 4) |
| `pkg/config/config.go` | | 1, 2 minus the `WriteOngoingGames` lines | those lines (7) |

```python
#!/usr/bin/env python3
"""hunks.py list <file>            -- number the hunks of master..ref for a file
   hunks.py apply <file> 1,3,4     -- apply exactly those hunks to the worktree"""
import subprocess, sys
M, R = "origin/master", "08da29c33"
def hunks(f):
    d = subprocess.run(["git","diff","-U3",M,R,"--",f],capture_output=True,text=True,check=True).stdout
    lines = d.splitlines(keepends=True)
    i = next(k for k,l in enumerate(lines) if l.startswith("@@"))
    head, hs, cur = "".join(lines[:i]), [], []
    for l in lines[i:]:
        if l.startswith("@@") and cur: hs.append("".join(cur)); cur=[]
        cur.append(l)
    hs.append("".join(cur))
    return head, hs
cmd, f = sys.argv[1], sys.argv[2]
head, hs = hunks(f)
if cmd == "list":
    for n,h in enumerate(hs,1):
        body=h.splitlines()
        ch=[l for l in body[1:] if l[:1] in "+-"]
        print(f"#{n} {body[0][:90]}  (+{sum(l[0]=='+' for l in ch)} -{sum(l[0]=='-' for l in ch)})")
        for l in ch[:6]: print("    "+l[:110])
else:
    pick=[int(x) for x in sys.argv[3].split(",")]
    patch=head+"".join(hs[n-1] for n in pick)
    subprocess.run(["git","apply","--recount","-"],input=patch,text=True,check=True)
    print(f"applied {pick} of {len(hs)} hunks to {f}")
```

## Progress

Kept up to date at the top of the stack. `TestStandings` in
`pkg/pair/standings` also fails on master and is excluded from "full suite".

Migrations reach production when a PR merges (the build's migration task runs
them); code reaches it only when the API task definition is updated. The
Deployed column is the code.


| # | Branch | PR | Ledger checked | Tests | Merged | Deployed |
|---|---|---|---|---|---|---|
| 1 | `referee-split-1-automod` | #2026 | 2026-10-10 (re-cut) | full suite ✅ | ✅ | ✅ migration 2026-10-10; code 2026-10-12 (`liwords-api:317`) |
| 1b | `referee-split-1b-games-uuid` | #2027 | 2026-10-09 | store + gameplay ✅ | ✅ | ✅ 2026-10-10 (index built by hand first; schema `202610090004`) |
| 2 | `referee-split-2-referee` | #2028 | 2026-10-10 (review fixes) | full suite ✅ (corpus tests skip) | ✅ | ✅ 2026-10-12 (`liwords-api:317`) |
| 3 | `referee-split-3-shadows` | #2029 | 2026-10-10 | full suite ✅; gameplay with flags on, matches master | ✅ | ✅ 2026-10-12 (`liwords-api:317`, `SHADOW_TURNS` and `SHADOW_TURNS_LOAD` on) |
| 4 | `referee-split-4-atomic-save` | #2053 | 2026-10-12 | full suite ✅; gameplay with flags on: no mismatches | ✅ 2026-10-22 | |
| 5 | `referee-split-5-session-lock` | #2065 | 2026-10-22 | full suite ✅; gameplay with flags on: no mismatches | | |
| 6 | `referee-split-6-no-cache` | | | | | |
| 7 | `referee-split-7-ongoing-games` | | | | | |

After the split (see "After the split"):

| Phase | What | Flag | Needs | Started | Flag on |
|---|---|---|---|---|---|
| 1 | xwordgame decides live moves alongside macondo, compared, serves nothing | `SHADOW_REFEREE` | step 3 deployed, its shadows quiet | | |
| 2 | xwordgame serves, real-time first, correspondence last | `XWORDGAME_REFEREE` | phase 1 quiet for a week; steps 4-6 | | |
| 3 | macondo off the live path | none | phase 2 on for every game mode | | |
