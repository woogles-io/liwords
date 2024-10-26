package standings_test

import (
	"fmt"
	"testing"

	"github.com/matryer/is"
	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
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

	req = pairtestutils.CreateDefaultPairRequest()
	standings = pkgstnd.CreateInitialStandings(req)
	for i := 0; i < int(req.Players); i++ {
		assertPlayerRecord(is, standings, i, -1, 0, 0)
		for j := i + 1; j < int(req.Players); j++ {
			is.True(standings.CanCatch(int(req.Rounds), 1000, i, j))
		}
	}
	assertGibsonizedPlayers(is, standings, req, map[int]bool{})
	assertGetAllSegments(is, standings, req, [][]int{{0, 8}})
	// FIXME: pass in segments
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
	// FIXME: pass in segments
	simAndAssertStandingTotals(is, standings, 10, int(req.Players), int(req.Rounds), 0, 8)

	req = pairtestutils.CreateDefaultOddPairRequest()
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
	simAndAssertStandingTotals(is, standings, 50, int(req.Players), int(req.Rounds), 0, 7)

	req = pairtestutils.CreateLakegeorgeAfterRound13PairRequest()
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
	simAndAssertStandingTotals(is, standings, 50, int(req.Players)-1, 2, 1, 28)

	// canCatch + simFactorPairing + Simming for the following scenarios:
	// 1st gibsonized
	// 1st and 2nd gibsonized
	// 3rd gibsonized
	// 4th gibsonized
	// 1st and 4th gibsonized
	// 1nd and 4th and 8th
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
	for i := range expectedPairings {
		fmt.Print(" ", expectedPairings[i])
	}
	fmt.Println()
	is.Equal(len(expectedPairings), len(actualPairings))
	for i := range expectedPairings {
		is.Equal(expectedPairings[i], actualPairings[i])
	}
}

// Asserts the sums of wins and spreads from players [i, j)
func simAndAssertStandingTotals(is *is.I, standings *pkgstnd.Standings, sims int, maxFactor int, roundsRemaining int, i int, j int) {
	pairings := pkgstnd.GetPairingsForSegment(i, j, roundsRemaining, maxFactor)
	standings.Backup()
	standingsCopy := standings.Copy()
	expectedTotalWins := ((j - i) / 2) * roundsRemaining
	expectedTotalSpread := 0
	if (j-i)%2 == 1 {
		expectedTotalWins += roundsRemaining
		expectedTotalSpread += 50 * roundsRemaining
	}
	for simIdx := 0; simIdx < sims; simIdx++ {
		standings.SimSingleIteration(pairings, roundsRemaining, i, j)
		fmt.Println(standings.String(nil))
		winsSum := 0
		spreadSum := 0
		for k := i; k < j; k++ {
			winsSum += int(standings.GetPlayerWins(k)*2) - int(standingsCopy.GetPlayerWins(k)*2)
			spreadSum += standings.GetPlayerSpread(k) - standingsCopy.GetPlayerSpread(k)
		}
		is.Equal(winsSum, expectedTotalWins*2)
		is.Equal(spreadSum, expectedTotalSpread)
		standings.RestoreFromBackup()
	}
}
