# Reviewing the xwordgame branch

This is a guide for reviewing `xwordgame-referee-state`, not a summary of it.
It says what each piece is for, which decisions are judgment calls worth
arguing with, and what is deliberately unfinished.

**Scope:** 9,436 lines across 33 files. 3,889 lines of code and 4,273 of test in
the two new packages, plus the store layer, two config flags and one migration.
Nothing is wired into the live path yet: both feature flags default off, and
with them off not a single byte of behaviour changes.

## Why this exists

In May 2026 roughly 944 correspondence games were corrupted because live game
state was reconstructed by replaying an event log instead of being stored.
`buildHistoryFromTurns` produced a `GameHistory` with no `PlayState`; proto3's
zero value for that field is `PLAYING`, which silently overwrote
`WAITING_FOR_FINAL_PASS` and ended games with `CONSECUTIVE_ZEROES` and wrong
scores. See `docs/mikado/liwords_referee.md`.

The fix is not "replay more carefully". It is that a game has two authoritative
records and neither is derivable from the other:

- the **event log** is authoritative for *what happened*, and is what the S3
  archive, GCG export, replay and analysis are all built from;
- the **position** is authoritative for *where the game is now* — board, bag,
  racks, scores, counters, whose turn, and what phase.

The log genuinely cannot reconstruct the position. Not as an implementation
detail: `WAITING_FOR_FINAL_PASS`, the bag, the scoreless-turn counter and the
words formed by the last play change without producing events. This branch
stores the position instead of inferring it.

## Reading order

Start here; each part assumes the one before it.

| # | Read | For |
|---|---|---|
| 1 | `pkg/xwordgame/state.go` (package comment) | the two-records argument, then the wire format |
| 2 | `pkg/xwordgame/apply.go`, `challenge.go` | the state machine and the referee |
| 3 | `pkg/xwordbridge/bridge.go` | how a live macondo game becomes a `State` |
| 4 | `pkg/xwordbridge/replay.go` | how an archived game becomes a `State` |
| 5 | `pkg/stores/game/ongoing.go` | where it gets written, and why nothing can fail |
| 6 | `db/migrations/202608040001_ongoing_games.up.sql` | the storage shape |

## The decisions worth arguing with

These are places where I chose, and where a different choice was defensible.
They are the highest-value things to review.

**The bag is a multiset, not an ordered sequence** (`state.go`). An ordered bag
makes exchanges incoherent — the player returns tiles in the same operation they
draw, so any append scheme hands back their own discards. macondo already runs
`fixedOrder=false` in live play, so its stored order is meaningless anyway.
Consequence: two positions with the same tiles remaining always encode
identically, so digests compare directly. Also removes the need for macondo's
post-phony reshuffle: there is no order to leak.

**Cross-scores are computed, not cached** (`score.go`). macondo keeps a
per-square cross-score cache; we recompute. That drops the 464-line `cross_set`
package and 882 values from the snapshot, at the cost of walking the same
geometry twice when `ScoreWords` is called. `parity_macondo_test.go` is the
evidence the two agree.

**`Rules` is not part of `State`.** Lexicon, distribution, layout, variant and
challenge rule live beside the snapshot as plain columns, resolved once at load
into shared objects. Keeps `State` about the position and the row cheap.

**The five-point challenge rule has an explicit mode** (`challenge.go`). macondo
pays a flat five; `pkg/cwgame` pays five per word. `FivePointMode` names the
choice, and `FivePointPerPlay` is the zero value so a caller that says nothing
keeps today's live behaviour. An earlier version inferred the mode from whether
indices were supplied, which meant naming every word scored differently from
naming none — for the same action. `LegacyFivePointMode` reproduces cwgame's
inference in one named place, for replaying old annotated games.

**Time is modelled by its effects, not by a clock** (`clock.go`). Timers,
increments, overtime and deadlines stay in liwords. What moved here is what
changes the position: a penalty, a timeout, and `TimeAttributedTo` — which
removes the flip-record-flip hack at `pkg/gameplay/game.go:437`.

**Replay trusts the record over the rules** (`replay.go`). Historical games were
played under the lexicon and policies of their day. `TrustRecordedPlays` skips
the word check and the two-letter minimum; challenge verdicts and end-of-game
adjustments are read, not recomputed. **This is the one that most deserves
scrutiny**, because trusting the record is also how a replay can hide an engine
bug. The mitigation is `EndRackArithmeticWrong`, which counts the times our own
figure disagreed while the recorded one was applied, and fails the test if we
knew the rack and still got the number wrong. It is currently zero.

## What is deliberately not compared

`pkg/xwordbridge/corpus_test.go` excludes some fields when comparing against a
macondo *reconstruction*. Each exclusion is narrow and none apply to the
live-game differential, which compares everything. Check that these are still
true rather than taking them on faith:

- `turns[]`: `PlayTurn` increments per-player turn counts only in the exchange
  branch, where live `playMove` increments for every move type.
- `lastWordsFormed` once a game is over: macondo never clears it on a pass, an
  exchange, or in `PlayToTurn`'s ending, so a finished position advertises a
  play that cannot be challenged.
- `onTurn` and `scorelessTurns` once a game is over: meaningless, and macondo's
  reconstruction disagrees with its own live play on both.
- `playState`: checked against the database's `game_end_reason` instead, because
  macondo's reconstruction never runs end-of-game logic. That is a stronger
  assertion, not a weaker one.

## Evidence

Three independent harnesses, because each covers what the others cannot.

| Harness | What it proves |
|---|---|
| `pkg/xwordgame/*_test.go` | hand-computed scores and rules; a differential cannot catch a bug shared by both engines |
| `pkg/xwordbridge/bridge_test.go` | same random move into both engines, 13 configs, both board sizes, all nine distributions, 196 games |
| `pkg/xwordbridge/corpus_test.go` | 113,010 real games, 3.12M events, 67k challenges — **0 divergences** |

The corpus is production data and is not in the repository. The export queries
are in the file header; point `XWORDGAME_CORPUS` at a directory and the test
runs, otherwise it skips.

Mutation testing was used throughout: seed a deliberate bug, confirm the suite
fails. Roughly 90 mutations. Two lessons that shaped later work — a mutation
that fails to compile must be reported as invalid rather than caught, and a
mutation nothing can observe is an *equivalent mutant*, not a coverage gap.

## Bugs this found, and where

Worth reading as a list of things that were wrong, since each one is a place the
design could still be wrong.

*In this branch, found by the harnesses:*

- placement took the play's whole span off the rack, including played-through
  markers — demanded a blank per covered square
- exchange used the caller's tile slice after rewriting the rack, so the bag
  received whatever landed in those slots; counts still balanced, only tile
  conservation noticed
- the scoreless-turn penalty was charged against an invented opponent rack
- `ApplyPlacement` set `LastWordsFormed` *after* the going-out check, so ending
  a game cleared it and then put it straight back — 4,448 divergences from one
  line's position
- a proto3 zero read as an absent field: `Cumulative != 0` used as a presence
  test, double-applying a penalty for the one game in 113,010 whose final score
  was exactly zero. **The same mistake as the May 2026 incident**, and it hid
  better because `PLAYING` is common while a zero score is not

*In macondo, found by comparison:*

- `PlayTurn` loses per-player turn counts
- `lastWordsFormed` is never cleared on a pass or exchange, so `ChallengeEvent`
  would adjudicate a challenge against a play from two turns ago
- `PlayToTurn` advances the player on turn after non-move events
- reconstruction never runs end-of-game logic, so a game that ended on six
  scoreless turns comes back as `PLAYING` — the incident's shape exactly
- a triple challenge the challenger lost appends no event at all
- games containing since-delisted words cannot be opened at all: nine in the
  corpus, e.g. a 2021 ECWL game containing `SEZ`

*In liwords, found incidentally:*

- `pkg/league/force_finish_games.go` never calls `ArchiveAndCleanup`, so
  league-adjudicated games never reach S3 and their `game_turns` rows leak. 27
  of 71 adjudicated games affected; forfeits are 93,021/93,021 clean
- `games.uuid` has no unique constraint — nothing prevents a duplicate game ID
- `LockGame` (`pkg/stores/game/cache.go:428`) is a process-local mutex map and
  does nothing across app servers

## Known limits

Stated so they are not mistaken for oversights.

- **15 games** have end-of-game adjustments we could not have computed, because
  the tiles drawn in a final exchange are revealed only by the end-rack event
  that arrives after the penalty is due. A limit of replaying a log, not of the
  engine: live play holds both racks, which
  `TestStalemateChargesTheRealRackAfterAnExchange` demonstrates.
- **1 game** whose archived history says `PLAYING` with no final scores while
  the games table says it ended on a triple challenge. The record contradicts
  itself; no replayer can reconcile it.
- **Mid-game replay** has partly-guessed opponent racks between their moves,
  because the log does not record them. macondo has the identical limitation.
- `TurnNum` counts plies and is deliberately not macondo's `Turn()`, which
  indexes the event history and so also counts synthetic end-of-game events.

## Not done

- nothing is wired into `pkg/gameplay`; the flags exist and do nothing
- no read path — the snapshot is written and never read back
- `pkg/cwgame` is untouched, and the annotator still uses it
- the archival leak, the unique constraint and the process-local lock are all
  identified and unfixed

## Where the risk actually is

Not in the rules — those have 113,010 games of evidence. The risk is in the
parts with the least evidence:

1. **`pkg/stores/game/ongoing.go`** has no test at all. It is guarded by flags
   that default off and by a promise that it never returns an error, and both of
   those should be verified by reading rather than assumed.
2. **The migration and its indexes.** HOT-update behaviour was measured (95%,
   and 0% with an indexed column, which is why the index list is what it is),
   but the queries have no callers yet, so nothing has exercised them.
3. **The cutover contract.** Several guarantees currently live in commit
   messages: that archived histories keep carrying `PlayState`, `FinalScores`
   and `Winner`, and that end-rack events keep their `rack` field. Prose does
   not survive; these need to become tests before the cutover, not after.
