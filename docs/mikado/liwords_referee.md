# Liwords Referee: eliminating the state-sources problem

Companion to `game_storage_v2.md` (architecture) and `remove-game-caches.dot` (mikado graph).
This doc explains *why* the `PortMacondoReferee` mikado node is load-bearing, using the
Phase 3b bug as a concrete case study.

---

## The Phase 3b incident (2026-05-19)

Phase 3b introduced `buildHistoryFromTurns` — a read path that reconstructed the in-memory
game state from `game_turns` rows instead of the `games.history` bytea blob. It was
reverted the next day (Phase 3b rollback) after producing widespread data corruption
in correspondence games. Scale: ~944 games left permanently unarchived, 49 confirmed
`archive-verify-mismatch` events in 10 hours.

### Exact mechanism

For games with a **non-VOID challenge rule**, when a player goes out via tile placement,
macondo does NOT immediately end the game. Instead it sets an intermediate state:

```
PlayState = WAITING_FOR_FINAL_PASS
```

The opponent must then pass (or challenge). On that pass, macondo transitions to
`GAME_OVER`, appends `END_RACK_PTS` to the event log, and sets `FinalScores`. Only
at that point does `performEndgameDuties` fire in liwords.

This intermediate state is **not encoded in any `game_turns` row**. It lives solely as
a field on macondo's internal `game.Game` struct (and is persisted to bytea only when
`Set()` writes `history.PlayState`).

`buildHistoryFromTurns` constructed a `GameHistory` proto from the turns rows but did
not set `PlayState` (the field does not exist in the turns data):

```go
hist := &macondopb.GameHistory{
    Events:        events,   // from game_turns
    ChallengeRule: ...,
    // PlayState NOT SET — proto zero value = PLAYING
}
```

Back in `Get()`, this proto's `PlayState` is read as `histPlayState`:

```go
histPlayState := hist.GetPlayState()   // = PLAYING (zero value)

mcg, _ := macondogame.NewFromHistory(hist, rules, len(hist.Events))
// NewFromHistory replays all events. After the going-out tile play it
// correctly computes WAITING_FOR_FINAL_PASS internally.

// Then the existing "restore stored state" override fires:
entGame.SetPlaying(histPlayState)            // = PLAYING ← wrong
entGame.History().PlayState = histPlayState  //   clobbers WAITING_FOR_FINAL_PASS
```

The override at those two lines exists because `NewFromHistory` only sees move events —
it cannot reconstruct states like `GAME_OVER` from a timeout or resign. Restoring
`histPlayState` from the stored bytea is correct for the bytea path. But when the
history comes from `buildHistoryFromTurns`, the proto zero value for `PlayState` is
indistinguishable from an intent to be in `PLAYING` state.

Result: the opponent's "final pass" is processed as a regular mid-game pass.
`scorelessTurns` increments. The game loops through six consecutive scoreless turns
and ends incorrectly as `CONSECUTIVE_ZEROES` with wrong scores and a corrupted event
tail.

### Why only correspondence games?

For cached (real-time) games: when the opponent calls `HandleEvent`, `Get()` returns
the **same in-memory entity** that the going-out player left in `WAITING_FOR_FINAL_PASS`.
No DB reload — no `buildHistoryFromTurns` — state is correct.

For correspondence games: every `HandleEvent` calls `Get()` which does a fresh DB load
(the comment at `game.go:632`: "correspondence games bypass the in-memory cache — each
`Get()` returns a new object"). The fresh load hits `buildHistoryFromTurns`, loses
`WAITING_FOR_FINAL_PASS`, and the bug fires.

---

## The underlying problem: state lives everywhere

The incident is a symptom of a deeper structural issue. At any point during a live game,
the authoritative state is scattered across at least five distinct locations:

| Source | What it holds |
|---|---|
| `macondogame.Game` (in-memory) | Board, racks, bag, scores, `scorelessTurns`, `lastWordsFormed`, `PlayState`, `backupMode` |
| `games` row | `history` bytea (serialized `GameHistory`), `game_end_reason`, `last_known_racks`, `winner_idx`, `quickdata` jsonb, `timers` jsonb |
| `game_turns` rows | Individual move events (jsonb), indexed by `turn_idx`. No `PlayState`, no `scorelessTurns`. |
| `entity.Game` wrapper | Timers, correspondence flag, tournament data, `GameEndReason`, `WinnerIdx`, `Stats` |
| `histPlayState` override in `Get()` | A compensating hack: restores the DB-persisted `PlayState` after macondo replay would overcompute it |

No single source of truth. Every read path must correctly fuse all five, and every
write path must keep them consistent. The `histPlayState` override is one of several
such compensating hacks:

- **`performEndgameDuties` forcibly calls `g.Game.SetPlaying(GAME_OVER)`** at line 87
  of `end.go`, because a timeout or resign does not produce any game event — macondo
  replay of the event log would compute `PLAYING` even though the game is over.

- **`last_known_racks` column** was added to `games` specifically because the turns
  path couldn't derive the current rack state on its own (it needs an extra column).

- **`AddFinalScoresToHistory` is called conditionally** in `performEndgameDuties`
  because macondo sometimes sets `FinalScores` inside `PlayMove` (VOID rule, or
  six-zeroes) and sometimes does not (non-VOID rule, where it waits for the final pass).
  The condition `len(FinalScores) == 0 || len(Events) > evtIdxBeforePenalties` tries
  to paper over this inconsistency.

- **`WAITING_FOR_FINAL_PASS`** is a macondo-internal state that has no representation
  in the `games` row schema (only in the bytea blob) and no representation in
  `game_turns` at all. It is completely invisible to any read path that doesn't use
  the bytea.

Each of these hacks is a coupling point where a future change (new read path, new
termination condition, new game type) can silently produce the wrong state.

---

## The fix: own the state machine in liwords

The `PortMacondoReferee` node in `remove-game-caches.dot` is the right structural fix.
The goal: macondo becomes a pure calculation library (move validation, tile scoring,
cross-score computation, rack draw) and liwords owns the state machine.

Concretely:

**Today:** liwords calls `macondogame.PlayMove`, then reads macondo's internal
`PlayState` to decide what to do next. macondo decides when a game is over.

**After PortMacondoReferee:** liwords calls macondo for scoring/validation only.
Liwords's own referee (`pkg/game/`) drives the state machine:

```
type PlayState int

const (
    Playing              PlayState = iota
    WaitingForFinalPass
    GameOver
)
```

Liwords knows about *all* termination conditions:
- Going-out → transition to `WaitingForFinalPass` (non-VOID) or `GameOver` (VOID)
- Final pass → transition from `WaitingForFinalPass` to `GameOver`
- Six consecutive scoreless turns → `GameOver` (CONSECUTIVE_ZEROES)
- Timeout → `GameOver` (TIME)
- Resign → `GameOver` (RESIGNED)
- Adjudication → `GameOver` (ADJUDICATED)

`PlayState` becomes a **first-class liwords concept** persisted as part of the `games`
row (a proper column, not buried inside the bytea blob), and included in every
`game_turns` write so the turns read path never needs to infer it.

This eliminates:
- The `histPlayState` override hack in `Get()`.
- The `performEndgameDuties` force-`SetPlaying(GAME_OVER)`.
- The `WAITING_FOR_FINAL_PASS` invisibility problem.
- Any future turns read path having to reconstruct `PlayState` from event replay.

The `AddFinalScoresToHistory` conditionality simplifies too: liwords controls when
`END_RACK_PTS` events are appended (it calls macondo for the point computation), so
the condition is always explicit, not an inference over whether macondo already did it.

---

## Migration path

This is the `PortMacondoReferee` mikado node, which the dot file shows depends on
`MigrateNativeGamesToPkgGame` and `MigrateAnnotatorToPkgGame`. The `game_storage_v2.md`
doc has the full design for `pkg/game/`.

The Phase 3b revert bought time. The safe re-enablement of a turns read path requires:

1. `pg_advisory_xact_lock` per game wrapping AppendTurns + Set in a single transaction
   (closes the race window for all games, not just correspondence).
2. `PlayState` persisted as a column on `games` (or derived from liwords's own state
   machine, not from macondo replay).
3. Either of: liwords referee in `pkg/game/` owns `PlayState` transitions, OR
   `buildHistoryFromTurns` explicitly reads `PlayState` from the `games` row
   (a cheaper intermediate fix, but still leaves macondo in charge of the state machine).

The cheapest short-term fix is option 3b: add `play_state` as a column on `games`,
write it in every `Set()` call, and have `buildHistoryFromTurns` read it. That
eliminates the proto-zero-value ambiguity without porting the referee. But it adds yet
another column to the already-overloaded `games` row, and the `performEndgameDuties`
force-SetPlaying and the `AddFinalScoresToHistory` conditionality problems remain.

The right fix is the referee port. It should happen before the turns read path is
re-enabled.
