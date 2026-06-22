package league

// Phase-3 pinpoint: from the replay data, find one state where the maxflow
// heuristic (CalculatePossibleRanks, used when a cluster has >bruteForceThreshold
// games) returns an OVER-TIGHT bound versus the exact brute-force oracle, then
// delta-debug-minimize it to the smallest standings+games that still fail. The
// printed case becomes a synthetic, data-free unit test for the fix.
//
// Env-gated like the replay test; never runs on CI.

import (
	"os"
	"sort"
	"testing"
)

func sortStandings(s []standingInfo) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].points != s[j].points {
			return s[i].points > s[j].points
		}
		if s[i].spread != s[j].spread {
			return s[i].spread > s[j].spread
		}
		return s[i].userID < s[j].userID
	})
}

// feasLimit caps the oracle's 3^games enumeration. The starting state must have
// its largest cluster in (bruteForceThreshold, feasLimit], i.e. maxflow is used
// yet exact brute is still cheap.
const pinpointFeasLimit = 12

func toGamePairs(st []standingInfo, unf []unfinishedGame) []gamePair {
	idx := make(map[int32]int, len(st))
	for i, s := range st {
		idx[s.userID] = i
	}
	var g []gamePair
	for _, u := range unf {
		a, okA := idx[u.player0ID]
		b, okB := idx[u.player1ID]
		if okA && okB {
			g = append(g, gamePair{a, b})
		}
	}
	return g
}

// overTightPlayer returns the index of a player whose maxflow bound is tighter
// than the exact bound (best too high or worst too low), or -1. Requires the
// largest cluster in (bruteForceThreshold, feasLimit] so maxflow is exercised
// and the oracle is affordable.
func overTightPlayer(st []standingInfo, unf []unfinishedGame) int {
	games := toGamePairs(st, unf)
	clusters := buildBruteForceClusters(st, games)
	mc := maxClusterGames(clusters)
	if mc <= bruteForceThreshold || mc > pinpointFeasLimit {
		return -1
	}
	exact := bruteForceRanksFromClusters(clusters, st, games)
	maxf := CalculatePossibleRanks(st, unf)
	for i := range st {
		if maxf[i].BestRank > exact[i].BestRank || maxf[i].WorstRank < exact[i].WorstRank {
			return i
		}
	}
	return -1
}

func countGamesOf(id int32, unf []unfinishedGame) int {
	c := 0
	for _, u := range unf {
		if u.player0ID == id || u.player1ID == id {
			c++
		}
	}
	return c
}

func withoutGame(st []standingInfo, unf []unfinishedGame, gi int) ([]standingInfo, []unfinishedGame) {
	g := unf[gi]
	newUnf := make([]unfinishedGame, 0, len(unf)-1)
	newUnf = append(newUnf, unf[:gi]...)
	newUnf = append(newUnf, unf[gi+1:]...)
	newSt := make([]standingInfo, len(st))
	copy(newSt, st)
	for i := range newSt {
		if newSt[i].userID == g.player0ID || newSt[i].userID == g.player1ID {
			newSt[i].gamesRemaining--
		}
	}
	return newSt, newUnf
}

func withoutPlayer(st []standingInfo, pi int) []standingInfo {
	newSt := make([]standingInfo, 0, len(st)-1)
	newSt = append(newSt, st[:pi]...)
	newSt = append(newSt, st[pi+1:]...)
	return newSt
}

// minimizeCase greedily removes games (decrementing gamesRemaining) and
// zero-game players while an over-tight player still exists.
func minimizeCase(st []standingInfo, unf []unfinishedGame) ([]standingInfo, []unfinishedGame) {
	for changed := true; changed; {
		changed = false
		for i := range unf {
			cst, cunf := withoutGame(st, unf, i)
			if overTightPlayer(cst, cunf) >= 0 {
				st, unf = cst, cunf
				changed = true
				break
			}
		}
		if changed {
			continue
		}
		for i := range st {
			if countGamesOf(st[i].userID, unf) != 0 {
				continue
			}
			cst := withoutPlayer(st, i)
			if overTightPlayer(cst, unf) >= 0 {
				st = cst
				changed = true
				break
			}
		}
	}
	return st, unf
}

func TestRankBoundsPinpoint(t *testing.T) {
	path := os.Getenv("LEAGUE_REPLAY_CSV")
	if path == "" {
		t.Skip("set LEAGUE_REPLAY_CSV to the exported league-all.csv to run")
	}
	played, unplayed, order := loadSchedule(t, path)

	// Scan for the first over-tight maxflow state.
	var foundSt []standingInfo
	var foundUnf []unfinishedGame
	found := false
	for _, k := range order {
		if found {
			break
		}
		scanDivision(played[k], unplayed[k], func(st []standingInfo, unf []unfinishedGame) bool {
			if overTightPlayer(st, unf) >= 0 {
				foundSt, foundUnf = st, unf
				found = true
				return true // stop
			}
			return false
		})
	}
	if !found {
		t.Skip("no over-tight maxflow state found within feasLimit; raise pinpointFeasLimit")
	}

	t.Logf("starting case: n=%d games=%d", len(foundSt), len(foundUnf))
	st, unf := minimizeCase(foundSt, foundUnf)
	printCase(t, st, unf)
	t.Fatalf("minimal over-tight maxflow case: n=%d games=%d (see log above)", len(st), len(unf))
}

// scanDivision walks a division's played prefix, invoking visit with the
// (sorted standings, remaining games) at each snapshot; stops early if visit
// returns true. Mirrors replayDivision's state construction.
func scanDivision(played []replayGame, unplayed []unfinishedGame, visit func([]standingInfo, []unfinishedGame) bool) {
	var players []int32
	seen := map[int32]bool{}
	totalGP := map[int32]int{}
	add := func(p int32) {
		if !seen[p] {
			seen[p] = true
			players = append(players, p)
		}
		totalGP[p]++
	}
	for _, g := range played {
		add(g.p0)
		add(g.p1)
	}
	for _, u := range unplayed {
		add(u.player0ID)
		add(u.player1ID)
	}
	n := len(players)
	expected := CalculateExpectedGamesPerPlayer(n)
	for _, p := range players {
		if totalGP[p] != expected {
			return
		}
	}

	type acc struct{ wins, draws, spread int }
	st := make(map[int32]*acc, n)
	playedCnt := make(map[int32]int, n)
	for _, p := range players {
		st[p] = &acc{}
	}
	for k := 0; k <= len(played); k++ {
		standings := make([]standingInfo, 0, n)
		for _, p := range players {
			a := st[p]
			standings = append(standings, standingInfo{
				userID:         p,
				points:         a.wins*2 + a.draws,
				spread:         a.spread,
				gamesRemaining: totalGP[p] - playedCnt[p],
			})
		}
		sortStandings(standings)
		unf := make([]unfinishedGame, 0, len(unplayed)+len(played)-k)
		unf = append(unf, unplayed...)
		for _, g := range played[k:] {
			unf = append(unf, unfinishedGame{player0ID: g.p0, player1ID: g.p1})
		}
		if visit(standings, unf) {
			return
		}
		if k < len(played) {
			g := played[k]
			playedCnt[g.p0]++
			playedCnt[g.p1]++
			st[g.p0].spread += g.s0 - g.s1
			st[g.p1].spread += g.s1 - g.s0
			switch g.result {
			case 1:
				st[g.p0].wins++
			case -1:
				st[g.p1].wins++
			default:
				st[g.p0].draws++
				st[g.p1].draws++
			}
		}
	}
}

func printCase(t *testing.T, st []standingInfo, unf []unfinishedGame) {
	remap := make(map[int32]int32, len(st))
	for i, s := range st {
		remap[s.userID] = int32(i + 1)
	}
	t.Log("standings := []standingInfo{")
	for _, s := range st {
		t.Logf("\tsi(%d, %d, %d, %d),", remap[s.userID], s.points, s.spread, s.gamesRemaining)
	}
	t.Log("}")
	t.Log("games := []unfinishedGame{")
	for _, u := range unf {
		t.Logf("\tuf(%d, %d),", remap[u.player0ID], remap[u.player1ID])
	}
	t.Log("}")

	games := toGamePairs(st, unf)
	clusters := buildBruteForceClusters(st, games)
	exact := bruteForceRanksFromClusters(clusters, st, games)
	maxf := CalculatePossibleRanks(st, unf)
	for i := range st {
		tag := ""
		if maxf[i].BestRank > exact[i].BestRank || maxf[i].WorstRank < exact[i].WorstRank {
			tag = "  <-- OVER-TIGHT"
		}
		t.Logf("player %d (pts=%d spread=%d gr=%d): maxflow %d-%d  exact %d-%d%s",
			remap[st[i].userID], st[i].points, st[i].spread, st[i].gamesRemaining,
			maxf[i].BestRank, maxf[i].WorstRank, exact[i].BestRank, exact[i].WorstRank, tag)
	}
}
