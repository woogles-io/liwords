package pair

import (
	"strconv"
	"strings"
	"testing"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestCOP(t *testing.T) {
	is := is.New(t)
	req := defaultPairRequest()
	addRoundPairings(req, "4 5 6 7 0 1 2 3")
	addRoundResults(req, "300 250 400 500 400 300 425 200")
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
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
