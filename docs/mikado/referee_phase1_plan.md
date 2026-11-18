# Referee phase 1: xwordgame decides live moves alongside macondo

Status: plan for approval, nothing built (2026-11-17).

Phase 1 of "After the split" in `pr1975_split.md`; the work items are in #2037.
macondo keeps deciding and serving every move. Behind `SHADOW_REFEREE`,
xwordgame decides the same move in the background, and every disagreement is
logged. Phase 2 (xwordgame serves) starts only after this has been quiet for a
week including a weekend, across real challenges under each rule in use.

## What gets compared

For every move liwords applies, with the flag on:

1. **Accept or reject.** Both engines accept the move, or both reject it. A
   rejection is compared by kind (invalid words, illegal placement, tiles not
   on rack, wrong player), not by message.
2. **The events.** The events macondo appended to the history, field by field,
   against the events xwordgame's result turns into: type, cumulative score,
   rack, position, direction, played tiles, score, bonus, words formed, lost
   score, end-rack points, and the history's `PlayState`, `FinalScores` and
   `Winner` when the move ends the game.
3. **The position.** `xwordbridge.StateFromGame` of macondo's game after the
   move, against xwordgame's state, with `CompareStates`.

Any difference logs one `referee-shadow-mismatch` line naming the game, the
move, and every field that differed, and increments a counter. Agreements
count too, per move kind and challenge rule, so "zero mismatches" can be read
against how many challenges, timeouts and endings were actually checked.

## Design

### The input is the live game, not the turns table

The shadow takes its "before" position from the game in memory
(`StateFromGame(&entGame.Game)`, taken synchronously before macondo applies the
move), not from `StateFromTurns`.

- **Mismatches point at the referee.** Loading from turns is already checked
  by `SHADOW_TURNS_LOAD`. Mixing the two would make every mismatch ambiguous.
- **No database read,** so no extra load and no race with the next move. The
  existing turns shadow had to work around exactly that race.
- **Phase 2 swaps the input** to `StateFromTurns`, which by then has its own
  week of clean shadows.

The one exception is a challenge, which needs the position *before* the
challenged play (below).

### Drawn tiles

macondo and xwordgame draw replacement tiles with different random sources, so
their racks after a play will almost never match. Two ways to handle it:

- **Compare around the draw (recommended).** Check that xwordgame drew the
  same *number* of tiles, that macondo's drawn tiles were in the bag before
  the move, and then set xwordgame's rack and bag to macondo's before comparing
  the rest. Simple, and it doesn't couple the shadow to the bag's internals.
- **Script xwordgame's `Rand`** to pick the tiles macondo drew. `Rand` is only
  `IntN(n)`, so this needs the bag's order inside the shadow. More exact,
  more fragile.

### Challenges need the position before the challenged play

#2037's first item. The replay already computes it (`prev` in `replay.go`) but
`StateFromTurns` doesn't return it. For the shadow, replay the *in-memory*
history up to the challenged play with `ReplayHistory`, which needs no database
read. Then correct the racks before restoring it:

- the challenger keeps their current rack (a phony doesn't change it);
- the challengee gets back the rack recorded in their placement event;
- the bag is recomputed from the tiles that are left.

Phase 2 needs the same function fed from `StateFromTurns`, so it lives next to
the replay, not in the shadow.

### The event emitter is the permanent part

Turning `ApplyResult` into macondo-style `GameEvent`s is what phase 2 writes
to `game_turns` and to the history. It belongs with the turns read path
(`ReplayHistory`, `DecodeTurn`, `MoveFromEvent`), which phase 3 moves into
xwordgame or a storage package. Put it in `pkg/xwordbridge/emit.go` for now,
with a header saying it's permanent, unlike `StateFromGame`, `Compare` and
`quirks.go`.

### Where it hooks in

- `PlayMove` (`pkg/gameplay/game.go`): tile plays, passes and exchanges, for
  players and bots alike (`goHandleBotMove` calls `PlayMove`).
- `handleChallenge`: challenges, the phony's return, challenge bonuses, and
  the turn lost on an unsuccessful challenge.
- `TimedOut`: timeouts, including the one while waiting for the final pass.
- The end of a game: end-rack points or penalties, the overtime penalty
  (10 × max overtime), and the winner recomputed after penalties.
- **Rejected moves:** where `PlayMove` returns the `ValidateMove` error, the
  shadow still runs and compares the rejection.

Each hook snapshots what it needs on the caller's goroutine and hands it to a
background goroutine, like `SpawnShadowCompare` does.

### Shadows must never take the API down

liwords has no panic-recovery middleware, and a panic in any goroutine kills
the whole process. The background goroutines in `pkg/stores/game` don't
recover today: `shadowCompareTurns` (`db.go`), the load shadow's `work`, and
the two archive goroutines in `s3.go`. Those are running in production now.
The first PR adds a small `safeGo` helper that recovers, logs the panic with
the game ID, and counts it, and uses it for all of those. The referee shadow
uses it from the start.

## The PRs, in order

Each one widens what the shadow checks, so real traffic is checked as early as
possible instead of waiting for everything.

1. **`safeGo` for the existing shadows and archive goroutines.** Small, and
   worth having whether or not phase 1 goes ahead.
2. **Rules fixes from #2037 that would otherwise drown the comparison:**
   - liwords never sends an exchange of zero tiles;
   - a range check on tile codes (an undesignated blank, or one outside the
     alphabet);
   - `ApplyTimeout`: a timeout while waiting for the final pass is an
     automatic pass, not a forfeit, and the timed-out player must be on turn;
   - `RulesFor` can be told a lexicon is required, so live callers fail
     instead of rejecting every play.
3. **Client move parsing:** a `ClientGameplayEvent` to an `xwordgame.Move`,
   replacing `CreateAndScorePlacementMove` for the shadow, including
   spelled-out letters on occupied squares turned into play-through markers.
   Tested against macondo's parsing on the same events.
4. **The event emitter,** with the parity differential
   (`TestStateMachineParityAgainstMacondo`) extended to compare events, not
   just positions.
5. **The shadow itself,** behind `SHADOW_REFEREE`, for tile plays, passes,
   exchanges and rejections. Other move kinds count as "skipped". **Deploy and
   turn it on here.**
6. **Challenges:** the position before the challenged play, returned tiles
   for `ServerChallengeResultEvent`, and challenges added to the differential
   for all six rules, including plays that go out and triple challenges. This
   closes the gap in #2037: challenge decisions have never been compared
   against macondo's.
7. **Clock and endings:** timeouts, the overtime penalty, end-rack points and
   penalties, and the winner after penalties.

The week-long bar starts when PR 7 is on.

## Left out of phase 1

- **WordSmog.** macondo checks WordSmog words against the `.kad` anagram file,
  and xwordgame against the `.kwg`. Fixing that brings macondo's `alphadawg`
  into `rules.go`, which is meant to outlive macondo. Phase 1 counts WordSmog
  moves as skipped; the fix belongs to phase 2, with the wider WordSmog
  differential.
- **Everything in #2037's phase 2 and 3 lists:** `InvalidWordsError` listing
  the words, empty-row decoding, dealing new games, moving the protos.

## Decisions for the user

- Compare around the draw, or script `Rand`? (Recommended: around the draw.)
- Is leaving WordSmog out of phase 1 acceptable?
- Turn `SHADOW_REFEREE` on in production as soon as PR 5 deploys, or wait
  until challenges (PR 6) are in?
