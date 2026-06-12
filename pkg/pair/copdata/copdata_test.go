package copdata_test

import (
	"strings"
	"testing"

	"github.com/matryer/is"
	pkgcopdata "github.com/woogles-io/liwords/pkg/pair/copdata"
	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"golang.org/x/exp/rand"
)

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
	is.Equal(copdata.HighestRankHopefully[6], 3)
	is.Equal(copdata.HighestRankAbsolutely[6], 3)
	is.Equal(copdata.HighestRankHopefully[7], 4)
	is.Equal(copdata.HighestRankAbsolutely[7], 3)
	is.Equal(copdata.HighestRankHopefully[17], 9)
	is.Equal(copdata.HighestRankAbsolutely[17], 7)
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
	is.Equal(copdata.HighestRankHopefully[23], 16)
	is.Equal(copdata.HighestRankAbsolutely[23], 15)
	is.Equal(copdata.LowestRankAbsolutely[0], 0)
	is.Equal(copdata.LowestRankAbsolutely[1], 1)
	is.Equal(copdata.LowestRankAbsolutely[2], 9)
	is.Equal(copdata.LowestRankAbsolutely[17], 24)
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
	is.Equal(copdata.DestinysChild, -1)

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
