package standings_test

import (
	"fmt"
	"testing"

	"github.com/matryer/is"
	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	"github.com/woogles-io/liwords/pkg/pair/verifyreq"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// FIXME: should this be tested in tests for copdata?
func TestStandings(t *testing.T) {
	is := is.New(t)

	// Test empty standings
	req := pairtestutils.CreateDefaultPairRequest()
	standings := pkgstnd.CreateInitialStandings(req)
	for i := 0; i < int(req.Players); i++ {
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
	for i := 0; i < int(req.Players); i++ {
		assertPlayerRecord(is, standings, i, -1, 0, 0)
		for j := i + 1; j < int(req.Players); j++ {
			is.True(standings.CanCatch(int(req.Rounds), 1000, i, j))
		}
	}
	assertGibsonizedPlayers(is, standings, req, map[int]bool{})
	assertGetAllSegments(is, standings, req, [][]int{{0, 8}})
	factorPairings := pkgstnd.GetPairingsForSegment(0, 8, int(req.Rounds), int(req.Players))
	is.Equal(len(factorPairings), int(req.Rounds))
	assertFactorPairings(is, factorPairings[0], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, factorPairings[1], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, factorPairings[2], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, factorPairings[3], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, factorPairings[4], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, factorPairings[5], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, factorPairings[6], []int{0, 4, 1, 5, 2, 6, 3, 7})
	assertFactorPairings(is, factorPairings[7], []int{0, 3, 1, 4, 2, 5, 6, 7})
	assertFactorPairings(is, factorPairings[8], []int{0, 2, 1, 3, 4, 5, 6, 7})
	assertFactorPairings(is, factorPairings[9], []int{0, 1, 2, 3, 4, 5, 6, 7})

	// New 9 player tournament
	req = pairtestutils.CreateDefaultOddPairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	for i := 0; i < int(req.Players); i++ {
		assertPlayerRecord(is, standings, i, -1, 0, 0)
		for j := i + 1; j < int(req.Players); j++ {
			is.True(standings.CanCatch(int(req.Rounds), 1000, i, j))
		}
	}
	assertGibsonizedPlayers(is, standings, req, map[int]bool{})
	assertGetAllSegments(is, standings, req, [][]int{{0, 7}})
	factorPairings = pkgstnd.GetPairingsForSegment(0, 7, int(req.Rounds), int(req.Players))
	is.Equal(len(factorPairings), int(req.Rounds))
	assertFactorPairings(is, factorPairings[0], []int{0, 3, 1, 4, 2, 5, 6})
	assertFactorPairings(is, factorPairings[1], []int{0, 3, 1, 4, 2, 5, 6})
	assertFactorPairings(is, factorPairings[2], []int{0, 3, 1, 4, 2, 5, 6})
	assertFactorPairings(is, factorPairings[3], []int{0, 3, 1, 4, 2, 5, 6})
	assertFactorPairings(is, factorPairings[4], []int{0, 3, 1, 4, 2, 5, 6})
	assertFactorPairings(is, factorPairings[5], []int{0, 3, 1, 4, 2, 5, 6})
	assertFactorPairings(is, factorPairings[6], []int{0, 3, 1, 4, 2, 5, 6})
	assertFactorPairings(is, factorPairings[7], []int{0, 3, 1, 4, 2, 5, 6})
	assertFactorPairings(is, factorPairings[8], []int{0, 2, 1, 3, 4, 5, 6})
	assertFactorPairings(is, factorPairings[9], []int{0, 1, 2, 3, 4, 5, 6})

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
	assertGetAllSegments(is, standings, req, [][]int{{1, 28}})
	factorPairings = pkgstnd.GetPairingsForSegment(1, 28, 2, int(req.Players))
	is.Equal(len(factorPairings), 2)
	assertFactorPairings(is, factorPairings[0], []int{1, 3, 2, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27})
	assertFactorPairings(is, factorPairings[1], []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27})
	numSims := 1000
	results := standings.SimFactorPair(numSims, 2, 2, standings.GetGibsonizedPlayers(req))
	// Jackson is gibsonized and should always be first
	assertGibsonizedResult(is, results, numSims, 3, 0)
	// Jeffrey cannot get 12th as he is two wins back with two to go and
	// KOTH pairings will ensure at least one player with 6 wins will get
	// to 8.
	is.Equal(results[20][11], 0)
	assertResultSums(is, results, int(req.Players), numSims)

	// 1st and 2nd are gibsonized
	req = pairtestutils.CreateAlbanyAfterRound24PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	is.True(!standings.CanCatch(3, 1000, 0, 1))
	is.True(!standings.CanCatch(3, 1476, 0, 1))
	is.True(standings.CanCatch(3, 1477, 0, 1))
	is.True(!standings.CanCatch(3, 100000000, 1, 2))
	assertGibsonizedPlayers(is, standings, req, map[int]bool{0: true, 1: true})
	assertGetAllSegments(is, standings, req, [][]int{{2, 30}})
	factorPairings = pkgstnd.GetPairingsForSegment(2, 30, 3, int(req.Players))
	assertFactorPairings(is, factorPairings[0], []int{2, 5, 3, 6, 4, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29})
	assertFactorPairings(is, factorPairings[1], []int{2, 4, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29})
	assertFactorPairings(is, factorPairings[2], []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29})

	results = standings.SimFactorPair(numSims, 2, 3, standings.GetGibsonizedPlayers(req))
	assertGibsonizedResult(is, results, numSims, 0, 0)
	assertGibsonizedResult(is, results, numSims, 10, 1)
	assertResultSums(is, results, int(req.Players), numSims)

	// 3rd is gibsonized
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	assertPlayerRecord(is, standings, 16, 7, 9.5, -682)
	assertPlayerRecord(is, standings, 23, 11, 1, -1710)
	assertGibsonizedPlayers(is, standings, req, map[int]bool{2: true})
	assertGetAllSegments(is, standings, req, [][]int{{0, 2}, {3, 24}})
	factorPairings = pkgstnd.GetPairingsForSegment(0, 2, 2, int(req.Players))
	assertFactorPairings(is, factorPairings[0], []int{0, 1})
	assertFactorPairings(is, factorPairings[1], []int{0, 1})
	factorPairings = pkgstnd.GetPairingsForSegment(3, 24, 2, int(req.Players))
	assertFactorPairings(is, factorPairings[0], []int{3, 5, 4, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23})
	assertFactorPairings(is, factorPairings[1], []int{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23})
	results = standings.SimFactorPair(numSims, 2, 2, standings.GetGibsonizedPlayers(req))
	assertGibsonizedResult(is, results, numSims, 4, 2)
	assertResultSums(is, results, int(req.Players), numSims)

	// 4th is gibsonized
	req = pairtestutils.CreateAlbany4thGibsonizedAfterRound25PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	assertGibsonizedPlayers(is, standings, req, map[int]bool{3: true})
	assertGetAllSegments(is, standings, req, [][]int{{0, 4}, {4, 24}})
	factorPairings = pkgstnd.GetPairingsForSegment(0, 4, 2, int(req.Players))
	assertFactorPairings(is, factorPairings[0], []int{0, 2, 1, 3})
	assertFactorPairings(is, factorPairings[1], []int{0, 1, 2, 3})
	factorPairings = pkgstnd.GetPairingsForSegment(4, 24, 2, int(req.Players))
	assertFactorPairings(is, factorPairings[0], []int{4, 6, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23})
	assertFactorPairings(is, factorPairings[1], []int{4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23})
	results = standings.SimFactorPair(numSims, 2, 2, standings.GetGibsonizedPlayers(req))
	assertGibsonizedResult(is, results, numSims, 2, 3)
	assertResultSums(is, results, int(req.Players), numSims)

	// 1st and 4th are gibsonized
	req = pairtestutils.CreateAlbany1stAnd4thGibsonizedAfterRound25PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	assertGibsonizedPlayers(is, standings, req, map[int]bool{0: true, 3: true})
	assertGetAllSegments(is, standings, req, [][]int{{1, 3}, {4, 24}})
	factorPairings = pkgstnd.GetPairingsForSegment(1, 3, 2, int(req.Players))
	assertFactorPairings(is, factorPairings[0], []int{1, 2})
	assertFactorPairings(is, factorPairings[1], []int{1, 2})
	factorPairings = pkgstnd.GetPairingsForSegment(4, 24, 2, int(req.Players))
	assertFactorPairings(is, factorPairings[0], []int{4, 6, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23})
	assertFactorPairings(is, factorPairings[1], []int{4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23})
	results = standings.SimFactorPair(numSims, 2, 2, standings.GetGibsonizedPlayers(req))
	assertGibsonizedResult(is, results, numSims, 1, 0)
	assertGibsonizedResult(is, results, numSims, 2, 3)
	assertResultSums(is, results, int(req.Players), numSims)

	req = pairtestutils.CreateAlbany1stAnd4thAnd8thGibsonizedAfterRound25PairRequest()
	is.True(verifyreq.Verify(req) == nil)
	standings = pkgstnd.CreateInitialStandings(req)
	assertGibsonizedPlayers(is, standings, req, map[int]bool{0: true, 3: true, 7: true})
	assertGetAllSegments(is, standings, req, [][]int{{1, 3}, {4, 8}, {8, 24}})
	factorPairings = pkgstnd.GetPairingsForSegment(1, 3, 2, int(req.Players))
	assertFactorPairings(is, factorPairings[0], []int{1, 2})
	assertFactorPairings(is, factorPairings[1], []int{1, 2})
	factorPairings = pkgstnd.GetPairingsForSegment(4, 8, 2, int(req.Players))
	assertFactorPairings(is, factorPairings[0], []int{4, 6, 5, 7})
	assertFactorPairings(is, factorPairings[1], []int{4, 5, 6, 7})
	factorPairings = pkgstnd.GetPairingsForSegment(8, 24, 2, int(req.Players))
	assertFactorPairings(is, factorPairings[0], []int{8, 10, 9, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23})
	assertFactorPairings(is, factorPairings[1], []int{8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23})
	results = standings.SimFactorPair(numSims, 2, 2, standings.GetGibsonizedPlayers(req))
	assertGibsonizedResult(is, results, numSims, 1, 0)
	assertGibsonizedResult(is, results, numSims, 2, 3)
	assertGibsonizedResult(is, results, numSims, 10, 7)
	assertResultSums(is, results, int(req.Players), numSims)
	// fmt.Println(standings.String(req))
	//fmt.Println(standings.ResultsString(results, req))
	// FIXME: Group sums from starting standings
	// FIXME: test player removal
	// FIXME: only sim cashers
}

func assertGibsonizedResult(is *is.I, results [][]int, numSims int, playerIdx int, gibsonizedPos int) {
	numPlayers := len(results)
	for i := 0; i < numPlayers; i++ {
		if i == gibsonizedPos {
			is.Equal(results[playerIdx][gibsonizedPos], numSims)
		} else {
			is.Equal(results[playerIdx][i], 0)
		}
	}
}
func assertPlayerRecord(is *is.I, standings *pkgstnd.Standings, rank int, playerIdx int, wins float64, spread int) {
	if playerIdx >= 0 {
		is.Equal(standings.GetPlayerIndex(rank), playerIdx)
	}
	is.Equal(int(standings.GetPlayerWins(rank)*2), int(wins*2))
	is.Equal(standings.GetPlayerSpread(rank), spread)
}

func assertGibsonizedPlayers(is *is.I, standings *pkgstnd.Standings, req *pb.PairRequest, expectedGibsonizedPlayers map[int]bool) {
	actualGibsonizedPlayers := standings.GetGibsonizedPlayers(req)
	for i := range int(req.Players) {
		if expectedGibsonizedPlayers[i] != actualGibsonizedPlayers[i] {
			panic(fmt.Sprintf("expected %t, got %t for %d", expectedGibsonizedPlayers[i], actualGibsonizedPlayers[i], i))
		}
		is.Equal(expectedGibsonizedPlayers[i], actualGibsonizedPlayers[i])
	}
}

func assertGetAllSegments(is *is.I, standings *pkgstnd.Standings, req *pb.PairRequest, expectedSegments [][]int) {
	gibsonizedPlayers := standings.GetGibsonizedPlayers(req)
	actualSegments := standings.GetAllSegments(gibsonizedPlayers)
	is.Equal(len(expectedSegments), len(actualSegments))
	for i := range expectedSegments {
		is.Equal(expectedSegments[i], actualSegments[i])
	}
}

func assertFactorPairings(is *is.I, actualPairings []int, expectedPairings []int) {
	// for i := range expectedPairings {
	// 	fmt.Print(" ", expectedPairings[i])
	// }
	// fmt.Println()
	// for i := range actualPairings {
	// 	fmt.Print(" ", actualPairings[i])
	// }
	// fmt.Println()
	is.Equal(len(expectedPairings), len(actualPairings))
	for i := range expectedPairings {
		is.Equal(expectedPairings[i], actualPairings[i])
	}
}

func assertResultSums(is *is.I, results [][]int, dim int, total int) {
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
