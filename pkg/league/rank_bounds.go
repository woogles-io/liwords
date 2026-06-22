package league

import (
	"math"
	"math/bits"
	"slices"
	"sort"
	"sync"

	"github.com/jackc/pgx/v5/pgtype"
)

// gamePair represents an unfinished game between two players (by standing index)
type gamePair struct {
	a, b int
}

// RankBounds contains the possible finishing rank range for a player
type RankBounds struct {
	BestRank  int
	WorstRank int
}

// CalculatePossibleRanks computes tight rank bounds for all players in a
// division using the actual remaining pairings and max-flow to determine
// simultaneous feasibility.
//
// standings must already be sorted by rank. unfinishedGames contains the
// player0_id / player1_id pairs for games not yet completed.
func CalculatePossibleRanks(
	standings []standingInfo,
	unfinishedGames []unfinishedGame,
) []RankBounds {
	n := len(standings)
	if n == 0 {
		return nil
	}

	// Build player index map: userID → index in standings
	playerIdx := make(map[int32]int, n)
	for i, s := range standings {
		playerIdx[s.userID] = i
	}

	// Parse unfinished games into index pairs
	var games []gamePair
	for _, g := range unfinishedGames {
		a, okA := playerIdx[g.player0ID]
		b, okB := playerIdx[g.player1ID]
		if okA && okB {
			games = append(games, gamePair{a, b})
		}
	}

	// Fast path: partition players into rank-disjoint clusters; if every
	// cluster's unfinished-game count is below the brute-force threshold,
	// enumerate each cluster in parallel for tight bounds. This covers both
	// the simple "total games ≤ threshold" case and the harder "many games
	// but spread across multiple disjoint clusters" case (e.g. a top clique
	// and a separate bottom pair whose pts ranges don't overlap).
	clusters := buildBruteForceClusters(standings, games)
	if maxClusterGames(clusters) <= bruteForceThreshold {
		return bruteForceRanksFromClusters(clusters, standings, games)
	}

	// Otherwise dispatch each cluster by its size. A cluster only ever shrinks as
	// games complete (components split, point ranges tighten), so it crosses each
	// threshold below at most once and only in the tightening direction -- which
	// is what makes the displayed range provably monotone (never widens).
	//
	//   unfinished games <= bruteForceThreshold: enumerate every outcome (exact,
	//     tightest). Per cluster, so a small cluster beside a large one still gets
	//     exact bounds.
	//   else, cluster-local candidate count k <= localCandidateGate: the exact
	//     max-flow branch-and-bound (worst rank directly, best rank via the
	//     loss-score inversion). k bounds the B&B cost, and the flow runs on the
	//     cluster's own games only -- other clusters are rank-disjoint and add no
	//     flow, so this equals a division-wide flow on a smaller graph.
	//   else (large, dense cluster): the pairing-agnostic frontend bound. Loose,
	//     but sound and monotone, and always at least as loose as the B&B bound,
	//     so the loose->tight flip as the cluster shrinks past the gate only ever
	//     tightens the range.
	crossAbove := make([]int, len(clusters))
	for i := range clusters {
		for j := range clusters {
			if i != j && clusters[j].minPts > clusters[i].maxPts {
				crossAbove[i] += len(clusters[j].members)
			}
		}
	}

	// Reusable state for the (sequential) max-flow tier: flow graph (adj slices
	// grow and stabilize) and candIdx (avoids map allocation in inner loops).
	fg := newFlowGraph(2 + len(games) + n)
	candIdx := initSlice(n, -1)
	mir := mirrorForBest(standings) // best rank = worst rank on this mirror
	results := make([]RankBounds, n)
	inCluster := make([]bool, n)
	var wg sync.WaitGroup
	for ci := range clusters {
		c := &clusters[ci]
		if len(c.games) <= bruteForceThreshold {
			// Brute clusters are independent (disjoint members), so enumerate
			// them in parallel; the max-flow tier below stays on this goroutine.
			wg.Add(1)
			go func(ci int) {
				defer wg.Done()
				enumerateBruteForceCluster(&clusters[ci], standings, games, results, crossAbove[ci])
			}(ci)
			continue
		}

		clusterGames := make([]gamePair, len(c.games))
		for j, gidx := range c.games {
			clusterGames[j] = games[gidx]
		}
		for i := range inCluster {
			inCluster[i] = false
		}
		for _, m := range c.members {
			inCluster[m] = true
		}
		for _, m := range c.members {
			gi := decomposeGames(m, n, clusterGames)
			kReal := worstCandidateCountLocal(m, standings, gi, inCluster)
			kMirror := worstCandidateCountLocal(m, mir, gi, inCluster)
			if max(kReal, kMirror) <= localCandidateGate {
				results[m] = RankBounds{
					BestRank:  bestViaInversion(m, mir, gi, fg, candIdx),
					WorstRank: worstRankForPlayer(m, standings, gi, fg, candIdx),
				}
			} else {
				results[m] = frontendBound(m, standings)
			}
		}
	}
	wg.Wait()
	return results
}

// localCandidateGate is the largest cluster-local candidate count for which the
// exact max-flow branch-and-bound runs; above it a cluster's members fall back
// to the frontend bound. It is a performance budget, not a correctness constant:
// the threshold is chosen so the worst single division-snapshot stays about as
// fast as the brute-force worst case (machine-independent via the ratio of
// B&B-worst to brute-worst). Lowering it only loosens results -- still sound and
// monotone -- never produces a wrong bound. Empirically, k <= 13 keeps the worst
// snapshot brute-bound; k = 14 hits a single-player B&B blow-up.
const localCandidateGate = 13

// frontendBound is the pairing-agnostic possible-rank range for player p: for
// the best rank P wins all its games (others hold their points), for the worst
// rank P loses all (others win all). It ignores who plays whom, so it is loose,
// but it is sound (best <= achievable best, worst >= achievable worst) and
// monotone -- the above/below counts only grow as games complete (a counted
// player's points pass P's ceiling, or its ceiling drops below P's points, and
// neither reverts). It is also always at least as loose as the max-flow bound,
// so swapping a member to the max-flow tier as its cluster shrinks past the gate
// can only tighten, never widen. It is exactly the bound the web frontend can
// compute itself from points and games-remaining, hence the name.
func frontendBound(p int, standings []standingInfo) RankBounds {
	n := len(standings)
	pMax := standings[p].points + standings[p].gamesRemaining*2 // P wins all
	pMin := standings[p].points                                 // P loses all
	above, below := 0, 0
	for i := range standings {
		if i == p {
			continue
		}
		if standings[i].points > pMax {
			above++ // above P even if P wins out and this player stalls
		}
		if standings[i].points+standings[i].gamesRemaining*2 < pMin {
			below++ // below P even if this player wins out and P stalls
		}
	}
	return RankBounds{BestRank: 1 + above, WorstRank: n - below}
}

// standingInfo is the subset of standing data needed for rank calculation.
type standingInfo struct {
	userID         int32
	points         int // wins*2 + draws
	spread         int
	gamesRemaining int
}

// unfinishedGame holds the two player database IDs for an incomplete game.
type unfinishedGame struct {
	player0ID, player1ID int32
}

// StandingInfoFromRow converts a DB standings row to the lightweight struct.
func StandingInfoFromRow(userID int32, wins, draws, spread, gamesRemaining int32) standingInfo {
	return standingInfo{
		userID:         userID,
		points:         int(wins)*2 + int(draws),
		spread:         int(spread),
		gamesRemaining: int(gamesRemaining),
	}
}

// UnfinishedGameFromRow converts a DB row to unfinishedGame.
func UnfinishedGameFromRow(p0, p1 pgtype.Int4) unfinishedGame {
	return unfinishedGame{player0ID: p0.Int32, player1ID: p1.Int32}
}

// playerGameInfo holds precomputed game decomposition for a specific player P.
type playerGameInfo struct {
	gamesVsP     []int      // how many remaining games each player has vs P
	nonPGames    []gamePair // games not involving P
	nonPGamesCnt []int      // remaining non-P games per player
}

func decomposeGames(p int, n int, allGames []gamePair) playerGameInfo {
	gi := playerGameInfo{
		gamesVsP:     make([]int, n),
		nonPGamesCnt: make([]int, n),
	}
	for _, g := range allGames {
		if g.a == p {
			gi.gamesVsP[g.b]++
		} else if g.b == p {
			gi.gamesVsP[g.a]++
		} else {
			gi.nonPGames = append(gi.nonPGames, g)
		}
	}
	for _, g := range gi.nonPGames {
		gi.nonPGamesCnt[g.a]++
		gi.nonPGamesCnt[g.b]++
	}
	return gi
}

// ---------------------------------------------------------------------------
// worst rank: maximize the number of players finishing above P
// ---------------------------------------------------------------------------

// worstCandidateDeficit classifies opponent i for P's worst rank: the points i
// still needs from non-P games to finish at or above P, and whether it can reach
// that at all (a candidate). P is assumed to lose all its remaining games, so
// each opponent of P gains +2 per game vs P.
//
// When i would tie P on points, spread decides. How i can accumulate points
// matters: a win gives 2 points and improves spread (potentially a lot), a draw
// gives 1 point with no spread change. An odd deficit forces at least one draw,
// but with deficit >= 2 (or more than one game) i can include wins for arbitrary
// spread; the only forced-pure-draw case is deficit 1 with exactly 1 game.
func worstCandidateDeficit(p, i int, standings []standingInfo, gi playerGameInfo) (deficit int, isCandidate bool) {
	tieDeficit := standings[p].points - (standings[i].points + gi.gamesVsP[i]*2)
	switch {
	case standings[p].gamesRemaining > 0:
		// P has -inf worst spread, so any opponent at P's points is above P.
		deficit = max(0, tieDeficit)
	case gi.nonPGamesCnt[i] == 0:
		// i has no remaining games; its spread is fixed. Equal spread means
		// either order (arbitrary tiebreak), so a tie still makes i a candidate.
		if standings[i].spread >= standings[p].spread {
			deficit = max(0, tieDeficit)
		} else {
			deficit = max(0, tieDeficit+1) // worse spread: needs strict points lead
		}
	case standings[i].spread >= standings[p].spread:
		// Spread >= P's: draws preserve it, wins improve it. A tie suffices.
		deficit = max(0, tieDeficit)
	case tieDeficit == 1 && gi.nonPGamesCnt[i] == 1:
		// Exactly 1 game, needs 1 point -> must draw (no spread change); with
		// spread < P's it cannot win the tiebreak, so it must exceed P on points.
		deficit = max(0, tieDeficit+1)
	default:
		// Spread < P's but deficit >= 2 or multiple games: i can include wins for
		// arbitrary spread improvement, so a tie on points suffices.
		deficit = max(0, tieDeficit)
	}
	return deficit, deficit <= gi.nonPGamesCnt[i]*2
}

func worstRankForPlayer(p int, standings []standingInfo, gi playerGameInfo, fg *flowGraph, candIdx []int) int {
	n := len(standings)

	// Candidates: opponents that can individually finish at or above P (so could
	// be forced above P). The per-opponent classification lives in
	// worstCandidateDeficit, shared with the gate's cluster-local count so the
	// two never drift.
	var candidates []cand
	for i := range n {
		if i == p {
			continue
		}
		if deficit, ok := worstCandidateDeficit(p, i, standings, gi); ok {
			candidates = append(candidates, cand{i, deficit})
		}
	}

	if len(candidates) <= 1 {
		return 1 + len(candidates) // 0 or 1 candidates: trivial
	}

	// Sort candidates by deficit ascending (cheapest to satisfy first).
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].deficit < candidates[j].deficit
	})

	// The worst rank is 1 + (players guaranteed above P) + (minimum candidates
	// that can be forced above P). The minimum forced-above count is the size of
	// the candidate set minus the most that can simultaneously stay at/below P;
	// finding that maximum is a minimum-eviction problem, solved exactly by
	// branch-and-bound below. (A greedy eviction is faster but sub-optimal, and
	// its error is not monotone in cluster size, so it can produce a widening
	// bound. The exact search is kept cheap by the cluster-local-k gate in
	// CalculatePossibleRanks, which never routes a large candidate set here.)
	initial := len(candidates)
	minEvict := minEvictAbove(candidates, gi.nonPGames, fg, candIdx, bnbBudget)
	return 1 + (initial - minEvict)
}

// worstCandidateCountLocal counts P's worst-rank candidates restricted to P's
// own cluster (the inCluster mask). It is the branch-and-bound gate metric: the
// candidate set drives the eviction search, so its size governs B&B cost.
// Cross-cluster players are rank-disjoint -- they enter worstRankForPlayer's set
// as trivial deficit-0 candidates that are never evicted and add ~no cost -- so
// counting only in-cluster candidates gates on the true cost driver. Applied to
// the real standings it bounds worst-rank cost; applied to the loss-score mirror
// it bounds best-rank (inversion) cost, so one metric covers both directions.
func worstCandidateCountLocal(p int, standings []standingInfo, gi playerGameInfo, inCluster []bool) int {
	k := 0
	for i := range standings {
		if i == p || !inCluster[i] {
			continue
		}
		if _, ok := worstCandidateDeficit(p, i, standings, gi); ok {
			k++
		}
	}
	return k
}

// bnbBudget caps the branch-and-bound recursion depth (the eviction count). It
// is a finite-termination backstop only: the cluster-local-k gate already bounds
// the candidate set so the true minimum eviction never approaches this, so the
// budget never binds in production. (A depth cap is monotone -- under-counting
// evictions only loosens the bound -- so even if it did bind it could not widen
// the range; the gate, not the budget, is what keeps the search cheap.)
const bnbBudget = 64

// reachW builds the "every candidate gains its deficit of points from non-P
// games" max-flow and reports feasibility. A candidate's deficit is first reduced
// by 2 per game it plays against a non-candidate (an outright win it can always
// take); the remainder must come from within-set games, each of which supplies 2
// points to its two endpoints. The set is feasible iff the flow saturates every
// game edge.
func reachW(candidates []cand, nonPGames []gamePair, fg *flowGraph, candIdx []int) bool {
	k := len(candidates)
	if k == 0 {
		return true
	}
	for ci, c := range candidates {
		candIdx[c.idx] = ci
	}
	defer func() {
		for _, c := range candidates {
			candIdx[c.idx] = -1
		}
	}()

	type withinGame struct{ ci, cj int }
	var withinGames []withinGame
	externalCnt := make([]int, k)
	for _, g := range nonPGames {
		ciA, ciB := candIdx[g.a], candIdx[g.b]
		aIn, bIn := ciA >= 0, ciB >= 0
		if aIn && bIn {
			withinGames = append(withinGames, withinGame{ciA, ciB})
		} else if aIn {
			externalCnt[ciA]++
		} else if bIn {
			externalCnt[ciB]++
		}
	}

	adjustedDeficit := make([]int, k)
	totalNeeded := 0
	for ci, c := range candidates {
		adj := max(0, c.deficit-externalCnt[ci]*2)
		adjustedDeficit[ci] = adj
		totalNeeded += adj
	}
	if totalNeeded == 0 {
		return true
	}

	numGameNodes := len(withinGames)
	numNodes := 2 + numGameNodes + k
	source, sink := 0, 1
	playerNode := func(ci int) int { return 2 + numGameNodes + ci }
	gameNode := func(gi int) int { return 2 + gi }

	fg.reset(numNodes)
	for gi, wg := range withinGames {
		gn := gameNode(gi)
		fg.addEdge(source, gn, 2)
		fg.addEdge(gn, playerNode(wg.ci), 2)
		fg.addEdge(gn, playerNode(wg.cj), 2)
	}
	for ci := range k {
		if adjustedDeficit[ci] > 0 {
			fg.addEdge(playerNode(ci), sink, adjustedDeficit[ci])
		}
	}
	return fg.maxflow(source, sink) >= totalNeeded
}

// minimalInfeasibleAbove shrinks an infeasible subset of candidates (indices into
// candidates) to a minimal one: a subset that is still infeasible but becomes
// feasible if any single member is removed. The minimum eviction set must contain
// at least one member of every minimal infeasible subset, so branching on a small
// one keeps the branch factor low.
func minimalInfeasibleAbove(candidates []cand, idxs []int, nonPGames []gamePair, fg *flowGraph, candIdx []int) []int {
	keep := append([]int(nil), idxs...)
	for i := 0; i < len(keep); {
		trial := append(append([]int{}, keep[:i]...), keep[i+1:]...)
		sub := make([]cand, len(trial))
		for j, ci := range trial {
			sub[j] = candidates[ci]
		}
		if !reachW(sub, nonPGames, fg, candIdx) {
			keep = trial
		} else {
			i++
		}
	}
	return keep
}

// minEvictAbove returns the exact minimum number of candidates that must be
// dropped (finish at/below P) so the rest can all simultaneously reach W. It
// tests feasibility with reachW; if infeasible, it branches on a minimal
// infeasible subset, evicting each member in turn and recursing. Branching on a
// minimal infeasible subset (not the whole set, nor the min cut, which is not a
// reliable infeasible subset for this flow) is exact and keeps the branch factor
// small. budget bounds the recursion depth (see bnbBudget).
func minEvictAbove(candidates []cand, nonPGames []gamePair, fg *flowGraph, candIdx []int, budget int) int {
	if reachW(candidates, nonPGames, fg, candIdx) {
		return 0
	}
	if budget <= 0 {
		return 1
	}
	all := make([]int, len(candidates))
	for i := range all {
		all[i] = i
	}
	branch := minimalInfeasibleAbove(candidates, all, nonPGames, fg, candIdx)
	best := budget + 1
	sub := make([]cand, 0, len(candidates)-1)
	for _, ci := range branch {
		sub = sub[:0]
		sub = append(sub, candidates[:ci]...)
		sub = append(sub, candidates[ci+1:]...)
		if r := 1 + minEvictAbove(sub, nonPGames, fg, candIdx, best-2); r < best {
			best = r
			if best == 1 {
				break
			}
		}
	}
	return best
}

type cand struct {
	idx     int
	deficit int // points still needed from non-P games
}

// ---------------------------------------------------------------------------
// best rank via loss-score inversion
//
// Best-rank is the mirror of worst-rank. Computing it directly would need its
// own candidate set (players who could be forced BELOW P), whose count keys off
// P's point ceiling pts+2*remaining -- which FALLS as P's games resolve, so the
// count can rise. A rising candidate count is non-monotone: a gate on it could
// flip tight->loose and widen the range. So instead we invert. With loss-score
// 2L+D we have points = 2W+D = 2G-(2L+D) exactly (G = W+D+L, constant per player
// at season end, draws included), so ranking by points descending is identical
// to ranking by loss-score ascending. Hence
//
//	bestRank(P) = n + 1 - worstRank(P on the loss-score mirror)
//
// reusing the single monotone worst-rank routine. On the mirror, best-rank keys
// off P's loss FLOOR, which rises over the season -- so its candidate count is
// monotone, and one gate metric covers both directions. (See
// rank_bounds_design.md for the full rationale.)
// ---------------------------------------------------------------------------

// mirrorForBest returns the loss-score mirror of a full division: each player's
// points become its loss-score 2L+D and its spread is negated, so worst-rank on
// the mirror equals best-rank on the original. Requires a full division
// (constant total games per player) so expected = CalculateExpectedGamesPerPlayer
// is each player's total games; then played = expected - remaining and
// 2L+D = 2*expected - points - 2*remaining.
func mirrorForBest(standings []standingInfo) []standingInfo {
	n := len(standings)
	expected := CalculateExpectedGamesPerPlayer(n)
	m := make([]standingInfo, n)
	for i, s := range standings {
		m[i] = standingInfo{
			userID:         s.userID,
			points:         2*expected - s.points - 2*s.gamesRemaining,
			spread:         -s.spread,
			gamesRemaining: s.gamesRemaining,
		}
	}
	return m
}

// bestViaInversion computes P's best rank as n+1 minus P's worst rank on the
// loss-score mirror. mir must be mirrorForBest(standings); gi is index-based, so
// it is valid unchanged on the mirror (the unfinished games are the same).
func bestViaInversion(p int, mir []standingInfo, gi playerGameInfo, fg *flowGraph, candIdx []int) int {
	return len(mir) + 1 - worstRankForPlayer(p, mir, gi, fg, candIdx)
}

// ---------------------------------------------------------------------------
// Dinic's max-flow (O(V²E), better than Edmonds-Karp's O(VE²) for dense graphs)
// ---------------------------------------------------------------------------

type flowEdge struct {
	to, rev, cap int
}

type flowGraph struct {
	adj   [][]flowEdge
	level []int
	iter  []int
	queue []int
}

func newFlowGraph(n int) *flowGraph {
	return &flowGraph{
		adj:   make([][]flowEdge, n),
		level: make([]int, n),
		iter:  make([]int, n),
		queue: make([]int, 0, n),
	}
}

// reset clears the graph for reuse with a potentially different size.
func (g *flowGraph) reset(n int) {
	if cap(g.adj) >= n {
		g.adj = g.adj[:n]
		for i := range g.adj {
			g.adj[i] = g.adj[i][:0]
		}
	} else {
		g.adj = make([][]flowEdge, n)
	}
	if cap(g.level) >= n {
		g.level = g.level[:n]
	} else {
		g.level = make([]int, n)
	}
	if cap(g.iter) >= n {
		g.iter = g.iter[:n]
	} else {
		g.iter = make([]int, n)
	}
}

func (g *flowGraph) addEdge(from, to, cap int) {
	g.adj[from] = append(g.adj[from], flowEdge{to, len(g.adj[to]), cap})
	g.adj[to] = append(g.adj[to], flowEdge{from, len(g.adj[from]) - 1, 0})
}

// bfs builds the level graph from source s.
func (g *flowGraph) bfs(s, t int) bool {
	for i := range g.level {
		g.level[i] = -1
	}
	g.level[s] = 0
	g.queue = g.queue[:0]
	g.queue = append(g.queue, s)
	for len(g.queue) > 0 {
		u := g.queue[0]
		g.queue = g.queue[1:]
		for _, e := range g.adj[u] {
			if g.level[e.to] < 0 && e.cap > 0 {
				g.level[e.to] = g.level[u] + 1
				g.queue = append(g.queue, e.to)
			}
		}
	}
	return g.level[t] >= 0
}

// dfs sends flow along blocking paths in the level graph.
func (g *flowGraph) dfs(u, t, pushed int) int {
	if u == t {
		return pushed
	}
	for ; g.iter[u] < len(g.adj[u]); g.iter[u]++ {
		e := &g.adj[u][g.iter[u]]
		if g.level[e.to] == g.level[u]+1 && e.cap > 0 {
			d := g.dfs(e.to, t, min(pushed, e.cap))
			if d > 0 {
				e.cap -= d
				g.adj[e.to][e.rev].cap += d
				return d
			}
		}
	}
	return 0
}

func (g *flowGraph) maxflow(s, t int) int {
	total := 0
	for g.bfs(s, t) {
		for i := range g.iter {
			g.iter[i] = 0
		}
		for {
			f := g.dfs(s, t, math.MaxInt)
			if f == 0 {
				break
			}
			total += f
		}
	}
	return total
}

func initSlice(n, val int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = val
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// bruteForceThreshold caps when we enumerate every game outcome rather than
// use the max-flow heuristic. 3^g grows fast; at g=10 we have ~59k outcomes
// × O(n^2 * g) margin checks, which is still sub-second for realistic n.
// Above that, fall back to the heuristic (which has residual looseness
// noted in its docstring).
const bruteForceThreshold = 10

// Brute-force rank bounds
//
// Tight rank bounds for each player via explicit enumeration of every
// win/draw/loss assignment, partitioned into rank-disjoint clusters so the
// cost scales with max(g_c), not total g.
//
// Per outcome within a cluster:
//
//   - points per cluster member are determined exactly.
//
//   - for each tied pair (P, Q) two margin-feasibility checks decide whether
//     Q can (forced-above) or might (possibly-above) finish above P on
//     spread. The coefficient of each game's margin m_g in (Q.spread -
//     P.spread) is Δ_g = sign_Q(g) - sign_P(g) where sign is +1 if that
//     player wins g, -1 if they lose, 0 if draw or not in the game.
//     Margins are 0 for draws and ≥ 1 otherwise.
//
//     max Σ Δ_g*m_g is +∞ if any non-draw game has Δ_g > 0 (push m_g → ∞),
//     else Σ_{non-draw, Δ_g < 0} Δ_g (m_g = 1 minimizes the negative).
//     Q.spread >= P.spread iff Σ Δ m >= P.initial - Q.initial (possibly).
//     Q.spread >  P.spread iff Σ Δ m >  P.initial - Q.initial (strict).
//
// Per-outcome best rank for P = 1 + fixedAbove + strictAbove + forcedAboveTied.
// Per-outcome worst rank for P = 1 + fixedAbove + strictAbove + possiblyAboveTied.
// fixedAbove counts players in strictly-higher clusters (constant per cluster).
// Final bounds are min/max over all outcomes within P's cluster.
//
// Tight: handles asymmetric within-set spread drops, spread interactions
// between candidates, zero-sum situations, etc., because it enumerates
// realizations directly.

// maxClusterGames returns the largest cluster's game count, used by the
// dispatcher to decide if brute force is tractable.
func maxClusterGames(clusters []bfCluster) int {
	m := 0
	for _, c := range clusters {
		if len(c.games) > m {
			m = len(c.games)
		}
	}
	return m
}

// bruteForceRanksFromClusters enumerates outcomes per cluster in parallel and
// combines within-cluster ranks with fixed cross-cluster contributions.
func bruteForceRanksFromClusters(clusters []bfCluster, standings []standingInfo, games []gamePair) []RankBounds {
	n := len(standings)

	// Cross-cluster fixed above/below counts per cluster.
	crossAbove := make([]int, len(clusters))
	crossBelow := make([]int, len(clusters))
	for i := range clusters {
		for j := range clusters {
			if i == j {
				continue
			}
			if clusters[j].minPts > clusters[i].maxPts {
				crossAbove[i] += len(clusters[j].members)
			} else if clusters[j].maxPts < clusters[i].minPts {
				crossBelow[i] += len(clusters[j].members)
			}
		}
	}

	result := make([]RankBounds, n)
	var wg sync.WaitGroup
	for i := range clusters {
		wg.Add(1)
		go func(ci int) {
			defer wg.Done()
			enumerateBruteForceCluster(&clusters[ci], standings, games, result, crossAbove[ci])
		}(i)
	}
	wg.Wait()
	return result
}

// bfCluster is a rank-disjoint group of players for brute-force enumeration.
// members includes unfinished players connected by games AND finished players
// whose fixed pts fall inside the cluster's pts range. games indexes into the
// caller's []gamePair.
type bfCluster struct {
	members []int // player indices (global)
	games   []int // game indices (global, into the outer games slice)
	minPts  int
	maxPts  int
}

// buildBruteForceClusters partitions players into rank-disjoint groups.
//  1. Connected components over the unfinished-player graph.
//  2. Compute each component's pts range [minPts, maxPts] across its members.
//  3. Interval-merge components whose ranges overlap.
//  4. Absorb finished players whose pts fall inside a merged range.
//  5. Any remaining finished player (pts outside every merged range) becomes
//     its own singleton cluster — its rank is fully fixed.
func buildBruteForceClusters(standings []standingInfo, games []gamePair) []bfCluster {
	n := len(standings)

	// Union-find over player indices, linked by games.
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}

	inGame := make([]bool, n)
	gamesPerPlayer := make([]int, n)
	for _, g := range games {
		inGame[g.a] = true
		inGame[g.b] = true
		gamesPerPlayer[g.a]++
		gamesPerPlayer[g.b]++
		union(g.a, g.b)
	}

	// Build one cluster per component of unfinished players.
	rootToIdx := make(map[int]int)
	var clusters []bfCluster
	for i := 0; i < n; i++ {
		if !inGame[i] {
			continue
		}
		r := find(i)
		idx, ok := rootToIdx[r]
		if !ok {
			idx = len(clusters)
			rootToIdx[r] = idx
			clusters = append(clusters, bfCluster{minPts: math.MaxInt, maxPts: math.MinInt})
		}
		c := &clusters[idx]
		c.members = append(c.members, i)
		pmin := standings[i].points
		pmax := standings[i].points + 2*gamesPerPlayer[i]
		if pmin < c.minPts {
			c.minPts = pmin
		}
		if pmax > c.maxPts {
			c.maxPts = pmax
		}
	}
	for gi, g := range games {
		idx := rootToIdx[find(g.a)]
		clusters[idx].games = append(clusters[idx].games, gi)
	}

	// Interval-merge clusters with overlapping pts ranges.
	if len(clusters) > 1 {
		sort.Slice(clusters, func(i, j int) bool {
			return clusters[i].minPts < clusters[j].minPts
		})
		merged := clusters[:0]
		for _, c := range clusters {
			if len(merged) > 0 && merged[len(merged)-1].maxPts >= c.minPts {
				last := &merged[len(merged)-1]
				last.members = append(last.members, c.members...)
				last.games = append(last.games, c.games...)
				if c.maxPts > last.maxPts {
					last.maxPts = c.maxPts
				}
			} else {
				merged = append(merged, c)
			}
		}
		clusters = merged
	}

	// Absorb finished players into clusters whose range contains their pts.
	// Leftover finished players become singleton clusters (fixed rank).
	for i := 0; i < n; i++ {
		if inGame[i] {
			continue
		}
		p := standings[i].points
		absorbed := false
		for j := range clusters {
			if p >= clusters[j].minPts && p <= clusters[j].maxPts {
				clusters[j].members = append(clusters[j].members, i)
				absorbed = true
				break
			}
		}
		if !absorbed {
			clusters = append(clusters, bfCluster{
				members: []int{i},
				minPts:  p,
				maxPts:  p,
			})
		}
	}

	return clusters
}

// enumerateBruteForceCluster runs 3^len(cluster.games) enumeration over the
// cluster's games, computing rank bounds for each cluster member. fixedAbove
// is the count of players in strictly-higher clusters (contributes a constant
// to every member's rank).
func enumerateBruteForceCluster(c *bfCluster, standings []standingInfo, allGames []gamePair, result []RankBounds, fixedAbove int) {
	m := len(c.members)
	g := len(c.games)

	// Initialize local best/worst for cluster members. Indexed by cluster
	// member index (0..m-1); we map back to global at the end.
	best := make([]int, m)
	worst := make([]int, m)
	for i := 0; i < m; i++ {
		best[i] = len(standings) + 1
		worst[i] = 0
	}

	// Precompute per-cluster-member the global game indices they participate
	// in. Not strictly needed for correctness, but lets us keep winMask/
	// loseMask sized to m rather than n.
	memberIdx := make(map[int]int, m)
	for mi, gi := range c.members {
		memberIdx[gi] = mi
	}

	points := make([]int, m)
	winMask := make([]uint64, m)
	loseMask := make([]uint64, m)

	// baseSpread is constant across leaves; tied is reset per player.
	// localGames holds each cluster game's two members in local indices.
	baseSpread := make([]int, m)
	maxPts := 0
	for i := range m {
		baseSpread[i] = standings[c.members[i]].spread
		if pts := standings[c.members[i]].points; pts > maxPts {
			maxPts = pts
		}
	}
	tied := make([]int, 0, m)
	localGames := make([][2]int, g)
	for gi := range g {
		gm := allGames[c.games[gi]]
		localGames[gi] = [2]int{memberIdx[gm.a], memberIdx[gm.b]}
	}
	js := newJointScratch(m)
	winsCnt := make([]int, m)
	lossCnt := make([]int, m)
	coupledVal := make([]bool, maxPts+2*g+1)
	// order + valOffset implement a per-leaf counting sort of members by points,
	// so strictly-above and tied-on-points are read off contiguous groups in
	// O(m + pointsRange) rather than rescanned per player in O(m^2). Sized by the
	// points range and cluster size, never by a fixed division size.
	order := make([]int, m)
	valOffset := make([]int, maxPts+2*g+1)

	total := 1
	for i := 0; i < g; i++ {
		total *= 3
	}

	for k := 0; k < total; k++ {
		for i := 0; i < m; i++ {
			points[i] = standings[c.members[i]].points
			winMask[i] = 0
			loseMask[i] = 0
		}
		x := k
		for localGi := 0; localGi < g; localGi++ {
			// localGames already holds both endpoints' local member indices.
			a, b := localGames[localGi][0], localGames[localGi][1]
			bit := uint64(1) << localGi
			switch x % 3 {
			case 0:
				points[a]++
				points[b]++
			case 1:
				points[a] += 2
				winMask[a] |= bit
				loseMask[b] |= bit
			case 2:
				points[b] += 2
				winMask[b] |= bit
				loseMask[a] |= bit
			}
			x /= 3
		}

		for mi := range m {
			winsCnt[mi] = bits.OnesCount64(winMask[mi])
			lossCnt[mi] = bits.OnesCount64(loseMask[mi])
		}
		// Mark each points value with a decided game between two players both at
		// that value -- the only values where a fixed P needs the joint
		// closed-form rather than independent per-member spread boxes.
		clear(coupledVal)
		for gi, gp := range localGames {
			a, b := gp[0], gp[1]
			if points[a] == points[b] && (winMask[a]|winMask[b])&(uint64(1)<<gi) != 0 {
				coupledVal[points[a]] = true
			}
		}

		// Counting sort members by points into `order` (ascending), so each
		// points value is a contiguous group. Then strictly-above is the suffix
		// past the group and the group itself is the tied-on-points set -- read
		// off in O(m + pointsRange) instead of an O(m^2) per-player rescan.
		clear(valOffset)
		for i := range m {
			valOffset[points[i]]++
		}
		acc := 0
		for v := range valOffset {
			cnt := valOffset[v]
			valOffset[v] = acc
			acc += cnt
		}
		for i := range m {
			v := points[i]
			order[valOffset[v]] = i
			valOffset[v]++
		}

		for lo := 0; lo < m; {
			v := points[order[lo]]
			hi := lo + 1
			for hi < m && points[order[hi]] == v {
				hi++
			}
			strictAbove := m - hi // members past this group have strictly higher points
			coupled := coupledVal[v]
			for gi := lo; gi < hi; gi++ {
				p := order[gi]
				var minAbove, maxGE int
				if standings[c.members[p]].gamesRemaining > 0 {
					// Free P: per-pair is exact in aggregate (the lose-all and
					// win-all leaves give P's true rank extremes). Fast path.
					for ti := lo; ti < hi; ti++ {
						if ti == gi {
							continue
						}
						q := order[ti]
						forced, possible := spreadOrdering(
							winMask[p], loseMask[p], winMask[q], loseMask[q],
							baseSpread[p], baseSpread[q],
						)
						if forced {
							minAbove++
						}
						if possible {
							maxGE++
						}
					}
				} else if coupled {
					// Fixed P tied with a coupled set: the per-pair view over-counts,
					// so use the joint closed-form. tied = this group minus P.
					tied = tied[:0]
					for ti := lo; ti < hi; ti++ {
						if ti != gi {
							tied = append(tied, order[ti])
						}
					}
					minAbove, maxGE = jointFixedTied(p, tied, winMask, baseSpread, localGames, js)
				} else {
					// Fixed P, no coupling: each tied opponent's reachable-spread
					// box is independent and exact.
					sprP := baseSpread[p]
					for ti := lo; ti < hi; ti++ {
						if ti == gi {
							continue
						}
						q := order[ti]
						hiq := baseSpread[q] - lossCnt[q] // max spread: no win -> lose minimally
						if winsCnt[q] > 0 {
							hiq = 1 << 30 // a win runs spread to +inf
						}
						loq := baseSpread[q] + winsCnt[q] // min spread: no loss -> win minimally
						if lossCnt[q] > 0 {
							loq = -(1 << 30)
						}
						if hiq >= sprP {
							maxGE++
						}
						if loq > sprP {
							minAbove++
						}
					}
				}
				bestRank := 1 + fixedAbove + strictAbove + minAbove
				worstRank := 1 + fixedAbove + strictAbove + maxGE
				if bestRank < best[p] {
					best[p] = bestRank
				}
				if worstRank > worst[p] {
					worst[p] = worstRank
				}
			}
			lo = hi
		}
	}

	for i := 0; i < m; i++ {
		result[c.members[i]] = RankBounds{BestRank: best[i], WorstRank: worst[i]}
	}
}

// spreadInf is a sentinel for an unbounded (+/-inf) reachable spread. At 1<<50
// it dwarfs any real spread sum yet leaves int64 headroom for summing one per
// cluster member (a sum stays safe up to thousands of members, far beyond any
// division), so finite values never reach it and the sums never overflow.
const spreadInf = int64(1) << 50

// spreadOrdering decides, for one brute leaf, whether a points-tied opponent Q
// sits above a FREE player P (P has remaining games) on spread. Across all
// leaves the lose-all and win-all leaves give P its true rank extremes, and
// there this per-pair test is exact -- so for a free P it yields the correct
// brute bound even though it ignores the joint coupling in intermediate leaves.
// (A FIXED P has no such extreme leaf; see jointFixedTied.)
//   - possible: Q can reach spread >= P (counts toward P's worst rank).
//   - forced:   Q must exceed P on spread (counts toward P's best rank).
//
// A non-draw game Q wins, or P loses, lets the margin run to +inf
// (maxUnbounded); P winning or Q losing lets it run to -inf (minUnbounded).
func spreadOrdering(pWins, pLosses, qWins, qLosses uint64, pSpread, qSpread int) (forced, possible bool) {
	maxUnbounded := (qWins | pLosses) != 0
	minUnbounded := (pWins | qLosses) != 0
	if maxUnbounded {
		possible = true
	} else {
		possible = qSpread >= pSpread
	}
	if minUnbounded {
		forced = false
	} else {
		forced = qSpread > pSpread
	}
	return forced, possible
}

// jointScratch holds reusable buffers for jointFixedTied so the brute's inner
// loop allocates nothing. One per enumerateBruteForceCluster goroutine.
type jointScratch struct {
	parent, wins, losses []int
	isTied, external     []bool
	comp                 []int
	lo, hi, costs        []int64
}

func newJointScratch(m int) *jointScratch {
	return &jointScratch{
		parent:   make([]int, m),
		wins:     make([]int, m),
		losses:   make([]int, m),
		isTied:   make([]bool, m),
		external: make([]bool, m),
		comp:     make([]int, 0, m),
		lo:       make([]int64, 0, m),
		hi:       make([]int64, 0, m),
		costs:    make([]int64, 0, m),
	}
}

func findUF(parent []int, x int) int {
	for parent[x] != x {
		parent[x] = parent[parent[x]]
		x = parent[x]
	}
	return x
}

// jointFixedTied computes, for a FIXED player P (no remaining games, so P's
// spread is the constant baseSpread[p]) in one brute leaf, the min number of
// points-tied players that MUST finish strictly above P (best rank) and the max
// that CAN finish at-or-above P (worst rank), accounting for the joint zero-sum
// coupling of shared games that the per-pair view misses.
//
// Tied players are grouped into components by their internal (tied-vs-tied)
// non-draw games. A component is OPEN if any member also has a non-draw game to
// a non-tied player: that game leaks spread out of the component, and the leak
// propagates through the internal edges, so every member can independently
// reach its own spread box -- count each member by its box. A CLOSED component
// has no such leak, so its total spread is conserved; the count is then a
// box-constrained sum-feasibility (maxAtOrAbove / maxAtOrBelow).
func jointFixedTied(p int, tied []int, winMask []uint64, baseSpread []int, localGames [][2]int, js *jointScratch) (minAbove, maxGE int) {
	if len(tied) == 0 {
		return 0, 0
	}
	sprP := int64(baseSpread[p])
	parent, wins, losses := js.parent, js.wins, js.losses
	isTied, external := js.isTied, js.external
	for _, q := range tied {
		parent[q] = q
		wins[q] = 0
		losses[q] = 0
		isTied[q] = true
		external[q] = false
	}
	defer func() {
		for _, q := range tied {
			isTied[q] = false
		}
	}()

	for gi, gp := range localGames {
		bit := uint64(1) << gi
		a, b := gp[0], gp[1]
		var w, l int
		switch {
		case winMask[a]&bit != 0:
			w, l = a, b
		case winMask[b]&bit != 0:
			w, l = b, a
		default:
			continue // draw: no spread coupling
		}
		if isTied[w] {
			wins[w]++
		}
		if isTied[l] {
			losses[l]++
		}
		switch {
		case isTied[w] && isTied[l]:
			if rw, rl := findUF(parent, w), findUF(parent, l); rw != rl {
				parent[rw] = rl
			}
		case isTied[w]:
			external[w] = true
		case isTied[l]:
			external[l] = true
		}
	}

	// Group tied players into coupled components (by union-find root). Sorting
	// by root makes same-component members contiguous, so each component is
	// handled once in O(|tied| log |tied|) -- no per-root rescan of the set.
	comp := append(js.comp[:0], tied...)
	js.comp = comp
	slices.SortFunc(comp, func(a, b int) int { return findUF(parent, a) - findUF(parent, b) })
	for i := 0; i < len(comp); {
		r := findUF(parent, comp[i])
		lo, hi := js.lo[:0], js.hi[:0]
		closed := true
		var sumC int64
		j := i
		for j < len(comp) && findUF(parent, comp[j]) == r {
			t := comp[j]
			if external[t] {
				closed = false
			}
			var loq, hiq int64
			if wins[t] > 0 {
				hiq = spreadInf
			} else {
				hiq = int64(baseSpread[t] - losses[t])
			}
			if losses[t] > 0 {
				loq = -spreadInf
			} else {
				loq = int64(baseSpread[t] + wins[t])
			}
			lo = append(lo, loq)
			hi = append(hi, hiq)
			sumC += int64(baseSpread[t])
			j++
		}
		js.lo, js.hi = lo, hi
		if closed {
			maxGE += maxAtOrAbove(lo, hi, sumC, sprP, js.costs)
			minAbove += len(lo) - maxAtOrBelow(lo, hi, sumC, sprP, js.costs)
		} else {
			for k := range lo {
				if hi[k] >= sprP {
					maxGE++
				}
				if lo[k] > sprP {
					minAbove++
				}
			}
		}
		i = j
	}
	return minAbove, maxGE
}

// maxAtOrAbove returns the largest count of members that can have spread >= c
// given each spread is in [lo_i, hi_i] and they sum to the conserved S. Placing
// a member at-or-above c costs max(c,lo_i)-lo_i over its floor; greedily admit
// the cheapest while the minimum achievable sum stays <= S. costs is scratch.
func maxAtOrAbove(lo, hi []int64, s, c int64, costs []int64) int {
	var base int64
	for _, l := range lo {
		base += l
	}
	costs = costs[:0]
	for i := range lo {
		if hi[i] >= c {
			floor := lo[i]
			if c > floor {
				floor = c
			}
			costs = append(costs, floor-lo[i])
		}
	}
	slices.Sort(costs)
	cnt := 0
	cur := base
	for _, ct := range costs {
		cur += ct
		if cur <= s {
			cnt++
		} else {
			break
		}
	}
	return cnt
}

// maxAtOrBelow is the mirror of maxAtOrAbove: the largest count of members that
// can have spread <= c, given the boxes and the conserved sum S.
func maxAtOrBelow(lo, hi []int64, s, c int64, costs []int64) int {
	var base int64
	for _, h := range hi {
		base += h
	}
	costs = costs[:0]
	for i := range hi {
		if lo[i] <= c {
			ceil := hi[i]
			if c < ceil {
				ceil = c
			}
			costs = append(costs, hi[i]-ceil)
		}
	}
	slices.Sort(costs)
	cnt := 0
	cur := base
	for _, ct := range costs {
		cur -= ct
		if cur >= s {
			cnt++
		} else {
			break
		}
	}
	return cnt
}
