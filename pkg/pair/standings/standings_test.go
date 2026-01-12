package standings_test

import (
	"testing"

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
	simResults, pairErr := standings.SimFactorPairAll(req, copRand, 10, int(req.ValidPlayers), -1, nil)
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
	assertFactorPairings(is, simResults.Pairings[8], []int{0, 2, 1, 3, 4, 5, 6, 7})
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
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, 10, int(req.ValidPlayers), -1, nil)
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
	assertFactorPairings(is, simResults.Pairings[8], []int{0, 2, 1, 3, 4, 5, 6, 7})
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
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, -1, nil)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{1, 3, 2, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 0})
	assertFactorPairings(is, simResults.Pairings[1], []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 0})
	for i := 0; i < int(req.ValidPlayers); i++ {
		is.Equal(simResults.GibsonGroups[i], 0)
	}
	// Jackson is gibsonized and should always be first
	assertGibsonizedResult(is, simResults.FinalRanks, numSims, 0)
	// Jeffrey cannot get 12th as he is two wins back with two to go and
	// KOTH pairings will ensure at least one player with 6 wins will get
	// to 8.
	is.Equal(simResults.FinalRanks[20][11], 0)
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
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 3, -1, nil)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{2, 5, 3, 6, 4, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 0, 1})
	assertFactorPairings(is, simResults.Pairings[1], []int{2, 4, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 0, 1})
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
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, -1, nil)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{0, 1, 3, 5, 4, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 2})
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
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, -1, nil)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{0, 2, 1, 3, 4, 6, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23})
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
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, -1, nil)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{1, 2, 4, 6, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 0, 3})
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
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, -1, nil)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	assertFactorPairings(is, simResults.Pairings[0], []int{1, 2, 4, 6, 5, 7, 8, 10, 9, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 0, 3})
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
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, 1, nil)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(simResults.HighestControlLossRankIdx, -1)

	req = pairtestutils.CreateAlbanyjuly4th2024AfterRound21PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	numSims = 10000
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 2, 6, nil)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(simResults.HighestControlLossRankIdx, 4)
	numSims = 1000

	req = pairtestutils.CreateBellevilleCSWAfterRound12PairRequest()
	// Give the player in 2nd more spread to trigger control loss
	req.DivisionResults[0].Results[0] = 1800
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	numSims = 5000
	simResults, pairErr = standings.SimFactorPairAll(req, copRand, numSims, 3, 5, nil)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(simResults.HighestControlLossRankIdx, -1)
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
