package tournament

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// The default players, in seed order.
const (
	will   = "Will"
	josh   = "Josh"
	conrad = "Conrad"
	jesse  = "Jesse"
)

func laggedRoundControls(methods []pb.PairingMethod, laggedRounds ...int) []*pb.RoundControl {
	roundControls := defaultRoundControls(len(methods))
	for i, method := range methods {
		roundControls[i].PairingMethod = method
	}
	for _, round := range laggedRounds {
		roundControls[round].PairingLag = 1
	}
	return roundControls
}

// submitScore submits a result in which the player with the higher score
// wins.
func submitScore(tc *ClassicDivision, round int, p1 string, p2 string, p1Score int, p2Score int) error {
	p1Result, p2Result := pb.TournamentGameResult_WIN, pb.TournamentGameResult_LOSS
	if p2Score > p1Score {
		p1Result, p2Result = p2Result, p1Result
	}
	_, err := tc.SubmitResult(round, p1, p2, p1Score, p2Score, p1Result, p2Result,
		pb.GameEndReason_STANDARD, false, 0, "")
	return err
}

func isWooglesError(err error, code pb.WooglesError) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), entity.NewWooglesError(code).Error()+entity.WooglesErrorDelimiter)
}

func TestLaggedPairings(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_SWISS,
		pb.PairingMethod_SWISS,
		pb.PairingMethod_KING_OF_THE_HILL,
	}, 2, 3)
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))

	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.True(!tc.IsRoundPaired(1))
	is.True(!tc.IsRoundPaired(2))

	// Completing round 1 pairs round 2 and also round 3, which is lagged
	// and so is paired from the results through round 1.
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	is.NoErr(equalPairingStrings([][]string{{conrad, will}, {jesse, josh}}, tc.getPlayerPairings(1)))
	// Round 3 avoids both the round 1 and round 2 matchups.
	is.NoErr(equalPairingStrings([][]string{{conrad, josh}, {jesse, will}}, tc.getPlayerPairings(2)))
	is.True(!tc.IsRoundPaired(3))

	// Round 3 can be opened while round 2 is still being played.
	is.NoErr(tc.StartRound(true))
	is.NoErr(tc.IsRoundStartable())
	is.NoErr(tc.StartRound(true))
	is.Equal(tc.GetCurrentRound(), 2)

	// Round 4 needs round 2 to be complete.
	err = tc.IsRoundStartable()
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_ROUND_NOT_COMPLETE))
	is.True(strings.Contains(err.Error(), entity.WooglesErrorDelimiter+"2"+entity.WooglesErrorDelimiter))

	// Round 1 is closed, but round 2 is still open.
	err = submitScore(tc, 0, will, josh, 450, 400)
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_NONAMENDMENT_PAST_RESULT))
	is.NoErr(submitScore(tc, 1, will, conrad, 500, 400))

	// Round 3 results that come in before round 2 is complete are not used
	// to pair round 4.
	is.NoErr(submitScore(tc, 2, will, jesse, 400, 900))
	is.NoErr(submitScore(tc, 2, conrad, josh, 300, 600))
	is.True(!tc.IsRoundPaired(3))
	finished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(!finished)

	is.NoErr(submitScore(tc, 1, josh, jesse, 410, 400))
	// King of the hill from the standings after round 2. The standings
	// after round 3 would pair Josh with Will and Jesse with Conrad.
	is.NoErr(equalPairingStrings([][]string{{conrad, will}, {jesse, josh}}, tc.getPlayerPairings(3)))

	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 3, will, conrad, 400, 300))
	is.NoErr(submitScore(tc, 3, josh, jesse, 400, 300))
	finished, err = tc.IsFinished()
	is.NoErr(err)
	is.True(finished)
}

func TestLaggedAustralianDraw(t *testing.T) {
	is := is.New(t)

	// TestLaggedPairings with the Australian Draw after the first round.
	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_AUSTRALIAN_DRAW,
		pb.PairingMethod_AUSTRALIAN_DRAW,
		pb.PairingMethod_AUSTRALIAN_DRAW,
	}, 2, 3)
	// The director form leaves MaxRepeats at 0 for the Australian Draw,
	// which avoids every repeat. This helper's default is Swiss's 1.
	for _, rc := range roundControls[1:] {
		rc.MaxRepeats = 0
	}
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))

	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	// Round 2 pairs each player with the highest-standing player they have
	// not met.
	is.NoErr(equalPairingStrings([][]string{{conrad, will}, {jesse, josh}}, tc.getPlayerPairings(1)))
	// Round 3 is paired from the same standings, and avoids the round 2
	// matchups even though round 2 has not been played.
	is.NoErr(equalPairingStrings([][]string{{conrad, josh}, {jesse, will}}, tc.getPlayerPairings(2)))
	is.True(!tc.IsRoundPaired(3))

	// Round 3 can be opened while round 2 is still being played.
	is.NoErr(tc.StartRound(true))
	is.NoErr(tc.StartRound(true))
	is.Equal(tc.GetCurrentRound(), 2)

	is.NoErr(submitScore(tc, 1, will, conrad, 500, 400))
	is.NoErr(submitScore(tc, 2, will, jesse, 400, 900))
	is.NoErr(submitScore(tc, 2, conrad, josh, 300, 600))
	is.True(!tc.IsRoundPaired(3))

	is.NoErr(submitScore(tc, 1, josh, jesse, 410, 400))
	// Everyone has now met everyone, so round 4 forgives the oldest round
	// first and repeats the round 1 matchups.
	is.NoErr(equalPairingStrings([][]string{{conrad, jesse}, {josh, will}}, tc.getPlayerPairings(3)))

	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 3, will, josh, 400, 300))
	is.NoErr(submitScore(tc, 3, conrad, jesse, 400, 300))
	finished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(finished)
}

func TestLaggedWinGroups(t *testing.T) {
	is := is.New(t)

	// TestLaggedAustralianDraw with Win Groups after the first round. Win
	// Groups needs the same round-by-round history, and in this field its own
	// rule comes to the same pairings.
	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_WIN_GROUPS,
		pb.PairingMethod_WIN_GROUPS,
		pb.PairingMethod_WIN_GROUPS,
	}, 2, 3)
	// The director form leaves MaxRepeats at 0 for Win Groups, which avoids
	// every repeat. This helper's default is Swiss's 1.
	for _, rc := range roundControls[1:] {
		rc.MaxRepeats = 0
	}
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))

	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	// Round 2: the winners conrad and will form one group, the losers jesse
	// and josh the other.
	is.NoErr(equalPairingStrings([][]string{{conrad, will}, {jesse, josh}}, tc.getPlayerPairings(1)))
	// Round 3 is paired from the same standings while round 2 is still being
	// played. conrad and will have just been paired, so the next two players
	// join their group, whose top half then plays its bottom half.
	is.NoErr(equalPairingStrings([][]string{{conrad, josh}, {jesse, will}}, tc.getPlayerPairings(2)))
	is.True(!tc.IsRoundPaired(3))

	is.NoErr(tc.StartRound(true))
	is.NoErr(tc.StartRound(true))
	is.Equal(tc.GetCurrentRound(), 2)

	is.NoErr(submitScore(tc, 1, will, conrad, 500, 400))
	is.NoErr(submitScore(tc, 2, will, jesse, 400, 900))
	is.NoErr(submitScore(tc, 2, conrad, josh, 300, 600))
	is.True(!tc.IsRoundPaired(3))

	is.NoErr(submitScore(tc, 1, josh, jesse, 410, 400))
	// Everyone has now met everyone, so round 4 forgives the oldest round
	// first and repeats the round 1 matchups.
	is.NoErr(equalPairingStrings([][]string{{conrad, jesse}, {josh, will}}, tc.getPlayerPairings(3)))

	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 3, will, josh, 400, 300))
	is.NoErr(submitScore(tc, 3, conrad, jesse, 400, 300))
	finished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(finished)
}

func TestLaggedPairingsFinishRequiresEveryRound(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
	}, 2)
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	is.NoErr(tc.StartRound(true))
	is.NoErr(tc.StartRound(true))

	// A director enters every final round scorecard before the last
	// scorecard from the round before it.
	for _, pairing := range tc.getPlayerPairings(2) {
		is.NoErr(submitScore(tc, 2, pairing[0], pairing[1], 400, 300))
	}
	complete, err := tc.IsRoundComplete(2)
	is.NoErr(err)
	is.True(complete)
	finished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(!finished)

	for _, pairing := range tc.getPlayerPairings(1) {
		is.NoErr(submitScore(tc, 1, pairing[0], pairing[1], 400, 300))
	}
	finished, err = tc.IsFinished()
	is.NoErr(err)
	is.True(finished)
}

func TestLaggedPairingsReadyGate(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
	}, 2)
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	is.NoErr(tc.StartRound(true))
	is.NoErr(tc.StartRound(true))

	// Will has not finished his round 2 game yet.
	_, _, err = tc.SetReadyForGame(will, "conn1", 2, 0, false)
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_PREVIOUS_GAME_UNFINISHED))

	// Round 2 is still open, so its games can still be started.
	_, _, err = tc.SetReadyForGame(josh, "conn2", 1, 0, false)
	is.NoErr(err)
	_, _, err = tc.SetReadyForGame(will, "conn1", 0, 0, false)
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_SET_GAME_ROUND_NUMBER))

	is.NoErr(submitScore(tc, 1, conrad, will, 400, 500))
	_, bothReady, err := tc.SetReadyForGame(will, "conn1", 2, 0, false)
	is.NoErr(err)
	is.True(!bothReady)

	// Getting unready is always allowed.
	_, _, err = tc.SetReadyForGame(jesse, "conn3", 2, 0, true)
	is.NoErr(err)
}

func TestLaggedSecondRound(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
	}, 1)
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.NoErr(err)

	// A lagged second round is paired from the initial standings as soon
	// as the round controls are set.
	initialPairings := [][]string{{conrad, jesse}, {josh, will}}
	is.NoErr(equalPairingStrings(initialPairings, tc.getPlayerPairings(1)))

	// It can be opened while the first round is still being played.
	is.NoErr(tc.StartRound(true))
	is.NoErr(tc.IsRoundStartable())

	is.NoErr(submitScore(tc, 0, will, josh, 300, 450))
	is.NoErr(submitScore(tc, 0, conrad, jesse, 300, 500))
	is.NoErr(equalPairingStrings(initialPairings, tc.getPlayerPairings(1)))

	// Re-pairing it after round 1 results are in still uses the initial
	// standings.
	_, err = tc.PairRound(1, false)
	is.NoErr(err)
	is.NoErr(equalPairingStrings(initialPairings, tc.getPlayerPairings(1)))
}

func TestLaggedPairingsValidation(t *testing.T) {
	is := is.New(t)

	kothRounds := func(n int) []pb.PairingMethod {
		methods := make([]pb.PairingMethod, n)
		for i := range methods {
			methods[i] = pb.PairingMethod_KING_OF_THE_HILL
		}
		return methods
	}

	roundControls := laggedRoundControls(kothRounds(4))
	roundControls[2].PairingLag = 2
	_, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_INVALID_PAIRING_LAG))

	_, err = compactNewClassicDivision(defaultPlayers, laggedRoundControls(kothRounds(4), 0), false)
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_LAGGED_FIRST_ROUND))

	_, err = compactNewClassicDivision(defaultPlayers, laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_ELIMINATION,
		pb.PairingMethod_ELIMINATION,
	}, 1), false)
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_LAGGED_ELIMINATION))

	copRoundControls := func(laggedRound int) []*pb.RoundControl {
		methods := kothRounds(6)
		for i := 3; i < 6; i++ {
			methods[i] = pb.PairingMethod_PAIRING_METHOD_COP
		}
		roundControls := laggedRoundControls(methods, laggedRound)
		for i := 3; i < 6; i++ {
			roundControls[i].DivisionSims = 1000
			roundControls[i].ControlLossSims = 1000
			roundControls[i].PlacePrizes = 1
			roundControls[i].ControlLossThreshold = 0.25
			roundControls[i].GibsonSpreads = []int32{250}
			roundControls[i].HopefulnessThresholds = []float64{0.1}
		}
		return roundControls
	}
	_, err = compactNewClassicDivision(defaultPlayers, copRoundControls(4), false)
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_LAGGED_COP_LAST_ROUNDS))
	_, err = compactNewClassicDivision(defaultPlayers, copRoundControls(5), false)
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_LAGGED_COP_LAST_ROUNDS))
	_, err = compactNewClassicDivision(defaultPlayers, copRoundControls(3), false)
	is.NoErr(err)

	// Changing a round's lag during the tournament is validated too.
	tc, err := compactNewClassicDivision(defaultPlayers, laggedRoundControls(kothRounds(3)), false)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	roundTwoPairings := tc.getPlayerPairings(1)

	_, _, err = tc.SetSingleRoundControls(0, &pb.RoundControl{Round: 0, PairingLag: 1,
		PairingMethod: pb.PairingMethod_KING_OF_THE_HILL, GamesPerRound: 1})
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_LAGGED_FIRST_ROUND))

	// Lagging the third round makes it pairable from the round 1 results,
	// which are complete, so it is paired right away. The second round's
	// pairings are untouched.
	resp, _, err := tc.SetSingleRoundControls(2, &pb.RoundControl{Round: 2, PairingLag: 1,
		PairingMethod: pb.PairingMethod_KING_OF_THE_HILL, GamesPerRound: 1})
	is.NoErr(err)
	is.True(len(resp.DivisionPairings) > 0)
	is.True(tc.IsRoundPaired(2))
	is.NoErr(equalPairingStrings(roundTwoPairings, tc.getPlayerPairings(1)))

	// It can be opened while round 2 is played.
	is.NoErr(tc.StartRound(true))
	is.NoErr(tc.IsRoundStartable())

	// Changing the lag of a round that is already paired leaves its
	// pairings alone.
	roundThreePairings := tc.getPlayerPairings(2)
	resp, _, err = tc.SetSingleRoundControls(2, &pb.RoundControl{Round: 2,
		PairingMethod: pb.PairingMethod_KING_OF_THE_HILL, GamesPerRound: 1})
	is.NoErr(err)
	is.Equal(len(resp.DivisionPairings), 0)
	is.NoErr(equalPairingStrings(roundThreePairings, tc.getPlayerPairings(2)))
}

func TestLaggedPairingsAutoStart(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
	}, 2)
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.Equal(tc.GetCurrentRound(), 0)

	// Completing round 1 opens round 2, and also round 3, which is lagged.
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	is.Equal(tc.GetCurrentRound(), 2)
}

func TestLaggedPairingsAddAndRemovePlayers(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
	}, 2)
	tc, err := compactNewClassicDivision(makeTournamentPersons(map[string]int32{
		will: 10000, josh: 3000, conrad: 2200, jesse: 2100}), roundControls, false)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	is.True(tc.IsRoundPaired(1))
	is.True(tc.IsRoundPaired(2))

	// The lagged third round is already paired, so it is re-paired along
	// with the second round.
	_, err = tc.AddPlayers(makeTournamentPersons(map[string]int32{"Matt": 1000}))
	is.NoErr(err)
	is.NoErr(validatePairings(tc, 1))
	is.NoErr(validatePairings(tc, 2))

	_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{jesse: 2100}))
	is.NoErr(err)
	is.NoErr(validatePairings(tc, 1))
	is.NoErr(validatePairings(tc, 2))
	isBye, err := tc.pairingIsBye(jesse, 2)
	is.NoErr(err)
	is.True(isBye)
}

func TestLaggedPairingsCOPRequest(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
	})
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 1, will, conrad, 450, 400))
	is.NoErr(submitScore(tc, 1, josh, jesse, 500, 300))
	is.NoErr(tc.StartRound(true))

	xhr, err := tc.GetXHRResponse()
	is.NoErr(err)
	cfg := &COPIntermediateConfig{GibsonSpread: []int{250}, HopefulnessThreshold: []float64{0.1}}

	// A lagged fourth round sends the pairings through round 3 but the
	// results only through round 2.
	req, err := TournamentDivisionToCOPRequest(xhr, 3, 1, cfg)
	is.NoErr(err)
	is.Equal(len(req.DivisionPairings), 3)
	is.Equal(len(req.DivisionResults), 2)

	req, err = TournamentDivisionToCOPRequest(xhr, 3, 2, cfg)
	is.NoErr(err)
	is.Equal(len(req.DivisionPairings), 3)
	is.Equal(len(req.DivisionResults), 3)

	_, err = TournamentDivisionToCOPRequest(xhr, 3, 3, cfg)
	is.True(err != nil)
}

// TestPreLagDivisionFixture checks that a division saved before pairing
// lag existed loads and plays out exactly as before.
func TestPreLagDivisionFixture(t *testing.T) {
	is := is.New(t)

	bts, err := os.ReadFile("./testdata/pre_lag_division.json")
	is.NoErr(err)
	tc := &ClassicDivision{}
	is.NoErr(json.Unmarshal(bts, tc))
	for _, rc := range tc.RoundControls {
		is.Equal(rc.PairingLag, uint32(0))
	}
	is.Equal(tc.GetCurrentRound(), 0)

	// Round 2 was already paired from the round 1 results.
	is.NoErr(equalPairingStrings([][]string{{conrad, will}, {jesse, josh}}, tc.getPlayerPairings(1)))
	is.True(!tc.hasAnyPairings(2))

	is.NoErr(tc.StartRound(true))
	err = submitScore(tc, 0, will, josh, 450, 400)
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_NONAMENDMENT_PAST_RESULT))

	is.NoErr(submitScore(tc, 1, will, conrad, 300, 400))
	err = tc.IsRoundStartable()
	is.True(isWooglesError(err, pb.WooglesError_TOURNAMENT_ROUND_NOT_COMPLETE))
	is.NoErr(submitScore(tc, 1, josh, jesse, 500, 300))
	is.NoErr(equalPairingStrings([][]string{{conrad, josh}, {jesse, will}}, tc.getPlayerPairings(2)))

	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 2, conrad, josh, 400, 300))
	is.NoErr(submitScore(tc, 2, will, jesse, 400, 300))
	finished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(finished)

	// Saving it again doesn't add the new field.
	bts, err = json.Marshal(tc)
	is.NoErr(err)
	is.True(!strings.Contains(string(bts), "pairing_lag"))
}

// Turning on lag for a future round in the middle of an event must not
// leave that round unpaired.
func TestLaggedPairingsLagTurnedOnMidEvent(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
	})
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	is.Equal(tc.GetCurrentRound(), 1)

	// While round 2 is being played, the director lags round 3. It is
	// paired from the round 1 results right away, and AutoStart opens it.
	_, _, err = tc.SetSingleRoundControls(2, &pb.RoundControl{Round: 2, PairingLag: 1,
		PairingMethod: pb.PairingMethod_KING_OF_THE_HILL, GamesPerRound: 1})
	is.NoErr(err)
	is.True(tc.IsRoundPaired(2))
	is.Equal(tc.GetCurrentRound(), 2)

	for _, pairing := range tc.getPlayerPairings(1) {
		is.NoErr(submitScore(tc, 1, pairing[0], pairing[1], 400, 300))
	}
	is.Equal(tc.GetCurrentRound(), 2)
}

// Pairing a round fills in a forfeit for every removed player, which goes
// through SubmitResult. That must not pair or open other rounds in the
// middle of the pairing.
func TestLaggedPairingsRemovedPlayerAutoStart(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
	}, 2, 3)
	tc, err := compactNewClassicDivision(defaultPlayersOdd, roundControls, true)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{"Matt": 2000}))
	is.NoErr(err)

	keysBefore := tc.PairingKeyInt
	for _, pairing := range tc.getPlayerPairings(0) {
		if len(pairing) == 2 {
			_, err = tc.SubmitResult(0, pairing[0], pairing[1], 400, 300,
				pb.TournamentGameResult_WIN, pb.TournamentGameResult_LOSS,
				pb.GameEndReason_STANDARD, false, 0, "")
			is.NoErr(err)
		}
	}
	// Completing round 1 pairs and opens round 2 and the lagged round 3.
	// Each round is paired once: two games and a forfeit for the removed
	// player, so six new pairings in all.
	is.Equal(tc.GetCurrentRound(), 2)
	is.NoErr(validatePairings(tc, 1))
	is.NoErr(validatePairings(tc, 2))
	is.Equal(tc.PairingKeyInt-keysBefore, 6)
}

// Pairings a director deletes stay deleted until the director pairs the
// round, even while results for other rounds keep coming in.
func TestLaggedPairingsDeletedPairingsStayDeleted(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
	}, 2, 3)
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	is.NoErr(tc.StartRound(true))
	is.True(tc.IsRoundPaired(2))

	is.NoErr(tc.DeletePairings(2))
	roundTwoPairings := tc.getPlayerPairings(1)
	is.NoErr(submitScore(tc, 1, roundTwoPairings[0][0], roundTwoPairings[0][1], 400, 300))
	is.True(!tc.hasAnyPairings(2))

	// Completing round 2 leaves round 3 for the director. The lagged round
	// 4, whose basis round 2 is, waits too, since its repeats depend on the
	// round 3 pairings.
	is.NoErr(submitScore(tc, 1, roundTwoPairings[1][0], roundTwoPairings[1][1], 400, 300))
	is.True(!tc.hasAnyPairings(2))
	is.True(!tc.hasAnyPairings(3))
}

// With every player but one removed, each round is complete as soon as it
// is paired, so completing one round pairs every round after it. That
// still happens one round at a time, each round paired once.
func TestLaggedPairingsByesCascade(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
	}, 2, 3)
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{josh: 3000, conrad: 2200, jesse: 2100}))
	is.NoErr(err)

	keysBefore := tc.PairingKeyInt
	_, err = tc.SubmitResult(0, will, josh, 400, 300, pb.TournamentGameResult_WIN,
		pb.TournamentGameResult_LOSS, pb.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)
	_, err = tc.SubmitResult(0, conrad, jesse, 400, 300, pb.TournamentGameResult_WIN,
		pb.TournamentGameResult_LOSS, pb.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Rounds 2 to 5 are each paired once, with a bye and three forfeits.
	is.Equal(tc.PairingKeyInt-keysBefore, 16)
	for round := 1; round < 5; round++ {
		is.NoErr(validatePairings(tc, round))
	}
	finished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(finished)
}

// A lagged round whose basis round completes while the round before it
// still has to be paired by hand is paired once a director pairs that
// round, and AutoStart never rejects a result because a round can't open.
func TestLaggedPairingsWaitingOnPreviousRound(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_MANUAL,
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_KING_OF_THE_HILL,
	}, 2)
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	// Round 2 is MANUAL and unpaired, so AutoStart can't open it, and
	// round 3, whose basis round 1 now is, waits for round 2's pairings.
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	is.Equal(tc.GetCurrentRound(), 0)
	is.True(!tc.hasAnyPairings(2))

	// The director pairs round 2 by hand, which makes round 3 pairable.
	_, err = tc.SetPairing(will, conrad, 1, pb.TournamentGameResult_NO_RESULT)
	is.NoErr(err)
	resp, err := tc.PairLaggedRoundAfter(1)
	is.NoErr(err)
	is.Equal(len(resp.DivisionPairings), 0) // round 2 isn't fully paired yet
	_, err = tc.SetPairing(josh, jesse, 1, pb.TournamentGameResult_NO_RESULT)
	is.NoErr(err)
	resp, err = tc.PairLaggedRoundAfter(1)
	is.NoErr(err)
	is.True(len(resp.DivisionPairings) > 0)
	is.True(tc.IsRoundPaired(2))

	// Once round 2 is open, AutoStart opens round 3 alongside it.
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 1, will, conrad, 400, 300))
	is.Equal(tc.GetCurrentRound(), 2)
	is.NoErr(submitScore(tc, 1, josh, jesse, 400, 300))
}

// Without lag, AutoStart that can't open the next round, because it is a
// MANUAL round nobody has paired, doesn't reject the result that completed
// the round.
func TestAutoStartDoesNotRejectResults(t *testing.T) {
	is := is.New(t)

	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		pb.PairingMethod_MANUAL,
		pb.PairingMethod_KING_OF_THE_HILL,
	})
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 0, will, josh, 450, 400))
	is.NoErr(submitScore(tc, 0, conrad, jesse, 500, 300))
	complete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(complete)
	is.Equal(tc.GetCurrentRound(), 0)
}

// laggedRepeatLimitedSwiss returns round controls for two players where
// the lagged third round can't be paired: it may not repeat a pairing, and
// the only possible pairing has been played already.
func laggedRepeatLimitedSwiss(second pb.PairingMethod) []*pb.RoundControl {
	roundControls := laggedRoundControls([]pb.PairingMethod{
		pb.PairingMethod_KING_OF_THE_HILL,
		second,
		pb.PairingMethod_SWISS,
	}, 2)
	roundControls[2].AllowOverMaxRepeats = false
	roundControls[2].MaxRepeats = 1
	return roundControls
}

// Pairing a lagged round is follow-on work, so failing to pair it doesn't
// reject the result that made it pairable.
func TestLaggedPairingFailureDoesNotRejectResult(t *testing.T) {
	is := is.New(t)

	tc, err := compactNewClassicDivision(makeTournamentPersons(map[string]int32{will: 2000, josh: 1000}),
		laggedRepeatLimitedSwiss(pb.PairingMethod_KING_OF_THE_HILL), false)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	is.NoErr(submitScore(tc, 0, will, josh, 400, 300))
	is.True(tc.IsRoundPaired(1))
	is.True(!tc.IsRoundPaired(2))
}

// Pairing a round clears it first, so a failed pairing must put the round
// back: a lagged round the director paired by hand keeps those pairings
// when re-pairing it from a result fails, and nothing in the response
// suggests otherwise.
func TestFailedPairingKeepsRoundPairings(t *testing.T) {
	is := is.New(t)

	tc, err := compactNewClassicDivision(makeTournamentPersons(map[string]int32{will: 2000, josh: 1000}),
		laggedRepeatLimitedSwiss(pb.PairingMethod_KING_OF_THE_HILL), false)
	is.NoErr(err)
	is.NoErr(tc.StartRound(true))
	// The director pairs the lagged round 3 by hand, repeat and all.
	_, err = tc.SetPairing(will, josh, 2, pb.TournamentGameResult_NO_RESULT)
	is.NoErr(err)
	is.True(tc.IsRoundPaired(2))
	roundThree := slices.Clone(tc.Matrix[2])

	// The result completing round 1 makes round 3 pairable, and the Swiss
	// pairing fails, since it may not repeat a pairing.
	resp, err := tc.SubmitResult(0, will, josh, 400, 300, pb.TournamentGameResult_WIN,
		pb.TournamentGameResult_LOSS, pb.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)
	is.Equal(tc.TakeAutoPairingFailures(), []int{2})
	is.Equal(tc.Matrix[2], roundThree)
	for _, key := range roundThree {
		is.True(tc.PairingMap[key] != nil)
	}
	is.NoErr(validatePairings(tc, 2))
	for _, pairing := range resp.DivisionPairings {
		is.True(pairing.Round != 2)
	}

	// Asking for it directly fails the same way, and also leaves it alone.
	_, err = tc.PairRound(2, false)
	is.True(err != nil)
	is.Equal(tc.Matrix[2], roundThree)
	is.NoErr(validatePairings(tc, 2))
}
