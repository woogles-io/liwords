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

	sc := newScratch(n, len(games))
	results := make([]RankBounds, n)
	for p := 0; p < n; p++ {
		if standings[p].gamesRemaining > 0 {
			bestPts := standings[p].points + standings[p].gamesRemaining*2
			// Min ceiling excluding P
			pCeil := standings[p].points + standings[p].gamesRemaining*2
			minCeilExP := minCeil1
			if pCeil == minCeil1 {
				minCeilExP = minCeil2
			}
			if bestPts >= maxFloor && minCeilExP >= standings[p].points {
				results[p] = RankBounds{1, n}
				continue
			}
		}
		results[p] = rankBoundsForPlayer(p, standings, games, sc)
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
// Slices are reused across players via scratch to avoid repeated allocation.
type playerGameInfo struct {
	gamesVsP     []int     // how many remaining games each player has vs P
	nonPGames    []gamePair // games not involving P
	nonPGamesCnt []int     // remaining non-P games per player
}

// scratch holds pre-allocated buffers reused across per-player computations.
type scratch struct {
	gamesVsP        []int      // len n
	nonPGamesCnt    []int      // len n
	nonPGames       []gamePair // cap totalGames
	effectivePts    []int      // len n
	inSet           []bool     // len n
	candidates      []cand
	belowCandidates []stayBelow
	playerFlow      []int // len n, reused in checkFeasibility
	candIdx         []int // len n, maps player index → candidate index (-1 if not)
	flow            *flowGraph
}

func newScratch(n, totalGames int) *scratch {
	// Max flow graph size: 2 (source/sink) + totalGames (game nodes) + n (player nodes)
	maxNodes := 2 + totalGames + n
	return &scratch{
		gamesVsP:     make([]int, n),
		nonPGamesCnt: make([]int, n),
		nonPGames:    make([]gamePair, 0, totalGames),
		effectivePts: make([]int, n),
		inSet:        make([]bool, n),
		playerFlow:   make([]int, n),
		candIdx:      initSlice(n, -1),
		flow:         newFlowGraph(maxNodes),
	}
}

func (s *scratch) decomposeGames(p int, n int, allGames []gamePair) playerGameInfo {
	// Clear reused slices
	for i := range s.gamesVsP {
		s.gamesVsP[i] = 0
		s.nonPGamesCnt[i] = 0
	}
	s.nonPGames = s.nonPGames[:0]

	for _, g := range allGames {
		if g.a == p {
			s.gamesVsP[g.b]++
		} else if g.b == p {
			s.gamesVsP[g.a]++
		} else {
			s.nonPGames = append(s.nonPGames, g)
		}
	}
	for _, g := range s.nonPGames {
		s.nonPGamesCnt[g.a]++
		s.nonPGamesCnt[g.b]++
	}
	return playerGameInfo{
		gamesVsP:     s.gamesVsP,
		nonPGames:    s.nonPGames,
		nonPGamesCnt: s.nonPGamesCnt,
	}
}

// rankBoundsForPlayer computes both best and worst rank for player P,
// sharing the game decomposition between the two computations.
func rankBoundsForPlayer(p int, standings []standingInfo, allGames []gamePair, sc *scratch) RankBounds {
	n := len(standings)
	gi := sc.decomposeGames(p, n, allGames)
	return RankBounds{
		BestRank:  bestRankForPlayer(p, standings, gi, sc),
		WorstRank: worstRankForPlayer(p, standings, gi, sc),
	}
}

// ---------------------------------------------------------------------------
// worst rank: maximize the number of players finishing above P
// ---------------------------------------------------------------------------

func worstRankForPlayer(p int, standings []standingInfo, gi playerGameInfo, sc *scratch) int {
	n := len(standings)
	W := standings[p].points // P loses all remaining → keeps current points

	// After P loses all, each opponent of P gets +2 per game vs P.
	for i := range standings {
		sc.effectivePts[i] = standings[i].points + gi.gamesVsP[i]*2
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
	sc.candidates = sc.candidates[:0]
	for i := 0; i < n; i++ {
		if i == p {
			continue
		}

		tieDeficit := W - sc.effectivePts[i] // points needed to match P
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
			sc.candidates = append(sc.candidates, cand{i, deficit})
		}
	}

	if len(sc.candidates) <= 1 {
		return 1 + len(sc.candidates) // 0 or 1 candidates: trivial
	}

	// Sort candidates by deficit ascending (cheapest to satisfy first).
	sort.Slice(sc.candidates, func(i, j int) bool {
		return sc.candidates[i].deficit < sc.candidates[j].deficit
	})

	// Iteratively check feasibility via max-flow and remove infeasible candidates.
	for i := range sc.inSet {
		sc.inSet[i] = false
	}
	for _, c := range sc.candidates {
		sc.inSet[c.idx] = true
	}

	for {
		feasible, infeasibleIdx := checkFeasibility(sc.candidates, sc.inSet, sc.effectivePts, gi.nonPGames, W, sc.flow, sc.playerFlow, sc.candIdx)
		if feasible {
			break
		}
		// Remove the infeasible candidate with the highest deficit
		if infeasibleIdx >= 0 {
			sc.inSet[sc.candidates[infeasibleIdx].idx] = false
			sc.candidates = append(sc.candidates[:infeasibleIdx], sc.candidates[infeasibleIdx+1:]...)
		}
		if len(sc.candidates) == 0 {
			break
		}
	}

	return 1 + len(sc.candidates)
}

// checkFeasibility uses max-flow to determine whether all candidates in the set
// can simultaneously reach W points from non-P games.
//
// Returns (true, -1) if feasible, or (false, idxToRemove) if not.
func checkFeasibility(candidates []cand, inSet []bool, effectivePts []int, nonPGames []gamePair, W int, fg *flowGraph, playerFlow []int, candIdx []int) (bool, int) {
	k := len(candidates)
	if k == 0 {
		return true, -1
	}

	// Map candidate player indices → 0..k-1 via candIdx slice.
	// Caller must clear candIdx to -1 for non-candidates.
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
	externalCnt := make([]int, k) // external non-P games per candidate

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
	// Check how much flow each player actually received.
	for ci := 0; ci < k; ci++ {
		playerFlow[ci] = 0
	}
	for ci := 0; ci < k; ci++ {
		pn := playerNode(ci)
		for _, e := range fg.adj[pn] {
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

func bestRankForPlayer(p int, standings []standingInfo, gi playerGameInfo, sc *scratch) int {
	n := len(standings)
	B := standings[p].points + standings[p].gamesRemaining*2 // P wins all remaining

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
		} else if qWorst == B && !pHasGames && gi.nonPGamesCnt[i] == 0 &&
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
	sc.belowCandidates = sc.belowCandidates[:0]
	for i := 0; i < n; i++ {
		if i == p {
			continue
		}
		if standings[i].points > B {
			continue // already guaranteed above
		}

		// Determine whether Q at exactly B points could be above P.
		// If so, Q must stay strictly below B to guarantee being below P.
		qBeatsOnSpreadAtB := false
		if pHasGames {
			// P has unbounded best spread (wins by huge margins) → Q can
			// never beat P on spread at equal points.
			qBeatsOnSpreadAtB = false
		} else if gi.nonPGamesCnt[i] > 0 {
			// Q has remaining games. If forced to absorb points via wins,
			// Q's spread could change arbitrarily (a single win can shift
			// spread by hundreds). We can't guarantee Q stays below P on
			// spread at B points. To ensure the bound is correct, Q must
			// stay strictly below B.
			qBeatsOnSpreadAtB = true
		} else {
			// Both finished. Spread is fixed. Equal spread: Q could be
			// ranked either side of P, so treat as potentially above.
			qBeatsOnSpreadAtB = standings[i].spread >= standings[p].spread
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
		if absorb >= gi.nonPGamesCnt[i]*2 {
			absorb = gi.nonPGamesCnt[i] * 2
		}
		sc.belowCandidates = append(sc.belowCandidates, stayBelow{i, absorb})
	}

	// Sort by absorb capacity ascending (tightest constraint first — these are
	// the ones most likely to be forced above B).
	sort.Slice(sc.belowCandidates, func(i, j int) bool {
		return sc.belowCandidates[i].absorb < sc.belowCandidates[j].absorb
	})

	// Iteratively check if all belowCandidates can stay at ≤ B via max-flow.
	// If not, remove the one with the least absorb capacity (most constrained).
	for i := range sc.inSet {
		sc.inSet[i] = false
	}
	for _, c := range sc.belowCandidates {
		sc.inSet[c.idx] = true
	}

	for {
		feasible, removeIdx := checkBestFeasibility(sc.belowCandidates, gi.nonPGames, sc.flow, sc.candIdx)
		if feasible {
			break
		}
		if removeIdx >= 0 {
			sc.inSet[sc.belowCandidates[removeIdx].idx] = false
			sc.belowCandidates = append(sc.belowCandidates[:removeIdx], sc.belowCandidates[removeIdx+1:]...)
		}
		if len(sc.belowCandidates) == 0 {
			break
		}
	}

	// forced above = (n-1) - len(belowCandidates) who stayed, accounting for guaranteed above
	minForcedAbove := guaranteedAbove + ((n - 1 - guaranteedAbove) - len(sc.belowCandidates))

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

	fg.reset(numNodes)
	for gi, wg := range withinGames {
		gn := gameNode(gi)
		fg.addEdge(source, gn, 2)
		fg.addEdge(gn, playerNode(wg.ci), 2)
		fg.addEdge(gn, playerNode(wg.cj), 2)
	}
	for ci := 0; ci < k; ci++ {
		fg.addEdge(playerNode(ci), sink, candidates[ci].absorb)
	}

	flow := fg.maxflow(source, sink)
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
