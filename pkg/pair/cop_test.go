package pair

import (
	"strconv"
	"strings"
	"testing"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestCOPErrors(t *testing.T) {
	is := is.New(t)
	req := defaultPairRequest()
	req.Players = -1
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_COUNT_INSUFFICIENT)

	req = defaultPairRequest()
	req.Players = 0
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_COUNT_INSUFFICIENT)

	req = defaultPairRequest()
	req.Rounds = -1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_ROUND_COUNT_INSUFFICIENT)

	req = defaultPairRequest()
	req.Rounds = 0
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_ROUND_COUNT_INSUFFICIENT)

	req = defaultPairRequest()
	req.Players = 1 << PlayerSpreadOffset
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_COUNT_TOO_LARGE)

	req = defaultPairRequest()
	req.PlayerNames = []string{"a", "b", "c"}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_NAME_COUNT_INSUFFICIENT)

	req = defaultPairRequest()
	req.PlayerNames[5] = ""
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_NAME_EMPTY)

	req = defaultPairRequest()
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_PAIRINGS_THAN_ROUNDS)

	req = defaultPairRequest()
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_ALL_ROUNDS_PAIRED)

	req = defaultPairRequest()
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: []int32{4, 5, 6, 7}})
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_ROUND_PAIRINGS_COUNT)

	req = defaultPairRequest()
	addRoundPairings(req, "4 5 6 7 0 1 20 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_INDEX_OUT_OF_BOUNDS)

	req = defaultPairRequest()
	addRoundPairings(req, "4 5 -6 7 0 1 2 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_INDEX_OUT_OF_BOUNDS)

	req = defaultPairRequest()
	addRoundResultsAndPairings(req, "4 300 5 250 6 400 -1 500 0 400 1 300 2 425 3 200")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_UNPAIRED_PLAYER)

	req = defaultPairRequest()
	addRoundPairings(req, "4 5 6 7 0 1 -1 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PAIRING)

	req = defaultPairRequest()
	addRoundPairings(req, "4 5 6 7 0 1 1 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PAIRING)

	req = defaultPairRequest()
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_RESULTS_THAN_ROUNDS)

	req = defaultPairRequest()
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	addRoundResults(req, "400 300 250 400 300 425 200 500")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_RESULTS_THAN_PAIRINGS)

	req = defaultPairRequest()
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundResults(req, "400 300 250 400 300 500")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_ROUND_RESULTS_COUNT)

	req = defaultPairRequest()
	req.Classes = []int32{0, 1, 2, 3, 4, 5, 6, 7, 8}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_CLASSES_THAN_PLAYERS)

	req = defaultPairRequest()
	req.Classes = []int32{-1}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS)

	req = defaultPairRequest()
	req.Classes = []int32{0}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS)

	req = defaultPairRequest()
	req.Classes = []int32{8}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS)

	req = defaultPairRequest()
	req.Classes = []int32{3, 2}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MISORDERED_CLASS)

	req = defaultPairRequest()
	req.ClassPrizes = []int32{2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS_PRIZES_COUNT)

	req = defaultPairRequest()
	req.ClassPrizes = []int32{-1}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS_PRIZE)

	req = defaultPairRequest()
	req.GibsonSpreads = []int32{100, 100, 100, 100, 100, 100, 100, 100, 100}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_GIBSON_SPREAD_COUNT)

	req = defaultPairRequest()
	req.GibsonSpreads = []int32{-100}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_GIBSON_SPREAD)

	req = defaultPairRequest()
	req.ControlLossThreshold = 2.4
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_THRESHOLD)

	req = defaultPairRequest()
	req.ControlLossThreshold = -1.3
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_THRESHOLD)

	req = defaultPairRequest()
	req.HopefulnessThreshold = 2.4
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_HOPEFULNESS_THRESHOLD)

	req = defaultPairRequest()
	req.HopefulnessThreshold = -1.3
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_HOPEFULNESS_THRESHOLD)

	req = defaultPairRequest()
	req.DivisionSims = -1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_DIVISION_SIMS)

	req = defaultPairRequest()
	req.DivisionSims = 0
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_DIVISION_SIMS)

	req = defaultPairRequest()
	req.ControlLossSims = -1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_SIMS)

	req = defaultPairRequest()
	req.ControlLossSims = 0
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_SIMS)

	req = defaultPairRequest()
	req.PlacePrizes = -1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLACE_PRIZES)

	req = defaultPairRequest()
	req.PlacePrizes = 9
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLACE_PRIZES)
}

func TestCOPSuccess(t *testing.T) {
	is := is.New(t)
	req := defaultPairRequest()
	addRoundResultsAndPairings(req, "4 300 5 250 6 400 7 500 0 300 1 300 2 425 3 200")
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	req = defaultOddPairRequest()
	addRoundResultsAndPairings(req, "4 300 5 250 6 400 3 50 0 400 1 300 2 425")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

func addRoundPairings(request *pb.PairRequest, pairingsStr string) {
	pairings := strings.Fields(pairingsStr)
	roundPairings := &pb.RoundPairings{}
	for _, pairing := range pairings {
		pairingInt, err := strconv.Atoi(pairing)
		if err != nil {
			panic(err)
		}
		roundPairings.Pairings = append(roundPairings.Pairings, int32(pairingInt))
	}
	request.DivisionPairings = append(request.DivisionPairings, roundPairings)
}

func addRoundResults(request *pb.PairRequest, resultsStr string) {
	results := strings.Fields(resultsStr)
	roundResults := &pb.RoundResults{}
	for _, result := range results {
		resultInt, err := strconv.Atoi(result)
		if err != nil {
			panic(err)
		}
		roundResults.Results = append(roundResults.Results, int32(resultInt))
	}
	request.DivisionResults = append(request.DivisionResults, roundResults)
}

func addRoundResultsAndPairings(request *pb.PairRequest, combinedStr string) {
	fields := strings.Fields(combinedStr)

	roundPairings := &pb.RoundPairings{}
	roundResults := &pb.RoundResults{}

	if len(fields)%2 != 0 {
		panic("Input string must contain pairs of <opp_index> <player_score>")
	}

	for i := 0; i < len(fields); i += 2 {
		oppIndex, err := strconv.Atoi(fields[i])
		if err != nil {
			panic(err)
		}
		playerScore, err := strconv.Atoi(fields[i+1])
		if err != nil {
			panic(err)
		}

		roundPairings.Pairings = append(roundPairings.Pairings, int32(oppIndex))
		roundResults.Results = append(roundResults.Results, int32(playerScore))
	}

	request.DivisionPairings = append(request.DivisionPairings, roundPairings)
	request.DivisionResults = append(request.DivisionResults, roundResults)
}

func defaultPairRequest() *pb.PairRequest {
	request := &pb.PairRequest{
		PairMethod:           pb.PairMethod_COP,
		PlayerNames:          []string{"Alice", "Bob", "Charlie", "Dave", "Eric", "Frank", "Grace", "Holly"},
		DivisionPairings:     []*pb.RoundPairings{},
		DivisionResults:      []*pb.RoundResults{},
		Classes:              []int32{4},
		ClassPrizes:          []int32{2},
		GibsonSpreads:        []int32{300, 250, 200},
		ControlLossThreshold: 0.25,
		HopefulnessThreshold: 0.02,
		Players:              8,
		Rounds:               10,
		PlacePrizes:          2,
		DivisionSims:         1000,
		ControlLossSims:      1000,
		UseControlLoss:       false,
		AllowRepeatByes:      false,
	}
	return request
}

func defaultOddPairRequest() *pb.PairRequest {
	request := &pb.PairRequest{
		PairMethod:           pb.PairMethod_COP,
		PlayerNames:          []string{"Alice", "Bob", "Charlie", "Dave", "Eric", "Frank", "Grace"},
		DivisionPairings:     []*pb.RoundPairings{},
		DivisionResults:      []*pb.RoundResults{},
		Classes:              []int32{4},
		ClassPrizes:          []int32{2},
		GibsonSpreads:        []int32{300, 250, 200},
		ControlLossThreshold: 0.25,
		HopefulnessThreshold: 0.02,
		Players:              7,
		Rounds:               10,
		PlacePrizes:          2,
		DivisionSims:         1000,
		ControlLossSims:      1000,
		UseControlLoss:       false,
		AllowRepeatByes:      false,
	}
	return request
}
