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
| `pkg/gameplay/end.go` | **4** (`StageTurns`); the automod-after-save hunks are dropped |
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
   not, and need fixing in step 5 or 6 (a further deviation, to be recorded):
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

# Files the split changes on purpose, so they will never equal the reference.
# Each is explained under "Deliberate deviations" in docs/mikado/pr1975_split.md.
deviations="
pkg/stores/game/shadowturns_test.go
pkg/stores/game/db.go
pkg/gameplay/end.go
pkg/mod/automod.go
pkg/stores/models/models.go
db/migrations/202610090003_notoriousgames_unique.up.sql
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


| # | Branch | PR | Ledger checked | Tests | Merged | Deployed |
|---|---|---|---|---|---|---|
| 1 | `referee-split-1-automod` | #2026 | 2026-10-10 (re-cut) | | | |
| 1b | `referee-split-1b-games-uuid` | #2027 | 2026-10-09 | store + gameplay ✅ | | |
| 2 | `referee-split-2-referee` | #2028 | 2026-10-09 | full suite ✅ (corpus tests skip) | | |
| 3 | `referee-split-3-shadows` | #2029 | 2026-10-09 | full suite ✅; gameplay with flags on, matches master | | |
| 4 | `referee-split-4-atomic-save` | | | | | |
| 5 | `referee-split-5-session-lock` | | | | | |
| 6 | `referee-split-6-no-cache` | | | | | |
| 7 | `referee-split-7-ongoing-games` | | | | | |
