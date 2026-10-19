# Plan: three-tier fourth-quarter grouping in COP

## Goal

In the fourth quarter, split the field into three tiers that pair within themselves:

1. **Hopeful winners** (W): players still hopeful to finish 1st play each other.
2. **Hopeful cashers who aren't hopeful winners** (M): play each other.
3. **Everyone else** (R): play each other.

## What COP does today

The fourth-quarter grouping lives in the **CC** ("cash contention") weight policy, `pkg/pair/cop/cop.go:921`. It only has two tiers:

- A player is "hopeful" if `ri <= hopeToCashBoundary(pargs)` and they aren't Gibsonized (`cop.go:950`).
- Any edge between a hopeful and a non-hopeful player gets `majorPenalty`, halved when `rj` is the first rank past the boundary (`cop.go:952-962`).
- It's active only when `isLastQuarter && !isFinalRound` (`cop.go:941`). The final round falls back to KOTH-style RD weighting.
- The bottom-6 rule (12+ players) and the odd-1st-group leader bar live in the same policy.

Hopeful winners are a subset of hopeful cashers, so today a hopeful winner can freely play a casher who can't win. The only rules that keep winners together are narrow ones:

- **T4** (`computeTop4LockActive`, `cop.go:276`): a hard lock, only in the 2nd-to-last round, and only when exactly 4 players are hopeful for 1st.
- **The odd-1st-group bar** (`computeDisallowedLeaderOpponent`, `cop.go:182`): when the group hopeful for 1st has an odd size, the next player down is barred from playing the leader.
- **PC's inverse-distance pull** (`cop.go:835`): it works per player and uses small weights, so it doesn't enforce any grouping.

## Proposed design

### Tier definitions

All tiers are rank windows, matching how CC already treats the cash window: every rank inside the window counts, not just the players who are individually hopeful.

| Tier | Ranks | Source |
|---|---|---|
| W | `0 .. wTop` | `wTop = LowestPossibleHopeNth[0]`, plus one rank when the odd-1st-group rule promotes a player (see Parity) |
| M | `wTop+1 .. cTop` | `cTop = hopeToCashBoundary(pargs)`, which already includes the Q4 parity promotion, the bye adjustment and the bottom-6 clamp |
| R | everything else | |

Gibsonized players are never in W or M, which matches today's CC. A player is Gibsonized only when they're locked into their exact rank (`standings.GetGibsonizedPlayers`), so a Gibsonized player below the leader can't win, and the only Gibsonized player who could fall inside W is a Gibsonized leader. Excluding Gibsonized players from W is therefore a guard, not a policy choice.

Invariant: W ⊆ C (C is the whole cash window, `0..cTop`). If W's promotion pushes `wTop` past `cTop`, raise `cTop` to `wTop`, which leaves M empty.

When `PlacePrizes == 1`, or whenever `wTop == cTop`, M is empty and the behaviour should match today's.

### Where the change goes

1. Add a `computeQ4Tiers(pargs)` helper that returns `(wTop, cTop)`. Compute it once in `copMinWeightMatching` after `computeDisallowedLeaderOpponent` and `adjustLowestPossibleHopeCasherForBye` have run, and store it on `policyArgs`. This follows the pattern of the other `compute*` fields, so the weight handler doesn't recompute it for every edge.
2. In CC, replace the two-way `riHopeful != rjHopeful` check with a tier comparison:
   - `tier(r)` is W if `r <= wTop`, M if `r <= cTop`, otherwise R, and Gibsonized players always go to R.
   - If `tier(ri) != tier(rj)`, the edge gets `majorPenalty`. Halve it when `rj` is the first rank of its own tier (`wTop+1` or `cTop+1`), which extends today's halving at `cTop+1`.
   - Keep the existing early-outs unchanged: L3, the disallowed leader opponent and the bye. Leave the bottom-6 branch in place, though it can't be reached (see the CC rows in the table below).
3. Log the tiers once per round, e.g. `Q4 tiers: winners ranks 1-4, cashers ranks 5-8`, so the log explains any cross-tier major-penalty edge.

The rule stays a **weight**, not a constraint. CC's doc comment explains why: stacking hard rules can leave a player with no legal pairing. A soft major penalty means the round always pairs, and in the worst case one pairing crosses a tier.

### Parity

Every tier needs an even number of pairable players. Pairable means not Gibsonized and not the bye recipient.

- **W:** `computeDisallowedLeaderOpponent` already detects an odd group hopeful for 1st, after the bye and a Gibsonized leader are taken out, and picks rank `wTop+1` as the extra player. Under the new rule that player **joins W**, so `wTop` grows by one, and stays barred from the leader. Without this, the extra player would be major-penalized against every W opponent and every M opponent at once.
- **C as a whole:** already kept even by `hopefulToCashExtensionRankIdx` (copdata) and `adjustLowestPossibleHopeCasherForBye`.
- **M:** if W and C are both even, then M = C \ W is even too, so M needs no separate fix. This relies on W and C counting pairable players the same way. C excludes every Gibsonized player and the W check excludes only a Gibsonized leader, but those agree, because no other Gibsonized player can be in W (see Tier definitions). No recount is needed.

### Interactions and precedence

| Rule | Interaction | Plan |
|---|---|---|
| CC: hopeful-vs-hopeful branch (`cop.go:949-962`) | This is the branch being changed | Replace its two-way hopeful/non-hopeful check with the three-way tier check, rather than adding the tier check alongside it. |
| CC: odd-1st-group leader bar (`cop.go:932-935`) | The player it bars from the leader is the same player W's parity fix promotes into W | Keep it, and keep it ahead of the tier check: the promoted player joins W but stays barred from the leader, so they play another W member. |
| CC: bottom-6 branch (`cop.go:968-980`) | Unreachable today. The clamp in `hopeToCashBoundary` keeps every hopeful player out of the bottom 6, so a hopeful vs bottom-6 edge always hits the hopeful/non-hopeful mismatch first and returns there | Still unreachable with tiers, because W ⊆ C and C sits above the clamp. Leave it as a safety net; no change. |
| CC: bye early-out (`cop.go:946-948`) | The bye isn't in any tier | Keep it, so the bye isn't major-penalized against everyone. |
| F3 (Factor 3) | Forces 1v4 / 2v5 / 3v6 in the 2nd-to-last round, which may cross the W boundary | When `factor3ForcedPairings` is non-empty, skip the W/M split and fall back to today's two-tier CC. Every other leader-group rule already defers to F3. |
| T4 | A hard version of "W plays W" when W has exactly 4 players | Keep it. The two rules agree. |
| L3 | Forces leader vs rank 2, which may sit outside W | Already exempt at the top of CC. Keep that. |
| CL (control loss) | Forces the leader vs the destiny's child | The destiny's child is always chosen within `LowestPossibleHopeNth[0]`, so it's in W. No conflict. |
| TB / CB (byes) | Remove a player from a tier | Handled by the parity rules above. |
| KH, PP (forced and prepaired pairings) | Can pull one player out of each of two tiers | No special handling: the soft penalty absorbs it, with at most one more cross-tier pairing. Exempt the forced edge itself from CC if it shows up as noise in the logs. |
| PC | For winners at ranks above 0, the contention window `LowestPossibleHopeNth[ri]` can reach into M, so PC mildly prefers a cross-tier opponent | CC's major penalty dominates. No change needed. |
| Retry (`lowestHopeOverride`) | Only expands PC windows | Tiers read `copdata` directly, not the override. No change. |

## Open questions

Each question has a recommended default, so implementation can start without waiting on answers.

1. **Final round.** Should the tiers apply in the true final round, where CC is disabled today in favour of KOTH? *Recommendation: no. Keep the `!isFinalRound` gate.*
2. **Strength.** A soft `majorPenalty` or a hard constraint? *Recommendation: soft (see "Where the change goes").*
3. **F3 precedence.** *Recommendation: skip the W/M split when F3 fires (see the table).*
4. **Definition of "hopeful winner".** *Recommendation: `LowestPossibleHopeNth[0]`, as computed now (this already includes the runaway-leaders halving of the hopefulness bar).*

## Implementation steps

1. Add `wTop` and `cTop` fields to `policyArgs`, plus `computeQ4Tiers`, computed in `copMinWeightMatching` after the existing parity and bye adjustments.
2. Fold the promoted rank from `computeDisallowedLeaderOpponent` into W.
3. Update CC's handler and doc comment to use the three tiers. Keep the bottom-6 and leader-bar logic.
4. Add the tier log line.
5. Update the CC description in the Weights section of the COP Google Doc.

## Tests

Unit tests in `pkg/pair/cop/cop_test.go`:

- **Tier boundaries** from `computeQ4Tiers`:
  - basic W/M/R split
  - `PlacePrizes == 1` (M empty)
  - odd W promotes into W
  - bye recipient in W
  - bye recipient in M
  - Gibsonized leader (left out of W's parity count)
  - W promotion past `cTop` (C raised to match)
  - F3 fired (no split)
- **CC weights per edge type:**
  - zero within W, within M and within R
  - `majorPenalty` for W–M, W–R and M–R
  - halving at `wTop+1` and `cTop+1`
  - existing exemptions (L3, bye, disallowed leader opponent) unchanged
- **End to end with `COPPair`:** fourth-quarter standings with, say, 4 hopeful winners, 4 further hopeful cashers and 4 others, where today's code pairs a winner against a non-winning casher. Assert that every pairing stays within its tier. Outside Q4 and in the final round, assert the old behaviour.
- **Existing tests to re-check:**
  - `TestHopefulCasherGrouping`: should still pass, since its 6-player hopeful half doesn't care about the split.
  - `TestBottomSixHopefulOverlap`
  - `TestOddHopefulContenderGroup`
  - `TestCCWeightExemptsForcedLeaderVsThird`
  - `TestCOPWeights`: pins exact pairings that may change.
  - Scenarios 9–19 (contender parity)

## Validation

- `go test ./pkg/pair/...`
- `COP_SCENARIOS=1 go test ./pkg/pair/cop/ -run 'TestScenario'` (numbered and multiround scenarios)
- For the multiround and hypothetical runs (`TestScenarioMultiRound_*`, `TestScenarioHypo*`), compare before and after:
  - `OVERCONSTRAINED` count, which should stay at 0
  - how often a `majorPenalty` edge appears in the final pairings, which shows how often a cross-tier pairing was unavoidable
  - `Retry:` lines in the logs
