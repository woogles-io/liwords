package standings_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"
	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	"github.com/woogles-io/liwords/pkg/pair/verifyreq"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"golang.org/x/exp/rand"
)

func TestStandings(t *testing.T) {
	is := is.New(t)
	copRand := rand.New(rand.NewSource(0))
	var pairErr pb.PairError

	// Test empty standings
	req := pairtestutils.CreateDefaultPairRequest()
	standings := pkgstnd.CreateInitialStandings(req)
	for i := 0; i < int(req.ValidPlayers); i++ {
		is.Equal(standings.GetPlayerWins(i), 0.0)
		is.Equal(standings.GetPlayerSpread(i), 0)
	}

	// Test nonempty standings
	req = pairtestutils.CreateAlbanyjuly4th2024AfterRound21PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	assertPlayerRecord(is, standings, 0, 10, 16.5, 986)
	assertPlayerRecord(is, standings, 1, 4, 15, 677)
	assertPlayerRecord(is, standings, 2, 0, 14.5, 1133)
	assertPlayerRecord(is, standings, 3, 6, 14, 391)
	assertPlayerRecord(is, standings, 4, 1, 13, 676)
	assertPlayerRecord(is, standings, 5, 8, 12, 966)
	assertPlayerRecord(is, standings, 6, 9, 12, 929)
	assertPlayerRecord(is, standings, 7, 5, 12, 528)
	assertPlayerRecord(is, standings, 8, 7, 12, 523)
	assertPlayerRecord(is, standings, 9, 12, 12, 96)
	assertPlayerRecord(is, standings, 10, 2, 12, -69)
	assertPlayerRecord(is, standings, 11, 3, 11.5, 456)
	assertPlayerRecord(is, standings, 12, 28, 11, 589)
	assertPlayerRecord(is, standings, 13, 13, 11, 280)
	assertPlayerRecord(is, standings, 14, 17, 11, 233)
	assertPlayerRecord(is, standings, 15, 27, 11, -56)
	assertPlayerRecord(is, standings, 16, 20, 11, -105)
	assertPlayerRecord(is, standings, 17, 29, 11, -140)
	assertPlayerRecord(is, standings, 18, 11, 11, -267)
	assertPlayerRecord(is, standings, 19, 18, 11, -481)
	assertPlayerRecord(is, standings, 20, 14, 10, -296)
	assertPlayerRecord(is, standings, 21, 15, 9, 152)
	assertPlayerRecord(is, standings, 22, 19, 9, 123)
	assertPlayerRecord(is, standings, 23, 24, 9, -918)
	assertPlayerRecord(is, standings, 24, 21, 8, 55)
	assertPlayerRecord(is, standings, 25, 16, 7.5, -676)
	assertPlayerRecord(is, standings, 26, 22, 7, -357)
	assertPlayerRecord(is, standings, 27, 26, 5, -1506)
	assertPlayerRecord(is, standings, 28, 23, 4, -1569)
	assertPlayerRecord(is, standings, 29, 25, 2, -2353)

	// New 8 player tournament
	req = pairtestutils.CreateDefaultPairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	for i := 0; i < int(req.ValidPlayers); i++ {
		assertPlayerRecord(is, standings, i, -1, 0, 0)
		for j := i + 1; j < int(req.ValidPlayers); j++ {
			is.True(standings.CanCatch(int(req.Rounds), 1000, i, j))
		}
	}
	assertGibsonizedPlayers(is, standings, req, map[int]bool{})
	simResults, pairErr := standings.SimFactorPairAll(req, copRand, 10, int(req.ValidPlayers), -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(len(simResults.Pairings), int(req.Rounds))
	assertFactorPairings(is, simResults.Pairings[0], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[1], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[2], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[3], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[4], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[5], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[6], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[7], []int{0, 3, 1, 4, 2, 5, 6, 7})
	assertFactorPairings(is, simResults.Pairings[8], []int{0, 2, 1, 3, 4, 6, 5, 7})
	assertFactorPairings(is, simResults.Pairings[9], []int{0, 1, 2, 3, 4, 5, 6, 7})
	for i := 0; i < int(req.ValidPlayers); i++ {
		is.Equal(simResults.GibsonGroups[i], 0)
	}

	// New 9 player tournament
	req = pairtestutils.CreateDefaultOddPairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	for i := 0; i < int(req.ValidPlayers); i++ {
		assertPlayerRecord(is, standings, i, -1, 0, 0)
		for j := i + 1; j < int(req.ValidPlayers); j++ {
			is.True(standings.CanCatch(int(req.Rounds), 1000, i, j))
		}
	}
	assertGibsonizedPlayers(is, standings, req, map[int]bool{})
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, 10, int(req.ValidPlayers), -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(len(simResults.Pairings), int(req.Rounds))
	// The pairings will add an extra dummy player
	assertFactorPairings(is, simResults.Pairings[0], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[1], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[2], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[3], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[4], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[5], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[6], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, simResults.Pairings[7], []int{0, 3, 1, 4, 2, 5, 6, 7})
	assertFactorPairings(is, simResults.Pairings[8], []int{0, 2, 1, 3, 4, 6, 5, 7})
	assertFactorPairings(is, simResults.Pairings[9], []int{0, 1, 2, 3, 4, 5, 6, 7})
	for i := 0; i < int(req.ValidPlayers); i++ {
		is.Equal(simResults.GibsonGroups[i], 0)
	}

	// 1st is gibsonized
	req = pairtestutils.CreateLakeGeorgeAfterRound13PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	assertPlayerRecord(is, standings, 0, 3, 12, 980)
	assertPlayerRecord(is, standings, 1, 10, 9, 290)
	assertPlayerRecord(is, standings, 2, 4, 9, 245)
	assertPlayerRecord(is, standings, 3, 8, 9, 142)
	assertPlayerRecord(is, standings, 4, 1, 8, 695)
	assertPlayerRecord(is, standings, 5, 2, 8, 652)
	is.True(standings.CanCatch(3, 1000, 0, 1))
	is.True(standings.CanCatch(3, 700, 0, 1))
	is.True(standings.CanCatch(3, 690, 0, 1))
	is.True(!standings.CanCatch(3, 689, 0, 1))
	is.True(!standings.CanCatch(3, 600, 0, 1))
	is.True(standings.CanCatch(3, 1000, 0, 3))
	is.True(standings.CanCatch(3, 838, 0, 3))
	is.True(!standings.CanCatch(3, 837, 0, 3))
	is.True(!standings.CanCatch(3, 1000, 0, 4))
	is.True(!standings.CanCatch(3, 100000000, 0, 4))
	is.True(standings.CanCatch(3, 100000000, 1, 6))
	is.True(!standings.CanCatch(3, 100000000, 1, 20))
	is.True(!standings.CanCatch(3, 100000000, 12, 27))
	is.True(standings.CanCatch(3, 100000000, 13, 27))
	is.True(standings.CanCatch(3, 100000000, 13, 27))
	is.True(standings.CanCatch(3, 755, 13, 27))
	is.True(!standings.CanCatch(3, 754, 13, 27))
	assertGibsonizedPlayers(is, standings, req, map[int]bool{0: true})
	numSims := 10000
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{1, 3, 2, 4, 5, 16, 6, 17, 7, 18, 8, 19, 9, 20, 10, 21, 11, 22, 12, 23, 13, 24, 14, 25, 15, 26, 27, 0})
	assertFactorPairings(is, simResults.Pairings[1], []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 0})
	for i := 0; i < int(req.ValidPlayers); i++ {
		is.Equal(simResults.GibsonGroups[i], 0)
	}
	// Jackson is gibsonized and should always be first
	assertGibsonizedResult(is, simResults.FinalRanks, numSims, 0)
	assertResultSums(is, simResults.FinalRanks, int(req.ValidPlayers), numSims)

	// 1st and 2nd are gibsonized
	req = pairtestutils.CreateAlbanyCSWAfterRound24PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	is.True(!standings.CanCatch(3, 1000, 0, 1))
	is.True(!standings.CanCatch(3, 1476, 0, 1))
	is.True(standings.CanCatch(3, 1477, 0, 1))
	is.True(!standings.CanCatch(3, 100000000, 1, 2))
	assertGibsonizedPlayers(is, standings, req, map[int]bool{0: true, 1: true})
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 3, -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{2, 5, 3, 6, 4, 7, 8, 19, 9, 20, 10, 21, 11, 22, 12, 23, 13, 24, 14, 25, 15, 26, 16, 27, 17, 28, 18, 29, 0, 1})
	assertFactorPairings(is, simResults.Pairings[1], []int{2, 4, 3, 5, 6, 18, 7, 19, 8, 20, 9, 21, 10, 22, 11, 23, 12, 24, 13, 25, 14, 26, 15, 27, 16, 28, 17, 29, 0, 1})
	assertFactorPairings(is, simResults.Pairings[2], []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 0, 1})
	for i := 0; i < int(req.ValidPlayers); i++ {
		is.Equal(simResults.GibsonGroups[i], 0)
	}
	assertGibsonizedResult(is, simResults.FinalRanks, numSims, 0)
	assertGibsonizedResult(is, simResults.FinalRanks, numSims, 1)
	assertResultSums(is, simResults.FinalRanks, int(req.ValidPlayers), numSims)

	// 3rd is gibsonized
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	assertPlayerRecord(is, standings, 16, 7, 9.5, -682)
	assertPlayerRecord(is, standings, 22, 20, 7, -455)
	assertGibsonizedPlayers(is, standings, req, map[int]bool{2: true})
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{0, 1, 3, 5, 4, 6, 7, 15, 8, 16, 9, 17, 10, 18, 11, 19, 12, 20, 13, 21, 14, 22, 23, 2})
	assertFactorPairings(is, simResults.Pairings[1], []int{0, 1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 2})
	is.Equal(simResults.GibsonGroups[0], 1)
	is.Equal(simResults.GibsonGroups[1], 1)
	for i := 2; i < int(req.ValidPlayers); i++ {
		is.Equal(simResults.GibsonGroups[i], 0)
	}
	assertGibsonizedResult(is, simResults.FinalRanks, numSims, 2)
	assertResultSums(is, simResults.FinalRanks, int(req.ValidPlayers), numSims)

	// 4th is gibsonized
	req = pairtestutils.CreateAlbany4thGibsonizedAfterRound25PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	assertGibsonizedPlayers(is, standings, req, map[int]bool{3: true})
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{0, 2, 1, 3, 4, 6, 5, 7, 8, 16, 9, 17, 10, 18, 11, 19, 12, 20, 13, 21, 14, 22, 15, 23})
	assertFactorPairings(is, simResults.Pairings[1], []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23})
	is.Equal(simResults.GibsonGroups[0], 1)
	is.Equal(simResults.GibsonGroups[1], 1)
	is.Equal(simResults.GibsonGroups[2], 1)
	is.Equal(simResults.GibsonGroups[3], 1)
	for i := 4; i < int(req.ValidPlayers); i++ {
		is.Equal(simResults.GibsonGroups[i], 0)
	}
	assertGibsonizedResult(is, simResults.FinalRanks, numSims, 3)
	assertResultSums(is, simResults.FinalRanks, int(req.ValidPlayers), numSims)

	// 1st and 4th are gibsonized
	req = pairtestutils.CreateAlbany1stAnd4thGibsonizedAfterRound25PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	assertGibsonizedPlayers(is, standings, req, map[int]bool{0: true, 3: true})
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{1, 2, 4, 6, 5, 7, 8, 16, 9, 17, 10, 18, 11, 19, 12, 20, 13, 21, 14, 22, 15, 23, 0, 3})
	assertFactorPairings(is, simResults.Pairings[1], []int{1, 2, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 0, 3})
	is.Equal(simResults.GibsonGroups[0], 0)
	is.Equal(simResults.GibsonGroups[1], 1)
	is.Equal(simResults.GibsonGroups[2], 1)
	is.Equal(simResults.GibsonGroups[3], 0)
	for i := 4; i < int(req.ValidPlayers); i++ {
		is.Equal(simResults.GibsonGroups[i], 0)
	}
	assertGibsonizedResult(is, simResults.FinalRanks, numSims, 0)
	assertGibsonizedResult(is, simResults.FinalRanks, numSims, 3)
	assertResultSums(is, simResults.FinalRanks, int(req.ValidPlayers), numSims)

	// 1st and 4th and 8th are gibsonized
	req = pairtestutils.CreateAlbany1stAnd4thAnd8thGibsonizedAfterRound25PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	assertGibsonizedPlayers(is, standings, req, map[int]bool{0: true, 3: true, 7: true})
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{1, 2, 4, 6, 5, 7, 8, 10, 9, 11, 12, 18, 13, 19, 14, 20, 15, 21, 16, 22, 17, 23, 0, 3})
	assertFactorPairings(is, simResults.Pairings[1], []int{1, 2, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 0, 3})
	is.Equal(simResults.GibsonGroups[0], 0)
	is.Equal(simResults.GibsonGroups[1], 1)
	is.Equal(simResults.GibsonGroups[2], 1)
	is.Equal(simResults.GibsonGroups[3], 0)
	is.Equal(simResults.GibsonGroups[4], 2)
	is.Equal(simResults.GibsonGroups[5], 2)
	is.Equal(simResults.GibsonGroups[6], 2)
	is.Equal(simResults.GibsonGroups[7], 2)
	for i := 8; i < int(req.ValidPlayers); i++ {
		is.Equal(simResults.GibsonGroups[i], 0)
	}
	assertGibsonizedResult(is, simResults.FinalRanks, numSims, 0)
	assertGibsonizedResult(is, simResults.FinalRanks, numSims, 3)
	assertGibsonizedResult(is, simResults.FinalRanks, numSims, 7)
	assertResultSums(is, simResults.FinalRanks, int(req.ValidPlayers), numSims)

	// Test control loss

	// 3rd is gibsonized
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, 1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(simResults.HighestControlLossRankIdx, -1)

	req = pairtestutils.CreateAlbanyjuly4th2024AfterRound21PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	// This scenario evaluates 6 candidate ranks without ever finding a
	// control loss (see below), so the control-loss scan never breaks early
	// and runs the full worst case: 12 sim batches sharing a single 6s wall
	// clock budget (controlSimTimeLimit in standings.go). A smaller numSims
	// keeps this well clear of that budget under CPU contention while still
	// leaving a wide margin on the ratio assertions below.
	numSims = 2000
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, 6, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	// With many rounds remaining, control loss requires vsFirstTournamentWins == sims exactly.
	// Rank 4 wins ~99.9% of vsFirst sims (not 100%), so control loss is not detected.
	// Rank 3 also doesn't trigger because with KOTH, rank 3 naturally faces rank 0 and the
	// vsFirst/vsFactorPair difference is below the threshold.
	is.Equal(simResults.HighestControlLossRankIdx, -1)
	// Rank 4 is still evaluated by the binary search; confirm high vsFirst wins vs lower factor-pair wins.
	is.True(simResults.VsFirstWins[4] > int(float64(numSims)*0.97))
	is.True(simResults.VsFirstWins[4]-simResults.AllControlLosses[4] >= int(req.ControlLossThreshold*float64(numSims)))

	req = pairtestutils.CreateBellevilleCSWAfterRound12PairRequest()
	// Give the player in 2nd more spread to trigger control loss
	req.DivisionResults[0].Results[0] = 1800
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	numSims = 5000
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 3, 5, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	// Rank 2 (0-indexed rank 1) wins every vsFirst sim but only 3743/5000
	// vsFactorPair sims - a 25.1% gap, just over the 25% threshold. The old
	// binary search missed this real control loss because it assumed the
	// vsFirst/vsFactorPair gap was monotonic across ranks, which is exactly
	// the bug the exhaustive per-rank scan (see standings.go) fixed.
	is.Equal(simResults.HighestControlLossRankIdx, 1)
	numSims = 1000
}

func assertGibsonizedResult(is *is.I, results [][]int, numSims int, gibsonizedPos int) {
	is.Helper()
	numPlayers := len(results)
	for i := 0; i < numPlayers; i++ {
		if i == gibsonizedPos {
			is.Equal(results[gibsonizedPos][gibsonizedPos], numSims)
		} else {
			is.Equal(results[gibsonizedPos][i], 0)
		}
	}
}
func assertPlayerRecord(is *is.I, standings *pkgstnd.Standings, rank int, playerIdx int, wins float64, spread int) {
	is.Helper()
	if playerIdx >= 0 {
		is.Equal(standings.GetPlayerIndex(rank), playerIdx)
	}
	is.Equal(int(standings.GetPlayerWins(rank)*2), int(wins*2))
	is.Equal(standings.GetPlayerSpread(rank), spread)
}

func assertGibsonizedPlayers(is *is.I, standings *pkgstnd.Standings, req *pb.PairRequest, expectedGibsonizedPlayers map[int]bool) {
	is.Helper()
	actualGibsonizedPlayers := standings.GetGibsonizedPlayers(req)
	for i := range int(req.ValidPlayers) {
		is.Equal(expectedGibsonizedPlayers[i], actualGibsonizedPlayers[i])
	}
}

func assertFactorPairings(is *is.I, actualPairings []int, expectedPairings []int) {
	is.Helper()
	is.Equal(len(expectedPairings), len(actualPairings))
	for i := range expectedPairings {
		is.Equal(expectedPairings[i], actualPairings[i])
	}
}

func assertResultSums(is *is.I, results [][]int, dim int, total int) {
	is.Helper()
	is.Equal(len(results), dim)
	is.Equal(len(results[0]), dim)

	for _, row := range results {
		rowSum := 0
		for _, val := range row {
			rowSum += val
		}
		is.Equal(rowSum, total)
	}

	for col := 0; col < dim; col++ {
		colSum := 0
		for row := 0; row < dim; row++ {
			colSum += results[row][col]
		}
		is.Equal(colSum, total)
	}

}

// createPendingRoundRequest returns a request whose first round has results
// and whose remaining complete pairing rows are pending (paired, no results).
// GibsonSpread is small so the sims' spreads can't overturn the record gaps
// the tests below rely on.
func createPendingRoundRequest(numPlayers int, rounds int, removedPlayers []int32, resultPairings string, results string, pendingPairings ...string) *pb.PairRequest {
	names := make([]string, numPlayers)
	classes := make([]int32, numPlayers)
	for i := range names {
		names[i] = string(rune('A' + i))
	}
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                names,
		PlayerClasses:              classes,
		ClassPrizes:                []int32{},
		GibsonSpread:               100,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 int32(numPlayers),
		ValidPlayers:               int32(numPlayers - len(removedPlayers)),
		RemovedPlayers:             removedPlayers,
		Rounds:                     int32(rounds),
		PlacePrizes:                1,
		DivisionSims:               1000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 0,
	}
	pairtestutils.AddRoundPairingsStr(req, resultPairings)
	pairtestutils.AddRoundResultsStr(req, results)
	for _, pending := range pendingPairings {
		pairtestutils.AddRoundPairingsStr(req, pending)
	}
	return req
}

func TestPendingRoundPairs(t *testing.T) {
	is := is.New(t)

	// No pending rounds: same as every unlagged request.
	req := createPendingRoundRequest(4, 3, nil, "1 0 3 2", "300 500 200 500")
	is.Equal(pkgstnd.GetNumCompletePairings(req), 1)
	is.Equal(pkgstnd.GetNumPendingRounds(req), 0)
	is.True(pkgstnd.GetPendingRoundPairs(req) == nil)
	is.Equal(pkgstnd.CreateInitialStandings(req).GetNumPendingRounds(), 0)

	// Two pending rounds followed by a partially prepaired round, which is
	// not pending.
	req = createPendingRoundRequest(5, 6, []int32{4}, "1 0 3 2 4", "400 100 390 400 50",
		"1 0 2 4 3", "2 3 0 1 4", "-1 -1 -1 -1 -1")
	is.Equal(pkgstnd.GetNumCompletePairings(req), 3)
	is.Equal(pkgstnd.GetNumPendingRounds(req), 2)
	// Pairs involving the removed player 4 are skipped.
	is.Equal(pkgstnd.GetPendingRoundPairs(req), [][][2]int{
		{{0, 1}, {2, 2}},
		{{0, 2}, {1, 3}},
	})
	is.Equal(pkgstnd.CreateInitialStandings(req).GetNumPendingRounds(), 2)
}

// Ranks after round 1: D (+300), B (+200), A (-200), C (-300). The pending
// round pairs D-C and B-A (1st vs 4th, 2nd vs 3rd), which the sims must play
// instead of the KOTH rows they generate for the final game.
func TestPendingRoundUsesRealPairings(t *testing.T) {
	is := is.New(t)
	copRand := rand.New(rand.NewSource(0))

	req := createPendingRoundRequest(4, 2, nil, "1 0 3 2", "300 500 200 500", "1 0 3 2")
	standings := pkgstnd.CreateInitialStandings(req)
	is.Equal(standings.GetNumPendingRounds(), 1)
	is.Equal(standings.GetPlayerIndex(0), 3)
	is.Equal(standings.GetPlayerIndex(1), 1)
	simResults, pairErr := standings.SimFactorPairAll(req, copRand, 10000, 2, -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertResultSums(is, simResults.FinalRanks, 4, simResults.TotalSims)
	// B finishes 1st only by beating A while D does not beat C: about 25%.
	bFirst := float64(simResults.FinalRanks[1][0]) / float64(simResults.TotalSims)
	is.True(bFirst > 0.2 && bFirst < 0.3)
	// A and C can never finish 1st.
	is.Equal(simResults.FinalRanks[2][0], 0)
	is.Equal(simResults.FinalRanks[3][0], 0)

	// Without the pending round, the final game is KOTH (D vs B), so B
	// finishes 1st about half the time.
	req = createPendingRoundRequest(4, 2, nil, "1 0 3 2", "300 500 200 500")
	standings = pkgstnd.CreateInitialStandings(req)
	is.Equal(standings.GetNumPendingRounds(), 0)
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, 10000, 2, -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	bFirst = float64(simResults.FinalRanks[1][0]) / float64(simResults.TotalSims)
	is.True(bFirst > 0.45 && bFirst < 0.55)

	// The forced-winner sims also play the real pending pairing: B always
	// beats A, but only wins the tournament when D does not beat C. Under
	// KOTH, B would beat D and always win.
	req = createPendingRoundRequest(4, 2, nil, "1 0 3 2", "300 500 200 500", "1 0 3 2")
	standings = pkgstnd.CreateInitialStandings(req)
	kothPairings := [][]int{{0, 1, 2, 3}}
	numSims := 10000
	wins, pairErr := standings.RunParallelSimForceWinner(copRand, numSims, 1, 2, kothPairings, 1, true, time.Now().Add(time.Minute).UnixNano())
	is.Equal(pairErr, pb.PairError_SUCCESS)
	winPct := float64(wins) / float64(numSims)
	is.True(winPct > 0.45 && winPct < 0.6)
	// Forcing A, who can't finish 1st even after beating B, never wins.
	wins, pairErr = standings.RunParallelSimForceWinner(copRand, numSims, 1, 2, kothPairings, 0, true, time.Now().Add(time.Minute).UnixNano())
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(wins, 0)

	// RunSimsWithPairings plays the pending round in place of the first row
	// of the pairings it is given.
	finalRanks, totalSims, pairErr := standings.RunSimsWithPairings(copRand, 10000, 1, kothPairings)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	bFirst = float64(finalRanks[1][0]) / float64(totalSims)
	is.True(bFirst > 0.2 && bFirst < 0.3)
}

// A pending bye is always a win. Ranks after round 1: B (1-0 +200),
// C (0-1 -50, a forfeit bye), A (0-1 -200). C has the bye in the pending
// round, so C always ends 1-1 and always finishes exactly 2nd: with a coin
// flip, C would finish last whenever B beat A.
func TestPendingByeAlwaysWins(t *testing.T) {
	is := is.New(t)
	copRand := rand.New(rand.NewSource(0))

	req := createPendingRoundRequest(3, 2, nil, "1 0 2", "300 500 -50", "1 0 2")
	standings := pkgstnd.CreateInitialStandings(req)
	is.Equal(standings.GetPlayerIndex(1), 2)
	// The odd field gets an evener player in the sims.
	simResults, pairErr := standings.SimFactorPairAll(req, copRand, 10000, 1, -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.True(simResults.TotalSims > 0)
	assertResultSums(is, simResults.FinalRanks, 3, simResults.TotalSims)
	is.Equal(simResults.FinalRanks[1][1], simResults.TotalSims)
}

// A pending bye still counts as a win when the sims have no evener player,
// which happens when a removal makes the field even. E is removed: D's
// pending game against E is skipped, and C's pending bye is credited without
// an opponent. Ranks after round 1: A (+300), D (+10), C (-10), B (-300).
// C ends 1-1 and always finishes exactly 3rd.
func TestPendingByeWithoutEvener(t *testing.T) {
	is := is.New(t)
	copRand := rand.New(rand.NewSource(0))

	req := createPendingRoundRequest(5, 2, []int32{4}, "1 0 3 2 4", "400 100 390 400 50", "1 0 2 4 3")
	standings := pkgstnd.CreateInitialStandings(req)
	is.Equal(standings.GetNumPlayers(), 4)
	is.Equal(standings.GetPlayerIndex(2), 2)
	simResults, pairErr := standings.SimFactorPairAll(req, copRand, 10000, 2, -1, nil, -1)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.True(simResults.TotalSims > 0)
	assertResultSums(is, simResults.FinalRanks, 4, simResults.TotalSims)
	is.Equal(simResults.FinalRanks[2][2], simResults.TotalSims)
}

// createByeResultsRequest returns a request whose results rounds are all
// byes, so each player's record can be set directly: a positive score is a
// win, a negative one a loss and 0 no change. The pending round has the
// given pairings and no results.
func createByeResultsRequest(numPlayers int, rounds int, byeScores []string, pendingPairings string) *pb.PairRequest {
	byes := make([]string, numPlayers)
	for i := range byes {
		byes[i] = strconv.Itoa(i)
	}
	byePairings := strings.Join(byes, " ")
	req := createPendingRoundRequest(numPlayers, rounds, nil, byePairings, byeScores[0])
	for _, scores := range byeScores[1:] {
		pairtestutils.AddRoundPairingsStr(req, byePairings)
		pairtestutils.AddRoundResultsStr(req, scores)
	}
	pairtestutils.AddRoundPairingsStr(req, pendingPairings)
	return req
}

// The round being paired is resolved against the standings before the
// pending round, not after it. Before the pending round: D (2-0), B (1-1,
// +20), C (1-1, 0), A (0-2). The pending round is D-C and B-A, and the
// round being paired is 1st v 3rd and 2nd v 4th of those standings, which
// is D-C and B-A again. A can only finish 1st if D loses twice, but then C
// wins twice, so A never finishes 1st. Resolved against the standings after
// the pending round instead, the pairings would sometimes put A against D,
// and A could finish 1st.
func TestPairedRoundResolvedBeforePendingRound(t *testing.T) {
	is := is.New(t)
	copRand := rand.New(rand.NewSource(0))

	req := createByeResultsRequest(4, 4, []string{"-10 30 -10 10", "-10 -10 10 10"}, "1 0 3 2")
	standings := pkgstnd.CreateInitialStandings(req)
	is.Equal(standings.GetNumPendingRounds(), 1)
	for rankIdx, playerIdx := range []int{3, 1, 2, 0} {
		is.Equal(standings.GetPlayerIndex(rankIdx), playerIdx)
	}
	pairings := [][]int{{0, 1, 2, 3}, {0, 2, 1, 3}}
	finalRanks, totalSims, pairErr := standings.RunSimsWithPairings(copRand, 20000, 2, pairings)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.True(totalSims > 0)
	// A started 4th.
	is.Equal(finalRanks[3][0], 0)
}

// The control-loss simulations make their swap in the ranks before the
// pending round too: playing 1st means playing whoever led then. Before the
// pending round: E (2-0, +76), C (2-0, +48), D (1-0), B (1-0), A (0-1,
// -10), F (0-1, -45). The pending round is A-D and E-F, with byes for B and
// C. The round being paired is 1st v 4th, 2nd v 5th and 3rd v 6th; with A
// forced to win every game and to play 1st, it becomes A-E, C-B and F-D.
// A can reach 2 wins. C and B meet on 3 and 2 wins, so one of them goes into
// the KOTH final on at least 3, and either they or whoever beats them ends
// above A. A never wins the tournament. Swapped against the leader after
// the pending round instead, A sometimes does.
func TestForcedWinnerPlaysLeaderBeforePendingRound(t *testing.T) {
	is := is.New(t)
	copRand := rand.New(rand.NewSource(0))

	req := createByeResultsRequest(6, 5, []string{"-10 17 24 31 38 -45", "0 0 24 0 38 0"}, "3 1 2 0 5 4")
	standings := pkgstnd.CreateInitialStandings(req)
	is.Equal(standings.GetNumPendingRounds(), 1)
	for rankIdx, playerIdx := range []int{4, 2, 3, 1, 0, 5} {
		is.Equal(standings.GetPlayerIndex(rankIdx), playerIdx)
	}
	pairings := [][]int{{0, 1, 2, 3, 4, 5}, {0, 3, 5, 2, 1, 4}, {0, 1, 2, 3, 4, 5}}
	wins, pairErr := standings.RunParallelSimForceWinner(copRand, 20000, 3, 3, pairings, 0, true, time.Now().Add(time.Minute).UnixNano())
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(wins, 0)
}
