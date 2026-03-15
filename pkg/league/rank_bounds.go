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

	results := make([]RankBounds, n)
	for p := 0; p < n; p++ {
		results[p].WorstRank = worstRankForPlayer(p, standings, games)
		results[p].BestRank = bestRankForPlayer(p, standings, games)
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

// ---------------------------------------------------------------------------
// worst rank: maximize the number of players finishing above P
// ---------------------------------------------------------------------------

func worstRankForPlayer(p int, standings []standingInfo, allGames []gamePair) int {
	n := len(standings)
	W := standings[p].points // P loses all remaining → keeps current points

	// Separate games into P-games and non-P games.
	gamesVsP := make([]int, n) // how many remaining games each player has vs P
	var nonPGames []gamePair
	for _, g := range allGames {
		if g.a == p {
			gamesVsP[g.b]++
		} else if g.b == p {
			gamesVsP[g.a]++
		} else {
			nonPGames = append(nonPGames, g)
		}
	}

	// After P loses all, each opponent of P gets +2 per game vs P.
	effectivePts := make([]int, n)
	nonPGamesCnt := make([]int, n) // remaining non-P games per player
	for i := range standings {
		effectivePts[i] = standings[i].points + gamesVsP[i]*2
	}
	for _, g := range nonPGames {
		nonPGamesCnt[g.a]++
		nonPGamesCnt[g.b]++
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
		} else if nonPGamesCnt[i] == 0 {
			// Q has no remaining games. Spread is fixed.
			if standings[i].spread > standings[p].spread {
				deficit = max(0, tieDeficit) // tie suffices
			} else {
				deficit = max(0, tieDeficit+1) // need strict points advantage
			}
		} else if standings[i].spread >= standings[p].spread {
			// Q has remaining games and spread ≥ P's. Draws preserve it,
			// wins improve it. Tie on points suffices.
			deficit = max(0, tieDeficit)
		} else if tieDeficit == 1 && nonPGamesCnt[i] == 1 {
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

		if deficit <= nonPGamesCnt[i]*2 {
			candidates = append(candidates, cand{i, deficit})
		}
	}

	if len(candidates) == 0 {
		return 1
	}

	// Sort candidates by deficit ascending (cheapest to satisfy first).
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].deficit < candidates[j].deficit
	})

	// Iteratively check feasibility via max-flow and remove infeasible candidates.
	inSet := make([]bool, n)
	for _, c := range candidates {
		inSet[c.idx] = true
	}

	for {
		feasible, infeasibleIdx := checkFeasibility(candidates, inSet, effectivePts, nonPGames, W)
		if feasible {
			break
		}
		// Remove the infeasible candidate with the highest deficit
		if infeasibleIdx >= 0 {
			inSet[candidates[infeasibleIdx].idx] = false
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
func checkFeasibility(candidates []cand, inSet []bool, effectivePts []int, nonPGames []gamePair, W int) (bool, int) {
	k := len(candidates)
	if k == 0 {
		return true, -1
	}

	// Map candidate player indices → 0..k-1
	candMap := make(map[int]int, k)
	for ci, c := range candidates {
		candMap[c.idx] = ci
	}

	// Identify within-set games and external games for each candidate.
	type withinGame struct {
		ci, cj int // indices into candidates slice
	}
	var withinGames []withinGame
	externalCnt := make([]int, k) // external non-P games per candidate

	for _, g := range nonPGames {
		ciA, aIn := candMap[g.a]
		ciB, bIn := candMap[g.b]
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
		// Not enough points from within-set games, remove most expensive
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

	g := newFlowGraph(numNodes)
	for gi, wg := range withinGames {
		gn := gameNode(gi)
		g.addEdge(source, gn, 2)
		g.addEdge(gn, playerNode(wg.ci), 2)
		g.addEdge(gn, playerNode(wg.cj), 2)
	}
	for ci := 0; ci < k; ci++ {
		if adjustedDeficit[ci] > 0 {
			g.addEdge(playerNode(ci), sink, adjustedDeficit[ci])
		}
	}

	flow := g.maxflow(source, sink)
	if flow >= totalNeeded {
		return true, -1
	}

	// Not feasible. Find the candidate with the largest remaining deficit.
	// Check how much flow each player actually received.
	playerFlow := make([]int, k)
	for ci := 0; ci < k; ci++ {
		pn := playerNode(ci)
		for _, e := range g.adj[pn] {
			if e.to == sink {
				// Flow through this edge = original capacity - remaining capacity
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
	idx    int
	absorb int // max additional points Q can receive without exceeding B
}

// ---------------------------------------------------------------------------
// best rank: minimize the number of players forced above P
// ---------------------------------------------------------------------------

func bestRankForPlayer(p int, standings []standingInfo, allGames []gamePair) int {
	n := len(standings)
	B := standings[p].points + standings[p].gamesRemaining*2 // P wins all remaining

	// Separate P's games from other games
	gamesVsP := make([]int, n)
	var nonPGames []gamePair
	for _, g := range allGames {
		if g.a == p {
			gamesVsP[g.b]++
		} else if g.b == p {
			gamesVsP[g.a]++
		} else {
			nonPGames = append(nonPGames, g)
		}
	}

	nonPGamesCnt := make([]int, n)
	for _, g := range nonPGames {
		nonPGamesCnt[g.a]++
		nonPGamesCnt[g.b]++
	}

	// When P wins all, P's opponents each LOSE their game vs P (get 0 from it).
	// Q's effective points = Q.currentPoints (unchanged; the loss to P adds 0).
	// Q can absorb at most B - Q.currentPoints points at or below B.

	pHasGames := standings[p].gamesRemaining > 0

	// Count guaranteed above: Q's worst case (lose all remaining) still beats P.
	// Q is guaranteed above if:
	//   Q.worstPoints > B, OR
	//   Q.worstPoints == B AND Q always beats P on spread at B points
	// For the spread check: Q losing all remaining means Q's spread decreases,
	// so we can only guarantee the spread tiebreak if Q has no remaining games
	// (spread is fixed) and it's better than P's.
	guaranteedAbove := 0
	for i := 0; i < n; i++ {
		if i == p {
			continue
		}
		qWorst := standings[i].points
		if qWorst > B {
			guaranteedAbove++
		} else if qWorst == B && !pHasGames && nonPGamesCnt[i] == 0 &&
			standings[i].spread > standings[p].spread {
			guaranteedAbove++
		}
	}

	// Build the "stay below P" set. A player Q is below P when:
	//   Q.points < B, OR
	//   Q.points == B AND Q has worse spread than P (or P has +∞ spread)
	//
	// If Q would beat P on spread at B points, Q must stay strictly below B
	// (absorb = B - Q.points - 1). This matters for draws: a draw gives each
	// player 1 point with 0 spread change, so a player reaching B via a draw
	// keeps their existing spread.
	var belowCandidates []stayBelow
	for i := 0; i < n; i++ {
		if i == p {
			continue
		}
		if standings[i].points > B {
			continue // already guaranteed above
		}

		// Determine whether Q at exactly B points would be above P.
		// If so, Q must stay strictly below B to be "below P".
		qBeatsOnSpreadAtB := false
		if pHasGames {
			// P has +∞ best spread → Q can never beat P on spread at equal points.
			qBeatsOnSpreadAtB = false
		} else if nonPGamesCnt[i] > 0 {
			// Q has remaining games → Q's spread is uncertain. In the best case
			// for P, Q's spread could end up below P's. But in the worst case
			// (for best rank computation), Q could also draw their way to B with
			// unchanged spread. If Q's current spread already beats P, a draw
			// preserves that. So Q COULD beat P on spread at B.
			qBeatsOnSpreadAtB = standings[i].spread >= standings[p].spread
		} else {
			// Both finished. Spread is fixed.
			qBeatsOnSpreadAtB = standings[i].spread > standings[p].spread
		}

		maxBelow := B - standings[i].points
		if qBeatsOnSpreadAtB && maxBelow > 0 {
			// Q at B would beat P on spread, so Q must stay at B-1 or below.
			maxBelow--
		}
		if maxBelow < 0 {
			continue // Q is at B and beats P on spread → guaranteed above (handled above)
		}

		absorb := maxBelow
		if absorb >= nonPGamesCnt[i]*2 {
			absorb = nonPGamesCnt[i] * 2
		}
		belowCandidates = append(belowCandidates, stayBelow{i, absorb})
	}

	// Sort by absorb capacity ascending (tightest constraint first — these are
	// the ones most likely to be forced above B).
	sort.Slice(belowCandidates, func(i, j int) bool {
		return belowCandidates[i].absorb < belowCandidates[j].absorb
	})

	// Iteratively check if all belowCandidates can stay at ≤ B via max-flow.
	// If not, remove the one with the least absorb capacity (most constrained).
	inSet := make(map[int]bool, len(belowCandidates))
	for _, c := range belowCandidates {
		inSet[c.idx] = true
	}

	for {
		feasible, removeIdx := checkBestFeasibility(belowCandidates, inSet, nonPGames)
		if feasible {
			break
		}
		if removeIdx >= 0 {
			inSet[belowCandidates[removeIdx].idx] = false
			belowCandidates = append(belowCandidates[:removeIdx], belowCandidates[removeIdx+1:]...)
		}
		if len(belowCandidates) == 0 {
			break
		}
	}

	forcedAbove := 0
	for i := 0; i < n; i++ {
		if i == p && !inSet[i] && i != p {
			forcedAbove++
		}
	}
	// Actually: forced above = (n-1) - len(belowCandidates) who stayed, accounting for guaranteed above
	minForcedAbove := guaranteedAbove + ((n - 1 - guaranteedAbove) - len(belowCandidates))

	return max(1, 1+minForcedAbove)
}

// checkBestFeasibility checks whether all players in the set can absorb the
// points from within-set games without anyone exceeding their absorb limit.
//
// Uses max-flow: source → game nodes → player nodes → sink.
// Each game produces 2 points. Each player can absorb at most their limit.
// Feasible if max_flow = total within-set game points.
func checkBestFeasibility(candidates []stayBelow, inSet map[int]bool, nonPGames []gamePair) (bool, int) {
	k := len(candidates)
	if k == 0 {
		return true, -1
	}

	candMap := make(map[int]int, k)
	for ci, c := range candidates {
		candMap[c.idx] = ci
	}

	// Find within-set games
	type withinGame struct{ ci, cj int }
	var withinGames []withinGame
	for _, g := range nonPGames {
		ciA, aIn := candMap[g.a]
		ciB, bIn := candMap[g.b]
		if aIn && bIn {
			withinGames = append(withinGames, withinGame{ciA, ciB})
		}
		// Games between set and non-set: set player can lose (get 0), so no constraint
	}

	totalGamePoints := len(withinGames) * 2
	if totalGamePoints == 0 {
		return true, -1
	}

	// Quick check: total absorb capacity
	totalAbsorb := 0
	for _, c := range candidates {
		totalAbsorb += c.absorb
	}
	if totalAbsorb >= totalGamePoints {
		// Check via max-flow for graph structure constraints
	}

	numGameNodes := len(withinGames)
	numNodes := 2 + numGameNodes + k
	source, sink := 0, 1
	playerNode := func(ci int) int { return 2 + numGameNodes + ci }
	gameNode := func(gi int) int { return 2 + gi }

	g := newFlowGraph(numNodes)
	for gi, wg := range withinGames {
		gn := gameNode(gi)
		g.addEdge(source, gn, 2)
		g.addEdge(gn, playerNode(wg.ci), 2)
		g.addEdge(gn, playerNode(wg.cj), 2)
	}
	for ci := 0; ci < k; ci++ {
		g.addEdge(playerNode(ci), sink, candidates[ci].absorb)
	}

	flow := g.maxflow(source, sink)
	if flow >= totalGamePoints {
		return true, -1
	}

	// Not feasible. Remove the player with the smallest absorb capacity
	// (the biggest bottleneck).
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
// Edmonds-Karp max-flow
// ---------------------------------------------------------------------------

type flowEdge struct {
	to, rev, cap int
}

type flowGraph struct {
	adj [][]flowEdge
}

func newFlowGraph(n int) *flowGraph {
	return &flowGraph{adj: make([][]flowEdge, n)}
}

func (g *flowGraph) addEdge(from, to, cap int) {
	g.adj[from] = append(g.adj[from], flowEdge{to, len(g.adj[to]), cap})
	g.adj[to] = append(g.adj[to], flowEdge{from, len(g.adj[from]) - 1, 0})
}

func (g *flowGraph) maxflow(s, t int) int {
	n := len(g.adj)
	total := 0
	parent := make([]int, n)
	parentEdge := make([]int, n)
	for {
		// BFS to find augmenting path
		for i := range parent {
			parent[i] = -1
		}
		parent[s] = s
		queue := []int{s}
		for len(queue) > 0 && parent[t] == -1 {
			u := queue[0]
			queue = queue[1:]
			for i, e := range g.adj[u] {
				if parent[e.to] == -1 && e.cap > 0 {
					parent[e.to] = u
					parentEdge[e.to] = i
					queue = append(queue, e.to)
				}
			}
		}
		if parent[t] == -1 {
			break
		}
		// Find bottleneck
		pathFlow := math.MaxInt
		for v := t; v != s; v = parent[v] {
			c := g.adj[parent[v]][parentEdge[v]].cap
			if c < pathFlow {
				pathFlow = c
			}
		}
		// Update residual graph
		for v := t; v != s; v = parent[v] {
			e := &g.adj[parent[v]][parentEdge[v]]
			e.cap -= pathFlow
			g.adj[v][e.rev].cap += pathFlow
		}
		total += pathFlow
	}
	return total
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
