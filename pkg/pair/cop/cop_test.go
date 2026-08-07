package cop_test

import (
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
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
	is.True(resp.Pairings[3] == int32(2) || resp.Pairings[3] == int32(0))

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
	is.Equal(resp.Pairings[2], int32(2))

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
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
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
	req.DivisionSims = 5000
	req.ControlLossSims = 1000
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
}

// TestTopDownByePrecedence checks that top-down byes take precedence over
// KOTH-forced pairings (cash prize or class prize) in the final round. Before
// the fix, KH could force a pairing for the same player TB independently
// selected for the bye, which disallowed the bye pairing TB needed and the
// pairing KH needed on the same edge, producing OVERCONSTRAINED.
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

	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// P0 (rank 2, 0 prior byes) receives the top-down bye, even though cash
	// prize KOTH would otherwise force it to play P8 (rank 1).
	is.Equal(resp.Pairings[0], int32(0))
	// The rest of the field still gets KOTH-appropriate pairings.
	is.Equal(resp.Pairings[8], int32(5))
	is.Equal(resp.Pairings[5], int32(8))
	is.Equal(resp.Pairings[2], int32(4))
	is.Equal(resp.Pairings[4], int32(2))
	is.Equal(resp.Pairings[6], int32(7))
	is.Equal(resp.Pairings[7], int32(6))
	is.Equal(resp.Pairings[3], int32(1))
	is.Equal(resp.Pairings[1], int32(3))
}

func TestCOPWeights(t *testing.T) {
	is := is.New(t)

	req := pairtestutils.CreateBLSRound32PairRequest()
	req.Seed = 0
	is.Equal(verifyreq.Verify(req), nil)
	resp := cop.COPPair(req)
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
	// AlmostGibsonized Q4: player 4 pairs with player 7 (non-majorPenalty, no retry).
	// Player 5 pairs with player 9 (within contender range, no retry). Player 0 pairs
	// with player 18 via the retry: the first-pass edge (rank0, rank18) has weight
	// >= majorPenalty, which expands LowestPossibleHopeNth for those players, and the
	// second pass selects the same edge with a lower weight.
	req = pairtestutils.CreateAlmostGibsonizedPairRequest()
	req.Seed = 1
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	// whatnoloan and condorave still play (unchanged)
	is.Equal(resp.Pairings[1], int32(3))
	is.Equal(resp.Pairings[3], int32(1))
	// In the fourth quarter, RD is suppressed for cashers; PC weight drives player 4
	// to its nearest LowestPossibleHopeNth opponent (non-majorPenalty, no retry).
	is.Equal(resp.Pairings[4], int32(7))
	is.Equal(resp.Pairings[7], int32(4))
	// Player 5 pairs within contender range (no retry).
	is.Equal(resp.Pairings[5], int32(9))
	is.Equal(resp.Pairings[14], int32(13))

	// Kingston 2023 after round 15: player 0's first-pass pairing is within the
	// contender range (non-majorPenalty), so no retry is triggered.
	req = pairtestutils.CreateKingston2023AfterRound15PairRequest()
	req.Seed = 1
	resp = cop.COPPair(req)
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
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[4], int32(9))
	is.Equal(resp.Pairings[10], int32(0))
	is.Equal(resp.Pairings[13], int32(17))
	is.Equal(resp.Pairings[17], int32(13))

	// In Q4 (Rounds=10), non-casher players pair by RD only.
	req = pairtestutils.CreateAlmostGibsonizedPairRequest()
	req.Seed = 1
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(resp.Pairings[10], int32(17))
	is.Equal(resp.Pairings[13], int32(14))
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
	req.ControlLossSims = 1000
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

	req = pairtestutils.CreateLG2025Round15PairRequest()
	resp = cop.COPPair(req)
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
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	is.Equal(countByes(resp.Pairings), 0)

	// Odd player count: exactly one bye.
	req = makeSimpleReq(pb.PairMethod_PAIR_RANDOM, 7, 10)
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	is.Equal(countByes(resp.Pairings), 1)
}

func TestRoundRobin(t *testing.T) {
	is := is.New(t)

	// Round 0 produces valid pairings.
	req := makeSimpleReq(pb.PairMethod_PAIR_ROUND_ROBIN, 8, 10)
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	round0P0 := resp.Pairings[0]

	// Round 1 rotates the schedule so player 0's opponent differs.
	pairtestutils.AddNDummyRounds(req, 1)
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	is.True(resp.Pairings[0] != round0P0)

	// Odd player count: exactly one bye per round.
	req = makeSimpleReq(pb.PairMethod_PAIR_ROUND_ROBIN, 7, 10)
	resp = cop.COPPair(req)
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
		resp = cop.COPPair(req)
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
		resp = cop.COPPair(req2)
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
	resp := cop.COPPair(req)
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
	resp := cop.COPPair(req)
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
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)

	// Round 1 also valid (group members play their return match).
	pairtestutils.AddNDummyRounds(req, 1)
	resp = cop.COPPair(req)
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

	swissResp := cop.COPPair(req)
	is.Equal(swissResp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, swissResp.Pairings)

	copReq := pairtestutils.CreateDefaultPairRequest()
	copReq.PairMethod = pb.PairMethod_COP
	pairtestutils.AddNDummyRounds(copReq, 3)
	pairtestutils.AddRoundResultsStr(copReq, "430 370 420 380 415 385 410 390")
	pairtestutils.AddRoundResultsStr(copReq, "370 430 380 420 385 415 390 410")
	pairtestutils.AddRoundResultsStr(copReq, "420 380 430 370 405 395 400 400")
	copReq.Seed = req.Seed

	copResp := cop.COPPair(copReq)
	is.Equal(copResp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(swissResp.Pairings, copResp.Pairings)

	// Odd player count: exactly one bye, same as COP.
	req = pairtestutils.CreateDefaultOddPairRequest()
	req.PairMethod = pb.PairMethod_PAIR_SWISS
	swissResp = cop.COPPair(req)
	is.Equal(swissResp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, swissResp.Pairings)
	is.Equal(countByes(swissResp.Pairings), 1)
}

func TestTeamRoundRobin(t *testing.T) {
	is := is.New(t)

	req := makeSimpleReq(pb.PairMethod_PAIR_TEAM_ROUND_ROBIN, 8, 10)
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	round0P0 := resp.Pairings[0]

	// After one matchup rotation player 0's opponent changes.
	pairtestutils.AddNDummyRounds(req, 1)
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
	is.True(resp.Pairings[0] != round0P0)
}

func TestInterleavedRoundRobin(t *testing.T) {
	is := is.New(t)

	req := makeSimpleReq(pb.PairMethod_PAIR_INTERLEAVED_ROUND_ROBIN, 8, 10)
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)

	pairtestutils.AddNDummyRounds(req, 1)
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	checkSymmetric(t, resp.Pairings)
}

func TestMultiroundPairings(t *testing.T) {
	is := is.New(t)
	numPlayers := 8

	// With no existing pairings and N=3, multiround_pairings should contain 3 rounds worth of data.
	req := makeSimpleReq(pb.PairMethod_PAIR_ROUND_ROBIN, numPlayers, 10)
	req.InitialNonperfRounds = 3
	resp := cop.COPPair(req)
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
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), numPlayers)
	is.Equal(resp.MultiroundPairings, resp.Pairings)

	// RANDOM with N=2 should produce 2 rounds in multiround_pairings.
	req = makeSimpleReq(pb.PairMethod_PAIR_RANDOM, numPlayers, 10)
	req.InitialNonperfRounds = 2
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), 2*numPlayers)
	checkSymmetric(t, resp.MultiroundPairings[:numPlayers])
	checkSymmetric(t, resp.MultiroundPairings[numPlayers:])

	// Initial Fontes with N=3 should produce 3 rounds.
	req = makeSimpleReq(pb.PairMethod_PAIR_INITIAL_FONTES, numPlayers, 10)
	req.InitialNonperfRounds = 3
	resp = cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), 3*numPlayers)
	for i := 0; i < 3; i++ {
		checkSymmetric(t, resp.MultiroundPairings[i*numPlayers:(i+1)*numPlayers])
	}

	// AUTO with R=10, P=8: floor(10/7)*7=7 RR rounds, then COP for the remaining 3.
	autoReq := pairtestutils.CreateDefaultPairRequest()
	autoReq.PairMethod = pb.PairMethod_PAIR_AUTO
	rrRounds := int(autoReq.ValidPlayers) - 1 // 7
	rrRoundsTotal := (int(autoReq.Rounds) / rrRounds) * rrRounds // 7
	resp = cop.COPPair(autoReq)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), rrRoundsTotal*numPlayers)
	for i := 0; i < rrRoundsTotal; i++ {
		checkSymmetric(t, resp.MultiroundPairings[i*numPlayers:(i+1)*numPlayers])
	}

	// After the RR rounds are complete, auto should use COP for the remaining rounds.
	pairtestutils.AddNDummyRounds(autoReq, rrRoundsTotal)
	resp = cop.COPPair(autoReq)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.Pairings), numPlayers)

	// AUTO with R=14, P=8: floor(14/7)*7=14 RR rounds, no COP remainder.
	autoReq = pairtestutils.CreateDefaultPairRequest()
	autoReq.PairMethod = pb.PairMethod_PAIR_AUTO
	autoReq.Rounds = 14
	rrRoundsTotal = (int(autoReq.Rounds) / rrRounds) * rrRounds // 14
	resp = cop.COPPair(autoReq)
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
	resp = cop.COPPair(autoReq)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.MultiroundPairings), rrRoundsTotal*numPlayers)
	for i := 0; i < rrRoundsTotal; i++ {
		checkSymmetric(t, resp.MultiroundPairings[i*numPlayers:(i+1)*numPlayers])
	}
	pairtestutils.AddNDummyRounds(autoReq, rrRoundsTotal)
	resp = cop.COPPair(autoReq)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.Equal(len(resp.Pairings), numPlayers)

	// AUTO with R < P-1 should produce 3 initial-fontes rounds.
	autoReq = pairtestutils.CreateDefaultPairRequest()
	autoReq.PairMethod = pb.PairMethod_PAIR_AUTO
	autoReq.Rounds = int32(numPlayers - 2) // fewer than one full RR cycle
	resp = cop.COPPair(autoReq)
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
	resp = cop.COPPair(autoReq10)
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
	req := pairtestutils.CreateBellevilleCSW4thCLAfterRound12PairRequest()
	req.ControlLossActivationRound = 11
	req.Seed = 1
	req.ControlLossSims = 1000000000
	resp := cop.COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_TIMEOUT)
}
