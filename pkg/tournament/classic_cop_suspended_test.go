package tournament

import (
	"testing"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestClassicDivisionCOPSuspended(t *testing.T) {
	is := is.New(t)

	tc, err := newGibsonByeDivision(pb.PairingMethod_KING_OF_THE_HILL, 1)
	is.NoErr(err)
	tc.DivisionControls.SuspendedResult = pb.TournamentGameResult_FORFEIT_LOSS
	rc := tc.RoundControls[2]
	rc.PairingMethod = pb.PairingMethod_PAIRING_METHOD_COP
	rc.DivisionSims = 1000
	rc.ControlLossSims = 1000
	rc.PlacePrizes = 1
	rc.ControlLossThreshold = 0.25
	rc.GibsonSpreads = []int32{250}
	rc.HopefulnessThresholds = []float64{0.1}
	err = pairAndPlayWithBye(tc, []string{"a", "b", "c", "d", "e", "f", "g", "h"}, []int{30, 20, 10, 5}, "i", pb.TournamentGameResult_BYE, 0)
	is.NoErr(err)
	err = pairAndPlayWithBye(tc, []string{"a", "c", "b", "e", "d", "g", "f", "i"}, []int{30, 20, 10, 5}, "h", pb.TournamentGameResult_BYE, 1)
	is.NoErr(err)

	// A suspended player is given the suspended result, not left unpaired.
	is.NoErr(tc.DeletePairings(2))
	tc.Players.Persons[tc.PlayerIndexMap["i"]].Suspended = true
	_, err = tc.PairRound(2, false)
	is.NoErr(err)
	pairing, err := tc.getPairing("i", 2)
	is.NoErr(err)
	is.True(pairing != nil)
	opp, err := tc.opponentOf("i", 2)
	is.NoErr(err)
	is.Equal(opp, "i")
	is.Equal(pairing.Outcomes[0], pb.TournamentGameResult_FORFEIT_LOSS)
	for _, p := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		opp, err := tc.opponentOf(p, 2)
		is.NoErr(err)
		is.True(opp != p && opp != "i")
	}
}
