package cop

import (
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/pair/verifyreq"

	copdatapkg "github.com/woogles-io/liwords/pkg/pair/copdata"
	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestCOPErrors(t *testing.T) {
	is := is.New(t)

	req := pairtestutils.CreateDefaultPairRequest()
	req.ValidPlayers = -1
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_COUNT_INSUFFICIENT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ValidPlayers = 0
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_COUNT_INSUFFICIENT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.Rounds = -1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_ROUND_COUNT_INSUFFICIENT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.Rounds = 0
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_ROUND_COUNT_INSUFFICIENT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.AllPlayers = 100000
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_COUNT_TOO_LARGE)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerNames = []string{"a", "b", "c"}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_NAME_COUNT_INSUFFICIENT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerNames[5] = ""
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_NAME_EMPTY)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_PAIRINGS_THAN_ROUNDS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_ALL_ROUNDS_PAIRED)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: []int32{4, 5, 6, 7}})
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_ROUND_PAIRINGS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 20 7 0 1 2 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_INDEX_OUT_OF_BOUNDS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 -6 7 0 1 2 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_INDEX_OUT_OF_BOUNDS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundResultsAndPairingsStr(req, "4 300 5 250 6 400 7 500 0 400 1 300 2 425 3 200")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 -1 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_UNPAIRED_PLAYER)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 -1 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PAIRING)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 1 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PAIRING)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_RESULTS_THAN_ROUNDS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_RESULTS_THAN_PAIRINGS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 500")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_ROUND_RESULTS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{0, 0, 0, 0, 0, 0, 0}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{0}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{0, 0, 0, 0, 0, 0, 0, -1}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{0, 0, 0, 0, 0, 0, 0, 2}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ClassPrizes = []int32{-1}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS_PRIZE)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ClassPrizes = []int32{0}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS_PRIZE)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ClassPrizes = []int32{-1}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS_PRIZE)

	req = pairtestutils.CreateDefaultPairRequest()
	req.GibsonSpread = -100
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_GIBSON_SPREAD)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ControlLossThreshold = 2.4
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_THRESHOLD)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ControlLossThreshold = -1.3
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_THRESHOLD)

	req = pairtestutils.CreateDefaultPairRequest()
	req.HopefulnessThreshold = 2.4
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_HOPEFULNESS_THRESHOLD)

	req = pairtestutils.CreateDefaultPairRequest()
	req.HopefulnessThreshold = -1.3
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_HOPEFULNESS_THRESHOLD)

	req = pairtestutils.CreateDefaultPairRequest()
	req.HopefulnessThreshold = 0
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_HOPEFULNESS_THRESHOLD)

	req = pairtestutils.CreateDefaultPairRequest()
	req.DivisionSims = -1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_DIVISION_SIMS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.DivisionSims = 0
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_DIVISION_SIMS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ControlLossSims = -1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_SIMS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ControlLossSims = 0
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_SIMS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlacePrizes = -1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLACE_PRIZES)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlacePrizes = 9
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLACE_PRIZES)

	req = pairtestutils.CreateDefaultPairRequest()
	req.RemovedPlayers = []int32{0, 8, 1}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_REMOVED_PLAYER)

	req = pairtestutils.CreateDefaultPairRequest()
	req.RemovedPlayers = []int32{0, -1}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_REMOVED_PLAYER)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ControlLossActivationRound = -1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_ACTIVATION_ROUND)
}

func TestCOPConstraintPolicies(t *testing.T) {
	is := is.New(t)

	// Prepaired players
	req := pairtestutils.CreateBellevilleCSWAfterRound12PairRequest()
	req.Seed = 1
	pairtestutils.AddRoundPairingsStr(req, "-1 -1 -1 10 -1 -1 -1 -1 -1 11 3 9")
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[3], int32(10))
	is.Equal(resp.Pairings[9], int32(11))
	is.Equal(resp.Pairings[10], int32(3))
	is.Equal(resp.Pairings[11], int32(9))

	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	pairtestutils.AddRoundPairingsStr(req, "-1 -1 -1 14 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 3 -1 -1 -1 -1 -1 21 20 -1 -1")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[4], int32(4))
	is.Equal(resp.Pairings[3], int32(14))
	is.Equal(resp.Pairings[14], int32(3))
	is.Equal(resp.Pairings[20], int32(21))
	is.Equal(resp.Pairings[21], int32(20))

	// KOTH
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	req.Rounds = 26
	resp = COPPair(req)
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))

	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	req.Rounds = 26
	req.PlacePrizes = 8
	resp = COPPair(req)
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	is.Equal(resp.Pairings[5], int32(6))
	is.Equal(resp.Pairings[6], int32(5))
	is.Equal(resp.Pairings[8], int32(10))
	is.Equal(resp.Pairings[10], int32(8))

	// KOTH Class Prizes
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	req.Rounds = 26
	req.ClassPrizes = []int32{1, 1}
	// Create class B
	req.PlayerClasses[9] = 1
	req.PlayerClasses[2] = 1
	req.PlayerClasses[12] = 1
	req.PlayerClasses[15] = 1
	req.PlayerClasses[5] = 1
	req.PlayerClasses[8] = 1
	// Create class C
	req.PlayerClasses[14] = 2
	req.PlayerClasses[13] = 2
	req.PlayerClasses[17] = 2
	req.PlayerClasses[21] = 2
	req.PlayerClasses[23] = 2
	req.PlayerClasses[19] = 2
	req.PlayerClasses[20] = 2
	resp = COPPair(req)
	// Expect the normal KOTH casher pairings:
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	// No class B pairings because a class B player can cash
	// Expect class C KOTH pairings for 1 class prize:
	is.Equal(resp.Pairings[14], int32(13))
	is.Equal(resp.Pairings[13], int32(14))

	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	req.Rounds = 26
	req.ClassPrizes = []int32{2}
	// Create class B
	req.PlayerClasses[14] = 1
	req.PlayerClasses[13] = 1
	req.PlayerClasses[17] = 1
	req.PlayerClasses[21] = 1
	req.PlayerClasses[23] = 1
	req.PlayerClasses[19] = 1
	req.PlayerClasses[20] = 1
	resp = COPPair(req)
	// Expect the normal KOTH casher pairings:
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	// Expect class B KOTH pairings for 2 class prizes:
	is.Equal(resp.Pairings[14], int32(13))
	is.Equal(resp.Pairings[13], int32(14))
	is.Equal(resp.Pairings[17], int32(21))
	is.Equal(resp.Pairings[21], int32(17))

	// Class B - 1 player class gibsonized
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	req.Rounds = 26
	req.ClassPrizes = []int32{2}
	// Create class B
	req.PlayerClasses[5] = 1
	req.PlayerClasses[14] = 1
	req.PlayerClasses[13] = 1
	req.PlayerClasses[17] = 1
	req.PlayerClasses[21] = 1
	req.PlayerClasses[23] = 1
	req.PlayerClasses[19] = 1
	req.PlayerClasses[20] = 1
	resp = COPPair(req)
	// Expect the normal KOTH casher pairings:
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	// Expect class B KOTH pairings for 1 class prizes:
	// The top 1 in class B cannot be surpassed by other
	// players in class B, so there is only 1 class B pairing
	is.Equal(resp.Pairings[13], int32(14))
	is.Equal(resp.Pairings[14], int32(13))

	// Class B - 2 players class gibsonized
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	req.Rounds = 26
	req.ClassPrizes = []int32{3}
	// Create class B
	req.PlayerClasses[5] = 1
	req.PlayerClasses[8] = 1
	req.PlayerClasses[13] = 1
	req.PlayerClasses[17] = 1
	req.PlayerClasses[21] = 1
	req.PlayerClasses[23] = 1
	req.PlayerClasses[19] = 1
	req.PlayerClasses[20] = 1
	resp = COPPair(req)
	// Expect the normal KOTH casher pairings:
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	// Expect class B KOTH pairings for 2 class prizes:
	// The top 2 in class B cannot be surpassed by other
	// players in class B
	is.Equal(resp.Pairings[5], int32(8))
	is.Equal(resp.Pairings[8], int32(5))
	is.Equal(resp.Pairings[13], int32(17))
	is.Equal(resp.Pairings[17], int32(13))

	// Class B - 3 players class gibsonized
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	req.Rounds = 26
	req.ClassPrizes = []int32{4}
	// Create class B
	req.PlayerClasses[5] = 1
	req.PlayerClasses[8] = 1
	req.PlayerClasses[18] = 1
	req.PlayerClasses[14] = 1
	req.PlayerClasses[13] = 1
	req.PlayerClasses[17] = 1
	req.PlayerClasses[21] = 1
	req.PlayerClasses[23] = 1
	req.PlayerClasses[19] = 1
	req.PlayerClasses[20] = 1
	resp = COPPair(req)
	// Expect the normal KOTH casher pairings:
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	// Expect class B KOTH pairings for 4 class prizes:
	// The top 2 play for first
	// The 3rd class B player is "gibsonized" for class B 3rd
	// 4th and 5th in class B play for 4th in class B
	is.Equal(resp.Pairings[5], int32(8))
	is.Equal(resp.Pairings[8], int32(5))
	is.Equal(resp.Pairings[14], int32(13))
	is.Equal(resp.Pairings[13], int32(14))

	// Class B - 4 players class gibsonized
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	req.Rounds = 26
	req.ClassPrizes = []int32{4}
	// Create class B
	req.PlayerClasses[5] = 1
	req.PlayerClasses[6] = 1
	req.PlayerClasses[8] = 1
	req.PlayerClasses[10] = 1
	req.PlayerClasses[14] = 1
	req.PlayerClasses[13] = 1
	req.PlayerClasses[17] = 1
	req.PlayerClasses[21] = 1
	req.PlayerClasses[23] = 1
	req.PlayerClasses[19] = 1
	req.PlayerClasses[20] = 1
	resp = COPPair(req)
	// Expect the normal KOTH casher pairings:
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	// Expect class B KOTH pairings for 4 class prizes:
	// No one can catch the top 4 class B players, so they just
	// play a straight KOTH among themselves
	is.Equal(resp.Pairings[5], int32(6))
	is.Equal(resp.Pairings[6], int32(5))
	is.Equal(resp.Pairings[8], int32(10))
	is.Equal(resp.Pairings[10], int32(8))
	is.True(resp.Pairings[14] != int32(13))
	is.True(resp.Pairings[13] != int32(14))

	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	req.Rounds = 26
	req.ClassPrizes = []int32{2, 2}
	// Create class B
	req.PlayerClasses[5] = 1
	req.PlayerClasses[8] = 1
	req.PlayerClasses[10] = 1
	req.PlayerClasses[22] = 1
	// Create class C
	req.PlayerClasses[14] = 2
	req.PlayerClasses[13] = 2
	req.PlayerClasses[17] = 2
	req.PlayerClasses[21] = 2
	req.PlayerClasses[23] = 2
	req.PlayerClasses[19] = 2
	req.PlayerClasses[20] = 2
	resp = COPPair(req)
	// Expect the normal KOTH casher pairings:
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	// Expect class B KOTH pairings for 2 class prizes:
	is.Equal(resp.Pairings[5], int32(8))
	is.Equal(resp.Pairings[8], int32(5))
	is.Equal(resp.Pairings[10], int32(22))
	is.Equal(resp.Pairings[22], int32(10))
	// Expect class C KOTH pairings for 2 class prizes:
	is.Equal(resp.Pairings[14], int32(13))
	is.Equal(resp.Pairings[13], int32(14))
	is.Equal(resp.Pairings[17], int32(21))
	is.Equal(resp.Pairings[21], int32(17))

	// Class B - only 1 valid pairing
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Rounds = 26
	req.ClassPrizes = []int32{2}
	// Create class B
	req.PlayerClasses[5] = 1
	req.PlayerClasses[6] = 1
	req.PlayerClasses[10] = 1
	req.PlayerClasses[18] = 1
	req.PlayerClasses[14] = 1
	req.PlayerClasses[13] = 1
	req.PlayerClasses[17] = 1
	req.PlayerClasses[21] = 1
	req.PlayerClasses[23] = 1
	req.PlayerClasses[19] = 1
	req.PlayerClasses[20] = 1
	resp = COPPair(req)
	// Expect the normal KOTH casher pairings:
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	// Only 1 valid pairing since the 4th class B
	// player can't catch the top 2 class B players
	is.Equal(resp.Pairings[5], int32(6))
	is.Equal(resp.Pairings[6], int32(5))
	is.True(resp.Pairings[10] != int32(18))
	is.True(resp.Pairings[18] != int32(10))

	// Class B - only 2 valid pairings
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	req.Rounds = 26
	req.ClassPrizes = []int32{3}
	// Create class B
	req.PlayerClasses[5] = 1
	req.PlayerClasses[6] = 1
	req.PlayerClasses[8] = 1
	req.PlayerClasses[10] = 1
	req.PlayerClasses[13] = 1
	req.PlayerClasses[14] = 1
	req.PlayerClasses[17] = 1
	req.PlayerClasses[21] = 1
	req.PlayerClasses[23] = 1
	req.PlayerClasses[19] = 1
	req.PlayerClasses[20] = 1
	resp = COPPair(req)
	// Expect the normal KOTH casher pairings:
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	// Only 2 pairings of class B KOTH are made
	// since the 6th class B player can't catch the top 3 class B players
	is.Equal(resp.Pairings[5], int32(6))
	is.Equal(resp.Pairings[6], int32(5))
	is.Equal(resp.Pairings[8], int32(10))
	is.Equal(resp.Pairings[10], int32(8))
	is.True(resp.Pairings[13] != int32(14))
	is.True(resp.Pairings[14] != int32(13))

	// Class B - Some players guaranteed money
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Rounds = 26
	req.ClassPrizes = []int32{1}
	// Create class B
	req.PlayerClasses[0] = 1
	req.PlayerClasses[1] = 1
	req.PlayerClasses[4] = 1
	req.PlayerClasses[5] = 1
	req.PlayerClasses[8] = 1
	resp = COPPair(req)
	// Expect the normal KOTH casher pairings:
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	// Class B
	is.Equal(resp.Pairings[5], int32(8))
	is.Equal(resp.Pairings[8], int32(5))

	// Class B - Some players guaranteed money, 1 player not guaranteed money
	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Rounds = 26
	req.ClassPrizes = []int32{1}
	// Create class B
	req.PlayerClasses[0] = 1
	req.PlayerClasses[1] = 1
	req.PlayerClasses[4] = 1
	req.PlayerClasses[2] = 1
	req.PlayerClasses[5] = 1
	req.PlayerClasses[8] = 1
	resp = COPPair(req)
	// Expect the normal KOTH casher pairings:
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[2], int32(9))
	is.Equal(resp.Pairings[9], int32(2))
	is.Equal(resp.Pairings[12], int32(15))
	is.Equal(resp.Pairings[15], int32(12))
	// Class B
	is.True(resp.Pairings[5] != int32(8))
	is.True(resp.Pairings[8] != int32(5))

	// Control loss with player in 2nd
	req = pairtestutils.CreateBellevilleCSWAfterRound12PairRequest()
	req.ControlLossActivationRound = 12
	req.Seed = 2
	resp = COPPair(req)
	is.Equal(resp.Pairings[0], int32(3))
	is.Equal(resp.Pairings[3], int32(0))

	// Control loss with player in 4th
	req = pairtestutils.CreateBellevilleCSW4thCLAfterRound12PairRequest()
	req.ControlLossActivationRound = 11
	req.Seed = 1
	resp = COPPair(req)
	// The control loss should force 1st to play either 2nd or 3rd since 4th
	// isn't hopeful enough.
	is.True(resp.Pairings[3] == int32(2) || resp.Pairings[3] == int32(0))

	// Gibson groups and Gibson Bye
	req = pairtestutils.CreateAlbany1stAnd4thAnd8thGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	resp = COPPair(req)
	is.Equal(resp.Pairings[0], int32(4))
	is.Equal(resp.Pairings[4], int32(0))
	is.Equal(resp.Pairings[1], int32(1))
	is.Equal(resp.Pairings[11], int32(-1))
	resp.Pairings[11] = 11
	pairtestutils.AddRoundPairings(req, resp.Pairings)
	resp = COPPair(req)
	is.Equal(resp.Pairings[0], int32(4))
	is.Equal(resp.Pairings[4], int32(0))
	is.Equal(resp.Pairings[2], int32(2))

	// Gibson Bye
	req = pairtestutils.CreateAlbanyCSWAfterRound24OddPairRequest()
	is.Equal(verifyreq.Verify(req), nil)
	req.Seed = 1
	resp = COPPair(req)
	is.Equal(resp.Pairings[10], int32(10))
	pairtestutils.AddRoundPairings(req, resp.Pairings)
	resp = COPPair(req)
	is.Equal(resp.Pairings[10], int32(10))

	req = pairtestutils.CreateLakeGeorgeAfterRound13PairRequest()
	is.Equal(verifyreq.Verify(req), nil)

	// With 6 rounds remaining and KOTH as the last round, rank 4 (not rank 3) is
	// the lowest contender with control loss. Prepairing player 25 (rank 3) with
	// player 6 no longer conflicts with the Destiny's Child requirement.
	req = pairtestutils.CreateAlbanyjuly4th2024AfterRound21PairRequest()
	req.ControlLossActivationRound = 21
	req.Seed = 1
	pairings := make([]int32, req.AllPlayers)
	for i := range pairings {
		pairings[i] = -1
	}
	pairings[6] = 25
	pairings[25] = 6
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: pairings,
	})
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	// This is the second round that control loss is active, so first will
	// be force paired with the 2 lowest contenders. Therefore, prepairing the lowest
	// contender with someone else should not result in any errors.
	req = pairtestutils.CreateAlbanyjuly4th2024AfterRound21PairRequest()
	req.Rounds = 26
	req.ControlLossActivationRound = 20
	req.Seed = 1
	pairings = make([]int32, req.AllPlayers)
	for i := range pairings {
		pairings[i] = -1
	}
	pairings[6] = 25
	pairings[25] = 6
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: pairings,
	})
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Ben should be playing Wellington, the 2nd lowest contender
	is.Equal(resp.Pairings[0], int32(10))
	is.Equal(resp.Pairings[10], int32(0))

	// With only 4 rounds to go, first will again be paired with only the lowest contender.
	// Therefore, prepairing the lowest contender with someone else should result in an overconstrained error.
	req = pairtestutils.CreateBellevilleCSWAfterRound12PairRequest()
	req.ControlLossActivationRound = 12
	req.Seed = 1
	pairings = make([]int32, req.AllPlayers)
	for i := range pairings {
		pairings[i] = -1
	}
	pairings[0] = 10
	pairings[10] = 0
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: pairings,
	})
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_OVERCONSTRAINED)

	// With only 2 rounds to go, first will again be paired with only the lowest contender.
	// Therefore, prepairing the lowest contender with someone else should result in an overconstrained error.
	req = pairtestutils.CreateAlbanyjuly4th2024AfterRound21PairRequest()
	req.Rounds = 23
	req.ControlLossActivationRound = 20
	req.Seed = 1
	pairings = make([]int32, req.AllPlayers)
	for i := range pairings {
		pairings[i] = -1
	}
	pairings[4] = 25
	pairings[25] = 4
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: pairings,
	})
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_OVERCONSTRAINED)

	// Check that top down byes work
	req = pairtestutils.CreateAlbanyAfterRound16PairRequest()
	// The last pairings array in this request prepairs certain players, which
	// we need to remove for this test.
	req.DivisionPairings = req.DivisionPairings[:len(req.DivisionPairings)-1]
	req.TopDownByes = true
	req.AllowRepeatByes = false
	req.DivisionSims = 5000
	req.ControlLossSims = 1000
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Chris Sykes should have the bye
	is.Equal(resp.Pairings[1], int32(1))

	// Run the next round with the given pairings and check
	// that the next lowest player receives a bye.
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: resp.Pairings,
	})
	// Add a pairing for the removed player so the pairings are regarded
	// as complete
	req.DivisionPairings[len(req.DivisionPairings)-1].Pairings[11] = 11
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Zach should have the bye
	is.Equal(resp.Pairings[4], int32(4))

	// Run another round
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: resp.Pairings,
	})
	req.DivisionPairings[len(req.DivisionPairings)-1].Pairings[11] = 11
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Andy should have the bye
	is.Equal(resp.Pairings[2], int32(2))

	// Run another round
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: resp.Pairings,
	})
	req.DivisionPairings[len(req.DivisionPairings)-1].Pairings[11] = 11
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Billy should get the bye since Eric already received a bye
	is.Equal(resp.Pairings[5], int32(5))
}

// TestTopDownByePrecedence checks that top-down byes take precedence over
// KOTH-forced pairings (cash prize or class prize) in the final round. Before
// the fix, KH could force a pairing for the same player TB independently
// selected for the bye, which disallowed the bye pairing TB needed and the
// pairing KH needed on the same edge, producing OVERCONSTRAINED.
//
// It also guards against a second, subtler bug in the same area: the cash
// prize KOTH loop scans adjacent rank pairs, and originally just skipped the
// pair containing the bye recipient without removing the recipient from the
// scan - which silently dropped their neighbor (the player who should have
// been their KOTH opponent) from forcing entirely, letting the neighbor fall
// through to the general weighted matching and get paired with a player well
// outside cash contention. Below, P8 (rank 1) is that neighbor: without the
// fix it ends up paired with P5 (rank 7, not even hopeful to cash); with the
// fix it correctly reaches past the byeing P0 to pair with P2 (rank 3), the
// next real contender down.
func TestTopDownByePrecedence(t *testing.T) {
	is := is.New(t)

	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8"},
		PlayerClasses:              []int32{0, 0, 0, 1, 1, 1, 1, 1, 1},
		ClassPrizes:                []int32{1},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 9,
		ValidPlayers:               9,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               20000,
		ControlLossSims:            5000,
		ControlLossActivationRound: 8,
		AllowRepeatByes:            false,
		TopDownByes:                true,
		Seed:                       1,
	}
	pairtestutils.AddNDummyRounds(req, 7)
	res := "420 380 415 385 410 390 405 395 400"
	for range 7 {
		pairtestutils.AddRoundResultsStr(req, res)
	}

	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// P0 (rank 2, 0 prior byes) receives the top-down bye, even though cash
	// prize KOTH would otherwise force it to play P8 (rank 1).
	is.Equal(resp.Pairings[0], int32(0))
	// The rest of the field still gets KOTH-appropriate pairings: with P0
	// (rank 2) removed from the scan, P8 (rank 1) reaches past it to pair
	// with P2 (rank 3), the next real contender down, instead of being
	// dropped from KOTH forcing entirely.
	is.Equal(resp.Pairings[8], int32(2))
	is.Equal(resp.Pairings[2], int32(8))
	is.Equal(resp.Pairings[4], int32(6))
	is.Equal(resp.Pairings[6], int32(4))
	is.Equal(resp.Pairings[5], int32(7))
	is.Equal(resp.Pairings[7], int32(5))
	is.Equal(resp.Pairings[3], int32(1))
	is.Equal(resp.Pairings[1], int32(3))
}

// TestTopDownByeTakesPrecedenceOverGibsonBye checks that top-down byes take
// precedence over the Gibson bye. Before the fix, GB (which restricts the
// bye to Gibsonized players once someone has clinched 1st) disabled TB
// entirely whenever the Gibson bye applied, forcing a repeat bye onto the
// Gibsonized leader even when a bye-free player was available. Now TB is
// computed regardless of Gibson status, and GB defers to it.
func TestTopDownByeTakesPrecedenceOverGibsonBye(t *testing.T) {
	is := is.New(t)

	// 3 players. P0 gets a big enough lead (and a prior bye in round 1) to
	// clinch 1st with 1 round left. P2 gets byes in rounds 2 and 3 (allowed
	// to repeat since these are historical, already-played rounds), so P1 is
	// the only player with zero prior byes going into the final round.
	buildReq := func() *pb.PairRequest {
		req := &pb.PairRequest{
			PairMethod:                 pb.PairMethod_COP,
			PlayerNames:                []string{"P0", "P1", "P2"},
			PlayerClasses:              []int32{0, 0, 0},
			GibsonSpread:               50,
			ControlLossThreshold:       0.25,
			HopefulnessThreshold:       0.02,
			AllPlayers:                 3,
			ValidPlayers:               3,
			Rounds:                     4,
			PlacePrizes:                1,
			DivisionSims:               10000,
			ControlLossSims:            2000,
			ControlLossActivationRound: 10,
			AllowRepeatByes:            false,
			Seed:                       1,
		}
		// Round 1: P0 bye. P1 beats P2 by 100.
		pairtestutils.AddRoundPairingsStr(req, "0 2 1")
		pairtestutils.AddRoundResultsStr(req, "50 300 200")
		// Round 2: P2 bye. P0 crushes P1.
		pairtestutils.AddRoundPairingsStr(req, "1 0 2")
		pairtestutils.AddRoundResultsStr(req, "800 100 50")
		// Round 3: P2 bye again. P0 crushes P1 again. Going into round 4, P0
		// has 1 prior bye, P1 has 0, and P0 is clinched for 1st.
		pairtestutils.AddRoundPairingsStr(req, "1 0 2")
		pairtestutils.AddRoundResultsStr(req, "800 100 50")
		return req
	}

	// Without TopDownByes, GB forces the bye onto the Gibsonized leader P0,
	// repeating its round-1 bye.
	req := buildReq()
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[0], int32(0))

	// With TopDownByes, the bye instead goes to P1 (the only bye-free
	// player), even though P0 is still Gibsonized for 1st.
	req2 := buildReq()
	req2.TopDownByes = true
	resp2 := COPPair(req2)
	is.Equal(resp2.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp2.Pairings[1], int32(1))
	is.Equal(resp2.Pairings[0], int32(2))
	is.Equal(resp2.Pairings[2], int32(0))
}

// TestOddHopefulContenderGroup checks Feature 5: when the group of players
// still hopeful to reach 1st is odd, the next player down is barred from
// playing the leader; a lone hopeful contender is left alone unless the
// leader is gibsonized.
func TestOddHopefulContenderGroup(t *testing.T) {
	is := is.New(t)

	oddGroupReq := func() *pb.PairRequest {
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
			Seed:                       1,
		}
	}

	// Leader P8 has a 7-player hopeful-for-1st group (odd); P7 is pulled in
	// as the 8th (even it out) but barred from playing the leader.
	req := oddGroupReq()
	pairtestutils.AddRoundResultsAndPairingsStr(req, "1 500 0 400 3 450 2 400 5 450 4 400 7 450 6 400 9 450 8 400")
	pairtestutils.AddRoundResultsAndPairingsStr(req, "2 500 3 400 0 450 1 400 6 400 7 400 4 400 5 400 8 400 9 400")
	pairtestutils.AddRoundResultsAndPairingsStr(req, "3 500 2 400 1 450 0 400 7 400 6 400 5 400 4 400 9 400 8 400")
	pairtestutils.AddRoundResultsAndPairingsStr(req, "4 400 5 400 6 400 7 400 0 400 1 500 2 400 3 400 8 400 9 400")
	pairtestutils.AddRoundResultsAndPairingsStr(req, "5 400 4 400 7 400 6 400 1 400 0 500 3 400 2 400 9 400 8 400")
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(resp.Pairings[8] != int32(7))
	is.True(resp.Pairings[7] != int32(8))
	checkSymmetric(t, resp.Pairings)

	// Exception: exactly one hopeful contender (only the leader) and the
	// leader isn't gibsonized - no one is barred, and the natural weight
	// policies still pair the leader with rank 2 (1v2).
	req = &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7"},
		PlayerClasses:              make([]int32, 8),
		ClassPrizes:                []int32{2},
		GibsonSpread:               5000,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 8,
		ValidPlayers:               8,
		Rounds:                     8,
		PlacePrizes:                1,
		DivisionSims:               20000,
		ControlLossSims:            5000,
		ControlLossActivationRound: 8,
		Seed:                       1,
	}
	pairtestutils.AddNDummyRounds(req, 6)
	r1 := "700 300 420 380 415 385 410 390"
	r2 := "700 300 380 420 385 415 390 410"
	r3 := "300 700 380 420 385 415 390 410"
	pairtestutils.AddRoundResultsStr(req, r1)
	pairtestutils.AddRoundResultsStr(req, r2)
	pairtestutils.AddRoundResultsStr(req, r1)
	pairtestutils.AddRoundResultsStr(req, r2)
	pairtestutils.AddRoundResultsStr(req, r1)
	pairtestutils.AddRoundResultsStr(req, r3)
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[0], int32(7))
	is.Equal(resp.Pairings[7], int32(0))

	// Same single-contender setup, but the leader is gibsonized (tighter
	// GibsonSpread): the exception no longer applies, so P7 is pulled in and
	// barred from playing the leader. With PlacePrizes=1 here, "hopeful to
	// cash" and "hopeful for 1st" are the same group (both read
	// LowestPossibleHopeNth[0]), but GetPrecompData's parity promotion now
	// excludes the Gibsonized leader from its count (see
	// "Exclude Gibsonized players from hopeful-to-cash parity count"), so a
	// raw group of 1 (just the Gibsonized leader) counts as 0 - already even
	// - and no promotion fires to mask the leader-bar path here. P7 is
	// therefore genuinely barred from playing P0, so the natural weight
	// policies pair P0 with P6 instead (and P7 with P5). (See
	// TestOddHopefulContenderGroupVsFactor3 for leader-bar coverage at
	// PlacePrizes>1.)
	req.GibsonSpread = 200
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(resp.Pairings[0] != int32(7))
	is.True(resp.Pairings[7] != int32(0))
	is.Equal(resp.Pairings[0], int32(6))
	is.Equal(resp.Pairings[7], int32(5))
	checkSymmetric(t, resp.Pairings)
}

// TestOddHopefulContenderGroupVsFactor3 regression-tests a conflict the odd-
// hopeful-contender-group rule can have with Factor 3 expansion: F3 forces
// specific pairings among the top 6 (including the leader's), and the group
// rule must not also try to bar the leader's F3-forced opponent - it must
// stand down, mirroring the CL policy's existing precedent for F3.
func TestOddHopefulContenderGroupVsFactor3(t *testing.T) {
	is := is.New(t)

	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10", "P11"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.30,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 12,
		ValidPlayers:               12,
		Rounds:                     8,
		PlacePrizes:                3,
		DivisionSims:               20000,
		ControlLossSims:            5000,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	pairtestutils.AddNDummyRounds(req, 6)
	pairtestutils.AddRoundResultsStr(req, "420 400 415 400 414 400 510 400 422 400 420 400")
	pairtestutils.AddRoundResultsStr(req, "420 400 415 400 414 400 400 410 400 417 400 422")
	pairtestutils.AddRoundResultsStr(req, "420 400 415 400 414 400 510 400 422 400 420 400")
	pairtestutils.AddRoundResultsStr(req, "420 400 415 400 414 400 400 410 400 417 400 422")
	pairtestutils.AddRoundResultsStr(req, "420 400 400 426 400 427 510 400 422 400 420 400")
	pairtestutils.AddRoundResultsStr(req, "400 495 400 430 400 426 400 410 400 417 400 422")

	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// TestFactor3HopefulnessUsesAtLeastNotExact regression-tests a real-world bug
// (reported from a July 4th 2026 run, round 27): the F3 hopefulness check
// compared each of 4th/5th/6th's chance of landing on their *exact* target
// final rank (1st/2nd/3rd respectively) against the hopefulness threshold,
// instead of their chance of finishing at that rank *or better*. A player who
// is very likely to leapfrog straight past their target rank into a better
// one - and so is genuinely hopeful for it - could have a tiny exact-rank
// probability and incorrectly fail the check, causing F3 to be skipped when
// it should have expanded.
//
// This scenario reproduces that shape: 6th place (P6) trails 4th/5th by wins
// but is spread-competitive with 1st-3rd, so when it does crack the top 3 it
// usually leaps to 1st or 2nd rather than landing exactly on 3rd. With
// HopefulnessThreshold=0.10, 6th's chance of finishing *exactly* 3rd is only
// ~4%, but its chance of finishing 3rd-or-better is ~25% - so under the old
// exact-rank check F3 was wrongly skipped, and under the fixed cumulative
// check it correctly expands.
func TestFactor3HopefulnessUsesAtLeastNotExact(t *testing.T) {
	is := is.New(t)

	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10", "P11"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.99, // effectively disable the control-loss branch
		HopefulnessThreshold:       0.10,
		AllPlayers:                 12,
		ValidPlayers:               12,
		Rounds:                     8,
		PlacePrizes:                3,
		DivisionSims:               100000,
		ControlLossSims:            2000,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	pairtestutils.AddNDummyRounds(req, 6)
	// P0,P2,P4 (1st-3rd): win rounds 1-5, lose round 6 -> 5-1.
	// P6,P8 (4th,5th): win rounds 1-4, lose rounds 5-6 -> 4-2, small margins.
	// P10 (6th): win rounds 1-4, lose rounds 5-6 -> 4-2, but a much bigger
	// margin, so it's spread-competitive with 1st-3rd despite being ranked 6th.
	mk := func(a, b, c, d, e, f int) string {
		return fmt.Sprintf("%d 400 %d 400 %d 400 %d 400 %d 400 %d 400", 400+a, 400+b, 400+c, 400+d, 400+e, 400+f)
	}
	for range 4 {
		pairtestutils.AddRoundResultsStr(req, mk(18, 13, 8, 1, 1, 100))
	}
	pairtestutils.AddRoundResultsStr(req, mk(18, 13, 8, -1, -1, -100))
	pairtestutils.AddRoundResultsStr(req, mk(-18, -13, -8, -1, -1, -100))

	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "Factor 3 expansion:"))
	is.True(!strings.Contains(resp.Log, "Factor 3 skipped: hopefulness"))
}

// TestHopefulCasherGrouping checks Feature 2: in the last quarter of the
// tournament, players who are hopeful to cash play other hopeful-to-cash
// players, and players who aren't hopeful to cash play other non-hopeful
// players. P0,P2,P4,P6,P8,P10 are a clean 6-0 top half (all hopeful to cash);
// P1,P3,P5,P7,P9,P11 are a clean 0-6 bottom half (none hopeful to cash).
func TestHopefulCasherGrouping(t *testing.T) {
	is := is.New(t)

	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10", "P11"},
		PlayerClasses:              make([]int32, 12),
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.05,
		AllPlayers:                 12,
		ValidPlayers:               12,
		Rounds:                     8,
		PlacePrizes:                4,
		DivisionSims:               5000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 8,
		Seed:                       1,
	}
	pairtestutils.AddNDummyRounds(req, 6)
	res := "460 340 450 350 440 360 430 370 420 380 410 390"
	for range 6 {
		pairtestutils.AddRoundResultsStr(req, res)
	}

	// Q4 (2 rounds left of 8): no pairing crosses the hopeful/non-hopeful line.
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	hopeful := map[int32]bool{0: true, 2: true, 4: true, 6: true, 8: true, 10: true}
	for pi, opp := range resp.Pairings {
		is.Equal(hopeful[int32(pi)], hopeful[opp])
	}

	// Outside Q4 (Rounds=30, same 6-round history: 24 rounds left), CC doesn't
	// apply; confirm the pairing algorithm still succeeds and isn't required
	// to keep the same grouping.
	req.Rounds = 30
	req.ControlLossActivationRound = 30
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
}

// TestBottomSixMajorWeight checks Feature 3: in divisions of 12+ players,
// during the last quarter, a major weight discourages hopeful-to-cash
// players from playing players in the bottom 6 of the standings.
func TestBottomSixMajorWeight(t *testing.T) {
	is := is.New(t)

	makeReq := func(names []int) *pb.PairRequest {
		playerNames := make([]string, len(names))
		for i := range playerNames {
			playerNames[i] = fmt.Sprintf("P%d", i)
		}
		return &pb.PairRequest{
			PairMethod:                 pb.PairMethod_COP,
			PlayerNames:                playerNames,
			PlayerClasses:              make([]int32, len(names)),
			ClassPrizes:                []int32{2},
			GibsonSpread:               200,
			ControlLossThreshold:       0.25,
			HopefulnessThreshold:       0.05,
			AllPlayers:                 int32(len(names)),
			ValidPlayers:               int32(len(names)),
			Rounds:                     8,
			PlacePrizes:                4,
			DivisionSims:               5000,
			ControlLossSims:            1000,
			ControlLossActivationRound: 8,
			Seed:                       1,
		}
	}

	// 12 players (>= 12): P0,P2,P4,P6,P8,P10 are a clean 6-0 top half
	// (hopeful to cash); P1,P3,P5,P7,P9,P11 are a clean 0-6 bottom half,
	// which is also exactly the bottom 6 of the 12-player standings. No
	// pairing crosses that line.
	req := makeReq(make([]int, 12))
	pairtestutils.AddNDummyRounds(req, 6)
	res := "460 340 450 350 440 360 430 370 420 380 410 390"
	for range 6 {
		pairtestutils.AddRoundResultsStr(req, res)
	}
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	bottomSix := map[int32]bool{1: true, 3: true, 5: true, 7: true, 9: true, 11: true}
	hopeful := map[int32]bool{0: true, 2: true, 4: true, 6: true, 8: true, 10: true}
	for pi, opp := range resp.Pairings {
		if hopeful[int32(pi)] {
			is.True(!bottomSix[opp])
		}
	}

	// Same shape of scenario, but with only 11 players (< 12): the bottom-6
	// rule never applies, regardless of the standings shape.
	req = makeReq(make([]int, 11))
	req.PlacePrizes = 4
	pairtestutils.AddNDummyRounds(req, 6)
	res = "460 340 450 350 420 380 415 385 410 390 405"
	for range 6 {
		pairtestutils.AddRoundResultsStr(req, res)
	}
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
}

// TestBottomSixHopefulOverlap regression-tests the interaction between CC's
// hopeful-vs-hopeful rule (Feature 2) and its hopeful-vs-bottom-6 rule
// (Feature 3) when a bottom-6 player is still (mathematically) hopeful to
// cash. Before the fix, such a player had no non-major-penalty pairing
// available: the hopeful-vs-hopeful rule would major-penalize pairing them
// with a non-hopeful player, and the hopeful-vs-bottom-6 rule would
// major-penalize pairing them with a hopeful one. The fix clamps the
// hopeful-to-cash boundary these two rules use so the bottom 6 are never
// counted as hopeful in divisions of 12+, giving bottom-6 players a clean
// pairing among themselves.
func TestBottomSixHopefulOverlap(t *testing.T) {
	is := is.New(t)

	req := &pb.PairRequest{
		PairMethod:           pb.PairMethod_COP,
		PlayerNames:          []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10", "P11", "P12"},
		PlayerClasses:        make([]int32, 13),
		ClassPrizes:          []int32{2},
		GibsonSpread:         200,
		ControlLossThreshold: 0.25,
		// A generous HopefulnessThreshold and PlacePrizes push the hopeful-to-
		// cash boundary deep enough to otherwise reach into the bottom 6.
		HopefulnessThreshold:       0.2,
		AllPlayers:                 13,
		ValidPlayers:               13,
		Rounds:                     8,
		PlacePrizes:                8,
		DivisionSims:               5000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 8,
		Seed:                       1,
	}
	pairtestutils.AddNDummyRounds(req, 6)
	res := "460 340 450 350 420 380 415 385 410 390 405 395 400"
	for range 6 {
		pairtestutils.AddRoundResultsStr(req, res)
	}
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	// P11 (bottom 6, still hopeful under the unclamped boundary) gets a
	// clean pairing with another bottom-6 player (P10) rather than being
	// forced into a major-penalty pairing no matter who it plays. (Which
	// bottom-6 player specifically shifts with unrelated weight tuning - e.g.
	// hopefulCasherByeWeight - since that changes who draws the bye and
	// therefore how the remaining bottom-6 players pair off among
	// themselves - but the pairing stays bottom-six-vs-bottom-six either way,
	// which is what this test guards.)
	is.Equal(resp.Pairings[11], int32(10))
	is.Equal(resp.Pairings[10], int32(11))
}

// TestForcedContenderBye checks that when PC (no bye for contenders) and BR
// (no repeat byes, since AllowRepeatByes=false) would otherwise be in direct
// conflict - every bye-repeat-free player is a hopeful-to-cash contender -
// the bye is forced onto that contender instead of repeating someone else's
// bye. P0 wins every real game it plays and never gets a bye across 4 rounds
// (5 players, so someone sits out every round); P1-P4 each get exactly one
// bye in turn and finish with losing records, leaving P0 as the sole
// contender for the 1 cash prize and the only bye-repeat-free player.
func TestForcedContenderBye(t *testing.T) {
	is := is.New(t)

	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4"},
		PlayerClasses:              make([]int32, 5),
		ClassPrizes:                []int32{2},
		GibsonSpread:               5000,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 5,
		ValidPlayers:               5,
		Rounds:                     8,
		PlacePrizes:                1,
		DivisionSims:               5000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 8,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	// R1: P1 byes; P0 beats P2, P3 beats P4.
	pairtestutils.AddRoundResultsAndPairingsStr(req, "2 500 1 50 0 300 4 420 3 380")
	// R2: P2 byes; P0 beats P3, P4 beats P1.
	pairtestutils.AddRoundResultsAndPairingsStr(req, "3 500 4 380 2 50 0 300 1 420")
	// R3: P3 byes; P0 beats P4, P1 beats P2.
	pairtestutils.AddRoundResultsAndPairingsStr(req, "4 500 2 420 1 380 3 50 0 300")
	// R4: P4 byes; P0 beats P1, P2 beats P3.
	pairtestutils.AddRoundResultsAndPairingsStr(req, "1 500 0 300 3 420 2 380 4 50")

	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	// P0 (4-0, the sole contender, and the only player without a prior bye)
	// takes the bye rather than repeating one onto P1-P4.
	is.Equal(resp.Pairings[0], int32(0))
}

func TestCOPWeights(t *testing.T) {
	is := is.New(t)

	req := pairtestutils.CreateBLSRound32PairRequest()
	req.Seed = 0
	is.Equal(verifyreq.Verify(req), nil)
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Matt T should be playing Rasheed, since rank differences
	// for pairings with a gibsonized player are squared.
	is.Equal(resp.Pairings[9], int32(25))
	is.Equal(resp.Pairings[25], int32(9))

	// PC weight uses LowestPossibleHopeNth exclusively for all hopeful cashers.
	// The retry mechanism works at the matching level: if a selected edge has weight
	// >= majorPenalty, LowestPossibleHopeNth is incremented for both players and
	// the matching is retried. In the fourth quarter (2 rounds left of 10), cashers
	// use PC weight only (RD is suppressed).
	//
	// AlmostGibsonized Q4: player 0 pairs with player 18 via the retry: the
	// first-pass edge (rank0, rank18) has weight >= majorPenalty, which expands
	// LowestPossibleHopeNth for those players, and the second pass selects the
	// same edge with a lower weight. CC (hopeful-to-cash vs hopeful-to-cash,
	// also Q4-only) then reshuffles who pairs with whom among the remaining
	// non-majorPenalty options so hopeful cashers stay together.
	req = pairtestutils.CreateAlmostGibsonizedPairRequest()
	req.Seed = 1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// whatnoloan and condorave still play (unchanged)
	is.Equal(resp.Pairings[1], int32(3))
	is.Equal(resp.Pairings[3], int32(1))
	// In the fourth quarter, RD is suppressed for cashers; PC and CC weight
	// drive player 4 to player 9 (non-majorPenalty, no retry).
	is.Equal(resp.Pairings[4], int32(9))
	is.Equal(resp.Pairings[9], int32(4))
	// Player 5 pairs with player 14 (within contender range, no retry).
	is.Equal(resp.Pairings[5], int32(14))
	is.Equal(resp.Pairings[14], int32(5))

	// Kingston 2023 after round 15: player 0's first-pass pairing is within the
	// contender range (non-majorPenalty), so no retry is triggered.
	req = pairtestutils.CreateKingston2023AfterRound15PairRequest()
	req.Seed = 1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[0], int32(9))

	// In the fourth quarter (roundsRemaining*4 <= Rounds), cashers use PC weight only
	// and non-cashers use RD weight only. Outside the fourth quarter, RD applies to all.
	//
	// AlmostGibsonized: lowestPossibleHopeCasher = rank 7. Rounds=10 is Q4 (2*4=8 <= 10).
	// Outside Q4 (Rounds=12, 4*4=16 > 12): RD-only for all players.
	req = pairtestutils.CreateAlmostGibsonizedPairRequest()
	req.Rounds = 12
	req.Seed = 1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[4], int32(9))
	is.Equal(resp.Pairings[10], int32(0))
	is.Equal(resp.Pairings[13], int32(17))
	is.Equal(resp.Pairings[17], int32(13))

	// In Q4 (Rounds=10), non-casher players pair by RD (and CC, which agrees
	// here since both are non-hopeful-to-cash) rather than crossing into the
	// hopeful-to-cash group.
	req = pairtestutils.CreateAlmostGibsonizedPairRequest()
	req.Seed = 1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[10], int32(13))
	is.Equal(resp.Pairings[13], int32(10))
}

// TestCOPFinalRoundGibsonHopefulCasherRepeat reproduces a bug reported from a
// hypothetical 12-player/7-round scenario: in the final round, with the
// standings leader Gibsonized and a lower-ranked player promoted to "hopeful
// to cash" by the odd-hopeful-to-cash-players parity fix, the fourth-quarter
// cash-contention weighting (PC/CC) forced the Gibsonized leader into an
// unnecessary repeat pairing against the promoted hopeful casher, even though
// a repeat-free arrangement existed for the leftover pool. isFinalRound
// disables that cash-contention weighting in the true final round so the
// leftover pool (everyone not already forced into a KOTH pair by KH) falls
// back to plain rank-diff/repeat weighting instead.
func TestCOPFinalRoundGibsonHopefulCasherRepeat(t *testing.T) {
	is := is.New(t)

	req := pairtestutils.CreateHypothetical12p7rRound7PairRequest()
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	// P3 (rank 1, Gibsonized leader) pairs with P4 (rank 10) rather than
	// repeating against P7 (rank 8, the promoted hopeful casher).
	is.Equal(resp.Pairings[3], int32(4))
	is.Equal(resp.Pairings[4], int32(3))
	// KH's forced cash-prize KOTH pairs: 2v3, 4v5, 6v7 (ranks).
	is.Equal(resp.Pairings[10], int32(2))
	is.Equal(resp.Pairings[2], int32(10))
	is.Equal(resp.Pairings[11], int32(0))
	is.Equal(resp.Pairings[0], int32(11))
	is.Equal(resp.Pairings[8], int32(1))
	is.Equal(resp.Pairings[1], int32(8))
	// The rest of the leftover pool also pairs repeat-free.
	is.Equal(resp.Pairings[7], int32(5))
	is.Equal(resp.Pairings[5], int32(7))
	is.Equal(resp.Pairings[9], int32(6))
	is.Equal(resp.Pairings[6], int32(9))
}

func TestCOPSuccess(t *testing.T) {
	is := is.New(t)

	req := pairtestutils.CreateDefaultPairRequest()
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

func TestCOPProdBugs(t *testing.T) {
	// These are all tests for requests that created unexpected behavior
	// in prod
	is := is.New(t)

	// Test players prepaired with byes
	req := pairtestutils.CreateAlbanyAfterRound16PairRequest()
	req.Seed = 1
	req.ControlLossSims = 1000
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[0], int32(22))
	is.Equal(resp.Pairings[11], int32(-1))
	is.Equal(resp.Pairings[19], int32(19))

	// Test that back-to-back pairings are penalized correctly
	req = pairtestutils.CreateAlbanyCSWNewYearsAfterRound27PairRequest()
	req.Seed = 1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	// There are pairings for all rounds, but the last round is only partially
	// paired, so this should finish successfully
	req = pairtestutils.CreateAlbanyCSWNewYearsAfterRound27LastRoundPartiallyPairedPairRequest()
	req.Seed = 1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	req = pairtestutils.CreateAlbanyCSWNewYearsRound25PartiallyPairedPairRequest()
	req.Seed = 1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	req = pairtestutils.CreateAlmostGibsonizedPairRequest()
	req.Seed = 1
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// whatnoloan is not gibsonized and is the only player who can hopefully win
	// Therefore, whatnoloan needs to play condorave since condorave is the player ranked just below whatnoloan
	is.Equal(resp.Pairings[1], int32(3))
	is.Equal(resp.Pairings[3], int32(1))

	req = pairtestutils.CreateLG2025Round15PairRequest()
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

// makeSimpleReq builds a minimal PairRequest for non-COP pairing methods.
func makeSimpleReq(method pb.PairMethod, numPlayers, numRounds int) *pb.PairRequest {
	names := make([]string, numPlayers)
	classes := make([]int32, numPlayers)
	for i := range names {
		names[i] = fmt.Sprintf("P%d", i+1)
	}
	return &pb.PairRequest{
		PairMethod:    method,
		PlayerNames:   names,
		PlayerClasses: classes,
		AllPlayers:    int32(numPlayers),
		ValidPlayers:  int32(numPlayers),
		Rounds:        int32(numRounds),
		Seed:          42,
	}
}

// checkSymmetric verifies all players are paired and pairings are symmetric (byes allowed).
func checkSymmetric(t *testing.T, pairings []int32) {
	t.Helper()
	for i, opp := range pairings {
		if opp < 0 {
			t.Errorf("player %d is unpaired (got %d)", i, opp)
			continue
		}
		if int(opp) == i {
			continue // bye
		}
		if pairings[opp] != int32(i) {
			t.Errorf("pairings not symmetric: pairings[%d]=%d but pairings[%d]=%d", i, opp, opp, pairings[opp])
		}
	}
}

// countByes returns the number of players paired with themselves (byes).
func countByes(pairings []int32) int {
	n := 0
	for i, opp := range pairings {
		if opp == int32(i) {
			n++
		}
	}
	return n
}

func TestRandom(t *testing.T) {
	is := is.New(t)

	// Even player count: everyone paired, no byes.
	req := makeSimpleReq(pb.PairMethod_PAIR_RANDOM, 8, 10)
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	is.Equal(countByes(resp.Pairings), 0)

	// Odd player count: exactly one bye.
	req = makeSimpleReq(pb.PairMethod_PAIR_RANDOM, 7, 10)
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	is.Equal(countByes(resp.Pairings), 1)
}

func TestRoundRobin(t *testing.T) {
	is := is.New(t)

	// Round 0 produces valid pairings.
	req := makeSimpleReq(pb.PairMethod_PAIR_ROUND_ROBIN, 8, 10)
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	round0P0 := resp.Pairings[0]

	// Round 1 rotates the schedule so player 0's opponent differs.
	pairtestutils.AddNDummyRounds(req, 1)
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	is.True(resp.Pairings[0] != round0P0)

	// Odd player count: exactly one bye per round.
	req = makeSimpleReq(pb.PairMethod_PAIR_ROUND_ROBIN, 7, 10)
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	is.Equal(countByes(resp.Pairings), 1)

	// Exhaustive cycle check: 8 players over 7 rounds (one full cycle) should
	// have every pair {i,j} appear exactly once.
	numPlayers := 8
	cycle := numPlayers - 1
	firstCycle := make([][]int32, cycle)
	for r := 0; r < cycle; r++ {
		req = makeSimpleReq(pb.PairMethod_PAIR_ROUND_ROBIN, numPlayers, 20)
		pairtestutils.AddNDummyRounds(req, r)
		resp = COPPair(req)
		is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
		checkSymmetric(t, resp.Pairings)
		cp := make([]int32, len(resp.Pairings))
		copy(cp, resp.Pairings)
		firstCycle[r] = cp
	}
	pairSeen := make(map[[2]int]bool)
	for _, roundPairings := range firstCycle {
		for i, opp := range roundPairings {
			if int(opp) > i {
				key := [2]int{i, int(opp)}
				is.True(!pairSeen[key])
				pairSeen[key] = true
			}
		}
	}
	is.Equal(len(pairSeen), numPlayers*(numPlayers-1)/2)

	// Second cycle (rounds 7–13) must repeat the identical schedule.
	for r := 0; r < cycle; r++ {
		req2 := makeSimpleReq(pb.PairMethod_PAIR_ROUND_ROBIN, numPlayers, 20)
		pairtestutils.AddNDummyRounds(req2, cycle+r)
		resp = COPPair(req2)
		is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
		is.Equal(resp.Pairings, firstCycle[r])
	}
}

func TestKingOfTheHill(t *testing.T) {
	is := is.New(t)

	// Pairings: 0↔7, 1↔6, 2↔5, 3↔4.
	// Each winner has a unique spread so standings are unambiguous without
	// relying on player index as a tiebreaker.
	// Spreads: P0=+100, P1=+200, P2=+300, P3=+400, P4=-400, P5=-300, P6=-200, P7=-100.
	// Standings: 3(+400), 2(+300), 1(+200), 0(+100), 7(-100), 6(-200), 5(-300), 4(-400).
	// KOTH pairs consecutive ranks: 3 vs 2, 1 vs 0, 7 vs 6, 5 vs 4.
	req := makeSimpleReq(pb.PairMethod_PAIR_KING_OF_THE_HILL, 8, 10)
	pairtestutils.AddRoundResultsAndPairingsStr(req, "7 350 6 400 5 450 4 500 3 100 2 150 1 200 0 250")
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	is.Equal(resp.Pairings[3], int32(2))
	is.Equal(resp.Pairings[2], int32(3))
	is.Equal(resp.Pairings[1], int32(0))
	is.Equal(resp.Pairings[0], int32(1))
	is.Equal(resp.Pairings[7], int32(6))
	is.Equal(resp.Pairings[6], int32(7))
	is.Equal(resp.Pairings[5], int32(4))
	is.Equal(resp.Pairings[4], int32(5))
}

func TestFactor(t *testing.T) {
	is := is.New(t)

	// Standings rank order after 1 round: 3,2,1,0,7,6,5,4.
	// Factor=2: top 4 get factor pairings: pool[0](3) vs pool[2](1), pool[1](2) vs pool[3](0).
	// Bottom 4 (players 7,6,5,4) get Swiss among themselves.
	req := makeSimpleReq(pb.PairMethod_PAIR_FACTOR, 8, 10)
	req.Factor = 2
	pairtestutils.AddRoundPairingsStr(req, "7 6 5 4 3 2 1 0")
	pairtestutils.AddRoundResultsStr(req, "400 400 400 400 0 0 0 0")
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	is.Equal(resp.Pairings[3], int32(1))
	is.Equal(resp.Pairings[1], int32(3))
	is.Equal(resp.Pairings[2], int32(0))
	is.Equal(resp.Pairings[0], int32(2))
	for _, pi := range []int{7, 6, 5, 4} {
		opp := int(resp.Pairings[pi])
		is.True(opp >= 4 && opp <= 7)
	}
}

func TestInitialFontes(t *testing.T) {
	is := is.New(t)

	// 8 players, 3 initial-fontes rounds → 4 groups of 2, each doing round robin.
	req := makeSimpleReq(pb.PairMethod_PAIR_INITIAL_FONTES, 8, 10)
	req.InitialNonperfRounds = 3
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)

	// Round 1 also valid (group members play their return match).
	pairtestutils.AddNDummyRounds(req, 1)
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
}

// TestSwiss verifies that PAIR_SWISS delegates entirely to COP: it produces
// exactly the same pairings as calling COP directly with the same request.
// swissPair was removed because COP's weight policies (repeats, rank/win
// diff, etc.) subsume its simpler win/spread-diff weighting.
func TestSwiss(t *testing.T) {
	is := is.New(t)

	req := pairtestutils.CreateDefaultPairRequest()
	req.PairMethod = pb.PairMethod_PAIR_SWISS
	pairtestutils.AddNDummyRounds(req, 3)
	pairtestutils.AddRoundResultsStr(req, "430 370 420 380 415 385 410 390")
	pairtestutils.AddRoundResultsStr(req, "370 430 380 420 385 415 390 410")
	pairtestutils.AddRoundResultsStr(req, "420 380 430 370 405 395 400 400")

	swissResp := COPPair(req)
	is.Equal(swissResp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, swissResp.Pairings)

	copReq := pairtestutils.CreateDefaultPairRequest()
	copReq.PairMethod = pb.PairMethod_COP
	pairtestutils.AddNDummyRounds(copReq, 3)
	pairtestutils.AddRoundResultsStr(copReq, "430 370 420 380 415 385 410 390")
	pairtestutils.AddRoundResultsStr(copReq, "370 430 380 420 385 415 390 410")
	pairtestutils.AddRoundResultsStr(copReq, "420 380 430 370 405 395 400 400")
	copReq.Seed = req.Seed

	copResp := COPPair(copReq)
	is.Equal(copResp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(swissResp.Pairings, copResp.Pairings)

	// Odd player count: exactly one bye, same as COP.
	req = pairtestutils.CreateDefaultOddPairRequest()
	req.PairMethod = pb.PairMethod_PAIR_SWISS
	swissResp = COPPair(req)
	is.Equal(swissResp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, swissResp.Pairings)
	is.Equal(countByes(swissResp.Pairings), 1)
}

func TestTeamRoundRobin(t *testing.T) {
	is := is.New(t)

	req := makeSimpleReq(pb.PairMethod_PAIR_TEAM_ROUND_ROBIN, 8, 10)
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	round0P0 := resp.Pairings[0]

	// After one matchup rotation player 0's opponent changes.
	pairtestutils.AddNDummyRounds(req, 1)
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	is.True(resp.Pairings[0] != round0P0)
}

func TestInterleavedRoundRobin(t *testing.T) {
	is := is.New(t)

	req := makeSimpleReq(pb.PairMethod_PAIR_INTERLEAVED_ROUND_ROBIN, 8, 10)
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)

	pairtestutils.AddNDummyRounds(req, 1)
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
}

func TestMultiroundPairings(t *testing.T) {
	is := is.New(t)
	numPlayers := 8

	// With no existing pairings and N=3, multiround_pairings should contain 3 rounds worth of data.
	req := makeSimpleReq(pb.PairMethod_PAIR_ROUND_ROBIN, numPlayers, 10)
	req.InitialNonperfRounds = 3
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), 3*numPlayers)
	// Each round's slice should be a valid symmetric pairing.
	for i := 0; i < 3; i++ {
		checkSymmetric(t, resp.MultiroundPairings[i*numPlayers:(i+1)*numPlayers])
	}
	// Round robin should produce distinct schedules across rounds.
	is.True(resp.MultiroundPairings[0] != resp.MultiroundPairings[numPlayers])

	// With existing pairings, multiround_pairings is a copy of pairings.
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), numPlayers)
	is.Equal(resp.MultiroundPairings, resp.Pairings)

	// RANDOM with N=2 should produce 2 rounds in multiround_pairings.
	req = makeSimpleReq(pb.PairMethod_PAIR_RANDOM, numPlayers, 10)
	req.InitialNonperfRounds = 2
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), 2*numPlayers)
	checkSymmetric(t, resp.MultiroundPairings[:numPlayers])
	checkSymmetric(t, resp.MultiroundPairings[numPlayers:])

	// Initial Fontes with N=3 should produce 3 rounds.
	req = makeSimpleReq(pb.PairMethod_PAIR_INITIAL_FONTES, numPlayers, 10)
	req.InitialNonperfRounds = 3
	resp = COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), 3*numPlayers)
	for i := 0; i < 3; i++ {
		checkSymmetric(t, resp.MultiroundPairings[i*numPlayers:(i+1)*numPlayers])
	}

	// AUTO with R=10, P=8: floor(10/7)*7=7 RR rounds, then COP for the remaining 3.
	autoReq := pairtestutils.CreateDefaultPairRequest()
	autoReq.PairMethod = pb.PairMethod_PAIR_AUTO
	rrRounds := int(autoReq.ValidPlayers) - 1                    // 7
	rrRoundsTotal := (int(autoReq.Rounds) / rrRounds) * rrRounds // 7
	resp = COPPair(autoReq)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), rrRoundsTotal*numPlayers)
	for i := 0; i < rrRoundsTotal; i++ {
		checkSymmetric(t, resp.MultiroundPairings[i*numPlayers:(i+1)*numPlayers])
	}

	// After the RR rounds are complete, auto should use COP for the remaining rounds.
	pairtestutils.AddNDummyRounds(autoReq, rrRoundsTotal)
	resp = COPPair(autoReq)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.Pairings), numPlayers)

	// AUTO with R=14, P=8: floor(14/7)*7=14 RR rounds, no COP remainder.
	autoReq = pairtestutils.CreateDefaultPairRequest()
	autoReq.PairMethod = pb.PairMethod_PAIR_AUTO
	autoReq.Rounds = 14
	rrRoundsTotal = (int(autoReq.Rounds) / rrRounds) * rrRounds // 14
	resp = COPPair(autoReq)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), rrRoundsTotal*numPlayers)
	for i := 0; i < rrRoundsTotal; i++ {
		checkSymmetric(t, resp.MultiroundPairings[i*numPlayers:(i+1)*numPlayers])
	}

	// AUTO with R=17, P=8: floor(17/7)*7=14 RR rounds, then COP for rounds 14-16.
	autoReq = pairtestutils.CreateDefaultPairRequest()
	autoReq.PairMethod = pb.PairMethod_PAIR_AUTO
	autoReq.Rounds = 17
	rrRoundsTotal = (int(autoReq.Rounds) / rrRounds) * rrRounds // 14
	resp = COPPair(autoReq)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), rrRoundsTotal*numPlayers)
	for i := 0; i < rrRoundsTotal; i++ {
		checkSymmetric(t, resp.MultiroundPairings[i*numPlayers:(i+1)*numPlayers])
	}
	pairtestutils.AddNDummyRounds(autoReq, rrRoundsTotal)
	resp = COPPair(autoReq)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.Pairings), numPlayers)

	// AUTO with R < P-1 should produce 3 initial-fontes rounds.
	autoReq = pairtestutils.CreateDefaultPairRequest()
	autoReq.PairMethod = pb.PairMethod_PAIR_AUTO
	autoReq.Rounds = int32(numPlayers - 2) // fewer than one full RR cycle
	resp = COPPair(autoReq)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), 3*numPlayers)
	for i := 0; i < 3; i++ {
		checkSymmetric(t, resp.MultiroundPairings[i*numPlayers:(i+1)*numPlayers])
	}

	// AUTO with 10 players and R=8 (< rrRounds=9): rounds 0-2 use Initial
	// Fontes, and round 3 onward now uses COP directly (Swiss was removed).
	autoNames := []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9"}
	autoReq10 := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_PAIR_AUTO,
		PlayerNames:                autoNames,
		PlayerClasses:              make([]int32, 10),
		ClassPrizes:                []int32{2},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 10,
		ValidPlayers:               10,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               1000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 8,
	}
	pairtestutils.AddNDummyRounds(autoReq10, 3)
	resp = COPPair(autoReq10)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "round 3 >= 3, using COP"))
	is.True(!strings.Contains(resp.Log, "using Swiss"))
}

func TestCOPProf(t *testing.T) {
	if os.Getenv("COP_PROF") == "" {
		t.Skip("Skipping COP profiling test. Use 'COP_PROF=1 go test -run COPProf' to run it and 'go tool pprof cop.prof' to analyze the results.")
	}

	is := is.New(t)
	f, err := os.Create("cop.prof")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	req := pairtestutils.CreateAlbanyAfterRound15PairRequest()
	is.Equal(verifyreq.Verify(req), nil)
	req.ControlLossActivationRound = 15
	req.DivisionSims = 1000000
	req.ControlLossSims = 1000000
	pprof.StartCPUProfile(f)

	start := time.Now() // Start timing
	resp := COPPair(req)
	elapsed := time.Since(start)                               // Calculate elapsed time
	fmt.Printf("COPPair took %v ms\n", elapsed.Milliseconds()) // Print elapsed time in ms

	pprof.StopCPUProfile()
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

func TestCOPTime(t *testing.T) {
	if os.Getenv("COP_TIME") == "" {
		t.Skip("Skipping COP profiling test. Use 'COP_TIME=1 go test -run COPTime' to run it.")
	}
	is := is.New(t)
	req := pairtestutils.CreateAlbanyAfterRound15PairRequest()
	req.ControlLossActivationRound = 15
	req.DivisionSims = 200000
	req.ControlLossSims = 200000
	is.Equal(verifyreq.Verify(req), nil)
	start := time.Now() // Start timing
	resp := COPPair(req)
	elapsed := time.Since(start)                               // Calculate elapsed time
	fmt.Printf("COPPair took %v ms\n", elapsed.Milliseconds()) // Print elapsed time in ms
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

func TestCOPDebug(t *testing.T) {
	if os.Getenv("COP_DEBUG") == "" {
		t.Skip("Skipping COP debug test. Use 'COP_DEBUG=1 go test -run COPDebug' to run it.")
	}
	is := is.New(t)
	req := pairtestutils.CreateBellevilleCSW4thCLAfterRound12PairRequest()
	req.ControlLossActivationRound = 11
	req.Seed = 1
	req.ControlLossSims = 1000000000
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_TIMEOUT)
}

// TestSimplePairFinalPairingsSkipsSelfPairedBye covers a display bug in
// simplePair's "Final Pairings" table: simplePairOnce represents a bye as
// the player paired with themselves (allPlayerPairings[pi] == pi), matching
// the self-pairing bye convention used elsewhere (e.g. standings.go), but
// the table-building loop only recognized negative opponent indices as a
// bye. A byed player fell through undetected and got displayed as if
// playing themselves. Initial Fontes with an odd player count always
// produces exactly one bye (via the addBye padding in
// getInitialFontesPairings), so this reliably exercises the bug.
func TestSimplePairFinalPairingsSkipsSelfPairedBye(t *testing.T) {
	is := is.New(t)
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_PAIR_INITIAL_FONTES,
		PlayerNames:                testPlayerNames(9),
		PlayerClasses:              make([]int32, 9),
		AllPlayers:                 9,
		ValidPlayers:               9,
		Rounds:                     8,
		PlacePrizes:                1,
		GibsonSpread:               200,
		HopefulnessThreshold:       0.02,
		DivisionSims:               1000,
		ControlLossSims:            1000,
		ControlLossActivationRound: 6,
		InitialNonperfRounds:       3,
		Seed:                       1,
	}
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	inFinalPairings := false
	for _, line := range strings.Split(resp.Log, "\n") {
		if strings.Contains(line, "Final Pairings") {
			inFinalPairings = true
			continue
		}
		if !inFinalPairings || !strings.Contains(line, "|") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 6 {
			continue
		}
		name1 := strings.TrimSpace(fields[0])
		name2 := strings.TrimSpace(fields[3])
		if name1 != "" && name1 != "Player" && name1 == name2 {
			t.Errorf("player shown playing themselves in Final Pairings (byed player not skipped): %s", line)
		}
	}
}

func testPlayerNames(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("P%d", i)
	}
	return names
}

// TestAdjustLowestPossibleHopeCasherForByeRetractsRedundantPromotion covers
// the double-promotion bug seen in round_22.log: GetPrecompData promotes a
// player to fix an odd raw hopeful-to-cash count, and then the definite bye
// recipient turns out to be a genuine (non-promoted) member of that group,
// which already restores parity on its own - the promotion should be
// retracted, not extended further.
func TestAdjustLowestPossibleHopeCasherForByeRetractsRedundantPromotion(t *testing.T) {
	is := is.New(t)

	playerNodes := []int{0, 1, 2, 3, 4, 5}
	numPlayers := len(playerNodes)
	req := &pb.PairRequest{PlayerNames: testPlayerNames(numPlayers)}

	// Raw contender count was 3 (ranks 0-2, odd); GetPrecompData promoted
	// rank 3 to fix parity, so the boundary going in is rank index 3.
	copdata := &copdatapkg.PrecompData{
		HopefulToCashPromotedPlayerRankIdx: 3,
		GibsonizedPlayers:                  make([]bool, numPlayers),
	}

	// Case 1: the bye recipient is a genuine contender (rank 1), ranked
	// above the promoted player (rank 3) - the promotion is now redundant
	// and should be retracted back to rank index 2.
	pargs := &policyArgs{
		req:                      req,
		copdata:                  copdata,
		lowestPossibleHopeCasher: 3,
		topDownByePlayer:         playerNodes[1],
		forcedContenderByePlayer: -1,
	}
	var logsb strings.Builder
	newBoundary := adjustLowestPossibleHopeCasherForBye(pargs, playerNodes, numPlayers, &logsb)
	is.Equal(newBoundary, 2)
	is.True(strings.Contains(logsb.String(), "retracting"))

	// Case 2: the bye recipient IS the artificially-promoted player (rank
	// 3) - that promotion no longer does anything useful, so extend past
	// it instead of retracting.
	pargs2 := &policyArgs{
		req:                      req,
		copdata:                  copdata,
		lowestPossibleHopeCasher: 3,
		topDownByePlayer:         playerNodes[3],
		forcedContenderByePlayer: -1,
	}
	var logsb2 strings.Builder
	newBoundary2 := adjustLowestPossibleHopeCasherForBye(pargs2, playerNodes, numPlayers, &logsb2)
	is.Equal(newBoundary2, 4)

	// Case 3: no promotion had fired (raw count already even), and the bye
	// recipient is inside the group - behaves exactly as before, extending
	// by one.
	copdataNoPromotion := &copdatapkg.PrecompData{
		HopefulToCashPromotedPlayerRankIdx: -1,
		GibsonizedPlayers:                  make([]bool, numPlayers),
	}
	pargs3 := &policyArgs{
		req:                      req,
		copdata:                  copdataNoPromotion,
		lowestPossibleHopeCasher: 2,
		topDownByePlayer:         playerNodes[1],
		forcedContenderByePlayer: -1,
	}
	var logsb3 strings.Builder
	newBoundary3 := adjustLowestPossibleHopeCasherForBye(pargs3, playerNodes, numPlayers, &logsb3)
	is.Equal(newBoundary3, 3)

	// Case 4: the bye recipient is Gibsonized (e.g. a Gibsonized leader
	// trivially counted as hopeful for a lower cash place too). Gibsonized
	// players are already excluded from the parity-relevant contender count
	// (see GetPrecompData and PC/CC), so their removal via a bye - even
	// though their rank falls within the boundary - needs no adjustment.
	copdataGibsonBye := &copdatapkg.PrecompData{
		HopefulToCashPromotedPlayerRankIdx: -1,
		GibsonizedPlayers:                  []bool{true, false, false, false, false, false},
	}
	pargs4 := &policyArgs{
		req:                      req,
		copdata:                  copdataGibsonBye,
		lowestPossibleHopeCasher: 3,
		topDownByePlayer:         playerNodes[0],
		forcedContenderByePlayer: -1,
	}
	var logsb4 strings.Builder
	newBoundary4 := adjustLowestPossibleHopeCasherForBye(pargs4, playerNodes, numPlayers, &logsb4)
	is.Equal(newBoundary4, 3)
	is.Equal(logsb4.String(), "")
}

// TestComputeDisallowedLeaderOpponentByeAware covers the analogous bug for
// the hopeful-for-1st contender group: a bye landing inside an odd raw
// group already restores parity, so no extra player should be pulled in.
func TestComputeDisallowedLeaderOpponentByeAware(t *testing.T) {
	is := is.New(t)

	playerNodes := []int{0, 1, 2, 3, 4}
	copdata := &copdatapkg.PrecompData{
		LowestPossibleHopeNth: []int{2}, // group is ranks 0-2 (size 3, odd)
		GibsonizedPlayers:     []bool{false, false, false, false, false},
	}

	// Without a bye, the odd group of 3 pulls in the next player (rank 3)
	// as the barred/disallowed opponent - unchanged from before.
	pargsNoBye := &policyArgs{
		playerNodes:              playerNodes,
		copdata:                  copdata,
		topDownByePlayer:         -1,
		forcedContenderByePlayer: -1,
		forcedLeaderVsThird:      -1,
	}
	is.Equal(computeDisallowedLeaderOpponent(pargsNoBye), playerNodes[3])

	// With a bye landing on rank 2 (inside the group), the group's pairable
	// size is already even (2), so no extra player should be pulled in.
	pargsWithBye := &policyArgs{
		playerNodes:              playerNodes,
		copdata:                  copdata,
		topDownByePlayer:         playerNodes[2],
		forcedContenderByePlayer: -1,
		forcedLeaderVsThird:      -1,
	}
	is.Equal(computeDisallowedLeaderOpponent(pargsWithBye), -1)
}

// TestComputeDisallowedLeaderOpponentGibsonizedLeader covers the analogous
// bug to the hopeful-to-cash boundary's Gibson exclusion: GibsonizedPlayers[0]
// uniquely means "guaranteed to finish 1st", so a Gibsonized leader has
// already settled the race and shouldn't count toward the hopeful-for-1st
// group's pairable parity - unlike the cash boundary's blanket exclusion, a
// Gibson lock at any rank other than 0 doesn't resolve the 1st-place race
// and must not be excluded.
func TestComputeDisallowedLeaderOpponentGibsonizedLeader(t *testing.T) {
	is := is.New(t)

	playerNodes := []int{0, 1, 2, 3, 4, 5}

	// Raw group of 4 (ranks 0-3, even) with a Gibsonized leader: excluding
	// the leader leaves 3 genuine contenders (odd), so the next player
	// (rank 4) should be pulled in and barred - previously this returned -1
	// because the raw (Gibson-blind) count looked even.
	evenRawGibsonLeader := &policyArgs{
		playerNodes: playerNodes,
		copdata: &copdatapkg.PrecompData{
			LowestPossibleHopeNth: []int{3},
			GibsonizedPlayers:     []bool{true, false, false, false, false, false},
		},
		topDownByePlayer:         -1,
		forcedContenderByePlayer: -1,
		forcedLeaderVsThird:      -1,
	}
	is.Equal(computeDisallowedLeaderOpponent(evenRawGibsonLeader), playerNodes[4])

	// Raw group of 3 (ranks 0-2, odd) with a Gibsonized leader: excluding
	// the leader leaves 2 genuine contenders (even), so no extra player
	// should be pulled in - previously this wrongly promoted rank 3 because
	// the raw (Gibson-blind) count looked odd.
	oddRawGibsonLeader := &policyArgs{
		playerNodes: playerNodes,
		copdata: &copdatapkg.PrecompData{
			LowestPossibleHopeNth: []int{2},
			GibsonizedPlayers:     []bool{true, false, false, false, false, false},
		},
		topDownByePlayer:         -1,
		forcedContenderByePlayer: -1,
		forcedLeaderVsThird:      -1,
	}
	is.Equal(computeDisallowedLeaderOpponent(oddRawGibsonLeader), -1)

	// Raw group of exactly 1 (just the Gibsonized leader): preserved
	// behavior - still bars the next player (rank 1) from playing the
	// settled leader, to protect a genuine 2nd/3rd-place race. Naively
	// excluding the leader here would make this a no-op (0, even).
	soloGibsonLeader := &policyArgs{
		playerNodes: playerNodes,
		copdata: &copdatapkg.PrecompData{
			LowestPossibleHopeNth: []int{0},
			GibsonizedPlayers:     []bool{true, false, false, false, false, false},
		},
		topDownByePlayer:         -1,
		forcedContenderByePlayer: -1,
		forcedLeaderVsThird:      -1,
	}
	is.Equal(computeDisallowedLeaderOpponent(soloGibsonLeader), playerNodes[1])

	// A non-leader Gibsonized player (rank 2, locked at rank 2-or-better)
	// must NOT be excluded from the count: they're still a genuine
	// contender for 1st. Raw group of 4 (even) stays even, so no extra
	// player is pulled in - same as if nobody were Gibsonized.
	nonLeaderGibsonized := &policyArgs{
		playerNodes: playerNodes,
		copdata: &copdatapkg.PrecompData{
			LowestPossibleHopeNth: []int{3},
			GibsonizedPlayers:     []bool{false, false, true, false, false, false},
		},
		topDownByePlayer:         -1,
		forcedContenderByePlayer: -1,
		forcedLeaderVsThird:      -1,
	}
	is.Equal(computeDisallowedLeaderOpponent(nonLeaderGibsonized), -1)
}

// TestComputeTop4LockActive covers the top-4-must-play-each-other policy:
// with 2 rounds remaining, Factor 3 not applying, and exactly 4 hopeful
// contenders for 1st (LowestPossibleHopeNth[0] == 3), the policy should
// activate - and should stay off whenever any of those conditions doesn't
// hold, or when the bye lands inside the top 4 and leaves only 3 of them to
// pair off.
func TestComputeTop4LockActive(t *testing.T) {
	is := is.New(t)

	playerNodes := []int{0, 1, 2, 3, 4, 5, 6}
	baseCopdata := &copdatapkg.PrecompData{LowestPossibleHopeNth: []int{3}}

	// All conditions met: activates.
	is.True(computeTop4LockActive(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  baseCopdata,
		roundsRemaining:          2,
		topDownByePlayer:         -1,
		forcedContenderByePlayer: -1,
	}))

	// Factor 3 already fired: defers to it, doesn't also activate.
	is.True(!computeTop4LockActive(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  baseCopdata,
		roundsRemaining:          2,
		factor3ForcedPairings:    [][2]int{{0, 3}},
		topDownByePlayer:         -1,
		forcedContenderByePlayer: -1,
	}))

	// Not the 2nd-to-last round: doesn't activate.
	is.True(!computeTop4LockActive(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  baseCopdata,
		roundsRemaining:          3,
		topDownByePlayer:         -1,
		forcedContenderByePlayer: -1,
	}))

	// Group of hopeful-for-1st contenders isn't exactly 4 (here, 3): doesn't activate.
	is.True(!computeTop4LockActive(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  &copdatapkg.PrecompData{LowestPossibleHopeNth: []int{2}},
		roundsRemaining:          2,
		topDownByePlayer:         -1,
		forcedContenderByePlayer: -1,
	}))

	// The bye lands on one of the top 4 (rank 3), leaving only 3 of them to
	// pair off: doesn't activate.
	is.True(!computeTop4LockActive(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  baseCopdata,
		roundsRemaining:          2,
		topDownByePlayer:         playerNodes[3],
		forcedContenderByePlayer: -1,
	}))

	// The bye lands outside the top 4 (rank 4): still activates.
	is.True(computeTop4LockActive(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  baseCopdata,
		roundsRemaining:          2,
		topDownByePlayer:         playerNodes[4],
		forcedContenderByePlayer: -1,
	}))
}

// TestTop4LockConstraint verifies the "T4" constraint policy itself: once
// top4LockActive is set, it disallows every pairing between one of the top 4
// (ranks 0-3) and anyone outside that group, without forcing any specific
// matchup among the four.
func TestTop4LockConstraint(t *testing.T) {
	is := is.New(t)

	var t4Policy constraintPolicy
	for _, cp := range constraintPolicies {
		if cp.name == "T4" {
			t4Policy = cp
		}
	}
	is.True(t4Policy.handler != nil)

	playerNodes := []int{10, 11, 12, 13, 14, 15, 16}

	forced, disallowed := t4Policy.handler(&policyArgs{playerNodes: playerNodes, top4LockActive: false})
	is.Equal(len(forced), 0)
	is.Equal(len(disallowed), 0)

	forced, disallowed = t4Policy.handler(&policyArgs{playerNodes: playerNodes, top4LockActive: true})
	is.Equal(len(forced), 0)
	wantDisallowed := map[[2]int]bool{}
	for pri := 0; pri <= 3; pri++ {
		for prj := 4; prj < len(playerNodes); prj++ {
			wantDisallowed[[2]int{playerNodes[pri], playerNodes[prj]}] = true
		}
	}
	is.Equal(len(disallowed), len(wantDisallowed))
	for _, dp := range disallowed {
		is.True(wantDisallowed[dp])
	}
}

// TestComputeForcedLeaderVsThird covers the size-2 exception to the
// odd-hopeful-contender-group rule: when the leader and exactly one other
// player are hopeful for 1st, and that other player is receiving this
// round's bye, the leader is forced onto 3rd instead of being left without
// a hopeful opponent.
func TestComputeForcedLeaderVsThird(t *testing.T) {
	is := is.New(t)

	playerNodes := []int{0, 1, 2, 3, 4}
	baseCopdata := &copdatapkg.PrecompData{
		LowestPossibleHopeNth: []int{1}, // group is ranks 0-1 (size 2)
		GibsonizedPlayers:     []bool{false, false, false, false, false},
	}

	// Rank 1 (the only other hopeful-for-1st player) receives the top-down
	// bye: forces the leader onto rank 2.
	is.Equal(computeForcedLeaderVsThird(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  baseCopdata,
		topDownByePlayer:         playerNodes[1],
		forcedContenderByePlayer: -1,
	}), playerNodes[2])

	// Same, but via the forced contender bye instead of the top-down bye.
	is.Equal(computeForcedLeaderVsThird(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  baseCopdata,
		topDownByePlayer:         -1,
		forcedContenderByePlayer: playerNodes[1],
	}), playerNodes[2])

	// No bye at all this round: doesn't activate.
	is.Equal(computeForcedLeaderVsThird(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  baseCopdata,
		topDownByePlayer:         -1,
		forcedContenderByePlayer: -1,
	}), -1)

	// The bye lands somewhere other than rank 1: doesn't activate.
	is.Equal(computeForcedLeaderVsThird(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  baseCopdata,
		topDownByePlayer:         playerNodes[2],
		forcedContenderByePlayer: -1,
	}), -1)

	// Exactly 3 players hopeful for 1st (rank 3 already in the group):
	// still activates - rank 2's bye leaves the leader without their
	// strongest hopeful opponent regardless.
	is.Equal(computeForcedLeaderVsThird(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  &copdatapkg.PrecompData{LowestPossibleHopeNth: []int{2}, GibsonizedPlayers: baseCopdata.GibsonizedPlayers},
		topDownByePlayer:         playerNodes[1],
		forcedContenderByePlayer: -1,
	}), playerNodes[2])

	// More than 3 players hopeful for 1st: doesn't activate, defers to the
	// general odd-hopeful-contender-group rule instead.
	is.Equal(computeForcedLeaderVsThird(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  &copdatapkg.PrecompData{LowestPossibleHopeNth: []int{3}, GibsonizedPlayers: baseCopdata.GibsonizedPlayers},
		topDownByePlayer:         playerNodes[1],
		forcedContenderByePlayer: -1,
	}), -1)

	// Only the leader is hopeful for 1st (size 1): still activates - rank 2
	// isn't even in the hopeful group, but their bye still leaves the
	// leader without their strongest available opponent.
	is.Equal(computeForcedLeaderVsThird(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  &copdatapkg.PrecompData{LowestPossibleHopeNth: []int{0}, GibsonizedPlayers: baseCopdata.GibsonizedPlayers},
		topDownByePlayer:         playerNodes[1],
		forcedContenderByePlayer: -1,
	}), playerNodes[2])

	// Gibsonized leader: the race is already settled, so this defers to the
	// general rule rather than treating rank 1's bye as leaving the leader
	// without a hopeful opponent.
	is.Equal(computeForcedLeaderVsThird(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  &copdatapkg.PrecompData{LowestPossibleHopeNth: []int{1}, GibsonizedPlayers: []bool{true, false, false, false, false}},
		topDownByePlayer:         playerNodes[1],
		forcedContenderByePlayer: -1,
	}), -1)

	// Factor 3 already fired: defers to it entirely.
	is.Equal(computeForcedLeaderVsThird(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  baseCopdata,
		topDownByePlayer:         playerNodes[1],
		forcedContenderByePlayer: -1,
		factor3ForcedPairings:    [][2]int{{0, 2}},
	}), -1)

	// No rank 2 to promote (too few players): doesn't activate.
	shortPlayerNodes := []int{0, 1}
	is.Equal(computeForcedLeaderVsThird(&policyArgs{
		playerNodes:              shortPlayerNodes,
		copdata:                  baseCopdata,
		topDownByePlayer:         playerNodes[1],
		forcedContenderByePlayer: -1,
	}), -1)
}

// TestComputeDisallowedLeaderOpponentDefersToForcedLeaderVsThird verifies
// that computeDisallowedLeaderOpponent steps aside entirely - rather than
// also barring the same edge - whenever computeForcedLeaderVsThird's
// size-2 exception applies.
func TestComputeDisallowedLeaderOpponentDefersToForcedLeaderVsThird(t *testing.T) {
	is := is.New(t)

	playerNodes := []int{0, 1, 2, 3, 4}
	copdata := &copdatapkg.PrecompData{
		LowestPossibleHopeNth: []int{1},
		GibsonizedPlayers:     []bool{false, false, false, false, false},
	}
	is.Equal(computeDisallowedLeaderOpponent(&policyArgs{
		playerNodes:              playerNodes,
		copdata:                  copdata,
		topDownByePlayer:         playerNodes[1],
		forcedContenderByePlayer: -1,
		forcedLeaderVsThird:      playerNodes[2],
	}), -1)
}

// TestForcedLeaderVsThirdConstraint verifies the "L3" constraint policy
// itself: once forcedLeaderVsThird is set, it forces exactly the leader-vs-3rd
// pairing and nothing else.
func TestForcedLeaderVsThirdConstraint(t *testing.T) {
	is := is.New(t)

	var l3Policy constraintPolicy
	for _, cp := range constraintPolicies {
		if cp.name == "L3" {
			l3Policy = cp
		}
	}
	is.True(l3Policy.handler != nil)

	playerNodes := []int{10, 11, 12, 13, 14}

	forced, disallowed := l3Policy.handler(&policyArgs{playerNodes: playerNodes, forcedLeaderVsThird: -1})
	is.Equal(len(forced), 0)
	is.Equal(len(disallowed), 0)

	forced, disallowed = l3Policy.handler(&policyArgs{playerNodes: playerNodes, forcedLeaderVsThird: playerNodes[2]})
	is.Equal(len(disallowed), 0)
	is.Equal(len(forced), 1)
	is.Equal(forced[0], [2]int{playerNodes[0], playerNodes[2]})
}

// TestCCWeightExemptsForcedLeaderVsThird verifies the CC weight policy
// returns 0 - not a major penalty - for the edge that L3 forces, even
// though it would otherwise fall inside the fourth-quarter hopeful-vs-
// non-hopeful cash-contention major penalty.
func TestCCWeightExemptsForcedLeaderVsThird(t *testing.T) {
	is := is.New(t)

	var ccPolicy weightPolicy
	for _, wp := range weightPolicies {
		if wp.name == "CC" {
			ccPolicy = wp
		}
	}
	is.True(ccPolicy.handler != nil)

	playerNodes := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	pargs := &policyArgs{
		playerNodes:              playerNodes,
		copdata:                  &copdatapkg.PrecompData{GibsonizedPlayers: make([]bool, len(playerNodes))},
		req:                      &pb.PairRequest{Rounds: 10},
		roundPairingsRemaining:   1, // last quarter
		disallowedLeaderOpponent: -1,
		forcedLeaderVsThird:      playerNodes[2],
	}
	is.Equal(ccPolicy.handler(pargs, 0, 2), int64(0))
}
