# Splitting PR #1975

PR #1975 (`xwordgame-referee-state`) is ~15k lines across 86 files. It mixes
inert new packages, flag-gated evidence gathering, and several unconditional
changes to the move path. Shipped as one PR, the deploy order in
`xwordgame_remaining.md` cannot be followed: the transaction, the lock and the
cache removal have no flags, so they would all go live at once.

This document is the plan for landing it as a stack of PRs, one deployable and
revertible stage each, and the record of how each step was checked.

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
tree, except for migration filenames (see below) and this document. That final diff is what
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
| 1 | `referee-split-1-automod` | automod idempotency, automod after save, `ready_flag` out of `UpdateGame`, `notoriousgames` unique index | yes, small | none |
| 1b | `referee-split-1b-games-uuid` | `games.uuid` unique index migration | none **if** the manual step ran first | none |
| 2 | `referee-split-2-referee` | `pkg/xwordgame`, `pkg/xwordbridge` | none; nothing calls them | none |
| 3 | `referee-split-3-shadows` | rewritten `-shadow-turns`, new `-shadow-turns-load`, `GetGameWithTurns`, archive-contract log, `cmd/profile-game-load` | none with flags off | `SHADOW_TURNS`, `SHADOW_TURNS_LOAD` |
| 4 | `referee-split-4-atomic-save` | `StageTurns`; turn rows and the games row in one transaction; `CommitArchival` count check | **yes**: a failed `game_turns` write fails the move | none |
| 5 | `referee-split-5-session-lock` | session advisory lock replaces the in-process mutex map | **yes**: DB round trip per move, holds a pool connection, `ErrGameLockBusy` | none |
| 6 | `referee-split-6-no-cache` | remove the game LRU cache | **yes**: every load hits the DB | none |
| 7 | `referee-split-7-ongoing-games` | `ongoing_games` migration, queries, writer; mikado docs | none with the flag off | `WRITE_ONGOING_GAMES` |

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
| `pkg/mod/automod.go`, `pkg/mod/classify_test.go`, `pkg/mod/automod_accumulation_test.go`, `pkg/mod/automod_apply_test.go` | **1** |
| `pkg/mod/automod_test.go` (deleted in reference) | kept in 1, **deleted in 6**. See below |
| `pkg/stores/mod/db.go`, `db/queries/mod.sql` | **1** |
| `db/migrations/*_automod_verdicts.*`, `*_notoriousgames_unique.*` | **1** |
| `db/migrations/*_games_uuid_unique.*` | **1b** |
| `db/queries/games.sql` | 1 (`ready_flag`), **3** (`GetGameWithTurns`) |
| `pkg/gameplay/end.go` | 1 (automod after save), **4** (`StageTurns`) |
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
| `pkg/bus/gameplay.go`, `pkg/gameplay/meta_events.go` | **5** |
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
- **Step 3: `pkg/config/config.go` has `ShadowTurnsLoad` but not
  `WriteOngoingGames`.** The reference adds both in the same hunks; the second
  lands with `ongoing_games` in 7.
- **Step 1: `pkg/mod/automod_test.go` is kept.** The reference branch deletes
  it because it relies on the game cache sharing one `*entity.Game` between
  the test and the store, which stops being true in step 6. Until then it
  still runs and is extra coverage. Deleted in 6.

### Coverage the reference branch dropped

The replacement automod tests test `Classify` and `updateNotoriety` directly.
Two things the deleted end-to-end test checked are no longer covered anywhere:

- casual (unrated) games never reach automod. The gate is in
  `pkg/gameplay/end.go`.
- `performEndgameDuties` calls automod at all.

The old test is kept until step 6 (above). Before 6 merges, add a
`pkg/gameplay` test for both, or accept the gap explicitly.

## Problems found in the reference branch while splitting

Fixes for these are deliberate deviations from the reference. Each gets its own
commit and is listed here, so the final tree diff explains itself.

- **`SHADOW_TURNS` reports false mismatches on `turns[]`.** Found in step 3 by
  running the `pkg/gameplay` suite with `DUAL_WRITE_TURNS`, `SHADOW_TURNS` and
  `SHADOW_TURNS_LOAD` on, against master, step 3 and the reference.
  `SpawnShadowCompare` diffs with `CompareStates`, which excludes nothing, on
  the grounds that the game it compares against was played live in memory.
  That stops being true when the game was loaded from history: macondo's
  `NewFromHistory` loses per-player turn counts (`PlayTurn` only increments
  them for exchanges, see `xwordgame_review.md`). The reference produces
  `from_turns.turns[0]=1 live.turns[0]=0` in `TestDoubleChallengeGoodWord` and
  `TestQuickdata`. In production this hits every correspondence game now, and
  every game after step 6. It is a false alarm, not corruption, but it would
  bury real mismatches. **Open:** decide the fix before trusting the flag
  (exclude `turns[]` from this comparison, or fix `PlayTurn` in macondo, which
  `xwordgame_remaining.md` already lists).

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

1. *Watch* `automod-already-applied` at DEBUG. A steady trickle means
   something still judges games twice.
3. Turn on `SHADOW_TURNS` and `SHADOW_TURNS_LOAD`. *Watch*
   `shadow-turns-mismatch`, `shadow-load-mismatch` (ERROR, each names every
   disagreeing field), `shadow-load-torn`, `shadow-load-count-mismatch`,
   `archive-contract-violation`. Before step 4, torn reads are expected: that
   is the race step 4 fixes, and the count is the measurement of it.
4. *Watch* move error rates; `dual-write-turns-error` should turn into failed
   moves, not appear alongside them. `shadow-load-torn` should go to zero.
5. *Watch* `ErrGameLockBusy`, `game-unlock-failed`, pool saturation, move
   latency.
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
blob() { git rev-parse -q --verify "$1:$2" 2>/dev/null || echo none; }
git diff --name-only "$m" "$r" | while read -r f; do
  here=$(blob HEAD "$f"); mm=$(blob "$m" "$f"); rr=$(blob "$r" "$f")
  if [[ $f == db/migrations/* && $rr != none ]]; then
    hit=$(git ls-tree HEAD db/migrations/ | awk -v b="$rr" '$3==b{print $4}')
    [[ -n $hit ]] && { printf '%-14s %s  (as %s)\n' "= reference" "$f" "${hit##*/}"; continue; }
  fi
  if [[ $here == "$rr" ]]; then s="= reference"
  elif [[ $f == pkg/stores/models/* && $here != none ]] && diff -q \
       <(git show "HEAD:$f" | grep -v '^//   sqlc v') \
       <(git show "$r:$f" 2>/dev/null | grep -v '^//   sqlc v') >/dev/null; then
    s="= reference"   # differs only in the sqlc version header
  elif [[ $here == "$mm" ]]; then s="= master"
  else s="INTERMEDIATE"; fi
  printf '%-14s %s\n' "$s" "$f"
done
# Anything HEAD changed that the reference never touched is a leak.
git diff --name-only "$m" HEAD | while read -r f; do
  [[ $f == docs/mikado/pr1975_split.md ]] && continue
  [[ $f == db/migrations/* ]] && continue
  git diff --quiet "$m" "$r" -- "$f" && printf '%-14s %s\n' "NOT IN REF" "$f"
done
```

Migration files are matched by content rather than name, since their numbers
change (see above). `NOT IN REF` means a split branch changed a file the
reference never touched, which should never happen.

## Progress

Kept up to date at the top of the stack. `TestStandings` in
`pkg/pair/standings` also fails on master and is excluded from "full suite".


| # | Branch | PR | Ledger checked | Tests | Merged | Deployed |
|---|---|---|---|---|---|---|
| 1 | `referee-split-1-automod` | #2026 | 2026-10-09 | full suite ✅ | | |
| 1b | `referee-split-1b-games-uuid` | #2027 | 2026-10-09 | store + gameplay ✅ | | |
| 2 | `referee-split-2-referee` | #2028 | 2026-10-09 | full suite ✅ (corpus tests skip) | | |
| 3 | `referee-split-3-shadows` | #2029 | 2026-10-09 | full suite ✅; gameplay with flags on, matches master | | |
| 4 | `referee-split-4-atomic-save` | | | | | |
| 5 | `referee-split-5-session-lock` | | | | | |
| 6 | `referee-split-6-no-cache` | | | | | |
| 7 | `referee-split-7-ongoing-games` | | | | | |
