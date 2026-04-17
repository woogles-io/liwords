package league

import (
	"math"
	"sort"

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

	// Fast path: few enough unfinished games → brute-force every outcome.
	// Gives tight bounds including spread tiebreaks, whereas the heuristic
	// below has residual looseness in asymmetric within-set spread drops.
	if len(games) <= bruteForceThreshold {
		return bruteForceRanks(standings, games)
	}

	// Precompute for per-player fast path: players with remaining games
	// whose point range clearly spans [1, n] can skip max-flow.
	maxFloor := 0 // highest current points (worst case for leader)
	// Two smallest ceilings so we can get min-excluding-P in O(1).
	minCeil1, minCeil2 := math.MaxInt, math.MaxInt
	for _, s := range standings {
		if s.points > maxFloor {
			maxFloor = s.points
		}
		ceil := s.points + s.gamesRemaining*2
		if ceil <= minCeil1 {
			minCeil2 = minCeil1
			minCeil1 = ceil
		} else if ceil < minCeil2 {
			minCeil2 = ceil
		}
	}

	// Reusable state: flow graph (adj slices grow and stabilize) and
	// candIdx (avoids map allocation in inner loops).
	fg := newFlowGraph(2 + len(games) + n)
	candIdx := initSlice(n, -1)

	results := make([]RankBounds, n)
	for p := 0; p < n; p++ {
		if standings[p].gamesRemaining > 0 {
			bestPts := standings[p].points + standings[p].gamesRemaining*2
			pCeil := bestPts
			minCeilExP := minCeil1
			if pCeil == minCeil1 {
				minCeilExP = minCeil2
			}
			if bestPts >= maxFloor && minCeilExP >= standings[p].points {
				results[p] = RankBounds{1, n}
				continue
			}
		}
		gi := decomposeGames(p, n, games)
		results[p] = RankBounds{
			BestRank:  bestRankForPlayer(p, standings, gi, fg, candIdx),
			WorstRank: worstRankForPlayer(p, standings, gi, fg, candIdx),
		}
	}
	return results
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

func worstRankForPlayer(p int, standings []standingInfo, gi playerGameInfo, fg *flowGraph, candIdx []int) int {
	n := len(standings)
	W := standings[p].points // P loses all remaining → keeps current points

	// After P loses all, each opponent of P gets +2 per game vs P.
	effectivePts := make([]int, n)
	for i := range standings {
		effectivePts[i] = standings[i].points + gi.gamesVsP[i]*2
	}

	// Build candidate list: players that can individually surpass P.
	//
	// When Q ties P on points, spread decides. Whether Q can beat P on
	// spread depends on how Q accumulates points:
	//   - A win gives Q 2 points AND improves spread (potentially a lot).
	//   - A draw gives Q 1 point with 0 spread change.
	//
	// An odd deficit requires at least one draw. But as long as the deficit
	// is ≥ 3, Q can include wins alongside the draw(s), giving Q arbitrary
	// spread improvement. The only case where Q is forced into a pure draw
	// (no spread change) is deficit = 1 with exactly 1 remaining game.
	pHasGames := standings[p].gamesRemaining > 0
	var candidates []cand
	for i := 0; i < n; i++ {
		if i == p {
			continue
		}

		tieDeficit := W - effectivePts[i] // points needed to match P
		deficit := 0

		if pHasGames {
			// P has −∞ worst spread → any Q at W points is above P
			deficit = max(0, tieDeficit)
		} else if gi.nonPGamesCnt[i] == 0 {
			// Q has no remaining games. Spread is fixed.
			// Equal spread: Q could be ranked either side of P (username
			// tiebreak is arbitrary), so Q is a valid candidate.
			if standings[i].spread >= standings[p].spread {
				deficit = max(0, tieDeficit) // tie suffices
			} else {
				deficit = max(0, tieDeficit+1) // need strict points advantage
			}
		} else if standings[i].spread >= standings[p].spread {
			// Q has remaining games and spread ≥ P's. Draws preserve it,
			// wins improve it. Tie on points suffices.
			deficit = max(0, tieDeficit)
		} else if tieDeficit == 1 && gi.nonPGamesCnt[i] == 1 {
			// Q has exactly 1 remaining game and needs 1 point → must draw.
			// Draw doesn't change spread. Q.spread < P.spread → can't beat
			// P on spread. Must exceed P on points.
			deficit = max(0, tieDeficit+1)
		} else {
			// Q has remaining games and spread < P's, but with deficit ≥ 2
			// or multiple games, Q can include wins that give arbitrary
			// spread improvement. Tie on points suffices.
			deficit = max(0, tieDeficit)
		}

		if deficit <= gi.nonPGamesCnt[i]*2 {
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

	// Iteratively check feasibility via max-flow and remove infeasible candidates.
	for {
		feasible, infeasibleIdx := checkFeasibility(candidates, gi.nonPGames, fg, candIdx)
		if feasible {
			break
		}
		if infeasibleIdx >= 0 {
			candidates = append(candidates[:infeasibleIdx], candidates[infeasibleIdx+1:]...)
		}
		if len(candidates) == 0 {
			break
		}
	}

	return 1 + len(candidates)
}

// checkFeasibility uses max-flow to determine whether all candidates in the set
// can simultaneously reach W points from non-P games.
//
// Returns (true, -1) if feasible, or (false, idxToRemove) if not.
func checkFeasibility(candidates []cand, nonPGames []gamePair, fg *flowGraph, candIdx []int) (bool, int) {
	k := len(candidates)
	if k == 0 {
		return true, -1
	}

	// Map candidate player indices → 0..k-1 via candIdx slice.
	for ci, c := range candidates {
		candIdx[c.idx] = ci
	}
	defer func() {
		for _, c := range candidates {
			candIdx[c.idx] = -1
		}
	}()

	// Identify within-set games and external games for each candidate.
	type withinGame struct {
		ci, cj int // indices into candidates slice
	}
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

	// Compute adjusted deficit (what each candidate needs from within-set games).
	adjustedDeficit := make([]int, k)
	totalNeeded := 0
	for ci, c := range candidates {
		adj := max(0, c.deficit-externalCnt[ci]*2)
		adjustedDeficit[ci] = adj
		totalNeeded += adj
	}

	if totalNeeded == 0 {
		return true, -1
	}

	// Quick check: total points from within-set games
	if len(withinGames)*2 < totalNeeded {
		worst := -1
		worstDef := -1
		for ci := range candidates {
			if adjustedDeficit[ci] > worstDef {
				worstDef = adjustedDeficit[ci]
				worst = ci
			}
		}
		return false, worst
	}

	// Build max-flow network:
	//   0: source
	//   1: sink
	//   2 .. 2+numWithinGames-1: game nodes
	//   2+numWithinGames .. 2+numWithinGames+k-1: player nodes
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
	for ci := 0; ci < k; ci++ {
		if adjustedDeficit[ci] > 0 {
			fg.addEdge(playerNode(ci), sink, adjustedDeficit[ci])
		}
	}

	flow := fg.maxflow(source, sink)
	if flow >= totalNeeded {
		return true, -1
	}

	// Not feasible. Find the candidate with the largest remaining deficit.
	playerFlow := make([]int, k)
	for ci := 0; ci < k; ci++ {
		pn := playerNode(ci)
		for _, e := range fg.adj[pn] {
			if e.to == sink {
				playerFlow[ci] = adjustedDeficit[ci] - e.cap
				break
			}
		}
	}

	worst := -1
	worstGap := -1
	for ci := range candidates {
		gap := adjustedDeficit[ci] - playerFlow[ci]
		if gap > worstGap {
			worstGap = gap
			worst = ci
		}
	}
	return false, worst
}

type cand struct {
	idx     int
	deficit int // points still needed from non-P games
}

type stayBelow struct {
	idx        int
	absorb     int // max additional points Q can receive without exceeding B
	maxPerGame int // max points from a single game (1 = draws only, 2 = any outcome)
}

// ---------------------------------------------------------------------------
// best rank: minimize the number of players forced above P
//
// Algorithm:
//   1. Compute B = P's maximum possible score (P wins all remaining games).
//   2. Count players guaranteed above P (their floor > B).
//   3. Build "stay below" candidates: players that COULD finish below P.
//   4. Use max-flow to check if all candidates can simultaneously absorb
//      their within-set game points without exceeding B.
//
// Flow network:
//
//   Source ──2──> [game] ──cap──> [player] ──absorb──> Sink
//
//   - Each game node receives 2 points from the source.
//   - Game-to-player edge capacity = maxPerGame (1 or 2, see below).
//   - Player-to-sink edge capacity = absorb limit.
//   - Feasible iff max_flow == total within-set game points.
//
// Draws-only optimization (maxPerGame=1):
//   When Q has worse spread than P and can reach B with ≤ gamesRemaining
//   points, Q can get there entirely through draws (1 point each, 0 spread
//   change). Setting per-game capacity to 1 restricts the flow to draw
//   outcomes, correctly proving Q stays below P at B with preserved spread.
//
// Guarantees (best and worst rank combined):
//   - bestRank is achievable (the flow maps to real game outcomes).
//   - bestRank-1 is impossible (flow infeasibility = real infeasibility).
//   - worstRank+1 is always impossible (sound upper bound).
//
// Known limitation (worstRank):
//   worstRank can be pessimistic because it doesn't model that spread is
//   zero-sum between opponents. If Q beats R, Q's spread improves but R's
//   worsens by the same amount. With N candidates all needing spread
//   improvement in a round-robin, the sum of net spread changes is 0, so
//   at most N-1 can simultaneously improve. The algorithm may count all N.
//
//   In practice this is rarely significant because candidates with
//   spread already above P's need no improvement, and candidates with
//   external games (against non-candidates) can gain unlimited spread
//   from those. The error only applies to candidates whose games are
//   entirely within the candidate set AND who need spread improvement.
//
//   Fixing this would require adding spread feasibility as a linear
//   programming constraint alongside the max-flow point check:
//     for each game(i,j): spread_delta_i = -spread_delta_j
//     for each candidate i: initial_spread_i + sum(deltas) > P.spread
//   This is a system of linear inequalities, solvable in polynomial time
//   but significantly more complex than the current max-flow approach.
// ---------------------------------------------------------------------------

func bestRankForPlayer(p int, standings []standingInfo, gi playerGameInfo, fg *flowGraph, candIdx []int) int {
	n := len(standings)
	B := standings[p].points + standings[p].gamesRemaining*2 // P wins all remaining

	// When P wins all, P's opponents each LOSE their game vs P (get 0 from it).
	// Q's effective points = Q.currentPoints (unchanged; the loss to P adds 0).
	// Q can absorb at most B - Q.currentPoints points at or below B.

	pHasGames := standings[p].gamesRemaining > 0

	// First pass: classify each non-P player as guaranteedAbove, guaranteedBelow,
	// or an open candidate. Stored as bool slices so externalCnt can reference them.
	isGuaranteedAbove := make([]bool, n)
	isGuaranteedBelow := make([]bool, n)
	guaranteedAbove := 0
	guaranteedBelow := 0
	for i := 0; i < n; i++ {
		if i == p {
			continue
		}
		qWorst := standings[i].points
		if qWorst > B {
			isGuaranteedAbove[i] = true
			guaranteedAbove++
			continue
		}
		if qWorst == B && !pHasGames && gi.nonPGamesCnt[i] == 0 &&
			standings[i].spread > standings[p].spread {
			isGuaranteedAbove[i] = true
			guaranteedAbove++
			continue
		}

		// Q's ceiling when P wins all: games vs P yield 0, non-P games up to 2.
		maxPts := standings[i].points + gi.nonPGamesCnt[i]*2
		if maxPts < B {
			isGuaranteedBelow[i] = true
			guaranteedBelow++
			continue // Q cannot reach B on points
		}
		if maxPts == B {
			if pHasGames {
				// P's best spread is unbounded (wins by huge margins).
				// Q at B loses the spread tiebreak to P.
				isGuaranteedBelow[i] = true
				guaranteedBelow++
				continue
			}
			if gi.nonPGamesCnt[i] == 0 && standings[i].spread < standings[p].spread {
				// Q finished at B with worse spread. Loses tiebreak to P.
				isGuaranteedBelow[i] = true
				guaranteedBelow++
				continue
			}
		}
	}

	// externalCnt[i] = non-P games Q plays against a guaranteedBelow opponent.
	// These opponents cannot reach B, so we route all 2 game pts to them
	// (Q loses the game → +0 pts, arbitrary loss margin). A Q with spread
	// >= P's can still end at B below P on spread by taking this external
	// loss with a huge margin.
	externalCnt := make([]int, n)
	for _, g := range gi.nonPGames {
		if isGuaranteedBelow[g.a] && !isGuaranteedBelow[g.b] {
			externalCnt[g.b]++
		} else if isGuaranteedBelow[g.b] && !isGuaranteedBelow[g.a] {
			externalCnt[g.a]++
		}
	}

	// Build the "stay below P" set with per-player constraints:
	//
	//   Q.spread < P.spread, can reach B via draws (maxBelow ≤ games):
	//     maxPerGame=1 (draws preserve spread → Q below P at B)
	//
	//   Q.spread >= P.spread, externalCnt[i] >= 1:
	//     full maxBelow (external loss drops spread arbitrarily → Q at B below P)
	//
	//   Q.spread >= P.spread, no external:
	//     maxBelow-- (Q must stay strictly below B; a win lifts spread and
	//     draws-only preserves it above P)
	//
	//   P has games (unbounded best spread):
	//     no restriction (P always beats Q on spread at equal points)
	//
	//   both finished (no games):
	//     fixed spread comparison, maxBelow-- if Q.spread >= P.spread
	var belowCandidates []stayBelow
	for i := 0; i < n; i++ {
		if i == p || isGuaranteedAbove[i] || isGuaranteedBelow[i] {
			continue
		}

		// Determine whether Q at exactly B points could be above P.
		// If so, Q must stay strictly below B (maxBelow--) or use
		// draws only (maxPerGame=1) to guarantee being below P.
		maxBelow := B - standings[i].points
		maxPerGame := 2

		if pHasGames {
			// P has unbounded best spread (wins by huge margins) → Q can
			// never beat P on spread at equal points. No restriction needed.
		} else if gi.nonPGamesCnt[i] == 0 {
			// Both finished. Spread is fixed. Equal spread: Q could be
			// ranked either side of P, so treat as potentially above.
			if standings[i].spread >= standings[p].spread && maxBelow > 0 {
				maxBelow--
			}
		} else if standings[i].spread < standings[p].spread &&
			maxBelow <= gi.nonPGamesCnt[i] {
			// Q has remaining games but worse spread than P. Q can reach B
			// via draws only (each draw gives 1 point, preserves spread).
			// Restrict per-game capacity to 1 so the flow only allows draws.
			maxPerGame = 1
		} else if externalCnt[i] >= 1 {
			// Q has at least one game vs a guaranteedBelow opponent. We route
			// that game as a Q-loss (opponent takes both pts, still capped
			// below B). A sufficiently large loss margin drops Q's spread
			// below P's even when Q reaches B via wins on other games. No
			// decrement needed; Q can stay at B below P.
		} else {
			// Q at B could beat P on spread (spread >= P's, no external loss
			// to absorb margin). Stay strictly below B.
			if maxBelow > 0 {
				maxBelow--
			}
		}

		if maxBelow < 0 {
			continue // Q is at B and beats P on spread → guaranteed above
		}

		absorb := maxBelow
		if absorb >= gi.nonPGamesCnt[i]*2 {
			absorb = gi.nonPGamesCnt[i] * 2
		}
		belowCandidates = append(belowCandidates, stayBelow{i, absorb, maxPerGame})
	}

	// Sort by absorb capacity ascending (tightest constraint first).
	sort.Slice(belowCandidates, func(i, j int) bool {
		return belowCandidates[i].absorb < belowCandidates[j].absorb
	})

	// Iteratively check if all belowCandidates can stay below P via max-flow.
	for {
		feasible, removeIdx := checkBestFeasibility(belowCandidates, gi.nonPGames, fg, candIdx)
		if feasible {
			break
		}
		if removeIdx >= 0 {
			belowCandidates = append(belowCandidates[:removeIdx], belowCandidates[removeIdx+1:]...)
		}
		if len(belowCandidates) == 0 {
			break
		}
	}

	minForcedAbove := guaranteedAbove + ((n - 1 - guaranteedAbove - guaranteedBelow) - len(belowCandidates))
	return max(1, 1+minForcedAbove)
}

// checkBestFeasibility checks whether all players in the set can absorb the
// points from within-set games without anyone exceeding their absorb limit.
//
// Uses max-flow: source → game nodes → player nodes → sink.
// Each game produces 2 points. Each player can absorb at most their limit.
// Feasible if max_flow = total within-set game points.
func checkBestFeasibility(candidates []stayBelow, nonPGames []gamePair, fg *flowGraph, candIdx []int) (bool, int) {
	k := len(candidates)
	if k == 0 {
		return true, -1
	}

	for ci, c := range candidates {
		candIdx[c.idx] = ci
	}
	defer func() {
		for _, c := range candidates {
			candIdx[c.idx] = -1
		}
	}()

	// Find within-set games
	type withinGame struct{ ci, cj int }
	var withinGames []withinGame
	for _, g := range nonPGames {
		ciA, ciB := candIdx[g.a], candIdx[g.b]
		if ciA >= 0 && ciB >= 0 {
			withinGames = append(withinGames, withinGame{ciA, ciB})
		}
	}

	totalGamePoints := len(withinGames) * 2
	if totalGamePoints == 0 {
		return true, -1
	}

	// Quick check: if total absorb capacity can't cover all game points,
	// definitely infeasible — skip building the flow graph.
	totalAbsorb := 0
	for _, c := range candidates {
		totalAbsorb += c.absorb
	}
	if totalAbsorb < totalGamePoints {
		worst := -1
		worstAbsorb := math.MaxInt
		for ci := range candidates {
			if candidates[ci].absorb < worstAbsorb {
				worstAbsorb = candidates[ci].absorb
				worst = ci
			}
		}
		return false, worst
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
		fg.addEdge(gn, playerNode(wg.ci), candidates[wg.ci].maxPerGame)
		fg.addEdge(gn, playerNode(wg.cj), candidates[wg.cj].maxPerGame)
	}
	for ci := 0; ci < k; ci++ {
		fg.addEdge(playerNode(ci), sink, candidates[ci].absorb)
	}

	flow := fg.maxflow(source, sink)
	if flow >= totalGamePoints {
		return true, -1
	}

	// Not feasible. Remove the player with the smallest absorb capacity.
	worst := -1
	worstAbsorb := math.MaxInt
	for ci := range candidates {
		if candidates[ci].absorb < worstAbsorb {
			worstAbsorb = candidates[ci].absorb
			worst = ci
		}
	}
	return false, worst
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

// bruteForceRanks enumerates every win/draw/loss assignment across all
// unfinished games and computes tight rank bounds for each player.
//
// For each outcome:
//
//   - points per player are determined exactly.
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
// Per-outcome best rank for P = 1 + strictAbove + forcedAboveTied.
// Per-outcome worst rank for P = 1 + strictAbove + possiblyAboveTied.
// Final bounds are min/max over all outcomes.
//
// Tight: handles asymmetric within-set spread drops (e.g. small win + huge
// loss), zero-sum spread interactions between candidates, and every other
// residual corner because it enumerates actual realizations.
func bruteForceRanks(standings []standingInfo, games []gamePair) []RankBounds {
	n := len(standings)
	g := len(games)

	best := make([]int, n)
	worst := make([]int, n)
	for i := range best {
		best[i] = n + 1
		worst[i] = 0
	}

	points := make([]int, n)
	outcome := make([]int, g) // 0 = draw, 1 = g.a wins, 2 = g.b wins

	total := 1
	for i := 0; i < g; i++ {
		total *= 3
	}

	for k := 0; k < total; k++ {
		x := k
		for i := 0; i < g; i++ {
			outcome[i] = x % 3
			x /= 3
		}

		for i := 0; i < n; i++ {
			points[i] = standings[i].points
		}
		for gi := 0; gi < g; gi++ {
			a, b := games[gi].a, games[gi].b
			switch outcome[gi] {
			case 0:
				points[a]++
				points[b]++
			case 1:
				points[a] += 2
			case 2:
				points[b] += 2
			}
		}

		for p := 0; p < n; p++ {
			strictAbove := 0
			forcedAboveTied := 0
			possiblyAboveTied := 0
			for q := 0; q < n; q++ {
				if q == p {
					continue
				}
				if points[q] > points[p] {
					strictAbove++
					continue
				}
				if points[q] < points[p] {
					continue
				}
				forced, possible := spreadOrdering(q, p, games, outcome, standings)
				if forced {
					forcedAboveTied++
				}
				if possible {
					possiblyAboveTied++
				}
			}
			bestRank := 1 + strictAbove + forcedAboveTied
			worstRank := 1 + strictAbove + possiblyAboveTied
			if bestRank < best[p] {
				best[p] = bestRank
			}
			if worstRank > worst[p] {
				worst[p] = worstRank
			}
		}
	}

	result := make([]RankBounds, n)
	for i := 0; i < n; i++ {
		result[i] = RankBounds{BestRank: best[i], WorstRank: worst[i]}
	}
	return result
}

// spreadOrdering decides, for a fixed outcome, two things about a pair (q, p)
// tied on final points:
//   - forced: Q.spread > P.spread in every margin assignment.
//   - possible: Q.spread >= P.spread in at least one margin assignment
//     (equal spread counts as possibly-above via username tiebreak).
//
// It analyzes Σ Δ_g · m_g where Δ_g = sign_Q(g) - sign_P(g) and m_g ranges
// over admissible margins (0 for draws, ≥ 1 otherwise). See bruteForceRanks
// for the full reasoning.
func spreadOrdering(q, p int, games []gamePair, outcome []int, standings []standingInfo) (forced, possible bool) {
	minSumFinite := 0
	maxSumFinite := 0
	minUnbounded := false
	maxUnbounded := false

	for gi, gm := range games {
		out := outcome[gi]
		if out == 0 {
			continue
		}
		signP := 0
		switch {
		case gm.a == p:
			if out == 1 {
				signP = 1
			} else {
				signP = -1
			}
		case gm.b == p:
			if out == 2 {
				signP = 1
			} else {
				signP = -1
			}
		}
		signQ := 0
		switch {
		case gm.a == q:
			if out == 1 {
				signQ = 1
			} else {
				signQ = -1
			}
		case gm.b == q:
			if out == 2 {
				signQ = 1
			} else {
				signQ = -1
			}
		}
		delta := signQ - signP
		if delta > 0 {
			maxUnbounded = true
			minSumFinite += delta
		} else if delta < 0 {
			minUnbounded = true
			maxSumFinite += delta
		}
	}

	diff := standings[p].spread - standings[q].spread

	if maxUnbounded {
		possible = true
	} else {
		possible = maxSumFinite >= diff
	}
	if minUnbounded {
		forced = false
	} else {
		forced = minSumFinite > diff
	}
	return forced, possible
}
