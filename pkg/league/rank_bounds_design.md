# Possible-rank bounds: design rationale

`CalculatePossibleRanks` shows each league player a "possible final rank" range,
`bestRank..worstRank`. This file documents the guarantees, the algorithm, the
deliberate tradeoffs, and -- importantly -- the approaches that were tried and
rejected, so a future maintainer does not re-walk the dead ends.

## Reader's guide

This section is the whole story in plain language; the rest of the file is the
precise reference.

**The problem.** Each player in a league division sees a "possible final rank"
range, like "you can still finish anywhere from 3rd to 7th". Some games in the
division are still unplayed, so the final order is not yet fixed; the range says
which finishing positions are still reachable for that player. Users treat the
range as a *promise*, which forces two non-negotiable properties:

- **Soundness:** the player's actual final rank must always be inside the shown
  range. We may show a range that is too wide, but never one that excludes a rank
  they could really finish in.
- **Monotonicity:** as games finish, the range may only *narrow*. Showing "3rd to
  7th" today and "2nd to 7th" tomorrow is forbidden -- a range that grows reads as
  the promise being broken. (Showing "1st to last" is always safe but tells the
  user nothing; the job is to be tight *and* never break the two rules.)

Everything else in this file exists to make the range as tight as possible
without ever breaking those two rules.

**The exact-but-slow method (brute force).** The honest way to compute the range
is to consider every way the unplayed games could turn out and keep the player's
best and worst rank across all of them. Each unplayed game has three outcomes --
win, draw, or loss -- giving `3^(unplayed games)` combinations: fine for a handful
of games, hopeless past about ten, so brute force is used only for small
situations. But win/draw/loss is not the whole story -- players who tie on points
are separated by score spread, and spread depends on the *margins* of the
remaining games, both relative to the spread each player already has and to each
other (a game's margin helps the winner exactly as much as it hurts the loser, so
shared games couple players). We do **not** enumerate margin sizes on top of the
`3^g` outcomes -- that is unbounded, and sampling a few sizes is unreliable (see
the joint-margin section). Instead, for each of the `3^g` outcomes we check which
spread orderings among the tied players are actually *achievable*. So the cost is
`3^g` outcomes each times a cheap feasibility check -- not `3^g` times a number of
margins.

**Cheaper safe methods for bigger situations.** When there are too many unplayed
games to enumerate, we fall back to cheaper methods that are still *safe* (never
exclude a reachable rank, never widen later), even if they sometimes report a
range slightly wider than the true one:

- **The "frontend" bound** is the simplest possible safe answer: assume the
  player wins all their remaining games while everyone else loses all of theirs
  (for the best rank), and the reverse (for the worst rank), ignoring who
  actually plays whom. It is called the *frontend* bound because it is exactly
  what the web frontend could compute on its own, knowing only each player's
  points and games remaining -- no pairing data. It is the loosest of the three,
  but trivially safe and only ever tightens.
- **The middle tier (max-flow / branch-and-bound)** is smarter but still cheap.
  The key sub-question for the worst rank is "what is the fewest rivals who could
  still leapfrog this player?" That turns out to be a network-flow feasibility
  question, and solving it *exactly* is a branch-and-bound search. This tier is
  tighter than the frontend bound and almost as good as brute force, used in the
  medium range where brute force is too slow.

**Why this stays monotone.** We pick which method to use from the size of the
player's *cluster* -- the group of players whose order relative to one another is
still in doubt. A cluster can only *shrink* as games finish, never grow, so we
only ever switch from a looser method to a tighter one, exactly once, and never
back. That is what guarantees the range never widens.

**What was tried and rejected, and why:**

- **A greedy shortcut for the leapfrog count** (a fast approximation instead of
  the exact branch-and-bound). Rejected: it is sometimes wrong in a direction
  that makes the range *grow* in a later snapshot, which breaks monotonicity. The
  exact search is used instead, kept cheap by only running it on small-enough
  clusters.
- **Remembering the last range and refusing to widen.** Would give monotonicity
  trivially, but needs new database columns to store prior ranges -- a schema
  migration and downtime. Rejected; the cluster-shrink argument gives
  monotonicity for free, with no stored state.
- **A permutation ("Hall") post-pass:** a clever extra tightening -- across the
  whole division, if a rank can belong to only one player, pin it to that player,
  and repeat. It is correct, cheap, and safe, but when measured on real league
  data it recovered *nothing* (the other tiers already capture that tightness in
  practice), so it would be pure dead code. Rejected.
- **Capping the exact search by work or time.** Stopping branch-and-bound after a
  fixed number of steps, or when a timer trips, bounds the cost but is either
  non-monotone (the cap is not tied to cluster size) or non-deterministic (the
  same situation could get the tight answer on one page load and the loose one on
  the next). Rejected; the size-based gate is the deterministic, monotone way to
  cap cost.

**One subtlety even brute force got wrong.** The exact brute force originally
reasoned about score spread one pair of players at a time, which misses that two
players sharing a game cannot both swing its margin their own way. That made even
the "exact" method occasionally too wide. The fix (the joint-margin section
below) reasons about the shared spread jointly.

**If you change anything here,** re-run the verification harness (last section).
It checks both hard guarantees mechanically against real game histories and an
independent oracle -- the safety net, since the algorithm is intricate enough
that code review alone will not catch a subtle break.

## Guarantees

- **Soundness (HARD).** `displayed_best <= actual_best <= actual_worst <=
  displayed_worst` at all times. Whatever results the unfinished games take
  (under the spread model below), the player's real final rank is always inside
  the displayed `[best, worst]`. Both ends are correct; we never exclude an
  achievable rank.
- **Monotonicity (HARD).** As games complete, the range may only *tighten*. It
  must never widen (e.g. show `worst=7` then later `worst=8`). This is the
  feature's promise -- users read the range as a guarantee, so widening it is a
  broken promise, not a cosmetic glitch. (The trivial `1..n` is always sound but
  useless; the point is to be tight *and* monotone.)
- **Tightest-feasible (SOFT).** Subject to the two hard guarantees, the range
  should be as narrow as possible. A too-wide but sound+monotone range is
  acceptable; a tighter range that is ever unsound or non-monotone is not. When
  the two conflict, tightness yields.

## Normal-termination spread model (deliberate)

A decided game contributes a **margin of at least 1, strictly**: the winner
gains strictly positive spread, the loser strictly negative; only a draw is
zero. The bounds reason about spread under this model.

We deliberately do **not** model the timeout edge case where a leading player
times out and loses *with positive spread* (a negative-spread "win" for the
opponent). Timeouts are rare and discouraged; in that rare case the displayed
range may be slightly too tight. Do not "fix" the spread logic to cover it
without revisiting this decision -- it is a choice, not an oversight.

- A cheater penalty that adjusts scores so the cheater loses by some margin is
  consistent with this model (the winner keeps positive spread).
- A planned time-bank-auto-pass mechanic (a timed-out player auto-passes on a
  fresh turn; forced adjudication after two weeks uses the current spread, so the
  leader wins) would make normal play exact -- no negative-spread wins.

## The bounds as extremal achievable ranks

Define `achievable(P, r)` = "there is some completion of the unfinished games
(results and margins, under the spread model) in which player P finishes in rank
`r`". Then:

    bestRank(P)  = smallest r with achievable(P, r)
    worstRank(P) = largest  r with achievable(P, r)

Conceptually you could compute either end by scanning from the extreme:
`while !achievable(P, best) { best++ }` and
`while !achievable(P, worst) { worst-- }`. Two independent verification oracles
evaluate `achievable` exactly: an exhaustive enumeration (the indisputable
ground truth, tiny cases only) and a Fourier-Motzkin feasibility test (which
scales to the whole brute regime). Production does **not** scan rank-by-rank:
each tier computes the extremal achievable rank in one shot (brute by
enumeration, B&B by minimum eviction), which is cheaper. Every tier and oracle
is a different way to decide the same `achievable`, so they cross-check each
other.

## Clusters: the monotonicity linchpin

Partition players into **clusters** -- maximal groups whose relative order is
still undetermined (`buildBruteForceClusters`). A cluster occupies a *consecutive
rank band* `[1+crossAbove, n-crossBelow]`; players outside it are rank-disjoint
(guaranteed all-above or all-below by points alone). Members share the band but
have different individual sub-ranges, so a cluster is **not** "everyone is
`1..size`".

A cluster's size -- its member count and its unfinished-game count -- only ever
*shrinks* as games complete: components can only split, and the point ranges
`[pts, pts + 2*remaining]` only tighten, so non-overlapping stays
non-overlapping. This is what makes a size-based gate monotone: it flips from the
looser tier to the tighter tier exactly once and never back.

## Three-tier dispatch (per cluster)

Each cluster is handled by one of three algorithms, chosen by size:

| tier     | when (per cluster)            | soundness | tightness                        |
|----------|-------------------------------|-----------|----------------------------------|
| brute    | unfinished games <= 10        | exact     | exact in both directions         |
| B&B      | local candidate count k <= 13 | sound     | exact eviction; flow-model loose |
| frontend | otherwise (large clusters)    | sound     | loosest (pairing-agnostic)       |

- **brute** enumerates the `3^games` win/draw/loss outcomes; within each it does
  *not* enumerate score margins (unbounded, and sampling is unreliable) but checks
  which spread orderings of the tied players are achievable (the joint-margin
  reasoning below) -- so it is `3^games` leaves times a cheap per-leaf feasibility
  check. Exact: it captures spread coupling *and* the cross-player permutation
  constraints the per-player heuristics miss. Affordable only while `3^games` is
  small, hence the `<= 10` cap.
- **B&B** (branch-and-bound, primer below) models the bound as a minimum eviction
  over P's candidate set and solves it exactly. Polynomial per node but
  exponential in the worst case, so gated by `k`. Sound; looser than brute only
  because the underlying flow model cannot represent every joint-margin
  realization and ignores permutation constraints -- never unsound in the
  achievable direction.
- **frontend** is the pairing-agnostic bound (named for the bound the web
  frontend can compute on its own, from points and games-remaining only):
  `best = 1 + |{Q: pts_Q > P_ceiling}|`, `worst = n - |{Q: Q_ceiling < P_pts}|`
  (a ceiling is `pts + 2*remaining`). Loose but sound, monotone (the counts only
  grow), and provably wider than the B&B range in both directions -- so the
  loose->tight flip as a cluster shrinks past the gate only ever tightens.

The brute/B&B/frontend gates are a **performance budget**, not correctness
constants: the thresholds are chosen so the worst single division-snapshot stays
about as fast as today's brute worst case (machine-independent via the ratio of
B&B-worst to brute-worst). Lowering a gate only loosens the result (still
sound+monotone); it never makes a wrong answer.

The thresholds are measured: brute runs at `<= 10` unfinished games per cluster
(`bruteForceThreshold`), and the B&B at cluster-local `k <= 13`
(`localCandidateGate`). Over the full game-history replay the worst single
division-snapshot is ~29ms and lands on a *B&B* cluster; the brute worst case (a
bunched `m=26`, `g=10` cluster) is ~24ms after its per-leaf counting sort, so the
budget is B&B-bound, not brute-bound. `k = 14` hits a single-player B&B blow-up
(a fully-bunched division), which is why the gate sits at 13; the brute stays at
`<= 10` because `3^11` leaves explode regardless of the per-leaf speed. Replaying
every completed division-season in completion order, the monotonicity violations
drop from 1220 (the reported bug) to 0, with zero final-result mismatches.

### What `k` is (the B&B gate metric)

`k` is P's **candidate count**: the number of *other* players whose final order
relative to P is still genuinely undetermined -- they could finish above or below
P depending on the unfinished results. (Guaranteed-above and guaranteed-below
players are not candidates.) For worst-rank, candidates are the players who could
be *forced above* P; that count drives the eviction search, so the B&B cost is
governed by `k`, which is why the gate is on `k`.

- **local k:** counted within P's own cluster only. Cross-cluster players are
  rank-disjoint, so they would inflate `k` while adding ~zero eviction cost;
  excluding them gates on the true cost driver.
- **unified across both ends:** via the inversion below, best-rank uses the same
  candidate count on a transformed view, so one metric
  `k = max(k_above_real, k_above_mirror)` gates both directions. It is monotone
  both ways (shown below), so the gate stays monotone.
- `k` is bounded by the cluster size, which only shrinks over the season. There
  is no assumed maximum division size; the gate copes with any `k` (above the
  threshold falls back to frontend).

### Primer: branch-and-bound (B&B) minimum eviction

The worst rank P can reach is `(players guaranteed above P) + (minimum number of
candidates that can be forced above P) + 1`. "Forced above" means a feasible
assignment of the unfinished results exists in which those candidates all end up
above P. Finding the *minimum* such set is the hard part:

- It is a minimum-vertex-cover-like problem (each unfinished game between two
  candidates is a constraint coupling them), which is **NP-hard** in general.
- A **greedy** approximation (repeatedly evict the highest-gain candidate) is
  fast but sub-optimal, and -- critically -- its error is **not monotone** in
  cluster size, so greedy bounds can *widen* between page loads. Rejected (see
  below).
- **Branch-and-bound** solves it exactly: test feasibility with a max-flow; if
  infeasible, find a *minimal infeasible subset* of candidates and branch on
  evicting each member of just that subset (a small branch factor), recursing.
  Exact, and bounded to the gated regime (`k <= 13`) so the exponential worst
  case never runs in production.

The feasibility test is a max-flow; B&B wraps it; the rank routine wraps the B&B.

### Primer: the max-flow feasibility test (what the graph is, why it works)

Max-flow is used in exactly **one** place: the B&B (medium) tier. Brute uses
enumeration plus the closed-form conservation rule, the frontend tier is pure
counting, and the verification oracle is an LP -- none of them use max-flow.

The question the flow answers (worst-rank case) is: *can this set of candidates
all stay at or below P on points?* Model it as routing the points that the
unfinished games will award:

- a **source** and a **sink**;
- one **game node** per unfinished game between two candidates, fed from the
  source by an edge of capacity 2 (the two league points at stake in a game);
- each game node feeds its **two player nodes** (the candidates playing it),
  capacity = the points that player could take from the game (a win is 2, a draw
  1);
- each player node drains to the sink with capacity = that player's **slack**:
  how many more points it can gain and still stay at or below P. (This is where
  the per-candidate spread tiebreak is folded in: a candidate that would tie P on
  points is kept below or not according to current spread.)

The candidates can all stay below P iff the **max-flow saturates every game
edge** (total flow = `2 * games`): every game's points find a home without
pushing any candidate past P. If the flow falls short, the **min-cut** names a
set of games whose points cannot all be absorbed -- so at least one of those
candidates is forced above P. That cut yields the minimal infeasible subset the
B&B branches on. (It is the standard max-flow/min-cut, i.e. transportation /
Hall, argument: infeasibility = an over-subscribed candidate set.)

**The flow models points, not joint spread.** It uses each candidate's current
spread only for the per-candidate tiebreak (the slack above), not the joint
zero-sum coupling of shared-game margins that brute handles. That coupling only
bites when P is *fixed* (no games left) and tied inside a *closed* group, which
in a medium/large cluster is rare (P usually still has games to move its own
spread) and is exactly what the brute tier (`<= 10`) covers exactly. So the B&B
bound is sound and, in practice, tight on spread; any residual looseness is
measured against the oracle and revisited only if material. Modeling joint spread
in the flow does not fit cleanly anyway -- spread matters only among players tied
on points, and this tier deliberately does not fix the points outcome -- so it is
left to brute.

### Why invert for best-rank instead of computing it directly

Best-rank is the mirror of worst-rank, and we compute it *as* one rather than
writing a second eviction routine -- for a concrete monotonicity reason, not just
code reuse:

- Computing best-rank *directly* needs its own candidate set (players who could
  be forced *below* P). That count keys off P's point **ceiling**
  (`pts + 2*remaining`), which *falls* over the season as P's games resolve -- so
  previously-guaranteed-below players can re-enter candidacy and the count can
  **rise**. A rising candidate count is non-monotone: a `k`-gate on it could flip
  tight->loose = widen = break the hard guarantee.
- The fix is the **win/loss inversion**. With **loss-score = 2L + D** (not just
  L), `points = 2W + D = 2G - (2L + D)` exactly (`G = W + D + L`, constant per
  player at season end; draws included). So ranking by points descending is
  identical to ranking by loss-score ascending. Best-rank-by-points therefore
  equals worst-rank-by-loss-score on a mirrored view (points -> loss-score,
  spread negated, rank flipped):

      bestRank(P) = n + 1 - worstRankExact(P, mirror)

  where the mirror sends `points -> 2*expected - points - 2*remaining` and
  `spread -> -spread`. On the mirror, best-rank keys off P's loss **floor**,
  which *rises* over time -- so its candidate count is **monotone**. One routine,
  monotone in both directions.

## Tightness by tier (when the sound bounds are also exact)

The bounds are always sound and monotone. They are *additionally* exact (as
tight as reality allows) depending on which tier the player's cluster falls in:

- **brute (cluster <= 10 games): exact in both directions.** After the
  joint-margin fix below, the enumeration realizes every legal outcome including
  the joint spread coupling of shared games and the cross-player permutation
  constraints.
- **B&B (local k <= 13): sound, exact eviction, but the flow model can be
  slightly loose.** The flow cannot represent every joint-margin realization and
  ignores permutation constraints, so the bound may be a little wider than
  reality -- never narrower (never unsound). The one direction where the flow
  model *can* be unsound (an over-tight best-rank in a tiny cluster) is covered
  by the brute tier (`<= 10`), which always wins there.
- **frontend (large clusters): loosest, but still sound + monotone.** Early in a
  season, when everyone is bunched in one big cluster, players land here and the
  range is close to the whole cluster band. As games finish and the cluster
  shrinks past the gates, each player's range tightens monotonically
  (frontend -> B&B -> brute), never widening.

So tightness improves over the course of a season, monotonically, as clusters
shrink -- which is exactly when users care about a precise range (near the end).

## Exact-brute joint-margin fix

The brute is the exact ground truth, but the original `spreadOrdering` reasons
about spread **per pair** of players -- effectively giving each player an
independent spread range -- which **misses the joint zero-sum coupling of shared
games**. Two players who share a game cannot both push that game's margin their
own way, so the per-pair brute is sound but can be too *wide*.

The fix reasons about spread **jointly** within a tied group:

- A player's reachable spread is a box `[lo, hi]` around the spread it already has
  (`start`): any win makes `hi = +inf`, any loss makes `lo = -inf`; otherwise
  `lo = start + wins`, `hi = start - losses`. We test *feasibility* of the needed
  spread ordering within these boxes -- we never enumerate concrete margin sizes.
- Within a **closed** component (every game of every member is internal to the
  tied group) the total spread is **conserved**, so not all members can sit at
  their extreme at once: the number that can stay `<= P` is limited by the
  conserved sum subject to the per-player boxes. (Example: a closed tied
  round-robin whose spread sum forces at least one member above P -- invisible to
  the per-pair view.)

This only bites when P itself is **fixed** on the relevant side (no remaining
game to push its own spread to the extreme) and is tied with a closed coupled
set; otherwise the per-pair view is already tight. It is verified against the
independent oracle below.

## Approaches considered and REJECTED

- **Clamp against stored prior bounds** (persist each player's last range, never
  widen vs it): needs new DB columns = a migration = downtime. Rejected; the
  cluster-shrink monotonicity argument needs no storage.
- **Greedy eviction (no exact solve):** sub-optimal, and its error is not
  monotone in cluster size, so it produces widening (non-monotone) bounds.
  Empirically matched the exact bound only up to ~19 unfinished games and first
  wobbled at 20. Dominated by gated B&B; dropped entirely.
- **Brute force above ~10 games:** `3^games` explodes (`3^19 ~ 1.2e9`).
- **B&B with a work-cap** (stop after N nodes): bounds time but BREAKS
  monotonicity -- the node budget is not monotone in cluster size, so the bound
  can widen.
- **B&B with a depth-cap:** monotone, but does not tame the blow-up -- the branch
  factor is unbounded, so `depth^branch` still explodes.
- **Dynamic time-budget gate** (run B&B, fall back when a timer trips):
  non-deterministic, hence non-monotone -- the same cluster could get the tight
  tier on one page load and the loose fallback on the next (server load),
  widening the range between loads.
- **Naive candidate-count gate:** the best-rank candidate count is non-monotone
  (see inversion above). Fixed by the loss-score inversion, which makes it
  monotone; only then is candidate-gating sound.
- **Hall / alldifferent post-pass:** a cheap, sound, monotone division-wide
  fixpoint that recovers cross-player permutation tightness the per-player tiers
  miss (if only one player's range contains rank `r`, force it; generalizes to
  size-w Hall intervals). Correct and fully implemented, but **measured zero
  reclaim** on the real corpus -- the per-player B&B/frontend bounds already
  capture all the permutation tightness that actually occurs -- so it would be
  dead code. Dropped. (The brute tier still gets permutation tightness for free
  via real enumeration.)
- **Finite-magnitude "brute of brute" oracle:** enumerating concrete margin
  magnitudes from a fixed set is unreliable -- too coarse a set is silently
  too-tight, and you can never prove the set is rich enough. Replaced by the
  exact Fourier-Motzkin oracle below.

## Verification

All env-gated (`LEAGUE_REPLAY_CSV`), skipped in CI; CI keeps synthetic regression
tests with no data file.

- **Exhaustive oracle:** for tiny divisions, enumerate every win/draw/loss
  outcome and every integer margin in `[1..B]` and read off the rank range --
  the indisputable ground truth.
- **Fourier-Motzkin oracle:** an independent exact `achievable(P, r)` via
  Fourier-Motzkin feasibility over the game margins (exact integer arithmetic;
  boolean feasibility, no witness). It scales past the exhaustive limit, is
  validated to match the exhaustive ground truth, and in turn validates the
  production brute on random small divisions.
- **Replay harness** (`monotonic_replay_test.go`): replays real completed
  division-seasons in game-completion order and asserts per-player monotonicity
  (best non-decreasing, worst non-increasing) plus a tie-aware final-result
  check.
- **Pinpoint** (`rank_bounds_pinpoint_test.go`): isolates a single over-tight
  state for debugging.
- **Per-commit metric deltas:** each change in this work is measured to *reduce*
  at least one of {monotonicity violations, soundness violations vs the oracle,
  range width} without regressing the two hard guarantees.
