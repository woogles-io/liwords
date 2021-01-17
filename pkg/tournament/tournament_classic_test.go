package tournament

import (
	"fmt"
	"github.com/matryer/is"
	"math/rand"
	"strings"
	"testing"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

var playerRatings = &realtime.TournamentPersons{Persons: map[string]int32{"Will": 1000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100, "Matt": 100}}
var playerStrings = []string{"Will", "Josh", "Conrad", "Jesse"}
var playersOddStrings = []string{"Will", "Josh", "Conrad", "Jesse", "Matt"}
var rounds = 2
var defaultFirsts = []realtime.FirstMethod{realtime.FirstMethod_MANUAL_FIRST, realtime.FirstMethod_MANUAL_FIRST}
var defaultGamesPerRound int32 = 1

func TestClassicDivisionZeroOrOnePlayers(t *testing.T) {
	// Division creation with zero or one players is a special
	// case that should not fail
	is := is.New(t)
	playerOZRatings := &realtime.TournamentPersons{Persons: map[string]int32{"One": 1000, "Two": 3000, "Three": 2200, "Jesse": 2100, "Matt": 100}}

	_, err := NewClassicDivision([]string{"One", "Two", "Three"}, playerOZRatings, defaultRoundControls(0), true)
	is.NoErr(err)

	_, err = NewClassicDivision([]string{}, playerOZRatings, defaultRoundControls(2), true)
	is.NoErr(err)

	_, err = NewClassicDivision([]string{"One"}, playerOZRatings, defaultRoundControls(2), true)
	is.NoErr(err)
}

func TestClassicDivisionRandom(t *testing.T) {
	// This test attempts to cover the basic
	// functions of a Classic Tournament

	roundControls := defaultRoundControls(rounds)

	is := is.New(t)

	roundControls = defaultRoundControls(rounds)

	tc, err := NewClassicDivision(playerStrings, playerRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	// Test getting a nonexistent round
	_, err = tc.getPairing("Josh", 9)
	is.True(err != nil)

	// Test getting a nonexistent player
	_, err = tc.getPairing("No one", 1)
	is.True(err != nil)

	playerPairings := tc.getPlayerPairings(0)
	player1 := playerPairings[0]
	player2 := playerPairings[1]
	player3 := playerPairings[2]
	player4 := playerPairings[3]

	pairing1, err := tc.getPairing(player1, 0)
	is.NoErr(err)
	pairing2, err := tc.getPairing(player3, 0)
	is.NoErr(err)

	expectedpairing1 := newClassicPairing(tc, player1, player2, 0)
	expectedpairing2 := newClassicPairing(tc, player3, player4, 0)

	// Submit result for an unpaired round
	err = tc.SubmitResult(1, player1, player2, 10000, -40, realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS, realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	// Submit result for players that didn't play each other
	err = tc.SubmitResult(0, player1, player3, 10000, -40, realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS, realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	// Submit a result for game index that is out of range
	err = tc.SubmitResult(0, player1, player2, 10000, -40, realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS, realtime.GameEndReason_STANDARD, false, 4, "")
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	// Submit a result before the tournament has started
	err = tc.SubmitResult(0, player1, player2, 10000, -40, realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS, realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	err = tc.StartRound()
	is.NoErr(err)

	// Submit a result for a paired round
	err = tc.SubmitResult(0, player1, player2, 10000, -40, realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS, realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// The result and record should have changed
	expectedpairing1.Games[0].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_WIN, realtime.TournamentGameResult_LOSS}
	expectedpairing1.Games[0].Scores[0] = 10000
	expectedpairing1.Games[0].Scores[1] = -40
	expectedpairing1.Games[0].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpairing1.Outcomes[0] = realtime.TournamentGameResult_WIN
	expectedpairing1.Outcomes[1] = realtime.TournamentGameResult_LOSS
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	// Attempt to submit the same result
	err = tc.SubmitResult(0, player1, player2, 10000, -40, realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS, realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	// Round 2 should not have been paired,
	// so attempting to submit a result for
	// it will throw an error.
	err = tc.SubmitResult(1, player1, player2, 10000, -40,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	// Amend the result
	err = tc.SubmitResult(0, player1, player2, 30, 900,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, true, 0, "")
	is.NoErr(err)

	// The result and record should be amended
	expectedpairing1.Games[0].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_LOSS,
			realtime.TournamentGameResult_WIN}
	expectedpairing1.Games[0].Scores[0] = 30
	expectedpairing1.Games[0].Scores[1] = 900
	expectedpairing1.Outcomes[0] = realtime.TournamentGameResult_LOSS
	expectedpairing1.Outcomes[1] = realtime.TournamentGameResult_WIN
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	// Submit the final result for round 1
	err = tc.SubmitResult(0, player3, player4, 1, 1,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_CANCELLED, false, 0, "")
	is.NoErr(err)

	expectedpairing2.Games[0].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_DRAW,
			realtime.TournamentGameResult_DRAW}
	expectedpairing2.Games[0].Scores[0] = 1
	expectedpairing2.Games[0].Scores[1] = 1
	expectedpairing2.Outcomes[0] = realtime.TournamentGameResult_DRAW
	expectedpairing2.Outcomes[1] = realtime.TournamentGameResult_DRAW
	expectedpairing2.Games[0].GameEndReason = realtime.GameEndReason_CANCELLED
	is.NoErr(equalPlayerRoundInfo(expectedpairing2, pairing2))

	roundIsComplete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Set pairings to test more easily
	err = tc.SetPairing(player1, player2, 1, false)
	is.NoErr(err)
	err = tc.SetPairing(player3, player4, 1, false)
	is.NoErr(err)

	pairing1, err = tc.getPairing(player1, 1)
	is.NoErr(err)
	pairing2, err = tc.getPairing(player3, 1)
	is.NoErr(err)

	expectedpairing1 = newClassicPairing(tc, player1, player2, 1)
	expectedpairing2 = newClassicPairing(tc, player3, player4, 1)

	// Round 2 should have been paired,
	// submit a result

	err = tc.SubmitResult(1, player1, player2, 0, 0,
		realtime.TournamentGameResult_FORFEIT_LOSS,
		realtime.TournamentGameResult_FORFEIT_LOSS,
		realtime.GameEndReason_FORCE_FORFEIT, false, 0, "")
	is.NoErr(err)

	expectedpairing1.Games[0].Scores[0] = 0
	expectedpairing1.Games[0].Scores[1] = 0
	expectedpairing1.Games[0].GameEndReason = realtime.GameEndReason_FORCE_FORFEIT
	expectedpairing1.Outcomes[0] = realtime.TournamentGameResult_FORFEIT_LOSS
	expectedpairing1.Outcomes[1] = realtime.TournamentGameResult_FORFEIT_LOSS
	expectedpairing1.Games[0].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_FORFEIT_LOSS,
			realtime.TournamentGameResult_FORFEIT_LOSS}
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	// Submit the final tournament results
	err = tc.SubmitResult(1, player3, player4, 50, 50,
		realtime.TournamentGameResult_BYE,
		realtime.TournamentGameResult_BYE,
		realtime.GameEndReason_CANCELLED, false, 0, "")
	is.NoErr(err)

	expectedpairing2.Games[0].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_BYE, realtime.TournamentGameResult_BYE}
	expectedpairing2.Games[0].Scores[0] = 50
	expectedpairing2.Games[0].Scores[1] = 50
	expectedpairing2.Games[0].GameEndReason = realtime.GameEndReason_CANCELLED
	expectedpairing2.Outcomes[0] = realtime.TournamentGameResult_BYE
	expectedpairing2.Outcomes[1] = realtime.TournamentGameResult_BYE
	is.NoErr(equalPlayerRoundInfo(expectedpairing2, pairing2))

	roundIsComplete, err = tc.IsRoundComplete(1)
	is.NoErr(err)
	is.True(roundIsComplete)

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(tournamentIsFinished)

	// Attempt to get the Standings
	// for an out of range round number
	_, err = tc.GetStandings(8)
	is.True(err != nil)

	// Standings are tested in the
	// King of the Hill Classic Tournament test.

	// Get the standings for round 1
	_, err = tc.GetStandings(0)
	is.NoErr(err)

	// Get the standings for round 2
	_, err = tc.GetStandings(1)
	is.NoErr(err)

	// Check that pairings are correct with an odd number of players
	tc, err = NewClassicDivision(playersOddStrings, playerRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))
}

func TestClassicDivisionKingOfTheHill(t *testing.T) {
	// This test is used to ensure that the standings are
	// calculated correctly and that King of the Hill
	// pairings are correct

	is := is.New(t)

	roundControls := defaultRoundControls(rounds)

	for i := 0; i < rounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_KING_OF_THE_HILL
	}

	tc, err := NewClassicDivision(playerStrings, playerRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	// Tournament should not be over

	player1 := playerStrings[0]
	player2 := playerStrings[1]
	player3 := playerStrings[3]
	player4 := playerStrings[2]

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(!tournamentIsFinished)
	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	err = tc.SubmitResult(0, player1, player2, 550, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(0, player3, player4, 300, 700,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 1
	standings, err := tc.GetStandings(0)
	is.NoErr(err)

	expectedstandings := []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player4, Wins: 1, Losses: 0, Draws: 0, Spread: 400},
		&realtime.PlayerStanding{Player: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 150},
		&realtime.PlayerStanding{Player: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -150},
		&realtime.PlayerStanding{Player: player3, Wins: 0, Losses: 1, Draws: 0, Spread: -400},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	// The next round should have been paired
	// Tournament should not be over

	tournamentIsFinished, err = tc.IsFinished()
	is.NoErr(err)
	is.True(!tournamentIsFinished)

	// Submit results for the round
	err = tc.SubmitResult(1, player1, player4, 670, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(1, player3, player2, 700, 700,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 2
	standings, err = tc.GetStandings(1)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player1, Wins: 2, Losses: 0, Draws: 0, Spread: 420},
		&realtime.PlayerStanding{Player: player4, Wins: 1, Losses: 1, Draws: 0, Spread: 130},
		&realtime.PlayerStanding{Player: player2, Wins: 0, Losses: 1, Draws: 1, Spread: -150},
		&realtime.PlayerStanding{Player: player3, Wins: 0, Losses: 1, Draws: 1, Spread: -400},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	// Tournament should be over

	tournamentIsFinished, err = tc.IsFinished()
	is.NoErr(err)
	is.True(tournamentIsFinished)
}

func TestClassicDivisionFactor(t *testing.T) {
	// This test is used to ensure that factor
	// pairings work correctly

	is := is.New(t)

	roundControls := []*realtime.RoundControl{}

	for i := 0; i < 2; i++ {
		roundControls = append(roundControls, &realtime.RoundControl{FirstMethod: realtime.FirstMethod_MANUAL_FIRST,
			PairingMethod:               realtime.PairingMethod_FACTOR,
			GamesPerRound:               defaultGamesPerRound,
			Round:                       int32(i),
			Factor:                      int32(i + 2),
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1})
	}
	playerLetterRatings := &realtime.TournamentPersons{Persons: map[string]int32{"h": 1000, "g": 3000, "f": 2200, "e": 2100, "d": 12, "c": 43, "b": 40, "a": 2}}
	tc, err := NewClassicDivision([]string{"h", "g", "f", "e", "d", "c", "b", "a"}, playerLetterRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	err = tc.StartRound()
	is.NoErr(err)

	// This should throw an error since it attempts
	// to amend a result that never existed
	err = tc.SubmitResult(0, "h", "f", 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, true, 0, "")
	is.True(err != nil)

	err = tc.SubmitResult(0, "h", "f", 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(0, "g", "e", 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(0, "d", "a", 700, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// This is an invalid factor for this number of
	// players and an error should be returned
	tc.RoundControls[1].Factor = 5

	err = tc.SubmitResult(0, "c", "b", 600, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	roundIsComplete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1
	standings, err := tc.GetStandings(0)
	is.NoErr(err)

	expectedstandings := []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: "h", Wins: 1, Losses: 0, Draws: 0, Spread: 400},
		&realtime.PlayerStanding{Player: "g", Wins: 1, Losses: 0, Draws: 0, Spread: 300},
		&realtime.PlayerStanding{Player: "d", Wins: 1, Losses: 0, Draws: 0, Spread: 200},
		&realtime.PlayerStanding{Player: "c", Wins: 1, Losses: 0, Draws: 0, Spread: 100},
		&realtime.PlayerStanding{Player: "b", Wins: 0, Losses: 1, Draws: 0, Spread: -100},
		&realtime.PlayerStanding{Player: "a", Wins: 0, Losses: 1, Draws: 0, Spread: -200},
		&realtime.PlayerStanding{Player: "e", Wins: 0, Losses: 1, Draws: 0, Spread: -300},
		&realtime.PlayerStanding{Player: "f", Wins: 0, Losses: 1, Draws: 0, Spread: -400},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	tc.RoundControls[1].Factor = 3

	err = tc.PairRound(1)
	is.NoErr(err)

	err = tc.StartRound()
	is.NoErr(err)

	// Standings should be: 1, 2, 5, 8, 7, 6, 4, 3

	err = tc.SubmitResult(1, "h", "c", 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(1, "g", "b", 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(1, "d", "a", 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(1, "e", "f", 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)
}

func TestClassicDivisionSwiss(t *testing.T) {
	// This test is used to ensure that round robin
	// pairings work correctly

	is := is.New(t)

	roundControls := []*realtime.RoundControl{}
	numberOfRounds := 7

	for i := 0; i < numberOfRounds; i++ {
		roundControls = append(roundControls, &realtime.RoundControl{FirstMethod: realtime.FirstMethod_MANUAL_FIRST,
			PairingMethod:               realtime.PairingMethod_SWISS,
			GamesPerRound:               defaultGamesPerRound,
			Round:                       int32(i),
			Factor:                      1,
			MaxRepeats:                  0,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1})
	}

	roundControls[0].PairingMethod = realtime.PairingMethod_KING_OF_THE_HILL

	tc, err := NewClassicDivision(playerStrings, playerRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	player1 := playerStrings[0]
	player2 := playerStrings[1]
	player3 := playerStrings[2]
	player4 := playerStrings[3]

	err = tc.StartRound()
	is.NoErr(err)

	err = tc.SubmitResult(0, player1, player2, 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(0, player3, player4, 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(1, player1, player3, 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(1, player2, player4, 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Since repeats only have a weight of 1,
	// player1 and player4 should be playing each other

	err = tc.SubmitResult(2, player1, player4, 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(2, player2, player3, 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	repeats, err := getRepeats(tc, 2)

	// Everyone should have played each other once at this point
	for _, v := range repeats {
		is.True(v == 1)
	}

	err = tc.SubmitResult(3, player2, player1, 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Use factor pairings to force deterministic pairings
	tc.RoundControls[4].PairingMethod = realtime.PairingMethod_FACTOR
	tc.RoundControls[4].Factor = 2

	err = tc.SubmitResult(3, player3, player4, 800, 700,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 4
	standings, err := tc.GetStandings(3)
	is.NoErr(err)

	expectedstandings := []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player1, Wins: 3, Losses: 1, Draws: 0, Spread: 800},
		&realtime.PlayerStanding{Player: player2, Wins: 3, Losses: 1, Draws: 0, Spread: 600},
		&realtime.PlayerStanding{Player: player3, Wins: 2, Losses: 2, Draws: 0, Spread: -300},
		&realtime.PlayerStanding{Player: player4, Wins: 0, Losses: 4, Draws: 0, Spread: -1100},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.SubmitResult(4, player1, player3, 900, 800,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Test that using the prohibitive weight will
	// lead to the correct pairings
	tc.RoundControls[6].AllowOverMaxRepeats = false
	tc.RoundControls[6].MaxRepeats = 2

	err = tc.SubmitResult(4, player4, player2, 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 5
	standings, err = tc.GetStandings(4)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player1, Wins: 4, Losses: 1, Draws: 0, Spread: 900},
		&realtime.PlayerStanding{Player: player2, Wins: 3, Losses: 2, Draws: 0, Spread: 300},
		&realtime.PlayerStanding{Player: player3, Wins: 2, Losses: 3, Draws: 0, Spread: -400},
		&realtime.PlayerStanding{Player: player4, Wins: 1, Losses: 4, Draws: 0, Spread: -800},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.SubmitResult(5, player1, player4, 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Once the next round is paired upon the completion
	// of round 6, an error will occur since there is
	// no possible pairing that does not give 3 repeats.
	tc.RoundControls[6].AllowOverMaxRepeats = false
	tc.RoundControls[6].MaxRepeats = 2

	err = tc.SubmitResult(5, player2, player3, 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(fmt.Sprintf("%s", err) == "prohibitive weight reached, pairings are not possible with these settings")

	tc.RoundControls[6].AllowOverMaxRepeats = true

	err = tc.PairRound(6)
	is.NoErr(err)

	roundControls = []*realtime.RoundControl{}
	numberOfRounds = 3
	// This test onlyworks for values of the form 2 ^ n
	numberOfPlayers := 32

	for i := 0; i < numberOfRounds; i++ {
		roundControls = append(roundControls, &realtime.RoundControl{FirstMethod: realtime.FirstMethod_MANUAL_FIRST,
			PairingMethod:               realtime.PairingMethod_KING_OF_THE_HILL,
			GamesPerRound:               defaultGamesPerRound,
			Round:                       int32(i),
			Factor:                      1,
			MaxRepeats:                  0,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1})
	}

	playerSwissRatings := &realtime.TournamentPersons{Persons: make(map[string]int32)}
	swissPlayers := []string{}
	for i := 1; i <= numberOfPlayers; i++ {
		swissPlayers = append(swissPlayers, fmt.Sprintf("%d", i))
		playerSwissRatings.Persons[fmt.Sprintf("%d", i)] = int32(1000 - i)
	}

	roundControls[2].PairingMethod = realtime.PairingMethod_SWISS

	tc, err = NewClassicDivision(swissPlayers, playerSwissRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	err = tc.StartRound()
	is.NoErr(err)

	for i := 0; i < numberOfPlayers; i += 2 {
		err = tc.SubmitResult(0, swissPlayers[i], swissPlayers[i+1], (numberOfPlayers*100)-i*100, 0,
			realtime.TournamentGameResult_WIN,
			realtime.TournamentGameResult_LOSS,
			realtime.GameEndReason_STANDARD, false, 0, "")
		is.NoErr(err)
	}

	for i := 0; i < numberOfPlayers; i += 4 {
		err = tc.SubmitResult(1, swissPlayers[i], swissPlayers[i+2], (numberOfPlayers*10)-i*10, 0,
			realtime.TournamentGameResult_WIN,
			realtime.TournamentGameResult_LOSS,
			realtime.GameEndReason_STANDARD, false, 0, "")
		is.NoErr(err)
	}
	for i := 1; i < numberOfPlayers; i += 4 {
		err = tc.SubmitResult(1, swissPlayers[i], swissPlayers[i+2], 0, (numberOfPlayers*10)-i*10,
			realtime.TournamentGameResult_LOSS,
			realtime.TournamentGameResult_WIN,
			realtime.GameEndReason_STANDARD, false, 0, "")
		is.NoErr(err)
	}

	// Get the standings for round 2
	standings, err = tc.GetStandings(1)
	is.NoErr(err)

	for i := 0; i < len(tc.Matrix[2]); i++ {
		pairingKey := tc.Matrix[2][i]
		pairing := tc.PairingMap[pairingKey]
		playerOne := pairing.Players[0]
		playerTwo := pairing.Players[1]
		var playerOneIndex int
		var playerTwoIndex int
		for i := 0; i < len(standings); i++ {
			standingsPlayer := standings[i].Player
			if playerOne == standingsPlayer {
				playerOneIndex = i
			} else if playerTwo == standingsPlayer {
				playerTwoIndex = i
			}
		}
		// Ensure players only played someone in with the same record
		playerOneStandings := standings[playerOneIndex]
		playerTwoStandings := standings[playerTwoIndex]
		is.True(playerOneStandings.Wins == playerTwoStandings.Wins)
		is.True(playerOneStandings.Losses == playerTwoStandings.Losses)
		is.True(playerOneStandings.Draws == playerTwoStandings.Draws)
	}
}

func TestClassicDivisionRoundRobin(t *testing.T) {
	// This test is used to ensure that round robin
	// pairings work correctly

	is := is.New(t)

	roundControls := []*realtime.RoundControl{}
	numberOfRounds := 6

	for i := 0; i < numberOfRounds; i++ {
		roundControls = append(roundControls, &realtime.RoundControl{FirstMethod: realtime.FirstMethod_MANUAL_FIRST,
			PairingMethod:               realtime.PairingMethod_ROUND_ROBIN,
			GamesPerRound:               defaultGamesPerRound,
			Round:                       int32(i),
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1})
	}

	tc, err := NewClassicDivision(playerStrings, playerRatings, roundControls, true)

	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))
	is.NoErr(validatePairings(tc, 1))
	is.NoErr(validatePairings(tc, 2))
	is.NoErr(validatePairings(tc, 3))
	is.NoErr(validatePairings(tc, 4))
	is.NoErr(validatePairings(tc, 5))

	// In a double round robin with 4 players,
	// everyone should have played everyone else twice.
	for _, player := range playerStrings {
		m := make(map[string]int)
		m[player] = 2

		for k := 0; k < len(tc.Matrix); k++ {
			opponent, err := tc.opponentOf(player, k)
			is.NoErr(err)
			m[opponent]++
		}
		for _, opponent := range playerStrings {
			var err error = nil
			if m[opponent] != 2 {
				err = fmt.Errorf("player %s didn't play %s exactly twice: %d", player, opponent, m[opponent])
			}
			is.NoErr(err)
		}
	}

	// Test Round Robin with an odd number of players (a bye)

	for i := 0; i < 4; i++ {
		roundControls = append(roundControls, &realtime.RoundControl{FirstMethod: realtime.FirstMethod_MANUAL_FIRST,
			PairingMethod:               realtime.PairingMethod_ROUND_ROBIN,
			GamesPerRound:               defaultGamesPerRound,
			Round:                       int32(i + 6),
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1})
	}

	tc, err = NewClassicDivision(playersOddStrings, playerRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))
	is.NoErr(validatePairings(tc, 1))
	is.NoErr(validatePairings(tc, 2))
	is.NoErr(validatePairings(tc, 3))
	is.NoErr(validatePairings(tc, 4))
	is.NoErr(validatePairings(tc, 5))
	is.NoErr(validatePairings(tc, 6))
	is.NoErr(validatePairings(tc, 7))
	is.NoErr(validatePairings(tc, 8))
	is.NoErr(validatePairings(tc, 9))

	// In a double round robin with 5 players,
	// everyone should have played everyone else twice
	// and everyone should have two byes
	for _, player := range playersOddStrings {
		m := make(map[string]int)
		// We don't assign the player as having played themselves
		// twice in this case because the bye will do that.

		for k := 0; k < len(tc.Matrix); k++ {
			opponent, err := tc.opponentOf(player, k)
			is.NoErr(err)
			m[opponent]++
		}
		for _, opponent := range playersOddStrings {
			var err error = nil
			if m[opponent] != 2 {
				err = fmt.Errorf("player %s didn't play %s exactly twice!", player, opponent)
			}
			is.NoErr(err)
		}
	}
}

func TestClassicDivisionInitialFontes(t *testing.T) {
	// This test only covers InitialFontes error conditions
	// and a single nonerror case. More tests can be
	// found in the pair package.

	is := is.New(t)

	roundControls := defaultRoundControls(rounds)

	for i := 1; i < rounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_INITIAL_FONTES
	}

	// InitialFontes can only be used in contiguous rounds
	// starting with round 1
	tc, err := NewClassicDivision(playerStrings, playerRatings, roundControls, true)
	is.True(err != nil)

	roundControls[0].PairingMethod = realtime.PairingMethod_INITIAL_FONTES

	// The number of InitialFontes pairings must be odd
	tc, err = NewClassicDivision(playerStrings, playerRatings, roundControls, true)
	is.True(err != nil)

	numberOfRoundsForInitialFontesTest := 4
	roundControls = defaultRoundControls(numberOfRoundsForInitialFontesTest)

	for i := 0; i < 3; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_INITIAL_FONTES
	}

	tc, err = NewClassicDivision(playerStrings, playerRatings, roundControls, true)
	is.NoErr(err)

	is.NoErr(validatePairings(tc, 0))
	is.NoErr(validatePairings(tc, 1))
	is.NoErr(validatePairings(tc, 2))
}

func TestClassicDivisionManual(t *testing.T) {
	is := is.New(t)

	roundControls := defaultRoundControls(rounds)

	roundControls[0].PairingMethod = realtime.PairingMethod_MANUAL
	roundControls[1].PairingMethod = realtime.PairingMethod_KING_OF_THE_HILL

	tc, err := NewClassicDivision(playerStrings, playerRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	player1 := playerStrings[0]
	player2 := playerStrings[1]
	player3 := playerStrings[2]
	player4 := playerStrings[3]

	// Check that round 1 is not paired
	for _, pairingKey := range tc.Matrix[0] {
		is.True(pairingKey == "")
	}

	// Pair round 1
	err = tc.SetPairing(player1, player2, 0, false)
	is.NoErr(err)
	err = tc.SetPairing(player3, player4, 0, false)
	is.NoErr(err)

	is.NoErr(validatePairings(tc, 0))

	// Amend a pairing
	err = tc.SetPairing(player2, player3, 0, false)
	is.NoErr(err)

	// Confirm that players 1 and 4 are now unpaired
	is.True(tc.Matrix[0][tc.PlayerIndexMap[player1]] == "")
	is.True(tc.Matrix[0][tc.PlayerIndexMap[player4]] == "")

	// Complete the round 1 pairings
	err = tc.SetPairing(player1, player4, 0, false)
	is.NoErr(err)

	is.NoErr(validatePairings(tc, 0))

	// Submit results for round 1

	err = tc.StartRound()
	is.NoErr(err)

	err = tc.SubmitResult(0, player2, player3, 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)
	err = tc.SubmitResult(0, player1, player4, 200, 450,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	roundIsComplete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1
	standings, err := tc.GetStandings(0)
	is.NoErr(err)

	expectedstandings := []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player4, Wins: 1, Losses: 0, Draws: 0, Spread: 250},
		&realtime.PlayerStanding{Player: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 100},
		&realtime.PlayerStanding{Player: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -100},
		&realtime.PlayerStanding{Player: player1, Wins: 0, Losses: 1, Draws: 0, Spread: -250},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	// Amend a result
	err = tc.SubmitResult(0, player1, player4, 500, 450,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, true, 0, "")
	is.NoErr(err)

	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1 again
	standings, err = tc.GetStandings(0)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 100},
		&realtime.PlayerStanding{Player: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 50},
		&realtime.PlayerStanding{Player: player4, Wins: 0, Losses: 1, Draws: 0, Spread: -50},
		&realtime.PlayerStanding{Player: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -100},
	}

	is.NoErr(equalStandings(expectedstandings, standings))
}

func TestClassicDivisionElimination(t *testing.T) {
	is := is.New(t)

	roundControls := defaultRoundControls(rounds)

	for i := 0; i < rounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_ELIMINATION
		roundControls[i].GamesPerRound = 3
	}

	// Try and make an elimination tournament with the wrong number of rounds
	tc, err := NewClassicDivision(playerStrings, playerRatings, roundControls[:1], true)
	is.True(err != nil)

	roundControls[0].PairingMethod = realtime.PairingMethod_RANDOM
	// Try and make an elimination tournament with other types
	// of pairings
	tc, err = NewClassicDivision(playerStrings, playerRatings, roundControls, true)
	is.True(err != nil)

	roundControls = defaultRoundControls(rounds)

	for i := 0; i < rounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_ELIMINATION
		roundControls[i].GamesPerRound = 3
	}

	tc, err = NewClassicDivision(playerStrings, playerRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	err = tc.StartRound()
	is.NoErr(err)

	is.NoErr(validatePairings(tc, 0))

	player1 := playerStrings[0]
	player2 := playerStrings[1]
	player3 := playerStrings[2]
	player4 := playerStrings[3]

	pairing1, err := tc.getPairing(player1, 0)
	is.NoErr(err)
	pairing2, err := tc.getPairing(player3, 0)
	is.NoErr(err)

	expectedpairing1 := newClassicPairing(tc, player1, player2, 0)
	expectedpairing2 := newClassicPairing(tc, player3, player4, 0)

	// Get the initial standings
	standings, err := tc.GetStandings(0)
	is.NoErr(err)

	// Ensure standings for Elimination are correct
	expectedstandings := []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player1, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
		&realtime.PlayerStanding{Player: player2, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
		&realtime.PlayerStanding{Player: player3, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
		&realtime.PlayerStanding{Player: player4, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	// The match is decided in two games
	err = tc.SubmitResult(0, player1, player2, 500, 490,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// The games should have changed
	expectedpairing1.Games[0].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_WIN,
			realtime.TournamentGameResult_LOSS}
	expectedpairing1.Games[0].Scores[0] = 500
	expectedpairing1.Games[0].Scores[1] = 490
	expectedpairing1.Games[0].GameEndReason = realtime.GameEndReason_STANDARD
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	err = tc.SubmitResult(0, player1, player2, 50, 0,
		realtime.TournamentGameResult_FORFEIT_WIN,
		realtime.TournamentGameResult_FORFEIT_LOSS,
		realtime.GameEndReason_FORCE_FORFEIT, false, 1, "")
	is.NoErr(err)

	// The outcomes should now be set
	expectedpairing1.Games[1].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_FORFEIT_WIN, realtime.TournamentGameResult_FORFEIT_LOSS}
	expectedpairing1.Games[1].Scores[0] = 50
	expectedpairing1.Games[1].Scores[1] = 0
	expectedpairing1.Games[1].GameEndReason = realtime.GameEndReason_FORCE_FORFEIT
	expectedpairing1.Outcomes[0] = realtime.TournamentGameResult_WIN
	expectedpairing1.Outcomes[1] = realtime.TournamentGameResult_ELIMINATED
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	roundIsComplete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(!roundIsComplete)

	// The match is decided in three games
	err = tc.SubmitResult(0, player3, player4, 500, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// The spread and games should have changed
	expectedpairing2.Games[0].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_WIN, realtime.TournamentGameResult_LOSS}
	expectedpairing2.Games[0].Scores[0] = 500
	expectedpairing2.Games[0].Scores[1] = 400
	expectedpairing2.Games[0].GameEndReason = realtime.GameEndReason_STANDARD
	is.NoErr(equalPlayerRoundInfo(expectedpairing2, pairing2))

	err = tc.SubmitResult(0, player3, player4, 400, 400,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 1, "")
	is.NoErr(err)

	// The spread and games should have changed
	expectedpairing2.Games[1].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_DRAW,
			realtime.TournamentGameResult_DRAW}
	expectedpairing2.Games[1].Scores[0] = 400
	expectedpairing2.Games[1].Scores[1] = 400
	expectedpairing2.Games[1].GameEndReason = realtime.GameEndReason_STANDARD
	is.NoErr(equalPlayerRoundInfo(expectedpairing2, pairing2))

	err = tc.SubmitResult(0, player3, player4, 450, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 2, "")
	is.NoErr(err)

	// The spread and games should have changed
	// The outcome and record should have changed
	expectedpairing2.Games[2].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_WIN, realtime.TournamentGameResult_LOSS}
	expectedpairing2.Games[2].Scores[0] = 450
	expectedpairing2.Games[2].Scores[1] = 400
	expectedpairing2.Games[2].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpairing2.Outcomes[0] = realtime.TournamentGameResult_WIN
	expectedpairing2.Outcomes[1] = realtime.TournamentGameResult_ELIMINATED
	is.NoErr(equalPlayerRoundInfo(expectedpairing2, pairing2))

	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1
	standings, err = tc.GetStandings(0)
	is.NoErr(err)

	// Elimination standings are based on wins and player order only
	// Losses are not recorded in Elimination standings
	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 60},
		&realtime.PlayerStanding{Player: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 150},
		&realtime.PlayerStanding{Player: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -60},
		&realtime.PlayerStanding{Player: player4, Wins: 0, Losses: 1, Draws: 0, Spread: -150},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	pairing1, err = tc.getPairing(player1, 1)
	is.NoErr(err)
	pairing2, err = tc.getPairing(player4, 1)
	is.NoErr(err)

	expectedpairing1 = newClassicPairing(tc, player1, player3, 1)

	// Half of the field should be eliminated

	// There should be no changes to the PRIs of players still
	// in the tournament. The Record gets carried over from
	// last round in the usual manner.
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	// The usual pri comparison method will fail since the
	// Games and Players are nil for elimianted players
	is.True(pairing2.Outcomes[0] == realtime.TournamentGameResult_ELIMINATED)
	is.True(pairing2.Outcomes[1] == realtime.TournamentGameResult_ELIMINATED)
	is.True(pairing2.Games == nil)
	is.True(pairing2.Players == nil)
	err = tc.SubmitResult(1, player1, player3, 500, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	expectedpairing1.Games[0].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_WIN, realtime.TournamentGameResult_LOSS}
	expectedpairing1.Games[0].Scores[0] = 500
	expectedpairing1.Games[0].Scores[1] = 400
	expectedpairing1.Games[0].GameEndReason = realtime.GameEndReason_STANDARD
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	err = tc.SubmitResult(1, player1, player3, 400, 600,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 1, "")
	is.NoErr(err)

	expectedpairing1.Games[1].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_LOSS,
			realtime.TournamentGameResult_WIN}
	expectedpairing1.Games[1].Scores[0] = 400
	expectedpairing1.Games[1].Scores[1] = 600
	expectedpairing1.Games[1].GameEndReason = realtime.GameEndReason_STANDARD
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	err = tc.SubmitResult(1, player1, player3, 450, 450,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 2, "")
	is.NoErr(err)

	expectedpairing1.Games[2].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_DRAW, realtime.TournamentGameResult_DRAW}
	expectedpairing1.Games[2].Scores[0] = 450
	expectedpairing1.Games[2].Scores[1] = 450
	expectedpairing1.Games[2].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpairing1.Outcomes[0] = realtime.TournamentGameResult_ELIMINATED
	expectedpairing1.Outcomes[1] = realtime.TournamentGameResult_WIN
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	roundIsComplete, err = tc.IsRoundComplete(1)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Amend a result
	err = tc.SubmitResult(1, player1, player3, 451, 450,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, true, 2, "")
	is.NoErr(err)

	expectedpairing1.Games[2].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_WIN,
			realtime.TournamentGameResult_LOSS}
	expectedpairing1.Games[2].Scores[0] = 451
	expectedpairing1.Games[2].Scores[1] = 450
	expectedpairing1.Games[2].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpairing1.Outcomes[0] = realtime.TournamentGameResult_WIN
	expectedpairing1.Outcomes[1] = realtime.TournamentGameResult_ELIMINATED
	is.NoErr(equalPlayerRoundInfo(expectedpairing1, pairing1))

	roundIsComplete, err = tc.IsRoundComplete(1)
	is.NoErr(err)
	is.True(roundIsComplete)

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(tournamentIsFinished)

	// Test ties and submitting tiebreaking results
	// Since this test is copied from above, the usual
	// validations are skipped, since they would be redundant.

	tc, err = NewClassicDivision(playerStrings, playerRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	err = tc.StartRound()
	is.NoErr(err)

	player1 = playerStrings[0]
	player2 = playerStrings[1]
	player3 = playerStrings[2]
	player4 = playerStrings[3]

	pairing2, err = tc.getPairing(player3, 0)
	is.NoErr(err)

	expectedpairing1 = newClassicPairing(tc, player1, player2, 0)
	expectedpairing2 = newClassicPairing(tc, player3, player4, 0)

	// The match is decided in two games
	err = tc.SubmitResult(0, player1, player2, 500, 490,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(0, player1, player2, 50, 0,
		realtime.TournamentGameResult_FORFEIT_WIN,
		realtime.TournamentGameResult_FORFEIT_LOSS,
		realtime.GameEndReason_FORCE_FORFEIT, false, 1, "")
	is.NoErr(err)

	// The next match ends up tied at 1.5 - 1.5
	// with both players having the same spread.
	err = tc.SubmitResult(0, player3, player4, 500, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(0, player3, player4, 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 1, "")
	is.NoErr(err)

	err = tc.SubmitResult(0, player3, player4, 500, 500,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 2, "")
	is.NoErr(err)

	expectedpairing2.Games[0].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_WIN,
			realtime.TournamentGameResult_LOSS}
	expectedpairing2.Games[1].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_LOSS,
			realtime.TournamentGameResult_WIN}
	expectedpairing2.Games[2].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_DRAW,
			realtime.TournamentGameResult_DRAW}
	expectedpairing2.Games[0].Scores[0] = 500
	expectedpairing2.Games[1].Scores[0] = 400
	expectedpairing2.Games[2].Scores[0] = 500
	expectedpairing2.Games[0].Scores[1] = 400
	expectedpairing2.Games[1].Scores[1] = 500
	expectedpairing2.Games[2].Scores[1] = 500
	expectedpairing2.Games[0].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpairing2.Games[1].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpairing2.Games[2].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpairing2.Outcomes[0] = realtime.TournamentGameResult_NO_RESULT
	expectedpairing2.Outcomes[1] = realtime.TournamentGameResult_NO_RESULT
	is.NoErr(equalPlayerRoundInfo(expectedpairing2, pairing2))

	// Round should not be over
	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(!roundIsComplete)

	// Submit a tiebreaking result, unfortunately, it's another draw
	err = tc.SubmitResult(0, player3, player4, 500, 500,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 3, "")
	is.NoErr(err)

	expectedpairing2.Games =
		append(expectedpairing2.Games,
			&realtime.TournamentGame{Scores: []int32{500, 500},
				Results: []realtime.TournamentGameResult{realtime.TournamentGameResult_DRAW,
					realtime.TournamentGameResult_DRAW}})
	expectedpairing2.Games[3].GameEndReason = realtime.GameEndReason_STANDARD
	is.NoErr(equalPlayerRoundInfo(expectedpairing2, pairing2))

	// Round should still not be over
	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(!roundIsComplete)

	// Attempt to submit a tiebreaking result, unfortunately, the game index is wrong
	err = tc.SubmitResult(0, player3, player4, 500, 500,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 5, "")
	is.True(err != nil)

	// Still wrong! Silly director (and definitely not code another layer up)
	err = tc.SubmitResult(0, player3, player4, 500, 500,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 2, "")
	is.True(err != nil)

	// Round should still not be over
	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(!roundIsComplete)

	// The players finally reach a decisive result
	err = tc.SubmitResult(0, player3, player4, 600, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 4, "")
	is.NoErr(err)

	// Round is finally over
	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1
	standings, err = tc.GetStandings(0)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 60},
		&realtime.PlayerStanding{Player: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 300},
		&realtime.PlayerStanding{Player: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -60},
		&realtime.PlayerStanding{Player: player4, Wins: 0, Losses: 1, Draws: 0, Spread: -300},
	}

	is.NoErr(equalStandings(expectedstandings, standings))
}

func TestClassicDivisionAddLatecomers(t *testing.T) {
	is := is.New(t)

	numberOfRounds := 5

	roundControls := defaultRoundControls(numberOfRounds)

	for i := 0; i < numberOfRounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_KING_OF_THE_HILL
	}

	tc, err := NewClassicDivision(playerStrings, playerRatings, roundControls, false)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	// Tournament should not be over

	player1 := playerStrings[0]
	player2 := playerStrings[1]
	player3 := playerStrings[3]
	player4 := playerStrings[2]

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(!tournamentIsFinished)

	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	err = tc.SubmitResult(0, player1, player2, 550, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(0, player3, player4, 300, 700,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")

	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	err = tc.SubmitResult(1, player1, player4, 670, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Add another player before the start of the next round
	err = tc.AddPlayers(&realtime.TournamentPersons{Persons: map[string]int32{"Bum": 50}})
	is.NoErr(err)

	err = tc.SubmitResult(1, player3, player2, 800, 700,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 2
	standings, err := tc.GetStandings(1)
	is.NoErr(err)

	expectedstandings := []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player1, Wins: 2, Losses: 0, Draws: 0, Spread: 420},
		&realtime.PlayerStanding{Player: player4, Wins: 1, Losses: 1, Draws: 0, Spread: 130},
		&realtime.PlayerStanding{Player: player3, Wins: 1, Losses: 1, Draws: 0, Spread: -300},
		&realtime.PlayerStanding{Player: "Bum", Wins: 0, Losses: 2, Draws: 0, Spread: -100},
		&realtime.PlayerStanding{Player: player2, Wins: 0, Losses: 2, Draws: 0, Spread: -250},
	}
	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	err = tc.SubmitResult(2, player1, player4, 400, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(2, player3, "Bum", 700, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// The bye result for player2 should have already been submitted
	standings, err = tc.GetStandings(2)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player1, Wins: 3, Losses: 0, Draws: 0, Spread: 520},
		&realtime.PlayerStanding{Player: player3, Wins: 2, Losses: 1, Draws: 0, Spread: 100},
		&realtime.PlayerStanding{Player: player4, Wins: 1, Losses: 2, Draws: 0, Spread: 30},
		&realtime.PlayerStanding{Player: player2, Wins: 1, Losses: 2, Draws: 0, Spread: -200},
		&realtime.PlayerStanding{Player: "Bum", Wins: 0, Losses: 3, Draws: 0, Spread: -500},
	}
	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	err = tc.SubmitResult(3, player1, player3, 400, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.AddPlayers(&realtime.TournamentPersons{Persons: map[string]int32{"Bummest": 50}})
	is.NoErr(err)

	err = tc.SubmitResult(3, player2, player4, 700, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.AddPlayers(&realtime.TournamentPersons{Persons: map[string]int32{"Bummer": 50}})
	is.NoErr(err)

	standings, err = tc.GetStandings(3)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player1, Wins: 4, Losses: 0, Draws: 0, Spread: 620},
		&realtime.PlayerStanding{Player: player2, Wins: 2, Losses: 2, Draws: 0, Spread: 200},
		&realtime.PlayerStanding{Player: player3, Wins: 2, Losses: 2, Draws: 0, Spread: 0},
		&realtime.PlayerStanding{Player: player4, Wins: 1, Losses: 3, Draws: 0, Spread: -370},
		&realtime.PlayerStanding{Player: "Bum", Wins: 1, Losses: 3, Draws: 0, Spread: -450},
		&realtime.PlayerStanding{Player: "Bummest", Wins: 0, Losses: 4, Draws: 0, Spread: -200},
		&realtime.PlayerStanding{Player: "Bummer", Wins: 0, Losses: 4, Draws: 0, Spread: -200},
	}
	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	err = tc.SubmitResult(4, player1, player2, 500, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(4, player3, player4, 300, 700,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")

	// Submit results for the round
	err = tc.SubmitResult(4, "Bum", "Bummest", 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	roundIsComplete, err := tc.IsRoundComplete(4)
	is.NoErr(err)
	is.True(roundIsComplete)

	err = tc.AddPlayers(&realtime.TournamentPersons{Persons: map[string]int32{"Guy": 50}})
	err = tc.AddPlayers(&realtime.TournamentPersons{Persons: map[string]int32{"Guyer": 400}})

	standings, err = tc.GetStandings(4)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player1, Wins: 5, Losses: 0, Draws: 0, Spread: 720},
		&realtime.PlayerStanding{Player: player2, Wins: 2, Losses: 3, Draws: 0, Spread: 100},
		&realtime.PlayerStanding{Player: player4, Wins: 2, Losses: 3, Draws: 0, Spread: 30},
		&realtime.PlayerStanding{Player: "Bum", Wins: 2, Losses: 3, Draws: 0, Spread: -250},
		&realtime.PlayerStanding{Player: player3, Wins: 2, Losses: 3, Draws: 0, Spread: -400},
		&realtime.PlayerStanding{Player: "Bummer", Wins: 1, Losses: 4, Draws: 0, Spread: -150},
		&realtime.PlayerStanding{Player: "Guy", Wins: 0, Losses: 5, Draws: 0, Spread: -250},
		&realtime.PlayerStanding{Player: "Guyer", Wins: 0, Losses: 5, Draws: 0, Spread: -250},
		&realtime.PlayerStanding{Player: "Bummest", Wins: 0, Losses: 5, Draws: 0, Spread: -400},
	}
	is.NoErr(equalStandings(expectedstandings, standings))
}

func TestClassicDivisionRemovePlayers(t *testing.T) {

	// Test Plan:

	// Player is in first
	// Player is removed - no present results, current round is repaired
	// Player is now in last
	// Ensure past rounds are not repaired
	// Prepaired future rounds become byes for the opponents
	// standings dependent pairings show player in last
	// Remove another player - present results for round, current round is not repaired
	// removed players always have forfeits, there are no byes
	// Remove more than one player at a time

	is := is.New(t)

	numberOfRounds := 12
	roundControls := defaultRoundControls(numberOfRounds)

	for i := 0; i < numberOfRounds; i++ {
		if i >= 2 && i <= 4 {
			roundControls[i].PairingMethod = realtime.PairingMethod_ROUND_ROBIN
		} else {
			roundControls[i].PairingMethod = realtime.PairingMethod_KING_OF_THE_HILL
		}
	}

	tc, err := NewClassicDivision(playerStrings, playerRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	err = tc.StartRound()
	is.NoErr(err)

	player1 := playerStrings[0]
	player2 := playerStrings[1]
	player3 := playerStrings[3]
	player4 := playerStrings[2]

	err = tc.SubmitResult(0, player1, player2, 500, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(0, player3, player4, 500, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	standings, err := tc.GetStandings(0)
	is.NoErr(err)

	expectedstandings := []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 200},
		&realtime.PlayerStanding{Player: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 100},
		&realtime.PlayerStanding{Player: player4, Wins: 0, Losses: 1, Draws: 0, Spread: -100},
		&realtime.PlayerStanding{Player: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -200},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.SubmitResult(1, player1, player3, 500, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(1, player2, player4, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.RemovePlayers(&realtime.TournamentPersons{Persons: map[string]int32{player1: 50}})
	is.NoErr(err)
	standings, err = tc.GetStandings(1)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player2, Wins: 1, Losses: 1, Draws: 0, Spread: 0},
		&realtime.PlayerStanding{Player: player3, Wins: 1, Losses: 1, Draws: 0, Spread: -100},
		&realtime.PlayerStanding{Player: player4, Wins: 0, Losses: 2, Draws: 0, Spread: -300},
		&realtime.PlayerStanding{Player: player1, Wins: 2, Losses: 0, Draws: 0, Spread: 400},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.SubmitResult(2, player3, player4, 500, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")

	err = tc.SubmitResult(2, player1, player2, 400, 400,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 0, "")

	// At this point the removed player is assigned forfeits

	err = tc.SubmitResult(3, player2, player4, 500, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(4, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	standings, err = tc.GetStandings(4)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player2, Wins: 3, Losses: 1, Draws: 1, Spread: 400},
		&realtime.PlayerStanding{Player: player3, Wins: 3, Losses: 2, Draws: 0, Spread: -150},
		&realtime.PlayerStanding{Player: player4, Wins: 1, Losses: 4, Draws: 0, Spread: -550},
		&realtime.PlayerStanding{Player: player1, Wins: 2, Losses: 2, Draws: 1, Spread: 300},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.SubmitResult(5, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.RemovePlayers(&realtime.TournamentPersons{Persons: map[string]int32{player4: 40}})
	is.NoErr(err)

	is.True(tc.PlayersProperties[tc.PlayerIndexMap[player1]].Removed)
	is.True(tc.PlayersProperties[tc.PlayerIndexMap[player4]].Removed)

	// Since this round had results, player4's bye against player1 remain unchanged
	pairing, err := tc.getPairing(player4, 5)
	is.NoErr(err)
	is.True(pairing.Games[0].Results[0] == realtime.TournamentGameResult_BYE)
	is.True(pairing.Games[0].Results[1] == realtime.TournamentGameResult_BYE)

	pairing, err = tc.getPairing(player1, 5)
	is.NoErr(err)
	is.True(pairing.Games[0].Results[0] == realtime.TournamentGameResult_FORFEIT_LOSS)
	is.True(pairing.Games[0].Results[1] == realtime.TournamentGameResult_FORFEIT_LOSS)

	standings, err = tc.GetStandings(5)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player2, Wins: 4, Losses: 1, Draws: 1, Spread: 600},
		&realtime.PlayerStanding{Player: player3, Wins: 3, Losses: 3, Draws: 0, Spread: -350},
		&realtime.PlayerStanding{Player: player1, Wins: 2, Losses: 3, Draws: 1, Spread: 250},
		&realtime.PlayerStanding{Player: player4, Wins: 2, Losses: 4, Draws: 0, Spread: -500},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	pairing, err = tc.getPairing(player1, 6)
	is.NoErr(err)
	is.True(pairing.Games[0].Results[0] == realtime.TournamentGameResult_FORFEIT_LOSS)
	is.True(pairing.Games[0].Results[1] == realtime.TournamentGameResult_FORFEIT_LOSS)

	err = tc.SubmitResult(6, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Starting Round 7 player4 gets forfeit losses

	err = tc.SubmitResult(7, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	err = tc.SubmitResult(8, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	standings, err = tc.GetStandings(8)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player2, Wins: 7, Losses: 1, Draws: 1, Spread: 1200},
		&realtime.PlayerStanding{Player: player3, Wins: 3, Losses: 6, Draws: 0, Spread: -950},
		&realtime.PlayerStanding{Player: player4, Wins: 3, Losses: 6, Draws: 0, Spread: -550},
		&realtime.PlayerStanding{Player: player1, Wins: 2, Losses: 6, Draws: 1, Spread: 100},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.RemovePlayers(&realtime.TournamentPersons{Persons: map[string]int32{player2: 10, player3: 60}})
	is.True(fmt.Sprintf("%s", err) == "cannot remove players as tournament would be empty")

	// Idiot director removed all but one player from the tournament
	err = tc.RemovePlayers(&realtime.TournamentPersons{Persons: map[string]int32{player2: 10}})

	err = tc.SubmitResult(9, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Tournament is now over as remaining player gets byes for the rest of the event

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(tournamentIsFinished)

	standings, err = tc.GetStandings(11)
	is.NoErr(err)

	expectedstandings = []*realtime.PlayerStanding{&realtime.PlayerStanding{Player: player3, Wins: 5, Losses: 7, Draws: 0, Spread: -1050},
		&realtime.PlayerStanding{Player: player2, Wins: 8, Losses: 3, Draws: 1, Spread: 1300},
		&realtime.PlayerStanding{Player: player4, Wins: 3, Losses: 9, Draws: 0, Spread: -700},
		&realtime.PlayerStanding{Player: player1, Wins: 2, Losses: 9, Draws: 1, Spread: -50},
	}
	is.NoErr(equalStandings(expectedstandings, standings))
}

func TestClassicDivisionFirsts(t *testing.T) {
	// Test
	//   Manual sets the correct firsts
	//   Random at least works
	//   Automatic pairs correctly when
	//     firsts aren't tied
	//     firsts are tied and seconds aren't tied
	//     firsts and seconds are tied
	//   Byes, forfeits do not change the first/second values
	is := is.New(t)

	firstRounds := 10

	firsts := []realtime.FirstMethod{realtime.FirstMethod_MANUAL_FIRST,
		realtime.FirstMethod_MANUAL_FIRST,
		realtime.FirstMethod_AUTOMATIC_FIRST,
		realtime.FirstMethod_AUTOMATIC_FIRST,
		realtime.FirstMethod_MANUAL_FIRST,
		realtime.FirstMethod_MANUAL_FIRST,
		realtime.FirstMethod_AUTOMATIC_FIRST,
		realtime.FirstMethod_AUTOMATIC_FIRST,
		realtime.FirstMethod_AUTOMATIC_FIRST,
		realtime.FirstMethod_RANDOM_FIRST}

	roundControls := []*realtime.RoundControl{}

	for i := 0; i < firstRounds; i++ {
		roundControls = append(roundControls, &realtime.RoundControl{FirstMethod: firsts[i],
			PairingMethod:               realtime.PairingMethod_MANUAL,
			GamesPerRound:               defaultGamesPerRound,
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1})
	}

	tc, err := NewClassicDivision(playerStrings, playerRatings, roundControls, false)
	is.NoErr(err)
	is.True(tc != nil)

	player1 := playerStrings[0]
	player2 := playerStrings[1]
	player3 := playerStrings[2]
	player4 := playerStrings[3]

	// Pair round 0

	playerOrder := []string{player1, player2, player3, player4}
	fs := []int{1, 0, 0, 1, 1, 0, 0, 1}
	round := 0
	is.NoErr(runFirstMethodRound(tc, playerOrder, fs, round, false))

	// Pair round 1

	playerOrder = []string{player1, player2, player3, player4}
	fs = []int{2, 0, 0, 2, 2, 0, 0, 2}
	round = 1
	is.NoErr(runFirstMethodRound(tc, playerOrder, fs, round, false))

	// Next two rounds should even out firsts and seconds
	// since they are automatic
	// Pair round 2

	playerOrder = []string{player2, player1, player4, player3}
	fs = []int{1, 2, 2, 1, 1, 2, 2, 1}
	round = 2
	is.NoErr(runFirstMethodRound(tc, playerOrder, fs, round, false))

	// Pair round 3

	playerOrder = []string{player2, player1, player4, player3}
	fs = []int{2, 2, 2, 2, 2, 2, 2, 2}
	round = 3
	is.NoErr(runFirstMethodRound(tc, playerOrder, fs, round, false))

	// Make a bye so we can setup the same number of firsts
	// but a different number of seconds
	// Pair round4

	playerOrder = []string{player1, player2, player3, player4}
	fs = []int{3, 2, 2, 3, 2, 2, 2, 2}
	round = 4
	is.NoErr(runFirstMethodRound(tc, playerOrder, fs, round, true))

	// Pair round 5

	playerOrder = []string{player1, player3, player2, player4}
	fs = []int{4, 2, 2, 3, 2, 3, 2, 2}
	round = 5
	is.NoErr(runFirstMethodRound(tc, playerOrder, fs, round, true))

	// Round 6 tests that if firsts are tied, seconds are a tiebreaker
	// Pair round 6

	playerOrder = []string{player2, player1, player4, player3}
	fs = []int{3, 3, 4, 3, 2, 3, 3, 3}
	round = 6
	is.NoErr(runFirstMethodRound(tc, playerOrder, fs, round, false))

	// Pair round 7

	playerOrder = []string{player2, player4, player3, player1}
	fs = []int{3, 4, 3, 3, 3, 3, 4, 3}
	round = 7
	is.NoErr(runFirstMethodRound(tc, playerOrder, fs, round, true))
}

func runFirstMethodRound(tc *ClassicDivision, playerOrder []string, fs []int, round int, useByes bool) error {
	err := tc.SetPairing(playerOrder[0], playerOrder[1], round, false)

	if err != nil {
		return err
	}

	if useByes {
		err = tc.SetPairing(playerOrder[2], playerOrder[2], round, false)

		if err != nil {
			return err
		}
		err = tc.SetPairing(playerOrder[3], playerOrder[3], round, false)

		if err != nil {
			return err
		}
	} else {
		err = tc.SetPairing(playerOrder[2], playerOrder[3], round, false)

		if err != nil {
			return err
		}
	}

	err = tc.StartRound()
	if err != nil {
		return err
	}

	err = completeManualRound(tc, round, playerOrder[0], playerOrder[1], playerOrder[2], playerOrder[3], useByes)

	if err != nil {
		return err
	}

	return checkFirsts(tc, playerOrder, fs, round)
}

func completeManualRound(tc *ClassicDivision, round int, player1 string, player2 string, player3 string, player4 string, useByes bool) error {

	err := tc.SubmitResult(round, player1, player2, 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")

	if err != nil {
		return err
	}

	// Results for byes are automatically submitted
	if !useByes {
		err = tc.SubmitResult(round, player3, player4, 200, 450,
			realtime.TournamentGameResult_LOSS,
			realtime.TournamentGameResult_WIN,
			realtime.GameEndReason_STANDARD, false, 0, "")

		if err != nil {
			return err
		}
	}

	roundIsComplete, err := tc.IsRoundComplete(round)
	if err != nil {
		return err
	}

	if !roundIsComplete {
		return fmt.Errorf("round %d is not complete.", round)
	}

	return err
}

func checkFirsts(tc *ClassicDivision, players []string, fs []int, round int) error {

	actualfs := []int{}

	for i := 0; i < len(players); i++ {
		playerfs := getPlayerFS(tc, players[i], round)
		actualfs = append(actualfs, playerfs...)
	}

	for i := 0; i < len(actualfs); i++ {
		if actualfs[i] != fs[i] {
			actualfsString := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(actualfs)), ", "), "[]")
			fsString := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(fs)), ", "), "[]")
			return fmt.Errorf("firsts and Seconds are not equal in round %d: %s, %s", round, fsString, actualfsString)
		}
	}
	return nil
}

func TestClassicDivisionRandomData(t *testing.T) {
	t.Skip()
	is := is.New(t)

	is.NoErr(runRandomTournaments(realtime.PairingMethod_RANDOM, false))
	is.NoErr(runRandomTournaments(realtime.PairingMethod_ROUND_ROBIN, false))
	is.NoErr(runRandomTournaments(realtime.PairingMethod_KING_OF_THE_HILL, false))
	is.NoErr(runRandomTournaments(realtime.PairingMethod_ELIMINATION, false))
	// Randomize the pairing method for each round
	// Given pairing method is irrelevant
	is.NoErr(runRandomTournaments(realtime.PairingMethod_MANUAL, true))
}

func runRandomTournaments(method realtime.PairingMethod, randomizePairings bool) error {
	for numberOfPlayers := 2; numberOfPlayers <= 512; numberOfPlayers++ {
		var numberOfRounds int
		if method == realtime.PairingMethod_ELIMINATION {
			// numberOfRounds will be -1 if number of players
			// is not of the form 2 ^ n
			numberOfRounds = logTwo(numberOfPlayers)
			if numberOfRounds < 0 {
				continue
			}
		} else {
			numberOfRounds = rand.Intn(10) + 10
		}

		roundControls := []*realtime.RoundControl{}

		for i := 0; i < numberOfRounds; i++ {
			roundControls = append(roundControls, &realtime.RoundControl{FirstMethod: realtime.FirstMethod_MANUAL_FIRST,
				PairingMethod:               method,
				GamesPerRound:               1,
				Factor:                      1,
				MaxRepeats:                  1,
				AllowOverMaxRepeats:         true,
				RepeatRelativeWeight:        1,
				WinDifferenceRelativeWeight: 1})
		}

		playersRatingsRandom := &realtime.TournamentPersons{Persons: make(map[string]int32)}
		playersRandom := []string{}
		for i := 0; i < numberOfPlayers; i++ {
			playersRandom = append(playersRandom, fmt.Sprintf("%d", i))
			playersRatingsRandom.Persons[fmt.Sprintf("%d", i)] = int32(i)
		}

		if method != realtime.PairingMethod_ELIMINATION {
			for i := 0; i < numberOfRounds; i++ {
				roundControls[i].FirstMethod = realtime.FirstMethod(rand.Intn(3))
			}
		}

		if randomizePairings {
			for i := 0; i < numberOfRounds; i++ {
				roundControls[i].PairingMethod = realtime.PairingMethod(rand.Intn(3))
			}
		}

		if method == realtime.PairingMethod_ELIMINATION {
			// Even-numbered games per rounds was leading to ties
			// which give inconclusive results, therefor not ending the round
			for i := 0; i < numberOfRounds; i++ {
				roundControls[i].GamesPerRound = int32((rand.Intn(5) * 2) + 1)
			}
		}

		tc, err := NewClassicDivision(playersRandom, playersRatingsRandom, roundControls, true)
		if err != nil {
			return err
		}

		for round := 0; round < numberOfRounds; round++ {

			err = validatePairings(tc, round)

			if err != nil {
				return err
			}

			pairings := tc.getPlayerPairings(round)
			for game := 0; game < int(tc.RoundControls[round].GamesPerRound); game++ {
				for l := 0; l < len(pairings); l += 2 {

					// The outcome might already be decided in an elimination tournament, skip the submission
					if method == realtime.PairingMethod_ELIMINATION {
						pairing, err := tc.getPairing(pairings[l], round)
						if err != nil {
							return err
						}
						if pairing.Outcomes[0] != realtime.TournamentGameResult_NO_RESULT &&
							pairing.Outcomes[1] != realtime.TournamentGameResult_NO_RESULT {
							continue
						}
					}

					// Byes have the results automatically submitted
					if pairings[l] == pairings[l+1] {
						continue
					}

					// For Elimination tournaments, force a decisive result
					// otherwise the round may not be over when we check for it
					var res1 realtime.TournamentGameResult
					var res2 realtime.TournamentGameResult
					if method == realtime.PairingMethod_ELIMINATION {
						if rand.Intn(2) == 0 {
							res1 = realtime.TournamentGameResult_WIN
							res2 = realtime.TournamentGameResult_LOSS
						} else {
							res1 = realtime.TournamentGameResult_LOSS
							res2 = realtime.TournamentGameResult_WIN
						}
					} else {
						res1 = realtime.TournamentGameResult(rand.Intn(6) + 1)
						res2 = realtime.TournamentGameResult(rand.Intn(6) + 1)
					}

					err = tc.SubmitResult(round,
						pairings[l],
						pairings[l+1],
						rand.Intn(300)+300,
						rand.Intn(300)+300,
						res1,
						res2,
						realtime.GameEndReason_STANDARD,
						false,
						game,
						"")
					if err != nil {
						fmt.Printf("(%d) error on round %d game %d pairing (%s, %s)",
							numberOfPlayers,
							round,
							game,
							pairings[l],
							pairings[l+1])
						return err
					}
				} // Pairings
			} // Games

			// Skip testing amendments for elimination events here.
			// Because of the random data a match may have gone from
			// decided to undecided based on the amendment. This will
			// cause a failure when the round is checked for completion.
			if method != realtime.PairingMethod_ELIMINATION {
				numberOfAmendments := rand.Intn(5)
				for l := 0; l < numberOfAmendments; l++ {
					randPairing := rand.Intn(len(pairings)/2) * 2
					err = tc.SubmitResult(round,
						pairings[randPairing],
						pairings[randPairing+1],
						rand.Intn(300)+300,
						rand.Intn(300)+300,
						realtime.TournamentGameResult(rand.Intn(6)+1),
						realtime.TournamentGameResult(rand.Intn(6)+1),
						realtime.GameEndReason_STANDARD,
						true,
						rand.Intn(int(tc.RoundControls[round].GamesPerRound)),
						"")
					if err != nil {
						return err
					}
				} // Amendments
			}

			roundIsComplete, err := tc.IsRoundComplete(round)
			if err != nil {
				return err
			}
			if !roundIsComplete {
				return fmt.Errorf("(%d) round %d is not complete (%d, %d)",
					numberOfPlayers, round, method, numberOfPlayers)
			}

			_, err = tc.GetStandings(round)
			if err != nil {
				return err
			}

		} // Tournament
		tournamentIsFinished, err := tc.IsFinished()
		if err != nil {
			return err
		}
		if !tournamentIsFinished {
			return fmt.Errorf("tournament is not complete (%d, %d)",
				method, numberOfPlayers)
		}
		if tc.RoundControls[0].PairingMethod == realtime.PairingMethod_ELIMINATION {
			standings, err := tc.GetStandings(numberOfRounds - 1)
			if err != nil {
				return err
			}
			bottomHalfSize := numberOfPlayers / 2
			eliminationPlayerIndex := numberOfPlayers - 1
			eliminatedInRound := 0
			for bottomHalfSize > 0 {
				for i := 0; i < bottomHalfSize; i++ {
					if int(standings[eliminationPlayerIndex].Wins) != eliminatedInRound {
						return fmt.Errorf("player has incorrect number of wins (%d, %d, %d)",
							eliminationPlayerIndex,
							eliminatedInRound,
							standings[eliminationPlayerIndex].Wins)
					}
					eliminationPlayerIndex--
				}
				eliminatedInRound++
				bottomHalfSize = bottomHalfSize / 2
			}
		}
	} // Number of players
	return nil
}

func validatePairings(tc *ClassicDivision, round int) error {
	// For each pairing, check that
	//   - Player's opponent is nonnull
	//   - Player's opponent's opponent is the player

	if round < 0 || round >= len(tc.Matrix) {
		return fmt.Errorf("round number out of range: %d", round)
	}

	for i, pairingKey := range tc.Matrix[round] {
		if pairingKey == "" {
			return fmt.Errorf("round %d player %d pairing nil", round, i)
		}
		pairing, ok := tc.PairingMap[pairingKey]
		if !ok {
			return fmt.Errorf("pairing key does not exist in pairing map: %s", pairingKey)
		}
		if pairing.Players == nil {
			// Some pairings can be nil for Elimination tournaments
			if tc.RoundControls[0].PairingMethod != realtime.PairingMethod_ELIMINATION {
				return fmt.Errorf("player %d is unpaired", i)
			} else {
				continue
			}
		}
		// Check that the pairing refs are correct
		opponent, err := tc.opponentOf(tc.Players[i], round)
		if err != nil {
			return err
		}
		opponentOpponent, err := tc.opponentOf(opponent, round)
		if err != nil {
			return err
		}
		if tc.Players[i] != opponentOpponent {
			return fmt.Errorf("player %s's opponent's (%s) opponent (%s) is not themself.",
				tc.Players[i],
				opponent,
				opponentOpponent)
		}
	}
	return nil
}

func equalStandings(sa1 []*realtime.PlayerStanding, sa2 []*realtime.PlayerStanding) error {

	if len(sa1) != len(sa2) {
		return fmt.Errorf("length of the standings are not equal: %d != %d", len(sa1), len(sa2))
	}

	for i := 0; i < len(sa1); i++ {
		s1 := sa1[i]
		s2 := sa2[i]
		err := equalStandingsRecord(s1, s2)
		if err != nil {
			return err
		}
	}
	return nil
}

func equalStandingsRecord(s1 *realtime.PlayerStanding, s2 *realtime.PlayerStanding) error {
	if s1.Player != s2.Player ||
		s1.Wins != s2.Wins ||
		s1.Losses != s2.Losses ||
		s1.Draws != s2.Draws ||
		s1.Spread != s2.Spread {
		return fmt.Errorf("standings do not match: (%s, %d, %d, %d, %d) != (%s, %d, %d, %d, %d)",
			s1.Player, s1.Wins, s1.Losses, s1.Draws, s1.Spread,
			s2.Player, s2.Wins, s2.Losses, s2.Draws, s2.Spread)
	}
	return nil
}

func (tc *ClassicDivision) getPlayerPairings(round int) []string {
	players := tc.Players
	pairingKeys := tc.Matrix[round]
	m := make(map[string]int)
	for _, player := range players {
		m[player] = 0
	}

	playerPairings := []string{}
	for _, pk := range pairingKeys {
		// An eliminated player could have nil for Players, skip them
		pairing := tc.PairingMap[pk]
		if pairing.Players != nil && m[pairing.Players[0]] == 0 {
			playerPairings = append(playerPairings, pairing.Players[0])
			playerPairings = append(playerPairings, pairing.Players[1])
			m[pairing.Players[0]] = 1
			m[pairing.Players[1]] = 1
		}
	}
	return playerPairings
}

func equalPlayerRoundInfo(p1 *realtime.PlayerRoundInfo, p2 *realtime.PlayerRoundInfo) error {
	// We are not concerned with ordering
	// Firsts and seconds are tested independently
	if (p1.Players[0] != p2.Players[0] && p1.Players[0] != p2.Players[1]) ||
		(p1.Players[1] != p2.Players[0] && p1.Players[1] != p2.Players[1]) {
		return fmt.Errorf("players are not the same: (%s, %s) != (%s, %s)",
			p1.Players[0],
			p1.Players[1],
			p2.Players[0],
			p2.Players[1])
	}
	if p1.Outcomes[0] != p2.Outcomes[0] || p1.Outcomes[1] != p2.Outcomes[1] {
		return fmt.Errorf("outcomes are not the same: (%d, %d) != (%d, %d)",
			p1.Outcomes[0],
			p1.Outcomes[1],
			p2.Outcomes[0],
			p2.Outcomes[1])
	}
	if len(p1.Games) != len(p2.Games) {
		return fmt.Errorf("number of games are not the same: %d != %d", len(p1.Games), len(p2.Games))
	}
	for i := 0; i < len(p1.Games); i++ {
		err := equalTournamentGame(p1.Games[i], p2.Games[i], i)
		if err != nil {
			return err
		}
	}
	return nil
}

func equalTournamentGame(t1 *realtime.TournamentGame, t2 *realtime.TournamentGame, i int) error {
	if t1.Scores[0] != t2.Scores[0] || t1.Scores[1] != t2.Scores[1] {
		return fmt.Errorf("scores are not the same at game %d: (%d, %d) != (%d, %d)",
			i,
			t1.Scores[0],
			t1.Scores[1],
			t2.Scores[0],
			t2.Scores[1])
	}
	if t1.Results[0] != t2.Results[0] || t1.Results[1] != t2.Results[1] {
		return fmt.Errorf("results are not the same at game %d: (%d, %d) != (%d, %d)",
			i,
			t1.Results[0],
			t1.Results[1],
			t2.Results[0],
			t2.Results[1])
	}
	if t1.GameEndReason != t2.GameEndReason {
		return fmt.Errorf("game end reasons are not the same for game %d: %d != %d", i, t1.GameEndReason, t2.GameEndReason)
	}
	return nil
}

func defaultRoundControls(numberOfRounds int) []*realtime.RoundControl {
	roundControls := []*realtime.RoundControl{}
	for i := 0; i < numberOfRounds; i++ {
		roundControls = append(roundControls, &realtime.RoundControl{FirstMethod: realtime.FirstMethod_MANUAL_FIRST,
			PairingMethod:               realtime.PairingMethod_RANDOM,
			GamesPerRound:               defaultGamesPerRound,
			Round:                       int32(i),
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1})
	}
	return roundControls
}

func (tc *ClassicDivision) printPriPairings(round int) {
	for _, pk := range tc.Matrix[round] {
		fmt.Printf("%p ", tc.PairingMap[pk])
		fmt.Println(tc.PairingMap[pk])
	}
}

func printStandings(standings []*realtime.PlayerStanding) {
	for _, standing := range standings {
		fmt.Println(standing)
	}
}

func logTwo(n int) int {
	res := 0
	for n > 1 {
		if n%2 != 0 {
			return -1
		}
		res++
		n = n / 2
	}
	return res
}
