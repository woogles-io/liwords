package cop_test

import (
	"fmt"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	"github.com/woogles-io/liwords/pkg/pair/verifyreq"

	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestCOPErrors(t *testing.T) {
	is := is.New(t)

	req := pairtestutils.CreateDefaultPairRequest()
	req.ValidPlayers = -1
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_COUNT_INSUFFICIENT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ValidPlayers = 0
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_ALL_ROUNDS_PAIRED)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: []int32{4, 5, 6, 7}})
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_ROUND_PAIRINGS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 20 7 0 1 2 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_INDEX_OUT_OF_BOUNDS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 -6 7 0 1 2 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_PLAYER_INDEX_OUT_OF_BOUNDS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundResultsAndPairingsStr(req, "4 300 5 250 6 400 7 500 0 400 1 300 2 425 3 200")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 -1 0 1 2 3")
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_UNPAIRED_PLAYER)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 -1 3")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PAIRING)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 1 3")
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_RESULTS_THAN_ROUNDS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 425 200 500")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_MORE_RESULTS_THAN_PAIRINGS)

	req = pairtestutils.CreateDefaultPairRequest()
	pairtestutils.AddRoundPairingsStr(req, "4 5 6 7 0 1 2 3")
	pairtestutils.AddRoundResultsStr(req, "400 300 250 400 300 500")
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_ROUND_RESULTS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{0, 0, 0, 0, 0, 0, 0}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{0}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS_COUNT)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{0, 0, 0, 0, 0, 0, 0, -1}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.PlayerClasses = []int32{0, 0, 0, 0, 0, 0, 0, 2}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_PLAYER_CLASS)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ClassPrizes = []int32{-1}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS_PRIZE)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ClassPrizes = []int32{0}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS_PRIZE)

	req = pairtestutils.CreateDefaultPairRequest()
	req.ClassPrizes = []int32{-1}
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CLASS_PRIZE)

	req = pairtestutils.CreateDefaultPairRequest()
	req.GibsonSpread = -100
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
	req.HopefulnessThreshold = 0
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

	req = pairtestutils.CreateDefaultPairRequest()
	req.ControlLossActivationRound = -1
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_INVALID_CONTROL_LOSS_ACTIVATION_ROUND)
}

func TestCOPConstraintPolicies(t *testing.T) {
	is := is.New(t)

	// Prepaired players
	req := pairtestutils.CreateBellevilleCSWAfterRound12PairRequest()
	req.Seed = 1
	pairtestutils.AddRoundPairingsStr(req, "-1 -1 -1 10 -1 -1 -1 -1 -1 11 3 9")
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[3], int32(10))
	is.Equal(resp.Pairings[9], int32(11))
	is.Equal(resp.Pairings[10], int32(3))
	is.Equal(resp.Pairings[11], int32(9))

	req = pairtestutils.CreateAlbany3rdGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	pairtestutils.AddRoundPairingsStr(req, "-1 -1 -1 14 -1 -1 -1 -1 -1 -1 -1 -1 -1 -1 3 -1 -1 -1 -1 -1 21 20 -1 -1")
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
	is.Equal(resp.Pairings[0], int32(3))
	is.Equal(resp.Pairings[3], int32(0))

	// Control loss with player in 4th
	req = pairtestutils.CreateBellevilleCSW4thCLAfterRound12PairRequest()
	req.ControlLossActivationRound = 11
	req.Seed = 1
	resp = cop.COPPair(req)
	// The control loss should force 1st to play either 2nd or 3rd since 4th
	// isn't hopeful enough.
	fmt.Println(resp.Log)
	is.True(resp.Pairings[3] == int32(2) || resp.Pairings[3] == int32(0))

	// Control loss with player in 4th
	req = pairtestutils.CreateBellevilleCSW4thCLAfterRound12PairRequest()
	req.ControlLossActivationRound = 11
	req.Seed = 1
	req.HopefulnessThreshold = 0.01
	resp = cop.COPPair(req)
	// The control loss should force 1st to play either 3rd or 4th, and
	// in this case should play 3rd because of repeats and other considerations
	is.Equal(resp.Pairings[4], int32(3))
	is.Equal(resp.Pairings[3], int32(4))

	// Gibson groups and Gibson Bye
	req = pairtestutils.CreateAlbany1stAnd4thAnd8thGibsonizedAfterRound25PairRequest()
	req.Seed = 1
	resp = cop.COPPair(req)
	is.Equal(resp.Pairings[0], int32(4))
	is.Equal(resp.Pairings[4], int32(0))
	is.Equal(resp.Pairings[1], int32(1))
	is.Equal(resp.Pairings[11], int32(-1))
	resp.Pairings[11] = 11
	pairtestutils.AddRoundPairings(req, resp.Pairings)
	resp = cop.COPPair(req)
	is.Equal(resp.Pairings[0], int32(4))
	is.Equal(resp.Pairings[4], int32(0))
	is.Equal(resp.Pairings[1], int32(1))

	// Gibson Bye
	req = pairtestutils.CreateAlbanyCSWAfterRound24OddPairRequest()
	is.Equal(verifyreq.Verify(req), nil)
	req.Seed = 1
	resp = cop.COPPair(req)
	is.Equal(resp.Pairings[10], int32(10))
	pairtestutils.AddRoundPairings(req, resp.Pairings)
	resp = cop.COPPair(req)
	is.Equal(resp.Pairings[10], int32(10))

	req = pairtestutils.CreateLakeGeorgeAfterRound13PairRequest()
	is.Equal(verifyreq.Verify(req), nil)

	// This is the first round that control loss is active, so first will
	// be force paired with the lowest contender. Therefore, prepairing the lowest
	// contender with someone else will result in an overconstrained error.
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
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_OVERCONSTRAINED)

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
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Ben should be playing Wellington, the 2nd lowest contender
	is.Equal(resp.Pairings[0], int32(10))
	is.Equal(resp.Pairings[10], int32(0))

	// With only 4 rounds to go, first will again be paired with only the lowest contender.
	// Therefore, prepairing the lowest contender with someone else should result in an overconstrained error.
	req = pairtestutils.CreateAlbanyjuly4th2024AfterRound21PairRequest()
	req.Rounds = 25
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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_OVERCONSTRAINED)

	// Check that top down byes work
	req = pairtestutils.CreateAlbanyAfterRound16PairRequest()
	// The last pairings array in this request prepairs certain players, which
	// we need to remove for this test.
	req.DivisionPairings = req.DivisionPairings[:len(req.DivisionPairings)-1]
	req.TopDownByes = true
	req.AllowRepeatByes = false
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Zach should have the bye
	is.Equal(resp.Pairings[4], int32(4))

	// Run another round
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: resp.Pairings,
	})
	req.DivisionPairings[len(req.DivisionPairings)-1].Pairings[11] = 11
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Andy should have the bye
	is.Equal(resp.Pairings[2], int32(2))

	// Run another round
	req.DivisionPairings = append(req.DivisionPairings, &pb.RoundPairings{
		Pairings: resp.Pairings,
	})
	req.DivisionPairings[len(req.DivisionPairings)-1].Pairings[11] = 11
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Billy should get the bye since Eric already received a bye
	is.Equal(resp.Pairings[5], int32(5))

	// FIXME: check that timeouts work
}

func TestCOPWeights(t *testing.T) {
	is := is.New(t)

	req := pairtestutils.CreateBLSRound32PairRequest()
	req.Seed = 0
	is.Equal(verifyreq.Verify(req), nil)
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// Matt T should be playing Michael F, since rank differences
	// for pairings with a gibsonized player are not cubed.
	is.Equal(resp.Pairings[9], int32(46))
	is.Equal(resp.Pairings[46], int32(9))
}

func TestCOPSuccess(t *testing.T) {
	is := is.New(t)

	req := pairtestutils.CreateDefaultPairRequest()
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

func TestCOPProdBugs(t *testing.T) {
	// These are all tests for requests that created unexpected behavior
	// in prod
	is := is.New(t)

	// Test players prepaired with byes
	req := pairtestutils.CreateAlbanyAfterRound16PairRequest()
	req.Seed = 1
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[0], int32(22))
	is.Equal(resp.Pairings[11], int32(-1))
	is.Equal(resp.Pairings[19], int32(19))

	// Test that back-to-back pairings are penalized correctly
	req = pairtestutils.CreateAlbanyCSWNewYearsAfterRound27PairRequest()
	req.Seed = 1
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	// There are pairings for all rounds, but the last round is only partially
	// paired, so this should finish successfully
	req = pairtestutils.CreateAlbanyCSWNewYearsAfterRound27LastRoundPartiallyPairedPairRequest()
	req.Seed = 1
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	req = pairtestutils.CreateAlbanyCSWNewYearsRound25PartiallyPairedPairRequest()
	req.Seed = 1
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

	req = pairtestutils.CreateAlmostGibsonizedPairRequest()
	req.Seed = 1
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// whatnoloan is not gibsonized and is the only player who can hopefully win
	// Therefore, whatnoloan needs to play condorave since condorave is the player ranked just below whatnoloan
	is.Equal(resp.Pairings[1], int32(3))
	is.Equal(resp.Pairings[3], int32(1))
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
	resp := cop.COPPair(req)
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
	resp := cop.COPPair(req)
	elapsed := time.Since(start)                               // Calculate elapsed time
	fmt.Printf("COPPair took %v ms\n", elapsed.Milliseconds()) // Print elapsed time in ms
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
}

func TestCOPDebug(t *testing.T) {
	if os.Getenv("COP_DEBUG") == "" {
		t.Skip("Skipping COP debug test. Use 'COP_DEBUG=1 go test -run COPDebug' to run it.")
	}
	is := is.New(t)
	req := pairtestutils.CreateAlmostGibsonizedPairRequest()
	req.Seed = 1
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// whatnoloan is not gibsonized and is the only player who can hopefully win
	// Therefore, whatnoloan needs to play condorave since condorave is the player ranked just below whatnoloan
	is.Equal(resp.Pairings[1], int32(3))
	is.Equal(resp.Pairings[3], int32(1))
}
