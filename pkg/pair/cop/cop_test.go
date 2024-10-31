package cop_test

import (
	"testing"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/pair/cop"

	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestCOPErrors(t *testing.T) {
	is := is.New(t)
	req := pairtestutils.CreateDefaultPairRequest()
	req.AllPlayers = -1
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_COUNT_INSUFFICIENT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.AllPlayers = 0
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_COUNT_INSUFFICIENT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.Rounds = -1
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_ROUND_COUNT_INSUFFICIENT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.Rounds = 0
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_ROUND_COUNT_INSUFFICIENT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.AllPlayers = 100000
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_COUNT_TOO_LARGE)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerNames = []string{"a", "b", "c"}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_NAME_COUNT_INSUFFICIENT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerNames[5] = ""
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_NAME_EMPTY)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_PAIRINGS_THAN_ROUNDS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_ALL_ROUNDS_PAIRED)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: []int32{4, 5, 6, 7}})
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_ROUND_PAIRINGS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairings(req, "4 5 20 7 0 1 2 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_INDEX_OUT_OF_BOUNDS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairings(req, "4 5 -6 7 0 1 2 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_INDEX_OUT_OF_BOUNDS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundResultsAndPairings(req, "4 300 5 250 6 400 -1 500 0 400 1 300 2 425 3 200")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_UNPAIRED_PLAYER)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 -1 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PAIRING)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 1 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PAIRING)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_RESULTS_THAN_ROUNDS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 425 200 500")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_RESULTS_THAN_PAIRINGS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairings(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundResults(req, "400 300 250 400 300 500")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_ROUND_RESULTS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.Classes = []int32{0, 1, 2, 3, 4, 5, 6, 7, 8}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_CLASSES_THAN_PLAYERS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.Classes = []int32{-1}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.Classes = []int32{0}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.Classes = []int32{8}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.Classes = []int32{3, 2}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MISORDERED_CLASS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ClassPrizes = []int32{2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS_PRIZES_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ClassPrizes = []int32{-1}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS_PRIZE)

	req = pairtestutils.CreateDefaultPairRequest()
	req.GibsonSpreads = []int32{100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_GIBSON_SPREAD_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.GibsonSpreads = []int32{-100}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_GIBSON_SPREAD)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ControlLossThreshold = 2.4
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_THRESHOLD)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ControlLossThreshold = -1.3
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_THRESHOLD)

	req = pairtestutils.CreateDefaultPairRequest()
	req.HopefulnessThreshold = 2.4
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_HOPEFULNESS_THRESHOLD)

	req = pairtestutils.CreateDefaultPairRequest()
	req.HopefulnessThreshold = -1.3
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_HOPEFULNESS_THRESHOLD)

	req = pairtestutils.CreateDefaultPairRequest()
	req.DivisionSims = -1
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_DIVISION_SIMS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.DivisionSims = 0
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_DIVISION_SIMS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ControlLossSims = -1
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_SIMS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ControlLossSims = 0
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_SIMS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlacePrizes = -1
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLACE_PRIZES)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlacePrizes = 9
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLACE_PRIZES)

	req = pairtestutils.CreateDefaultPairRequest()
	req.RemovedPlayers = []int32{0, 8, 1}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_REMOVED_PLAYER)

	req = pairtestutils.CreateDefaultPairRequest()
	req.RemovedPlayers = []int32{0, -1}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_REMOVED_PLAYER)
}

func TestCOPSuccess(t *testing.T) {
	is := is.New(t)
	req := pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundResultsAndPairings(req, "4 300 5 250 6 400 7 500 0 300 1 300 2 425 3 200")
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	req = pairtestutils.CreateDefaultOddPairRequest()
	pairtestutils.AddRoundResultsAndPairings(req, "4 300 5 250 6 400 3 50 0 400 1 300 2 425")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}
