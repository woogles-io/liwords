package league

import "testing"

// BenchmarkBruteWorst measures the brute-force tier's worst case: a large
// division bunched into a single cluster sitting at the brute-force game
// threshold, so 3^bruteForceThreshold leaves each rank every member. This is the
// hot path that sets the per-snapshot budget; the benchmark guards against perf
// regressions in enumerateBruteForceCluster. (26 members is a representative
// large division, not an assumed maximum.)
func BenchmarkBruteWorst(b *testing.B) {
	const n = 26
	st := make([]standingInfo, n)
	for i := range st {
		st[i] = standingInfo{userID: int32(i + 1), points: 20, spread: 0, gamesRemaining: 0}
	}
	var unf []unfinishedGame
	for i := range bruteForceThreshold { // a path of threshold games keeps it one cluster
		unf = append(unf, unfinishedGame{player0ID: int32(i + 1), player1ID: int32(i + 2)})
		st[i].gamesRemaining++
		st[i+1].gamesRemaining++
	}
	sortStandings(st)
	for b.Loop() {
		CalculatePossibleRanks(st, unf)
	}
}
