package cop

import (
	"fmt"
	"strings"
	"testing"

	"github.com/matryer/is"
	"golang.org/x/exp/rand"

	copdatapkg "github.com/woogles-io/liwords/pkg/pair/copdata"
	pkgstnd "github.com/woogles-io/liwords/pkg/pair/standings"
	pairtestutils "github.com/woogles-io/liwords/pkg/pair/testutils"
	"github.com/woogles-io/liwords/pkg/pair/verifyreq"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Tests for pending rounds: rounds that are fully paired but have no results
// yet, which is what a lagged pairing request (results through round S,
// pairings through round R-1 > S) looks like. The sims must play pending
// rounds with their real pairings.

// createPendingLeaderMatchupRequest builds an 8-player, 8-round request with
// 6 rounds of results. P0 is 6-0 (+300), P2 5-1 (+160), P4 4-2 (+60), P6 and
// P7 3-3, P5 2-4, P3 1-5 and P1 0-6. When withPendingRound is true, round 7
// is paired but has no results (P = 1), with 1st vs 2nd (P0 vs P2), and the
// request pairs the final round. Otherwise it is an ordinary request to pair
// round 7.
func createPendingLeaderMatchupRequest(withPendingRound bool) *pb.PairRequest {
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{},
		GibsonSpread:               200,
		ControlLossThreshold:       0.25,
		HopefulnessThreshold:       0.02,
		AllPlayers:                 8,
		ValidPlayers:               8,
		Rounds:                     8,
		PlacePrizes:                2,
		DivisionSims:               20000,
		ControlLossSims:            2000,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	pairtestutils.AddNDummyRounds(req, 6)
	for range 3 {
		pairtestutils.AddRoundResultsStr(req, "450 400 440 400 430 400 420 400")
	}
	pairtestutils.AddRoundResultsStr(req, "450 400 440 400 430 400 400 420")
	pairtestutils.AddRoundResultsStr(req, "450 400 440 400 400 430 400 420")
	pairtestutils.AddRoundResultsStr(req, "450 400 400 440 400 430 400 420")
	if withPendingRound {
		// P0 vs P2, P1 vs P3, P4 vs P7, P5 vs P6
		pairtestutils.AddRoundPairingsStr(req, "2 3 0 1 7 6 5 4")
	}
	return req
}

// With 1st vs 2nd in the pending round, one of them always ends with at
// least 7 wins (or both with 6, and they then meet in the final round), so
// 3rd (P4, at most 6 wins) can never finish 1st. If the sims factor-paired the
// pending round instead (1st vs 3rd), 3rd could.
func TestPendingRoundLeaderMatchup(t *testing.T) {
	is := is.New(t)

	req := createPendingLeaderMatchupRequest(true)
	is.True(verifyreq.Verify(req) == nil)
	is.Equal(pkgstnd.GetNumPendingRounds(req), 1)
	logsb := &strings.Builder{}
	copdata, pairErr := copdatapkg.GetPrecompData(req, rand.New(rand.NewSource(uint64(req.Seed))), logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(copdata.Standings.GetPlayerIndex(2), 4)
	is.True(copdata.BaselineTotalSims > 0)
	is.Equal(copdata.BaselineFinalRanks[2][0], 0)
	is.True(copdata.HighestRankAbsolutely[2] > 0)
	is.True(strings.Contains(logsb.String(), "Pending rounds: 1\n"))
	is.True(strings.Contains(logsb.String(), "Pending round 7 pairings"))
	is.True(strings.Contains(logsb.String(), "(#1) P0 vs (#3) P2"))

	// The same standings with no round in progress: the sims factor-pair
	// round 7, so 3rd can play 1st and can finish 1st.
	req = createPendingLeaderMatchupRequest(false)
	is.True(verifyreq.Verify(req) == nil)
	is.Equal(pkgstnd.GetNumPendingRounds(req), 0)
	logsb = &strings.Builder{}
	copdata, pairErr = copdatapkg.GetPrecompData(req, rand.New(rand.NewSource(uint64(req.Seed))), logsb)
	is.Equal(pairErr, pb.PairError_SUCCESS)
	is.Equal(copdata.Standings.GetPlayerIndex(2), 4)
	is.True(copdata.BaselineFinalRanks[2][0] > 0)
	is.True(strings.Contains(logsb.String(), "Pending rounds: 0\n"))
	is.True(!strings.Contains(logsb.String(), "Pending round 7 pairings"))

	// The full lagged request pairs the final round successfully.
	req = createPendingLeaderMatchupRequest(true)
	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	assertValidRoundPairings(is, req, resp.Pairings)
}

// The scenario from TestFactor3HopefulnessUsesAtLeastNotExact fires Factor 3
// when it pairs round 7 with round 6 complete. Lagged, the same results with
// round 7 in progress mean round 8 (the final round) is being paired, so
// Factor 3 must not fire.
func TestPendingRoundSkipsFactor3(t *testing.T) {
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
		DivisionSims:               20000,
		ControlLossSims:            2000,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	// 7 paired rounds, 6 with results.
	pairtestutils.AddNDummyRounds(req, 7)
	mk := func(a, b, c, d, e, f int) string {
		return fmt.Sprintf("%d 400 %d 400 %d 400 %d 400 %d 400 %d 400", 400+a, 400+b, 400+c, 400+d, 400+e, 400+f)
	}
	for range 4 {
		pairtestutils.AddRoundResultsStr(req, mk(18, 13, 8, 1, 1, 100))
	}
	pairtestutils.AddRoundResultsStr(req, mk(18, 13, 8, -1, -1, -100))
	pairtestutils.AddRoundResultsStr(req, mk(-18, -13, -8, -1, -1, -100))
	is.Equal(pkgstnd.GetRoundsRemaining(req), 2)
	is.Equal(pkgstnd.GetNumPendingRounds(req), 1)

	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "Pending rounds: 1\n"))
	is.True(strings.Contains(resp.Log, "Factor 3 skipped: round in progress"))
	is.True(!strings.Contains(resp.Log, "Factor 3 expansion:"))
	is.True(!strings.Contains(resp.Log, "Factor 3 Sim Results"))
	assertValidRoundPairings(is, req, resp.Pairings)
}

// A TSH-style lagged request: TSH can pair a round based on results two or
// more rounds back, so there can be more than one pending round. 9 players
// (byes every round, including both pending rounds), 10 rounds, results
// through round 6, pairings through round 8, pairing round 9.
func TestPendingRoundsTSHStyleTwoPending(t *testing.T) {
	is := is.New(t)

	numPlayers := 9
	req := &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                []string{"P0", "P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8"},
		PlayerClasses:              []int32{0, 0, 0, 0, 0, 0, 0, 0, 0},
		ClassPrizes:                []int32{},
		GibsonSpread:               250,
		ControlLossThreshold:       0.15,
		HopefulnessThreshold:       0.01,
		AllPlayers:                 int32(numPlayers),
		ValidPlayers:               int32(numPlayers),
		Rounds:                     10,
		PlacePrizes:                3,
		DivisionSims:               20000,
		ControlLossSims:            2000,
		ControlLossActivationRound: 6,
		AllowRepeatByes:            false,
		Seed:                       1,
	}
	// Round r: player r has the bye, the others are paired in index order.
	for r := range 8 {
		pairings := make([]int32, numPlayers)
		pairings[r] = int32(r)
		others := []int{}
		for p := range numPlayers {
			if p != r {
				others = append(others, p)
			}
		}
		for i := 0; i < len(others); i += 2 {
			pairings[others[i]] = int32(others[i+1])
			pairings[others[i+1]] = int32(others[i])
		}
		pairtestutils.AddRoundPairings(req, pairings)
		if r >= 6 {
			// Rounds 7 and 8 are pending.
			continue
		}
		results := make([]int32, numPlayers)
		results[r] = 50
		for i := 0; i < len(others); i += 2 {
			// The lower-numbered player wins by a varying margin.
			results[others[i]] = int32(400 + 10 + (r*7+i*13)%90)
			results[others[i+1]] = 400
		}
		req.DivisionResults = append(req.DivisionResults, &pb.RoundResults{Results: results})
	}
	is.True(verifyreq.Verify(req) == nil)
	is.Equal(pkgstnd.GetRoundsRemaining(req), 4)
	is.Equal(pkgstnd.GetNumPendingRounds(req), 2)
	is.Equal(pkgstnd.GetPendingRoundPairs(req), [][][2]int{
		{{0, 1}, {2, 3}, {4, 5}, {6, 6}, {7, 8}},
		{{0, 1}, {2, 3}, {4, 5}, {6, 8}, {7, 7}},
	})

	resp := COPPair(req)
	is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	is.True(strings.Contains(resp.Log, "Pending rounds: 2\n"))
	is.True(strings.Contains(resp.Log, "Pending round 7 pairings"))
	is.True(strings.Contains(resp.Log, "Pending round 8 pairings"))
	is.True(strings.Contains(resp.Log, "(#7) P6 vs BYE"))
	is.True(strings.Contains(resp.Log, "(#8) P7 vs BYE"))
	assertValidRoundPairings(is, req, resp.Pairings)
}

// assertValidRoundPairings checks that every player is paired exactly once
// (a bye is a self-pairing) and that the pairings are symmetric.
func assertValidRoundPairings(is *is.I, req *pb.PairRequest, pairings []int32) {
	is.Helper()
	is.Equal(len(pairings), int(req.AllPlayers))
	numByes := 0
	for playerIdx, oppIdx := range pairings {
		is.True(oppIdx >= 0 && int(oppIdx) < len(pairings))
		is.Equal(int(pairings[oppIdx]), playerIdx)
		if int(oppIdx) == playerIdx {
			numByes++
		}
	}
	is.Equal(numByes, int(req.ValidPlayers)%2)
}
