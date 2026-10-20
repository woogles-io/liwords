# Lagged Pairings Plan

## Goal

Today, round N+1 cannot be paired until every result for round N is in. With
lagged pairings, a director can say that round N+1 is paired from the results
through round N-1 instead. Round N+1 can then be paired, and opened, while
round N is still being played. This keeps fast finishers from waiting on the
slowest game of the round.

Directors choose the lag per round. Some rounds pair from the previous round's
results (the current behavior) and some pair from results two rounds back. A
division may therefore have results pending for two rounds at the same time.

Rounds are 1-based in prose and 0-based in code, as everywhere else in the
tournament package.

## Decisions

- A round's lag is 0 or 1. Nothing larger is supported.
- No migration, and no change to stored tournament data. Existing tournaments
  read every lag as 0, and every lag-0 code path behaves as it does today. See
  [Compatibility with existing tournaments](#compatibility-with-existing-tournaments).
- AutoStart opens lagged rounds automatically, including while the previous
  round is still being played.
- Changing a round's lag never re-pairs it. The director re-pairs with an
  explicit "Pair round".
- Players are told when a round's pairings are lagged.
- COP supports lag, but not on the last two rounds of a division. Woogles'
  round-control validation rejects that. The COP engine and the public pairing
  API do not, because TSH uses them for lagged pairings.
- Woogles builds lagged COP requests the same way TSH's client does
  (`COP.pm` in `jvc56/tournament_pairing_algorithms`).
- COP's simulations play the in-progress round with its real pairings, with
  byes as forced wins. This change is in the shared engine, so TSH's lagged
  requests benefit too.

## Terminology

- **Lag** (`pairing_lag`), per round: 0 means the round is paired from results
  through the previous round (the current behavior). 1 means it is paired from
  results through two rounds back.
- **Basis round** of round R: `B(R) = R - 1 - lag(R)`. This is the last round
  whose results feed R's pairings. A negative basis means "no results"
  (initial standings, ordered by seed).
- **Open round**: a round that has been started (`<= CurrentRound`) and not
  yet closed for normal results. A division has at most two open rounds:
  `CurrentRound - 1` (only when `CurrentRound` is lagged) and `CurrentRound`.
- **Complete prefix**: the largest k such that rounds 0..k are all complete.

## Behavior spec

1. **Pairing.** Round R is auto-paired as soon as rounds `0..B(R)` are all
   complete, if all of these hold:
   - R has not started.
   - R's method depends on standings and is not MANUAL.
   - R-1 already has pairings, so repeats and bye counts can see them.

   Standings come from round `B(R)`. Results already in for later rounds are
   ignored, so pairings never depend on which games happened to finish first.
   Repeats and bye history still cover every round through R-1, using
   pairings rather than results, so R does not repeat R-1's matchups.
2. **Opening a round.** Round R can be opened when it is fully paired and
   rounds `0..R-1-lag(R)` are complete. With lag 0 this is today's rule. With
   lag 1, R can open while R-1 is still in progress, but R-2 must be done.
   This rule also applies to standings-independent rounds (round robin and
   similar) that are marked lagged. For those rounds, the lag only relaxes the
   gate for opening the round.
3. **Per-player gate.** A player cannot ready up for their round R game until
   their own round R-1 game has a result. Each player still plays one game at
   a time, which keeps the "one active game per user" assumptions in the bus
   and UI true.
4. **Results.** Normal (non-amendment) results are accepted for any open
   round. In practice, a game that started in round N and ends after round N+1
   opened must still be recorded through `HandleTournamentGameEnded`.
5. **Rounds finish in order.** Because of the per-player gate, a player's round
   R game cannot finish before their round R-1 game. So in online play, round
   R never completes before round R-1. Results a director types in (IRL mode,
   score edits) skip that gate and can arrive in any order. For example, an IRL
   director might enter every final-round scorecard before the last
   penultimate-round card arrives. If the division then counted as finished,
   `SetResult` would reject the late card, because it refuses results once the
   tournament is finished. So lagged divisions treat a round as done only when
   it and every earlier round are complete: they use the complete prefix. This
   applies to finishing the division, to the pairing trigger, and to
   gibsonization. Directors can still enter scorecards in any order.
6. **Amendments** keep today's behavior. Amending a basis-round result after
   the dependent round was paired does not re-pair it automatically. The
   director can re-pair any round that has not started.
7. **AutoStart** opens every round as soon as it can be opened, so a lagged
   round opens while its predecessor is still in progress. Each round it opens
   sends a `TournamentRoundStarted` event.
8. **Changing lag** on a future round updates its controls only. Existing
   pairings stay until the director clicks "Pair round". The new lag applies
   at once to whether the round can be opened.
9. **Telling players.** Players see which rounds are lagged and which round's
   results their pairings come from (see [Telling players](#telling-players)).

### Example

Seven rounds. Rounds 5 to 7 are lagged.

| Event | Effect |
|---|---|
| Round 3 complete | Round 4 (lag 0) is paired. Round 5 (lag 1, basis 3) is paired too. |
| Round 4 in progress | Round 5 can be opened, because round 3 is complete. With AutoStart it opens right away. |
| A player finishes their round 4 game | They can ready for their round 5 game. Players still in round 4 cannot. |
| Round 4 complete | Round 6 (lag 1, basis 4) is paired. Once round 6 opens, it runs alongside round 5. |
| Round 5 complete | Round 7 (lag 1, basis 5) is paired. |
| Round 7 complete | The division is finished. Round 6 is already complete, since every player finished it before playing round 7. |

## Compatibility with existing tournaments

No migration, no backfill, and no conversion when data is read.

- Divisions are stored as JSON with `encoding/json`
  (`pkg/stores/tournament/db.go:79-105`, `:191-210`). The generated Go field
  is tagged `json:"pairing_lag,omitempty"`. Rows without the key decode to 0,
  and divisions with every lag at 0 are written without the key.
- Every change below takes one of two forms:
  - **Same at lag 0.** The new rule gives exactly today's result when every
    lag is 0. For example, `B(R) = R - 1` and "rounds `0..R-1-lag` complete"
    are today's formulas when lag is 0.
  - **Gated on `t.hasLaggedRounds()`.** The new code runs only for divisions
    with at least one round whose lag is above 0. Otherwise the existing code
    runs unchanged.

  The table in [Backend changes](#backend-changes) says which form each change
  takes.
- A director can turn on lag in the middle of a tournament (for a future
  round). The lagged code paths must therefore handle a history produced by
  the lag-0 code paths. They do, because lag 0 is a special case of the
  general rule.
- COP requests with no in-progress round produce the same pairings as today.
- One deliberate change also reaches lag-0 divisions. When AutoStart opens a
  round, `SetResult` now sends `TournamentRoundStarted`, which it currently
  never does. This changes no stored data.
- Add a test that loads a division JSON fixture written before this change and
  plays it to the end (see [Tests](#tests)).

## Data model

Add one field to `RoundControl` in `api/proto/ipc/tournament.proto`:

```proto
  // pairing_lag is how many extra rounds of results this round's pairings
  // lag behind: 0 (the default) pairs round R from results through round R-1,
  // and 1 pairs it from results through round R-2, so that round R can be
  // paired and opened while round R-1 is still being played. Only 0 and 1 are
  // valid.
  uint32 pairing_lag = 21;
```

- No new persisted state in `ClassicDivision`. `CurrentRound` keeps meaning
  "the most recently opened round". Open rounds and the complete prefix are
  derived from `CurrentRound`, the round controls and the pairing outcomes.
- Run `./gen.sh` (or the buf generate step) to regenerate the Go and TS
  bindings in `rpc/` and `liwords-ui/src/gen/`.

## Backend changes

All in `pkg/tournament/classic_division.go` unless noted.

| Change | Compatibility |
|---|---|
| Non-amendment result window in `SubmitResult` | Same at lag 0 |
| Standings refresh range in `SubmitResult` | Gated |
| Pairing trigger in `SubmitResult` | Gated |
| AutoStart loop | Same at lag 0 (plus the new event) |
| `PairRound` basis, initial standings, `canCatch` | Same at lag 0 |
| `prepair` | Same at lag 0 |
| `IsRoundReady` / `IsRoundStartable` | Same at lag 0 |
| `SetReadyForGame` open rounds and per-player gate | Same at lag 0 |
| `IsFinished` | Gated |
| `GetStandings` gibsonization scan | Gated |
| `AddPlayers` / `RemovePlayers` re-pairing | Gated |
| Validation | Only fires when lag > 0 |
| COP request builder | Same at lag 0 |

### New helpers

- `pairingLag(round int) int` returns 0 for out-of-range rounds.
- `pairingBasisRound(round int) int` returns `round - 1 - pairingLag(round)`.
- `hasLaggedRounds() bool` returns true if any round's lag is above 0.
- `completePrefix() int` returns the largest k with rounds 0..k complete, or -1.
- `firstOpenRound() int` returns `CurrentRound - pairingLag(CurrentRound)`.
- `pairNewlyPairableRounds(prefixBefore int) (*pb.DivisionPairingsResponse, error)`
  runs only in lagged divisions. For each unstarted round R in
  `[CurrentRound+1, min(CurrentRound+2, n-1)]`, in ascending order, it calls
  `PairRound(R, false)` when all of these hold:
  - `prefixBefore < B(R) <= completePrefix()`, meaning the basis just became
    complete.
  - R depends on standings and is not MANUAL.
  - R-1 has pairings.

  Ascending order matters. Completing round k can make both k+1 (lag 0) and
  k+2 (lag 1) pairable, and k+2 must see k+1's pairings.

### `SubmitResult` (`:449`)

- The non-amendment past-result check (`:508`) is
  `round < CurrentRound && !amend`. It becomes
  `round < firstOpenRound() && !amend`, which is the same check when lag is 0.
- Standings refresh (`:650`): lagged divisions run through `CurrentRound+2`,
  because a lagged round two ahead may already be paired. Other divisions keep
  `CurrentRound+1`.
- Pairing trigger (`:661-676`): lagged divisions record
  `prefixBefore := t.completePrefix()` before writing the result, then call
  `pairNewlyPairableRounds(prefixBefore)`. Other divisions keep today's
  "pair `round+1`" block exactly as it is. Both keep the `!finished && !amend`
  guard.
- AutoStart (`:669`) changes from a single `StartRound(true)` to "while
  `IsRoundStartable() == nil`, `StartRound`". With lag 0, at most one round
  can ever be opened here, so the loop does the same as today.

### `PairRound` (`:709`)

- Standings (`:746-754`): use `GetStandings(B(round))`. Round 0 keeps today's
  `standingsRound = 1` path. A lagged round 1 (basis -1) needs initial
  standings (all zeros, ordered by seed) even after round 0 results exist, so
  add a `getInitialRecords()` path for that case only.
- Repeats (`:756`): keep `getRepeats(t, round-1)`. The same applies to
  Australian Draw `getRepeatRounds`. Both read pairings, not results, so they
  already include the in-progress R-1 matchups.
- Gibsonization (`canCatch` at `:849`): remaining rounds must count every
  round whose results are not in the standings. Pass `B(round)+1` instead of
  `round`.
- COP (`:763`): pass the basis round through to `pairRoundWithCOP`. See
  [COP](#cop).

### `prepair` (`:220`)

- Today it pairs round 0 plus the standings-independent rounds. It should
  also pair any non-manual round whose basis is negative. That is only round
  1 (0-based), and only when it is lagged.

### Opening rounds

- `IsRoundReady` (`:1612`): require rounds `0..round-1-lag(round)` complete,
  instead of `0..round-1`.
- `IsRoundStartable` (`:1640`): today it requires `CurrentRound` to be
  complete. Instead, require round `CurrentRound - lag(CurrentRound+1)` (and
  so all earlier rounds) to be complete. Keep the
  `TOURNAMENT_ROUND_NOT_COMPLETE` error, reporting the earliest required round
  that is incomplete.
- `StartRound` is unchanged.

### `SetReadyForGame` (`:1694`)

- Replace `CurrentRound != round` (`:1702`) with
  `round < firstOpenRound() || round > CurrentRound`.
- Add the per-player gate. If `round-1 >= firstOpenRound()` and the player's
  `round-1` pairing has no outcome, reject with a new
  `TOURNAMENT_PREVIOUS_GAME_UNFINISHED` error. This can only happen in a lagged
  round.

### `IsFinished` (`:1804`)

- Lagged divisions use `completePrefix() == len(Matrix)-1`. Other divisions
  keep "last round complete". `possiblyEndTournament` (`tournament.go:1314`)
  depends on this.

### `GetStandings` gibsonization (`:1558`)

- In lagged divisions, the backward scan for `lastCompleteRound` becomes
  `completePrefix()`. That way, rounds that are complete only because results
  were entered out of order never feed gibsonization.

### `AddPlayers` / `RemovePlayers` (`:1241`, `:1367`)

- Both re-pair future rounds with
  `i == CurrentRound+1 || IsStandingsIndependent(pm)`. In lagged divisions,
  also re-pair `i == CurrentRound+2` when it is lagged and already paired.
  Keep ascending order.
- In `AddPlayers`, the past-round forfeits for new players (`i <= CurrentRound`)
  already cover an open `CurrentRound-1`.

### Validation (`SetRoundControls`, `validateRoundControl` at `:2334`)

These checks only fire when `pairing_lag > 0`:

- `pairing_lag > 1` is an error.
- Lag on round 1 (0-based round 0) is an error.
- Lag on an ELIMINATION round is an error.
- Lag on a COP round that is one of the division's last two rounds
  (`rc.Round >= totalRounds-2`) is an error. Until COP lag support lands
  (phase 3), lag on any COP round is an error.

`SetSingleRoundControls` (`:256`) goes through `validateRoundControl`, so
mid-tournament edits get the same checks. Add the new `WooglesError` codes to
`api/proto/ipc/errors.proto`, with frontend text in
`liwords-ui/src/store/constants.ts`.

### Service layer (`pkg/tournament/tournament.go`)

- `SetResult` (`:1154`): note `GetCurrentRound()` before calling
  `SubmitResult`. After `ts.Set`, call `sendDivisionStart` for each round
  AutoStart opened. No interface change is needed.
- `SetSingleRoundControls` (`:290`): no re-pairing (decision 8). Keep the
  `round >= CurrentRound+1` guard.
- `startDivisionChecks` (`:1366`) keeps `round == CurrentRound+1`. The real
  change is in `IsRoundStartable`.
- `PairRound` / `DeletePairings` keep the `round >= CurrentRound+1` guard.
- `HandleTournamentGameEnded`: no change. The game carries its own round, and
  the division now accepts results for open rounds.
- `export.go` (`:117`, `:195`, `:380`) iterates through `CurrentRound`. Check
  that an incomplete `CurrentRound-1` exports as "no score yet", the same way
  an incomplete `CurrentRound` does today.

## COP

### What TSH does today

TSH's COP client (`COP.pm`, `Run`) already sends lagged requests to woogles'
`pair_service.PairService/HandlePairRequest`. For "pair round R based on round
S", it sends:

- `division_pairings` for every round through the last paired round (R-1).
- `division_results` only through the based-on round S.
- `gibson_spread`, `hopefulness_threshold` and `control_loss_threshold`
  chosen from the per-round config arrays by
  `rounds_remaining = number_of_rounds - last_paired_round0 - 1`. That is the
  number of rounds left to pair.

The engine then derives two counts:

- **Games remaining**, `Rounds - len(DivisionResults)`
  (`standings.GetRoundsRemaining`). Used for the sim length, gibsonization,
  can-catch, cumulative gibson spread, the factor ceiling, and the
  final-round and penultimate-round checks.
- **Rounds left to pair**, `Rounds - CompletePairings`. Used for the
  last-quarter policies and control-loss activation.

Woogles keeps those counts as they are and builds its requests the way TSH
does.

### Woogles request (`pkg/tournament/utils.go:26`)

- `TournamentDivisionToCOPRequest` takes the basis round in addition to the
  round being paired.
- Pairings for rounds `0..R-1`: unchanged.
- Results only for rounds `0..B(R)`. Today `:53-68` emits results for every
  round below R, and a round with no results becomes 0-0 scores. At lag 0 the
  output is unchanged, since `B(R) = R-1`.
- Gibson spread and hopefulness are already indexed by
  `nrounds - (round+1)`, which counts rounds left to pair, the same as
  `COP.pm`. No change.
- Basis rounds are always complete in woogles. So `COP.pm`'s "partial results"
  warning and its 0-for-pending-score case never apply.
- `COPGibsonization[R]` is still stored and applied as today. When R is paired,
  R-1 has no results, so the standings through R match the standings through
  B(R) that COP ranked.

### Engine changes (`pkg/pair`)

The engine currently treats the in-progress round like any future round. It
factor-pairs that round in the simulations even though its real matchups are
known. The change:

- **Pending rounds.** `P = CompletePairings - len(DivisionResults)`. P is 0 in
  every request today except TSH lagged ones. Woogles sends P = 1. TSH may send
  more, so handle any P.
- **Simulations.** Every simulation entry point plays the P pending rounds
  with their real pairings first, then the generated rounds. The entry points
  are `SimFactorPairAll` → `runParallelSims`, `evaluatePlayerControlLoss` /
  `RunParallelSimForceWinner` → `simForceWinner`, and `RunSimsWithPairings`.
  - Store the pending rounds as player-index pairs, built from
    `req.DivisionPairings[len(results) .. CompletePairings-1]`. Simulation
    pairings are rank indices, and standings are re-sorted after every
    simulated round. So translate player indices to current ranks right before
    simulating each pending round.
  - `assignPairingsForSegment` still generates rows for every remaining game.
    Rows below P are replaced by the pending rounds. Row i keeps the factor
    `min(roundsRemaining - i, maxFactor)`. That gives the first generated row
    (i = P) a factor equal to the rounds left to pair, which is what it is.
  - Byes in a pending round, meaning a player paired with themselves (and
    with the evener in the simulation), are always simulated as a win for that
    player. Use a non-tie spread, the same as forced winners in `simRound`.
    Future-round byes in the simulation keep today's coin flip.
  - Skip removed players the same way `CreateInitialStandings` does.
- **Control-loss sims.** `simForceWinner` applies no vsFirst or
  factor-opponent swap on pending rows, because their pairings are fixed.
- **Factor 3** (`computeFactor3ForcedPairings`): skip it when P > 0, and log
  "Factor 3 skipped: round in progress". With P > 0, two games remaining means
  the round being paired is the final round, not the penultimate one that
  Factor 3 models. Woogles never sends such a request. For TSH, this stops
  Factor 3 from firing on a lagged final round.
- **Everything else** is unchanged: the round counts, KH, top-4 lock,
  control-loss thresholds, the back-to-back (BB) rule and the repeat and bye
  counts. BB and repeats already read every paired round, including the
  pending one. `verifyreq` already accepts `len(results) < len(pairings)`.
- **Logging.** Add "Pending rounds: P" and the pending pairings to the COP log.
- **Requests with P = 0** follow exactly the same code path as today.

### Woogles validation

A COP round that is one of the last two rounds cannot be lagged. This is
checked in `validateRoundControl` only, not in the engine and not in
`verifyreq`.

## Frontend changes (`liwords-ui/src`)

### Director round controls

- `tournament/director_tools/ghetto_tools/round_control_fields.tsx`: add a
  method-independent "Pair from results through" select ("previous round" or
  "two rounds back") next to the pairing-type select (`:118-160`). Disable it,
  with an explanation, for round 1, for Elimination, and for COP rounds that
  are one of the last two.
- `rdCtrlFromSetting` (`:273`): copy `pairingLag` before the per-method switch,
  and repeat the validation.
- `tournament/director_tools/pairing_methods.ts:14` `settingsEqual()` must
  compare `pairingLag`. Otherwise rounds that differ only by lag merge into one
  range.
- `ghetto_tools/round_controls.tsx:512-533`: the
  `roundControlsToDisplayArray` field copy must include `pairingLag`.
- After a director changes the lag of a round that is already paired, show:
  "Round X pairings were not changed. Use Pair round to re-pair it with the
  new setting."

### Derived state

- Add a selector `openRounds(division)` that returns `currentRound` plus
  `currentRound - 1` when `roundControls[currentRound].pairingLag` is 1 and
  that round is incomplete. Derive it from `currentRound`, the round controls
  and the pairing outcomes. No new server field is needed.
- Add a selector `playerActiveRound(division, player)` that returns the
  earliest open round where the player's pairing has no outcome, or else
  `currentRound`.

Both are equal to `currentRound` at lag 0, so lag-0 tournaments render as
today.

### Player-facing

- `store/selectors/tournament_selectors.ts:38-118` `tourneyStatus()`: derive
  the status from `playerActiveRound`, not `division.currentRound` (`:57`).
  The `ROUND_GAME_ACTIVE` check (`:85-94`) should match that round.
- `tournament/ready.ts:25`: send `playerActiveRound` as the round.
- `tournament/competitor_status.tsx`: build the messages from the player's
  active round.
- `tournament/enter_own_scores.tsx:72-115, 158, 254-271`: target the player's
  earliest round with no result, not `currentRound`.
- `store/reducers/tournament_reducer.ts:712-717`: accept ready messages for
  any open round, not only `currentRound`.

### Telling players

`roundControls` are already part of the division data every viewer receives,
so this is frontend only.

- On the pairings page header for a lagged round, show "Paired from results
  through round X" (X = `B(R)+1`, 1-based).
- In `competitor_status.tsx`, when a player finishes a game and the next round
  is a lagged round that is already open, show "Round N is open. Its pairings
  are based on results through round N-2." When a lagged round opens while the
  player's previous game is still going, show "Finish your round N-1 game to
  start round N."
- In the director's round-controls summary and the actions panel, show
  "(lagged)" next to lagged rounds.

### Pairings, standings, actions panel

- `tournament/pairings.tsx:306`: show Ready, Watch and Resume buttons for
  every open round. At `:576-581` and `:597`, style an open
  `currentRound - 1` as in progress, not completed.
- `tournament/standings.tsx`: mark the standings for open rounds as partial.
- `tournament/actions_panel.tsx:271-274`: when `pairingsTentative`, treat
  pairings as final once the basis round is complete, not only when
  `selectedRound <= currentRound`.
- `store/constants.ts:421`: reword the 1038 text so it reads correctly when the
  blocking round is two back.

### Frontend tests to update

`store/reducers/tournament_reducer.test.ts:135, 160` and
`tournament/standings.test.tsx:87-91`.

## Tests

### `pkg/tournament/tournament_classic_test.go`

- **Compatibility.** Load a pre-change division JSON fixture (add one under
  `pkg/tournament/testdata/`) and play it to the end. It must behave
  identically. All existing tests must pass unchanged.
- **Pairing trigger.** A lagged round R is paired the moment R-2 completes,
  with R-1 still pending. The standings used match those of R-2, even though
  some R-1 results are already in.
- **Repeats and byes.** Lagged round R does not repeat R-1 opponents (KotH with
  a small field forces this). Bye history includes R-1 byes.
- **Interleaved lags.** With lag 0 and lag 1 interleaved, completing round k
  pairs both k+1 (lag 0) and k+2 (lag 1), in that order.
- **Out-of-order entry (IRL mode).** Final-round results entered before the
  last penultimate-round result do not finish the division, and the late
  result is still accepted. A round made complete by out-of-order entry does
  not trigger pairing until the earlier round completes.
- **Result window.** `SubmitResult` without `amend` succeeds for an open
  `CurrentRound-1` and fails for a closed earlier round.
- **Opening rounds.** `IsRoundStartable` allows opening R while R-1 is
  incomplete only when R is lagged.
- **Ready gate.** `SetReadyForGame` rejects round R for a player whose R-1
  game has no result, and accepts it once the result is in.
- **Lagged round 1.** It is paired in `prepair`. Re-pairing it after round 0
  results are in still uses initial standings.
- **Gibsonization.** `canCatch` counts the extra unplayed round.
- **Player changes.** AddPlayers and RemovePlayers mid-tournament, with a
  lagged round already paired two ahead.
- **Validation.** Lag is rejected on round 1, on Elimination, on COP rounds in
  the last two rounds, and with values above 1.
- **AutoStart.** It opens a lagged round while its predecessor is in progress,
  and the service sends `TournamentRoundStarted`.
- **Changing lag.** Changing the lag of a paired round leaves its pairings
  untouched.
- **COP.** A lagged COP round sends results through B(R) and pairings through
  R-1.

### `pkg/pair/cop`

- **P = 0.** Existing scenarios produce identical output.
- **Pending matchup.** With P = 1, a scenario where the pending matchup decides
  hopefulness or control loss. For example, 1st plays 2nd in the pending
  round, so the sims must not factor-pair them apart.
- **Pending bye.** With P = 1, a pending bye always counts as a win in the
  sims.
- **Factor 3.** It is skipped when P > 0.
- **P = 2.** A TSH-style request with P = 2 works.

## Phases

1. **Backend core.** The proto field, helpers, `SubmitResult`, `PairRound`,
   gates for opening rounds and readying, `IsFinished`, the AutoStart event,
   validation (rejecting lag on every COP round for now), and tests. Usable
   through the API or Bruno.
2. **Frontend.** The director lag control, open-round handling for players,
   and telling players.
3. **COP.** The engine's pending-round simulations, the Factor 3 skip, the
   request builder passing results through B(R), and the scenario tests. Then
   narrow the COP validation to "not the last two rounds".
