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
// Only reached when len(games) > bruteForceThreshold. For smaller divisions
// CalculatePossibleRanks dispatches to bruteForceRanks, which enumerates
// outcomes and returns tight bounds covering every spread interaction.
//
// Algorithm:
//   1. Compute B = P's maximum possible score (P wins all remaining games).
//   2. Classify each non-P player as guaranteedAbove, guaranteedBelow, or
//      an open candidate.
//   3. Precompute externalCnt[i] = open candidate Q's non-P games vs a
//      guaranteedBelow opponent.
//   4. Build belowCandidates with per-player absorb caps and per-game caps,
//      tightened for spread (see below).
//   5. Use max-flow to check if all belowCandidates can simultaneously absorb
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
// Spread treatment when Q can reach B:
//
//   Q.spread < P.spread, draws-only path (maxPerGame=1):
//     Q reaches B via draws (1 pt each, 0 spread change), preserving
//     Q.spread < P.spread so Q sits below P on tiebreak.
//
//   Q.spread >= P.spread, externalCnt[Q] >= 1:
//     Q plays at least one guaranteedBelow opponent. Routing that game as
//     Q's loss (opponent takes both pts, still capped below B) lets Q
//     concede an arbitrary margin, dropping Q.finalSpread below P.spread
//     even when Q hits B via wins elsewhere. Full absorb=maxBelow.
//
//   Q.spread >= P.spread, no external:
//     Q at B ties P on points and beats P on spread (wins raise it, draws
//     preserve it). Decrement maxBelow so Q stays strictly below B on points.
//     This can still be pessimistic when Q's within-set opponents have
//     maxPerGame=2 (Q could realize 1W+1L with a huge loss margin, reaching
//     B at a dropped spread). For division sizes where this matters the
//     brute-force path handles the case; the heuristic leaves it loose.
//
// Guarantees:
//   - bestRank is achievable (flow outcomes map to real game outcomes, with
//     realizable margins for the external-loss case).
//   - bestRank-1 can be wrongly reported as impossible in the pessimistic
//     case above (Q.spread >= P's, no external, opponents maxPerGame=2).
//     Brute force covers small divisions; large divisions where this triggers
//     are very rare in practice.
//   - worstRank+1 is always impossible (sound upper bound).
//   - worstRank can be pessimistic when the candidate set is a closed
//     round-robin whose initial spread sum is too small for every member to
//     strictly beat P on tiebreak. Brute force covers small divisions.
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
			gm := allGames[c.games[localGi]]
			// Both endpoints are guaranteed cluster members because games are
			// assigned to clusters by the union-find root.
			a := memberIdx[gm.a]
			b := memberIdx[gm.b]
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

		for p := range m {
			strictAbove := 0
			tied = tied[:0]
			for q := range m {
				if q == p {
					continue
				}
				switch {
				case points[q] > points[p]:
					strictAbove++
				case points[q] == points[p]:
					tied = append(tied, q)
				}
			}
			var minAbove, maxGE int
			if standings[c.members[p]].gamesRemaining > 0 {
				// Free P: per-pair is exact in aggregate (the lose-all and
				// win-all leaves give P's true rank extremes). Fast path.
				for _, q := range tied {
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
			} else {
				// Fixed P: the per-pair view over-counts coupled tied players.
				// If no two of P's tied opponents are coupled (no decided game
				// this leaf between two players both at P's points), each is
				// independent and a per-member reachable-spread box is exact;
				// only genuine coupling needs the joint closed-form.
				if coupledVal[points[p]] {
					minAbove, maxGE = jointFixedTied(p, tied, winMask, baseSpread, localGames, js)
				} else {
					sprP := baseSpread[p]
					for _, q := range tied {
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
	}

	for i := 0; i < m; i++ {
		result[c.members[i]] = RankBounds{BestRank: best[i], WorstRank: worst[i]}
	}
}

// spreadInf is a sentinel for an unbounded (+/-inf) reachable spread. It is far
// larger than any real spread sum (a cluster has <= ~26 players), so finite
// values never reach it and sums of a handful of sentinels do not overflow.
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
