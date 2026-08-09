package copdata_test

import (
	"os"
	"strings"
	"testing"

	"github.com/matryer/is"
	pkgcopdata "github.com/woogles-io/liwords/pkg/pair/copdata"
	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"golang.org/x/exp/rand"
)

func TestMain(m *testing.M) {
	pkgstnd.NumSimWorkersOverride = 1
	os.Exit(m.Run())
}

func TestCOPPrecompData(t *testing.T) {
	is := is.New(t)
	copRand := rand.New(rand.NewSource(uint64(0)))

	var logsb strings.Builder
	var req *pb.PairRequest
	var copdata *pkgcopdata.PrecompData
	var pairErr pb.PairError

	// Empty division
	req = pairtestutils.CreateDefaultPairRequest()
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	for _, count := range copdata.PairingCounts {
		is.Equal(count, 0)
	}
	for _, count := range copdata.RepeatCounts {
		is.Equal(count, 0)
	}
	for _, rank := range copdata.HighestRankHopefully {
		is.Equal(rank, 0)
	}
	for _, rank := range copdata.HighestRankAbsolutely {
		is.Equal(rank, 0)
	}
	for _, group := range copdata.GibsonGroups {
		is.Equal(group, 0)
	}
	is.Equal(copdata.DestinysChild, -1)

	// Empty division
	req = pairtestutils.CreateDefaultOddPairRequest()
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	for _, count := range copdata.PairingCounts {
		is.Equal(count, 0)
	}
	for _, count := range copdata.RepeatCounts {
		is.Equal(count, 0)
	}
	for _, rank := range copdata.HighestRankHopefully {
		is.Equal(rank, 0)
	}
	for _, rank := range copdata.HighestRankAbsolutely {
		is.Equal(rank, 0)
	}
	for _, group := range copdata.GibsonGroups {
		is.Equal(group, 0)
	}
	is.Equal(copdata.DestinysChild, -1)

	req = pairtestutils.CreateLakeGeorgeAfterRound13PairRequest()
	req.ControlLossActivationRound = 10
	req.DivisionSims = 10000
	copRand.Seed(1)
	// 1st is gibsonized, so control loss should not be used
	// even though it's set to true in the request
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 23)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 14)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 7)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 22)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 1)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 20)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 21)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 25)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 0)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 3)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 4)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(3, 6)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 25)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 15)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 9)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 27)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 5)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 22)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 20)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 18)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 17)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 10)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 4)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 2)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 21)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 0)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 1)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 3)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 6)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 7)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 8)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 11)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 12)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 13)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 14)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 16)], 0)
	for _, count := range copdata.PairingCounts {
		is.True(count < 2)
	}
	for _, count := range copdata.RepeatCounts {
		is.Equal(count, 0)
	}
	is.Equal(copdata.HighestRankHopefully[0], 0)
	is.Equal(copdata.HighestRankAbsolutely[0], 0)
	for rank := 1; rank <= 5; rank++ {
		is.Equal(copdata.HighestRankHopefully[rank], 1)
		is.Equal(copdata.HighestRankAbsolutely[rank], 1)
	}
	is.Equal(copdata.HighestRankHopefully[6], 4)
	is.Equal(copdata.HighestRankAbsolutely[6], 3)
	is.Equal(copdata.HighestRankHopefully[7], 4)
	is.Equal(copdata.HighestRankAbsolutely[7], 3)
	is.Equal(copdata.HighestRankHopefully[17], 8)
	is.Equal(copdata.HighestRankAbsolutely[17], 6)
	for _, group := range copdata.GibsonGroups {
		is.Equal(group, 0)
	}
	is.Equal(copdata.LowestRankAbsolutely[0], 0)
	is.Equal(copdata.DestinysChild, -1)

	// 3rd gibsonized
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.ControlLossActivationRound = 25
	copRand.Seed(1)
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(1, 0)], 2)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(1, 1)], 0)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(1, 2)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(1, 3)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(1, 4)], 2)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(1, 5)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(1, 6)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(1, 7)], 1)
	is.Equal(copdata.RepeatCounts[1], 2)
	is.Equal(copdata.GibsonGroups[0], 1)
	is.Equal(copdata.GibsonGroups[1], 1)
	for rank := 2; rank < len(copdata.GibsonGroups); rank++ {
		is.Equal(copdata.GibsonGroups[rank], 0)
	}
	is.Equal(copdata.LowestRankAbsolutely[2], 2)
	is.Equal(copdata.DestinysChild, -1)

	// 1st and 2nd gibsonized
	req = pairtestutils.CreateAlbanyCSWAfterRound24PairRequest()
	req.ControlLossActivationRound = 25
	copRand.Seed(1)
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 25)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 29)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 9)], 1)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 4)], 2)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 6)], 2)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 10)], 2)
	is.Equal(copdata.PairingCounts[pkgcopdata.GetPairingKey(0, 1)], 2)
	is.Equal(copdata.RepeatCounts[0], 4)
	is.Equal(copdata.RepeatCounts[2], 3)
	is.Equal(copdata.RepeatCounts[25], 7)
	for _, group := range copdata.GibsonGroups {
		is.Equal(group, 0)
	}
	for rank := 0; rank <= 1; rank++ {
		is.Equal(copdata.HighestRankHopefully[rank], rank)
		is.Equal(copdata.HighestRankAbsolutely[rank], rank)
	}
	is.Equal(copdata.HighestRankHopefully[23], 15)
	is.Equal(copdata.HighestRankAbsolutely[23], 14)
	is.Equal(copdata.LowestRankAbsolutely[0], 0)
	is.Equal(copdata.LowestRankAbsolutely[1], 1)
	is.Equal(copdata.LowestRankAbsolutely[2], 10)
	is.Equal(copdata.LowestRankAbsolutely[17], 25)
	is.Equal(copdata.LowestRankAbsolutely[29], 29)
	is.Equal(copdata.DestinysChild, -1)

	// 4th gibsonized
	req = pairtestutils.CreateAlbany4thGibsonizedAfterRound25PairRequest()
	req.ControlLossActivationRound = 25
	copRand.Seed(1)
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	for rank := 0; rank < 4; rank++ {
		is.Equal(copdata.GibsonGroups[rank], 1)
	}
	for rank := 4; rank < len(copdata.GibsonGroups); rank++ {
		is.Equal(copdata.GibsonGroups[rank], 0)
	}
	is.Equal(copdata.LowestRankAbsolutely[3], 3)
	is.Equal(copdata.DestinysChild, 2)

	req = pairtestutils.CreateAlbany1stAnd4thGibsonizedAfterRound25PairRequest()
	req.ControlLossActivationRound = 26
	copRand.Seed(1)
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(copdata.DestinysChild, -1)
	is.Equal(copdata.GibsonGroups[0], 0)
	is.Equal(copdata.GibsonGroups[1], 1)
	is.Equal(copdata.GibsonGroups[2], 1)
	is.Equal(copdata.GibsonGroups[3], 0)
	for rank := 4; rank < len(copdata.GibsonGroups); rank++ {
		is.Equal(copdata.GibsonGroups[rank], 0)
	}
	is.Equal(copdata.DestinysChild, -1)

	req = pairtestutils.CreateAlbany1stAnd4thAnd8thGibsonizedAfterRound25PairRequest()
	req.ControlLossActivationRound = 25
	copRand.Seed(1)
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(copdata.DestinysChild, -1)
	is.Equal(copdata.GibsonGroups[0], 0)
	is.Equal(copdata.GibsonGroups[1], 1)
	is.Equal(copdata.GibsonGroups[2], 1)
	is.Equal(copdata.GibsonGroups[3], 0)
	is.Equal(copdata.GibsonGroups[4], 2)
	is.Equal(copdata.GibsonGroups[5], 2)
	is.Equal(copdata.GibsonGroups[6], 2)
	is.Equal(copdata.GibsonGroups[7], 2)
	for rank := 8; rank < len(copdata.GibsonGroups); rank++ {
		is.Equal(copdata.GibsonGroups[rank], 0)
	}
	is.Equal(copdata.DestinysChild, -1)

	req = pairtestutils.CreateBellevilleCSWAfterRound12PairRequest()
	req.ControlLossActivationRound = 15
	copRand.Seed(1)
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(copdata.DestinysChild, -1)
	for rank := 0; rank < len(copdata.GibsonGroups); rank++ {
		is.Equal(copdata.GibsonGroups[rank], 0)
	}

	req = pairtestutils.CreateBellevilleCSWAfterRound12PairRequest()
	req.ControlLossActivationRound = 12
	req.ControlLossThreshold = 0.5
	copRand.Seed(1)
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(copdata.DestinysChild, -1)
	for rank := 0; rank < len(copdata.GibsonGroups); rank++ {
		is.Equal(copdata.GibsonGroups[rank], 0)
	}

	req = pairtestutils.CreateBellevilleCSWAfterRound12PairRequest()
	req.ControlLossActivationRound = 12
	copRand.Seed(1)
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(copdata.DestinysChild, 1)
	for rank := 0; rank < len(copdata.GibsonGroups); rank++ {
		is.Equal(copdata.GibsonGroups[rank], 0)
	}
}

// hopefulnessTestReq builds a 10-player COP request with no history; callers
// add rounds to control the leader/2nd win% for TestRunawayLeadersHalveHopefulness.
func hopefulnessTestReq() *pb.PairRequest {
	return &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9"},
		PlayerClasses:              make([]int32, 10),
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.05,
		AllPlayers:                 10,
		ValidPlayers:               10,
		Rounds:                     10,
		PlacePrizes:                2,
		DivisionSims:               3000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 10,
	}
}

// TestRunawayLeadersHalveHopefulness checks that when the leader and 2nd
// place's combined simulated probability of finishing 1st exceeds 80%, other
// players only need half as many simulated wins to be considered hopeful
// contenders for 1st or 2nd (Feature 4). This directly shrinks/grows
// LowestPossibleHopeNth[0]/[1], which Feature 5 (see cop_test.go) builds its
// odd-contender-group logic on top of.
//
// This must be computed from FinalRanks (a true probability, bounded at
// 100% since only one player can finish 1st), not from the leader and 2nd's
// independent actual completed-game win rates, which can each approach 100%
// and so can sum past it - hence testing with Rounds set so only 2 rounds
// remain, rather than with a win% that would overstate the real chance
// either player finishes 1st.
func TestRunawayLeadersHalveHopefulness(t *testing.T) {
	is := is.New(t)

	// Leader (P8) and 2nd (P0), with 2 rounds remaining, combine for an 80.2%
	// simulated chance of finishing 1st: just over 80%, so the
	// hopeful-for-1st/2nd bar is halved.
	req := hopefulnessTestReq()
	req.Rounds = 7
	pairtestutils.AddRoundResultsAndPairingsStr(req, "1 500 0 400 3 450 2 400 5 450 4 400 7 450 6 400 9 450 8 400")
	pairtestutils.AddRoundResultsAndPairingsStr(req, "2 500 3 400 0 450 1 400 6 400 7 400 4 400 5 400 8 400 9 400")
	pairtestutils.AddRoundResultsAndPairingsStr(req, "3 500 2 400 1 450 0 400 7 400 6 400 5 400 4 400 9 400 8 400")
	pairtestutils.AddRoundResultsAndPairingsStr(req, "4 400 5 400 6 400 7 400 0 400 1 500 2 400 3 400 8 400 9 400")
	pairtestutils.AddRoundResultsAndPairingsStr(req, "5 400 4 400 7 400 6 400 1 400 0 500 3 400 2 400 9 400 8 400")

	copRand := rand.New(rand.NewSource(1))
	var logsb strings.Builder
	copdata, pairErr := pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.True(strings.Contains(logsb.String(), "Leader+2nd combined 1st-place% (80.2%) > 80%"))
	is.True(strings.Contains(logsb.String(), "halving the hopeful-for-1st/2nd bar to 325 (normally 650)"))
	is.Equal(copdata.LowestPossibleHopeNth[0], 2)
	is.Equal(copdata.LowestPossibleHopeNth[1], 6)

	// Before any rounds are played there's no simulation to compute a 1st-
	// place probability from, so the halving never triggers (guards the
	// numPlayers<2/TotalSims==0 cases).
	req = hopefulnessTestReq()
	copRand.Seed(1)
	logsb.Reset()
	copdata, pairErr = pkgcopdata.GetPrecompData(req, copRand, &logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.True(!strings.Contains(logsb.String(), "halving the hopeful-for-1st/2nd bar"))
}
