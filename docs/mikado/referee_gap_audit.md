# AuditRefereeGap: what the liwords referee actually needs from macondo

Mikado leaf node from `game_storage_v2.md`. Output: the concrete port list for
`pkg/xwordgame/`, produced by reading macondo v0.13.3 function by function.

Companion docs: `liwords_referee.md` (why), `game_storage_v2.md` (architecture).

---

## Summary

| | non-test lines | disposition |
|---|---|---|
| `macondo/board` | 1,523 | port ~40%, skip the movegen half |
| `macondo/move` | 544 | port, slimmed |
| `macondo/game` (`game.go`, `challenge.go`, `rules.go`, `turn.go`, `player.go`) | ~2,400 | port the referee half, **replace** the state machine |
| `macondo/cross_set` | 464 | **skip entirely** (see below) |
| `word-golib/tilemapping`, `word-golib/kwg` | — | **keep as a dependency**, do not port |

Estimated port: **~1,800–2,200 lines**, plus the ~500 already written
(`state.go`, `bag.go`).

macondo stays in `go.mod` regardless — puzzle generation (`pkg/puzzles`) and
analysis need move generation. The goal of this port is not deleting the
dependency; it is owning the state machine.

---

## Finding 1: compute cross-scores fresh; do not cache them, and do not store them

macondo's `board.ScoreWord` (board.go:982) reads `g.GetCrossScore(row, col, dir)`
— a per-square cache of the score of the perpendicular word already on the
board. Maintaining that cache is what `cross_set` exists for, and it is why
`game.PlayMove` has an `updateCrossSets bool` parameter and a
`PlayMoveNoCrossSet` variant.

Nothing about scoring *requires* that cache. It exists because a move generator
scores tens of thousands of candidate plays per turn and cannot afford to
re-walk the board for each one. **A referee scores exactly one move per turn**,
so the cache buys nothing and costs plenty.

The cost is concrete, and it is the deciding argument: a cache of derived state
has to live somewhere across a reload. Either we recompute it on every load — in
which case we have paid for the walk anyway and might as well compute on demand
— or we persist it in the snapshot. Persisting is the bad option:

- 21x21 x 2 directions = **882 values**. Even at a byte each that takes the
  worst-case snapshot from 549 bytes to ~1,430, up against the ~2KB TOAST
  threshold, entirely for data that is a pure function of the board.
- It reintroduces cached-derived-state that can go stale — the same defect class
  as the Phase 3b bug. Every additional remembered-rather-than-computed value is
  another thing a future read path can get wrong.

So: no cache, no cross-score columns, nothing in the snapshot. Score directly
from the words the move forms:

```
words := board.FormedWords(move)        // main word + every cross word
score := 0
for each word:
    walk its letters, applying letter/word multipliers only to freshly
    placed squares; sum
add 50 if tilesPlayed == RackTileLimit
```

At most 8 words of at most 21 letters. A few hundred nanoseconds, once per turn
— against a move that already costs a database round trip.

This removes:

- the entire `cross_set` package (464 lines) from the port,
- `GetCrossScore` / `SetCrossScore` / `ResetCrossScores` and the parallel
  cross-score arrays on the board,
- the `updateCrossSets` plumbing through `playMove`,
- `TraverseBackwardsForScore`.

It also removes a category of *cached derived state that can go stale*, which is
thematically the same defect class as the Phase 3b bug. Fewer places where the
truth is computed once and remembered.

**This is the single riskiest simplification in the port** — it is a
reimplementation of scoring, not a transcription. It is exactly what the corpus
parity harness is for: every archived game exercises it on every turn.

## Finding 2: `word-golib` stays

`tilemapping` (alphabet, letter distributions, `Rack`) and `kwg` (lexicon
lookup) are pure calculation with no state of their own — the "macondo becomes a
calculation library" half of the plan is already factored out into this module.
Porting them would be pointless churn.

The one exception is `tilemapping.Bag`, which `pkg/xwordgame/bag.go` replaces.
Reasons are documented at the top of that file: it has no serialisation story,
its counts array is sized to the distribution rather than fixed-width, and it
carries Monte Carlo machinery a live game does not need. Its *semantics* are
mirrored deliberately, in particular draw-before-return on exchange.

---

## `board` package

### Port

| function | note |
|---|---|
| `MakeBoard`, layouts | bonus-square layouts for CrosswordGame and SuperCrosswordGame |
| `Dim`, `GetSqIdx`, `PosExists` | geometry |
| `GetBonus`, `GetLetterMultiplier`, `GetWordMultiplier`, `BonusMuls` | scoring inputs |
| `SetLetter`, `GetLetter`, `HasLetter`, `Clear`, `IsEmpty` | already covered by `State.SetTileAt`/`TileAt`/`Board` |
| `LeftAndRightEmpty`, `WordEdge` | word-boundary walks |
| `PlaceMoveTiles`, `UnplaceMoveTiles`, `PlayMove` | tile placement |
| `ErrorIfIllegalPlay` | placement legality: on board, connected, not overwriting, first play covers centre |
| `FormedWords`, `formedCrossWord` | the core of validation and of the new scoring path |
| `TilesPlayed` | endgame detection |
| `Transpose` | vertical plays are scored as horizontal ones on a transposed board |
| `ToFEN` | CGP export |

### Skip

Cross sets and anchors are movegen-only: `GetCrossSet`, `SetCrossSet`,
`SetCrossSetLetter`, `ClearCrossSet`, `SetAllCrosses`, `ClearAllCrosses`,
`CrossSetsForDir`, `GetCrossSetIdx`, all `LeftExtSet`/`RightExtSet`,
`updateAnchors`, `UpdateAllAnchors`, `IsAnchor`, `updateAnchorsForMove`.

Cross scores go with Finding 1: `GetCrossScore`, `GetCrossScoreIdx`,
`SetCrossScore`, `ResetCrossScores`, `CrossScoresForDir`,
`TraverseBackwardsForScore`, `ScoreWord`.

Simulation support: `Copy`, `RestoreFromCopy`, `CopyFrom`, `PlaySmallMove`,
`TestSetTilesPlayed`.

---

## `game` package

### Port largely as-is (the calculator)

| function | note |
|---|---|
| `ValidateMove`, `validateTilePlayMove` | move legality |
| `ChallengeEvent` (challenge.go:32) | the five challenge rules; the trickiest logic in macondo and the least worth rewriting |
| `validateWords`, `ValidateWords` | lexicon lookup, variant-aware |
| `calculateRackPts` | endgame rack adjustments |
| `MoveFromEvent`, `EventFromMove` (turn.go) | event <-> move conversion; needed at the archival and bot boundaries |
| `modifyForPlaythrough` | play-through tile representation in events |
| `CalculateCoordsFromStringPosition` | coordinate parsing |
| `MaxCanExchange`, `ExchangeLimit` (rules.go) | exchange legality |
| `NewBasicGameRules` | slimmed: no `cross_set.Generator`, no `config.Config` |

### Replace, do not port (the state machine)

These are where liwords takes ownership. Porting them would carry the defect
across.

| function | why it is replaced |
|---|---|
| `playMove` (game.go:548) | 137 lines fusing scoring, bag draws, history append, `PlayState` transitions and scoreless-turn counting. Split: the calculation stays, the transitions become explicit `PlayState` moves against the persisted snapshot. |
| `handleConsecutiveScorelessTurns` | the six-zeroes rule becomes one explicit transition, not a side effect of playing a move |
| `endOfGameCalcs`, `AddFinalScoresToHistory` | the conditional `len(FinalScores) == 0 \|\| len(Events) > evtIdxBeforePenalties` in `performEndgameDuties` exists only because macondo sometimes sets final scores inside `PlayMove` and sometimes waits. Liwords will always know. |
| `SetPlaying`, `Playing` | replaced by the persisted `PlayState` on the snapshot; the `histPlayState` override in `db.go:323` and the forced `SetPlaying(GAME_OVER)` in `end.go` both disappear |
| `NewFromHistory`, `PlayToTurn`, `PlayTurn`, `PlayLatestEvent` | replay-to-derive-state is the bug. Retained **only** in the parity harness and as a repair tool, never on the live read path. |
| `NewFromSnapshot`, `SetRandomRack`, `ThrowRacksIn` | rack reconstruction hacks that exist because state was not stored |

### Skip

`SeedBag`, `SetUidFromSeed`, `SetEndgameMode`, `SetScorelessTurns`,
`SetMaxScorelessTurns` (sim/endgame-solver hooks), `PlaySmallMove*` and all
`tinymove` paths, `CreateAndScorePlacementMove` and `PlayScoringMove` (shell
conveniences), `AddNote`, `mlhelper.go` and `inference_helpers.go` (ML
inference), `backup.go` (`stateStack`, `BackupMode` — simulation undo),
`RecalculateBoard`, `FlipPlayers`, `RenamePlayer`.

The nick-keyed accessors (`PointsForNick`, `BingosForNick`, `TurnsForNick`) go
away; liwords indexes players by position.

---

## What `State` already covers

`pkg/xwordgame/state.go` + `bag.go` (~500 lines, written) already replaces:

- `stateBackup` (backup.go:60) — macondo's own enumeration of "the state you must
  save and restore" is board, bag, playing, scorelessTurns, onturn, turnnum,
  players. `State` is that set, plus `LastWordsFormed`.
- `playerState` (player.go) — rack, points, bingos, turns.
- `tilemapping.Bag` — as a multiset.

Note `lastWordsFormed`: macondo's comment at game.go:69 says it "does not need to
be backed up", and game.go:1130 has a patch commented *"prevent another +5 on app
restart"*. It is cross-restart state, and `State` persists it.

---

## Parity harness

Two tiers, because they validate different things.

**Tier 1 — offline corpus replay (the calculator).** Feed each archived
`GameHistory` through both engines event by event; compare board, scores, racks,
bag contents, tiles played, and end reason at every turn. 11.3M games in S3.
Catches every scoring, validation, challenge and endgame divergence, including
the rewritten cross-score-free scoring from Finding 1. Zero production risk.

Caveat: a `GameHistory` is an event log, so replaying it and comparing against a
state derived from that same log is circular for exactly the fields that are not
in the log. Tier 1 cannot validate `WAITING_FOR_FINAL_PASS`, `scorelessTurns`,
bag contents mid-game, or `lastWordsFormed`.

**Tier 2 — live shadow (the state machine).** Reuse `SpawnShadowCompare`
(`pkg/stores/game/db.go:404`), already wired into the move path at
`pkg/gameplay/game.go:565` behind `cfg.ShadowTurns`. Run the new engine
alongside the live one after every move and compare `State.Digest()`.

The existing comparison is too weak for this — it checks only
`wantTurn, wantP0, wantP1, wantEventCount` (db.go:446). Replace it with a full
digest comparison; log divergence with game UUID and turn index. This is the tier
that would have caught Phase 3b.

Only after Tier 2 runs clean on production traffic for a sustained period does
the live path switch over.
