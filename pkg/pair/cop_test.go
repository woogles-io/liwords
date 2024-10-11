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
	addRoundPairings(req, "4 5 20 7 0 1 2 3")
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

func TestCOPPrecompData(t *testing.T) {
	is := is.New(t)
	var logsb strings.Builder

	req := createKingston2023AfterRound15PairRequest()
	precompData, resp := getPrecompData(req, &logsb)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	is.Equal(precompData.gibsonizedPlayers[0], false)
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

func createKingston2023AfterRound15PairRequest() *pb.PairRequest {
	request := &pb.PairRequest{
		PairMethod:  pb.PairMethod_COP,
		Players:     26,
		Rounds:      23,
		PlayerNames: []string{"Jackson Smylie", "Joshua Sokol", "Arie Sinke", "Michael Fagen", "Max Panitch", "Lou Cornelis", "Eric Goldstein", "Jason Li", "Christopher Sykes", "Chloe Fatsis", "Deen Hergott", "Stefan Fatsis", "Noah Kalus", "Steve Ozorio", "Anna Miransky", "Lydia Keras", "Andre Popadynec", "Marcela Kadanka", "Agnes Kramer", "Richard Kirsch", "Joan Buma", "Gregg Bigourdin", "Sharmaine Farini", "Lilla Sinanan", "Mad Palazzo", "Trevor Sealy"},
		DivisionPairings: []*pb.RoundPairings{
			{Pairings: []int32{20, 22, 23, 25, 21, 24, 10, 14, 16, 15, 6, 17, 19, 18, 7, 9, 8, 11, 13, 12, 0, 4, 1, 2, 5, 3}},
			{Pairings: []int32{16, 13, 17, 19, 14, 15, 18, 21, 20, 24, 22, 23, 25, 1, 4, 5, 0, 2, 6, 3, 8, 7, 10, 11, 9, 12}},
			{Pairings: []int32{8, 6, 11, 12, 7, 9, 1, 4, 0, 5, 13, 2, 3, 10, 21, 24, 20, 23, 22, 25, 16, 14, 18, 17, 15, 19}},
			{Pairings: []int32{3, 18, 10, 0, 19, 7, 14, 5, 24, 16, 2, 22, 21, 20, 6, 23, 9, 25, 1, 4, 13, 12, 11, 15, 8, 17}},
			{Pairings: []int32{10, 7, 3, 2, 24, 18, 21, 1, 19, 25, 0, 15, 14, 22, 12, 11, 17, 16, 5, 8, 23, 6, 13, 20, 4, 9}},
			{Pairings: []int32{11, 19, 4, 15, 2, 14, 24, 8, 7, 13, 12, 0, 10, 9, 5, 3, 22, 18, 17, 1, 21, 20, 16, 25, 6, 23}},
			{Pairings: []int32{19, 12, 5, 14, 18, 2, 13, 17, 10, 11, 8, 9, 1, 6, 3, 22, 25, 7, 4, 0, 24, 23, 15, 21, 20, 16}},
			{Pairings: []int32{2, 17, 0, 4, 3, 19, 9, 10, 12, 6, 7, 20, 8, 15, 16, 13, 14, 1, 21, 5, 11, 18, 23, 22, 25, 24}},
			{Pairings: []int32{1, 0, 7, 17, 6, 8, 4, 2, 5, 12, 23, 18, 9, 19, 24, 16, 15, 3, 11, 13, 22, 25, 20, 10, 14, 21}},
			{Pairings: []int32{5, 8, 12, 9, 16, 0, 17, 13, 1, 3, 20, 21, 2, 7, 23, 19, 4, 6, 24, 15, 10, 11, 25, 14, 18, 22}},
			{Pairings: []int32{4, 3, 9, 1, 0, 17, 8, 18, 6, 2, 15, 13, 16, 11, 22, 10, 12, 5, 7, 23, 25, 24, 14, 19, 21, 20}},
			{Pairings: []int32{7, 9, 8, 11, 12, 6, 5, 0, 2, 1, 19, 3, 4, 25, 17, 20, 21, 14, 23, 10, 15, 16, 24, 18, 22, 13}},
			{Pairings: []int32{14, 5, 6, 21, 17, 1, 2, 16, 9, 8, 24, 12, 11, 23, 0, 25, 7, 4, 20, 22, 18, 3, 19, 13, 10, 15}},
			{Pairings: []int32{6, 2, 1, 8, 5, 4, 0, 9, 3, 7, 25, 14, 24, 17, 11, 18, 23, 13, 15, 20, 19, 22, 21, 16, 12, 10}},
			{Pairings: []int32{8, 6, 19, 9, 22, 11, 1, 12, 0, 3, 18, 5, 7, 16, 25, 21, 13, 20, 10, 2, 17, 15, 4, 24, 23, 14}},
		},
		DivisionResults: []*pb.RoundResults{
			{Results: []int32{572, 430, 504, 392, 413, 401, 427, 322, 422, 449, 376, 332, 355, 416, 438, 367, 224, 475, 389, 435, 324, 408, 397, 307, 355, 378}},
			{Results: []int32{395, 465, 502, 382, 440, 416, 445, 483, 530, 449, 420, 347, 533, 302, 287, 336, 329, 301, 300, 361, 316, 320, 379, 343, 341, 336}},
			{Results: []int32{388, 484, 494, 334, 396, 450, 467, 378, 352, 396, 409, 353, 392, 398, 364, 267, 477, 419, 551, 332, 347, 410, 289, 309, 398, 408}},
			{Results: []int32{402, 437, 441, 503, 340, 361, 456, 452, 421, 415, 301, 366, 404, 471, 420, 407, 267, 419, 413, 452, 403, 355, 314, 280, 414, 373}},
			{Results: []int32{378, 415, 407, 412, 568, 374, 453, 384, 412, 464, 444, 381, 339, 408, 479, 315, 388, 456, 417, 334, 457, 351, 402, 344, 324, 354}},
			{Results: []int32{460, 449, 417, 406, 379, 441, 332, 377, 455, 387, 405, 262, 469, 351, 345, 393, 467, 418, 360, 400, 381, 418, 256, 352, 482, 358}},
			{Results: []int32{342, 395, 414, 437, 393, 529, 494, 466, 392, 535, 356, 333, 426, 370, 406, 388, 353, 363, 419, 344, 404, 390, 286, 343, 409, 330}},
			{Results: []int32{513, 448, 413, 354, 498, 471, 458, 469, 486, 246, 341, 593, 328, 381, 308, 286, 425, 493, 379, 394, 344, 570, 378, 342, 427, 334}},
			{Results: []int32{410, 531, 476, 489, 353, 380, 433, 350, 385, 454, 314, 403, 425, 388, 380, 383, 387, 375, 503, 334, 401, 344, 404, 454, 327, 349}},
			{Results: []int32{434, 393, 479, 478, 459, 410, 433, 498, 387, 484, 430, 473, 329, 340, 566, 322, 440, 319, 414, 365, 312, 407, 393, 311, 431, 345}},
			{Results: []int32{453, 439, 405, 359, 386, 450, 327, 414, 488, 369, 496, 423, 452, 187, 446, 350, 387, 226, 316, 418, 359, 455, 389, 380, 394, 448}},
			{Results: []int32{436, 360, 386, 384, 444, 358, 433, 423, 445, 470, 330, 378, 452, 333, 380, 435, 407, 358, 448, 344, 376, 372, 243, 437, 437, 406}},
			{Results: []int32{470, 444, 353, 340, 381, 350, 432, 413, 435, 301, 378, 439, 367, 523, 367, 378, 370, 442, 442, 343, 331, 328, 329, 322, 365, 314}},
			{Results: []int32{400, 482, 374, 451, 409, 424, 442, 510, 413, 323, 514, 356, 414, 326, 359, 382, 496, 370, 326, 444, 414, 293, 445, 324, 355, 282}},
			{Results: []int32{442, 414, 477, 469, 360, 537, 504, 382, 400, 431, 386, 249, 405, 428, 402, 358, 267, 325, 428, 404, 408, 481, 356, 427, 368, 378}},
		},
		Classes:              []int32{4},
		ClassPrizes:          []int32{2},
		GibsonSpreads:        []int32{300, 250, 200},
		ControlLossThreshold: 0.25,
		HopefulnessThreshold: 0.02,
		PlacePrizes:          2,
		DivisionSims:         1000,
		ControlLossSims:      1000,
		UseControlLoss:       false,
		AllowRepeatByes:      false,
	}
	return request
}
