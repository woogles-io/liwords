package cop

import (
	"fmt"
	"strings"
	"testing"

	"github.com/matryer/is"

	copdatapkg "github.com/woogles-io/liwords/pkg/pair/copdata"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

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
	}
	is.Equal(computeDisallowedLeaderOpponent(pargsNoBye), playerNodes[3])

	// With a bye landing on rank 2 (inside the group), the group's pairable
	// size is already even (2), so no extra player should be pulled in.
	pargsWithBye := &policyArgs{
		playerNodes:              playerNodes,
		copdata:                  copdata,
		topDownByePlayer:         playerNodes[2],
		forcedContenderByePlayer: -1,
	}
	is.Equal(computeDisallowedLeaderOpponent(pargsWithBye), -1)
}
