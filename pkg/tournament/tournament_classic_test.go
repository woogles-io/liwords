package tournament

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/matryer/is"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/utilities"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

var defaultPlayers = makeTournamentPersons(map[string]int32{"Will": 10000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100})
var defaultPlayersOdd = makeTournamentPersons(map[string]int32{"Will": 10000, "Josh": 3000, "Conrad": 2200, "Jesse": 2100, "Matt": 2000})
var defaultRounds = 2
var defaultGamesPerRound int32 = 1

func TestClassicDivisionZeroOrOnePlayers(t *testing.T) {
	// Division creation with zero or one players is a special
	// case that should not fail
	is := is.New(t)
	playerOZRatings := makeTournamentPersons(map[string]int32{"One": 1000, "Two": 3000, "Three": 2200, "Jesse": 2100, "Matt": 100})

	_, err := compactNewClassicDivision(playerOZRatings, defaultRoundControls(0), true)
	is.True(err != nil)

	_, err = compactNewClassicDivision(playerOZRatings, defaultRoundControls(1), true)
	is.NoErr(err)

	_, err = compactNewClassicDivision(playerOZRatings, defaultRoundControls(2), true)
	is.NoErr(err)

	_, err = compactNewClassicDivision(playerOZRatings, defaultRoundControls(2), true)
	is.NoErr(err)
}

func TestClassicDivisionRandom(t *testing.T) {
	// This test attempts to cover the basic
	// functions of a Classic Tournament
	is := is.New(t)

	roundControls := defaultRoundControls(defaultRounds)

	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	// Test getting a nonexistent round
	_, err = tc.getPairing("Josh", 9)
	is.True(err != nil)

	// Test getting a nonexistent player
	_, err = tc.getPairing("No one", 1)
	is.True(err != nil)

	player1 := defaultPlayers.Persons[0].Id
	player2 := defaultPlayers.Persons[1].Id
	player3 := defaultPlayers.Persons[2].Id
	player4 := defaultPlayers.Persons[3].Id

	// Set pairings to test more easily
	_, err = tc.SetPairing(player1, player2, 0)
	is.NoErr(err)
	_, err = tc.SetPairing(player3, player4, 0)
	is.NoErr(err)

	pairing1, err := tc.getPairing(player1, 0)
	is.NoErr(err)
	pairing2, err := tc.getPairing(player3, 0)
	is.NoErr(err)

	expectedpairing1 := newClassicPairing(tc, 0, 1, 0)
	expectedpairing2 := newClassicPairing(tc, 2, 3, 0)

	// Submit result for an unpaired round
	_, err = tc.SubmitResult(1, player1, player2, 10000, -40, realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS, realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	// Submit result for players that didn't play each other
	_, err = tc.SubmitResult(0, player1, player3, 10000, -40, realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS, realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	// Submit a result for game index that is out of range
	_, err = tc.SubmitResult(0, player1, player2, 10000, -40, realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS, realtime.GameEndReason_STANDARD, false, 4, "")
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	// Submit a result before the tournament has started
	_, err = tc.SubmitResult(0, player1, player2, 10000, -40, realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS, realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	err = tc.StartRound()
	is.NoErr(err)

	// Submit a result for a paired round
	_, err = tc.SubmitResult(0, player1, player2, 10000, -40, realtime.TournamentGameResult_WIN,
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
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	// Attempt to submit the same result
	_, err = tc.SubmitResult(0, player1, player2, 10000, -40, realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS, realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	// Round 2 should not have been paired,
	// so attempting to submit a result for
	// it will throw an error.
	_, err = tc.SubmitResult(1, player1, player2, 10000, -40,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	// Amend the result
	_, err = tc.SubmitResult(0, player1, player2, 30, 900,
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
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	// Submit the final result for round 1
	_, err = tc.SubmitResult(0, player3, player4, 1, 1,
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
	is.NoErr(equalPairings(expectedpairing2, pairing2))

	roundIsComplete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Set pairings to test more easily
	_, err = tc.SetPairing(player1, player2, 1)
	is.NoErr(err)
	_, err = tc.SetPairing(player3, player4, 1)
	is.NoErr(err)

	pairing1, err = tc.getPairing(player1, 1)
	is.NoErr(err)
	pairing2, err = tc.getPairing(player3, 1)
	is.NoErr(err)

	expectedpairing1 = newClassicPairing(tc, 0, 1, 1)
	expectedpairing2 = newClassicPairing(tc, 2, 3, 1)

	// Round 2 should have been paired,
	// submit a result

	_, err = tc.SubmitResult(1, player1, player2, 0, 0,
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
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	// Submit the final tournament results
	_, err = tc.SubmitResult(1, player3, player4, 50, 50,
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
	is.NoErr(equalPairings(expectedpairing2, pairing2))

	roundIsComplete, err = tc.IsRoundComplete(1)
	is.NoErr(err)
	is.True(roundIsComplete)

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(tournamentIsFinished)

	// Attempt to get the Standings
	// for an out of range round number
	_, err = tc.GetStandings(8, false)
	is.True(err != nil)

	// Standings are tested in the
	// King of the Hill Classic Tournament test.

	// Get the standings for round 1
	_, err = tc.GetStandings(0, false)
	is.NoErr(err)

	// Get the standings for round 2
	_, err = tc.GetStandings(1, false)
	is.NoErr(err)

	oddPlayers := makeTournamentPersons(map[string]int32{"One": 1000, "Two": 3000, "Three": 2200, "Jesse": 2100, "Matt": 100})
	// Check that pairings are correct with an odd number of players
	tc, err = compactNewClassicDivision(oddPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))
}

func TestClassicDivisionSpreadCap(t *testing.T) {
	// This test is used to ensure that the standings are
	// calculated correctly and that King of the Hill
	// pairings are correct

	is := is.New(t)

	roundControls := defaultRoundControls(defaultRounds)

	for i := 0; i < defaultRounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_KING_OF_THE_HILL
	}

	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	// Change Spread Cap
	tc.DivisionControls.SpreadCap = 200

	// Tournament should not be over

	player1 := defaultPlayers.Persons[0].Id
	player2 := defaultPlayers.Persons[1].Id
	player3 := defaultPlayers.Persons[2].Id
	player4 := defaultPlayers.Persons[3].Id

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(!tournamentIsFinished)
	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	_, err = tc.SubmitResult(0, player1, player2, 500, 301,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, player3, player4, 300, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 1
	standings, err := tc.GetStandings(0, false)
	is.NoErr(err)

	expectedstandings := &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player4, Wins: 1, Losses: 0, Draws: 0, Spread: 200},
		{PlayerId: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 199},
		{PlayerId: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -199},
		{PlayerId: player3, Wins: 0, Losses: 1, Draws: 0, Spread: -200},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	// Submit results for the round
	_, err = tc.SubmitResult(1, player1, player4, 601, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(1, player3, player2, 250, 200,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 2
	standings, err = tc.GetStandings(1, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{
		{PlayerId: player1, Wins: 2, Losses: 0, Draws: 0, Spread: 399},
		{PlayerId: player4, Wins: 1, Losses: 1, Draws: 0, Spread: 0},
		{PlayerId: player3, Wins: 1, Losses: 1, Draws: 0, Spread: -150},
		{PlayerId: player2, Wins: 0, Losses: 2, Draws: 0, Spread: -249},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))
}

func TestClassicDivisionKingOfTheHill(t *testing.T) {
	// This test is used to ensure that the standings are
	// calculated correctly and that King of the Hill
	// pairings are correct

	is := is.New(t)

	roundControls := defaultRoundControls(defaultRounds)

	for i := 0; i < defaultRounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_KING_OF_THE_HILL
	}

	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	// Tournament should not be over

	player1 := defaultPlayers.Persons[0].Id
	player2 := defaultPlayers.Persons[1].Id
	player3 := defaultPlayers.Persons[2].Id
	player4 := defaultPlayers.Persons[3].Id

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(!tournamentIsFinished)
	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	_, err = tc.SubmitResult(0, player1, player2, 550, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, player3, player4, 300, 700,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 1
	standings, err := tc.GetStandings(0, false)
	is.NoErr(err)

	expectedstandings := &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player4, Wins: 1, Losses: 0, Draws: 0, Spread: 400},
		{PlayerId: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 150},
		{PlayerId: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -150},
		{PlayerId: player3, Wins: 0, Losses: 1, Draws: 0, Spread: -400},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	// The next round should have been paired
	// Tournament should not be over

	tournamentIsFinished, err = tc.IsFinished()
	is.NoErr(err)
	is.True(!tournamentIsFinished)

	// Submit results for the round
	_, err = tc.SubmitResult(1, player1, player4, 670, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(1, player3, player2, 700, 700,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 2
	standings, err = tc.GetStandings(1, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player1, Wins: 2, Losses: 0, Draws: 0, Spread: 420},
		{PlayerId: player4, Wins: 1, Losses: 1, Draws: 0, Spread: 130},
		{PlayerId: player2, Wins: 0, Losses: 1, Draws: 1, Spread: -150},
		{PlayerId: player3, Wins: 0, Losses: 1, Draws: 1, Spread: -400},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	// Tournament should be over

	tournamentIsFinished, err = tc.IsFinished()
	is.NoErr(err)
	is.True(tournamentIsFinished)
}

func TestClassicDivisionOverwriteByes(t *testing.T) {
	// This test is used to ensure that the standings are
	// calculated correctly and that King of the Hill
	// pairings are correct

	is := is.New(t)

	roundControls := defaultRoundControls(defaultRounds)

	for i := 0; i < defaultRounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_KING_OF_THE_HILL
	}

	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	// Tournament should not be over

	player1 := defaultPlayers.Persons[0].Id
	player2 := defaultPlayers.Persons[1].Id
	player3 := defaultPlayers.Persons[2].Id
	player4 := defaultPlayers.Persons[3].Id

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(!tournamentIsFinished)

	_, err = tc.SetPairing(player3, player3, 0)
	is.NoErr(err)

	pk1, err := tc.getPairingKey(player1, 0)
	is.NoErr(err)
	pk2, err := tc.getPairingKey(player2, 0)
	is.NoErr(err)
	_, err = tc.getPairingKey(player3, 0)
	is.NoErr(err)
	pk4, err := tc.getPairingKey(player4, 0)
	is.NoErr(err)

	is.True(pk1 == pk2)
	bye, err := tc.pairingIsBye(player3, 0)
	is.NoErr(err)
	is.True(bye)
	is.True(pk4 == "")

	// Overwrite these pairings
	_, err = tc.PairRound(0, true)
	is.NoErr(err)
	pk1, err = tc.getPairingKey(player1, 0)
	is.NoErr(err)
	pk2, err = tc.getPairingKey(player2, 0)
	is.NoErr(err)
	pk3, err := tc.getPairingKey(player3, 0)
	is.NoErr(err)
	pk4, err = tc.getPairingKey(player4, 0)
	is.NoErr(err)

	is.True(pk1 == pk2)
	is.True(pk3 == pk4)

	_, err = tc.SetPairing(player2, player2, 0)
	is.NoErr(err)

	// Don't overwrite the bye
	_, err = tc.PairRound(0, false)
	is.NoErr(err)

	pk1, err = tc.getPairingKey(player1, 0)
	is.NoErr(err)
	_, err = tc.getPairingKey(player2, 0)
	is.NoErr(err)
	pk3, err = tc.getPairingKey(player3, 0)
	is.NoErr(err)
	_, err = tc.getPairingKey(player4, 0)
	is.NoErr(err)

	is.True(pk1 == pk3)
	bye, err = tc.pairingIsBye(player2, 0)
	is.NoErr(err)
	is.True(bye)
	bye, err = tc.pairingIsBye(player4, 0)
	is.NoErr(err)
	is.True(bye)

	// Clear the pairings
	err = tc.DeletePairings(0)
	is.NoErr(err)
	pk1, err = tc.getPairingKey(player1, 0)
	is.NoErr(err)
	is.True(pk1 == "")
	pk2, err = tc.getPairingKey(player2, 0)
	is.NoErr(err)
	is.True(pk2 == "")
	pk3, err = tc.getPairingKey(player3, 0)
	is.NoErr(err)
	is.True(pk3 == "")
	pk4, err = tc.getPairingKey(player4, 0)
	is.NoErr(err)
	is.True(pk4 == "")
}

func pairAndPlay(tc *ClassicDivision, pairings []string, round int) (*realtime.DivisionPairingsResponse, error) {
	if len(pairings)%2 != 0 {
		return nil, fmt.Errorf("Cannot pair and play with the given pairings and scores")
	}

	err := tc.DeletePairings(round)
	if err != nil {
		return nil, err
	}

	pm := newPairingsMessage()

	for i := 0; i < len(pairings); i += 2 {
		newpm, err := tc.SetPairing(pairings[i], pairings[i+1], round)
		if err != nil {
			return nil, err
		}
		pm = combinePairingMessages(pm, newpm)
	}

	err = tc.StartRound()
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(pairings); i += 2 {
		newpm, err := tc.SubmitResult(round, pairings[i], pairings[i+1], 500, 400,
			realtime.TournamentGameResult_WIN,
			realtime.TournamentGameResult_LOSS,
			realtime.GameEndReason_STANDARD, false, 0, "")
		if err != nil {
			return nil, err
		}
		pm = combinePairingMessages(pm, newpm)
	}

	return pm, nil
}

func TestClassicDivisionGibson(t *testing.T) {
	is := is.New(t)

	roundControls := []*realtime.RoundControl{}

	for i := 0; i < 98; i++ {
		roundControls = append(roundControls, &realtime.RoundControl{FirstMethod: realtime.FirstMethod_AUTOMATIC_FIRST,
			PairingMethod: realtime.PairingMethod_KING_OF_THE_HILL,
			GamesPerRound: defaultGamesPerRound,
			Round:         int32(i)})
	}
	playerLetterRatings := makeTournamentPersons(map[string]int32{"h": 1000, "g": 3000, "f": 2200, "e": 2100, "d": 12, "c": 43, "b": 40, "a": 2})
	tc, err := compactNewClassicDivision(playerLetterRatings, roundControls, false)
	is.NoErr(err)
	is.True(tc != nil)

	// To get the compiler to stop complaining
	if tc != nil && tc.DivisionControls != nil {
		newDivisionControls := &realtime.DivisionControls{
			GameRequest:      &realtime.GameRequest{Lexicon: "CSW19", Rules: &realtime.GameRules{BoardLayoutName: entity.CrosswordGame, LetterDistributionName: "English", VariantName: "classic"}, InitialTimeSeconds: 25 * 60, IncrementSeconds: 0, ChallengeRule: macondopb.ChallengeRule_FIVE_POINT, GameMode: realtime.GameMode_REAL_TIME, RatingMode: realtime.RatingMode_RATED, RequestId: "yeet", OriginalRequestId: "originalyeet", MaxOvertimeMinutes: 10},
			Gibsonize:        true,
			MinimumPlacement: 200,
		}
		// Attempt to set a nonsensical minimum placement
		_, err = tc.SetDivisionControls(newDivisionControls)
		is.NoErr(err)
		newDivisionControls.MinimumPlacement = 2
		_, err = tc.SetDivisionControls(newDivisionControls)
		is.NoErr(err)
	}

	currentRound := 0

	_, err = pairAndPlay(tc, []string{"a", "b", "c", "d", "e", "f", "g", "h"}, currentRound)
	is.NoErr(err)
	currentRound++
	_, err = pairAndPlay(tc, []string{"a", "c", "e", "g", "b", "d", "f", "h"}, currentRound)
	is.NoErr(err)
	currentRound++
	_, err = pairAndPlay(tc, []string{"a", "e", "h", "g", "b", "f", "d", "c"}, currentRound)
	is.NoErr(err)
	currentRound++

	for i := 0; i < 30; i++ {
		winner := "c"
		loser := "f"
		if i%2 == 1 {
			winner, loser = loser, winner
		}
		_, err = pairAndPlay(tc, []string{"e", "d", "a", "g", winner, loser, "b", "h"}, currentRound)
		is.NoErr(err)
		currentRound++
	}

	for i := 0; i < 30; i++ {
		_, err = pairAndPlay(tc, []string{"a", "d", "f", "e", "c", "b", "g", "h"}, currentRound)
		is.NoErr(err)
		currentRound++
	}

	for i := 0; i < 18; i++ {
		_, err = pairAndPlay(tc, []string{"a", "c", "f", "e", "g", "b", "d", "h"}, currentRound)
		is.NoErr(err)
		currentRound++
	}

	// Two players, all gibsonized

	// Second place is 17 games behind with 17 games to go
	// and spread is not used, so no players should be gibsonized
	pm, err := tc.PairRound(currentRound, true)
	is.NoErr(err)
	currentPairings := tc.getPlayerPairings(currentRound)
	is.NoErr(equalPairingStrings(currentPairings, []string{"a", "f", "c", "g", "b", "e", "d", "h"}))
	is.True(len(pm.GibsonizedPlayers) == 0)

	// Turn gibsonization by spread on, with a low spread
	// cap, to trigger gibsonization. Player "a" should
	// be gibsonized now.
	tc.DivisionControls.GibsonSpread = 199
	pm, err = tc.PairRound(currentRound, true)
	is.NoErr(err)
	currentPairings = tc.getPlayerPairings(currentRound)
	is.NoErr(equalPairingStrings(currentPairings, []string{"a", "e", "f", "g", "b", "c", "d", "h"}))
	is.True(len(pm.GibsonizedPlayers) == 1)
	is.True(pm.GibsonizedPlayers["a"] == int32(currentRound))

	// Adjust the spread cap to remove the gibsonization
	tc.DivisionControls.GibsonSpread = 201
	pm, err = tc.PairRound(currentRound, true)
	is.NoErr(err)
	currentPairings = tc.getPlayerPairings(currentRound)
	is.NoErr(equalPairingStrings(currentPairings, []string{"a", "f", "c", "g", "b", "e", "d", "h"}))
	is.True(len(pm.GibsonizedPlayers) == 0)

	// Adjust the minimum placement so that only first matters
	// which means the when first is gibsonized, they will play second
	tc.DivisionControls.GibsonSpread = 199
	tc.DivisionControls.MinimumPlacement = 0
	pm, err = tc.PairRound(currentRound, true)
	is.NoErr(err)
	currentPairings = tc.getPlayerPairings(currentRound)
	is.NoErr(equalPairingStrings(currentPairings, []string{"a", "f", "c", "g", "b", "e", "d", "h"}))
	is.True(len(pm.GibsonizedPlayers) == 1)
	is.True(pm.GibsonizedPlayers["a"] == int32(currentRound))

	// Adjust the minimum placement to a value that makes no
	// sense but which shouldn't break anything
	tc.DivisionControls.GibsonSpread = 199
	tc.DivisionControls.MinimumPlacement = 7
	pm, err = tc.PairRound(currentRound, true)
	is.NoErr(err)
	currentPairings = tc.getPlayerPairings(currentRound)
	is.NoErr(equalPairingStrings(currentPairings, []string{"a", "h", "f", "g", "c", "e", "b", "d"}))
	is.True(len(pm.GibsonizedPlayers) == 1)
	is.True(pm.GibsonizedPlayers["a"] == int32(currentRound))

	// Switch back to games only gibsonization and a minimum placement of 2
	tc.DivisionControls.GibsonSpread = 0
	tc.DivisionControls.MinimumPlacement = 2

	for i := 0; i < 3; i++ {
		_, err = pairAndPlay(tc, []string{"a", "c", "f", "e", "g", "b", "d", "h"}, currentRound)
		is.NoErr(err)
		currentRound++
	}

	pm, err = tc.PairRound(currentRound, true)
	is.NoErr(err)
	currentPairings = tc.getPlayerPairings(currentRound)
	is.NoErr(equalPairingStrings(currentPairings, []string{"a", "f", "c", "g", "b", "e", "d", "h"}))
	is.True(len(pm.GibsonizedPlayers) == 2)
	is.True(pm.GibsonizedPlayers["a"] == int32(currentRound))
	is.True(pm.GibsonizedPlayers["f"] == int32(currentRound))

	for i := 0; i < 5; i++ {
		_, err = pairAndPlay(tc, []string{"a", "c", "f", "e", "g", "b", "d", "h"}, currentRound)
		is.NoErr(err)
		currentRound++
	}

	pm, err = tc.PairRound(currentRound, true)
	is.NoErr(err)
	currentPairings = tc.getPlayerPairings(currentRound)
	is.NoErr(equalPairingStrings(currentPairings, []string{"a", "f", "c", "g", "b", "e", "d", "h"}))
	is.True(len(pm.GibsonizedPlayers) == 4)
	is.True(pm.GibsonizedPlayers["a"] == int32(currentRound))
	is.True(pm.GibsonizedPlayers["f"] == int32(currentRound))
	is.True(pm.GibsonizedPlayers["c"] == int32(currentRound))
	is.True(pm.GibsonizedPlayers["g"] == int32(currentRound))

	for i := 0; i < 8; i++ {
		_, err = pairAndPlay(tc, []string{"a", "c", "f", "e", "g", "b", "d", "h"}, currentRound)
		is.NoErr(err)
		currentRound++
	}

	pm, err = tc.PairRound(currentRound, true)
	is.NoErr(err)
	currentPairings = tc.getPlayerPairings(currentRound)
	is.NoErr(equalPairingStrings(currentPairings, []string{"a", "f", "c", "g", "d", "e", "b", "h"}))
	is.True(len(pm.GibsonizedPlayers) == 5)
	is.True(pm.GibsonizedPlayers["a"] == int32(currentRound))
	is.True(pm.GibsonizedPlayers["f"] == int32(currentRound))
	is.True(pm.GibsonizedPlayers["c"] == int32(currentRound))
	is.True(pm.GibsonizedPlayers["g"] == int32(currentRound))

	// Test edge cases
	roundControls = []*realtime.RoundControl{}

	for i := 0; i < 2; i++ {
		roundControls = append(roundControls, &realtime.RoundControl{FirstMethod: realtime.FirstMethod_AUTOMATIC_FIRST,
			PairingMethod: realtime.PairingMethod_KING_OF_THE_HILL,
			GamesPerRound: defaultGamesPerRound,
			Round:         int32(i)})
	}
	playerLetterRatings = makeTournamentPersons(map[string]int32{"h": 1000, "g": 3000})
	tc, err = compactNewClassicDivision(playerLetterRatings, roundControls, false)
	is.NoErr(err)
	is.True(tc != nil)

	// To get the compiler to stop complaining
	if tc != nil && tc.DivisionControls != nil {
		newDivisionControls := &realtime.DivisionControls{
			GameRequest:      &realtime.GameRequest{Lexicon: "CSW19", Rules: &realtime.GameRules{BoardLayoutName: entity.CrosswordGame, LetterDistributionName: "English", VariantName: "classic"}, InitialTimeSeconds: 25 * 60, IncrementSeconds: 0, ChallengeRule: macondopb.ChallengeRule_FIVE_POINT, GameMode: realtime.GameMode_REAL_TIME, RatingMode: realtime.RatingMode_RATED, RequestId: "yeet", OriginalRequestId: "originalyeet", MaxOvertimeMinutes: 10},
			Gibsonize:        true,
			MinimumPlacement: 1,
		}
		_, err = tc.SetDivisionControls(newDivisionControls)
		is.NoErr(err)
	}

	currentRound = 0
	_, err = pairAndPlay(tc, []string{"g", "h"}, currentRound)
	is.NoErr(err)
	currentRound++

	pm, err = tc.PairRound(currentRound, true)
	is.NoErr(err)
	is.True(len(pm.GibsonizedPlayers) == 0)

	tc.DivisionControls.GibsonSpread = 10

	pm, err = tc.PairRound(currentRound, true)
	is.NoErr(err)
	is.True(len(pm.GibsonizedPlayers) == 1)
	is.True(pm.GibsonizedPlayers["g"] == int32(currentRound))
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
	playerLetterRatings := makeTournamentPersons(map[string]int32{"h": 1000, "g": 3000, "f": 2200, "e": 2100, "d": 12, "c": 43, "b": 40, "a": 2})
	tc, err := compactNewClassicDivision(playerLetterRatings, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	// Set pairings to test more easily
	_, err = tc.SetPairing("d", "a", 0)
	is.NoErr(err)
	_, err = tc.SetPairing("c", "b", 0)
	is.NoErr(err)

	is.NoErr(validatePairings(tc, 0))

	err = tc.StartRound()
	is.NoErr(err)

	// This should throw an error since it attempts
	// to amend a result that never existed
	_, err = tc.SubmitResult(0, "h", "f", 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, true, 0, "")
	is.True(err != nil)

	_, err = tc.SubmitResult(0, "h", "f", 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, "g", "e", 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, "d", "a", 700, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// This is an invalid factor for this number of
	// players and an error should be returned
	tc.RoundControls[1].Factor = 5

	_, err = tc.SubmitResult(0, "c", "b", 600, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(err != nil)

	roundIsComplete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1
	standings, err := tc.GetStandings(0, false)
	is.NoErr(err)

	expectedstandings := &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: "h", Wins: 1, Losses: 0, Draws: 0, Spread: 400},
		{PlayerId: "g", Wins: 1, Losses: 0, Draws: 0, Spread: 300},
		{PlayerId: "d", Wins: 1, Losses: 0, Draws: 0, Spread: 200},
		{PlayerId: "c", Wins: 1, Losses: 0, Draws: 0, Spread: 100},
		{PlayerId: "b", Wins: 0, Losses: 1, Draws: 0, Spread: -100},
		{PlayerId: "a", Wins: 0, Losses: 1, Draws: 0, Spread: -200},
		{PlayerId: "e", Wins: 0, Losses: 1, Draws: 0, Spread: -300},
		{PlayerId: "f", Wins: 0, Losses: 1, Draws: 0, Spread: -400},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	tc.RoundControls[1].Factor = 3

	_, err = tc.PairRound(1, true)
	is.NoErr(err)

	err = tc.StartRound()
	is.NoErr(err)

	// Standings should be: 1, 2, 5, 8, 7, 6, 4, 3

	_, err = tc.SubmitResult(1, "h", "c", 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(1, "g", "b", 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(1, "d", "a", 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(1, "e", "f", 400, 500,
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

	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	player1 := defaultPlayers.Persons[0].Id
	player2 := defaultPlayers.Persons[1].Id
	player3 := defaultPlayers.Persons[2].Id
	player4 := defaultPlayers.Persons[3].Id

	err = tc.StartRound()
	is.NoErr(err)

	_, err = tc.SubmitResult(0, player1, player2, 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, player3, player4, 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(1, player1, player3, 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(1, player2, player4, 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Since repeats only have a weight of 1,
	// player1 and player4 should be playing each other

	_, err = tc.SubmitResult(2, player1, player4, 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(2, player2, player3, 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	repeats, err := getRepeats(tc, 2)

	// Everyone should have played each other once at this point
	for _, v := range repeats {
		is.True(v == 1)
	}

	_, err = tc.SubmitResult(3, player2, player1, 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Use factor pairings to force deterministic pairings
	tc.RoundControls[4].PairingMethod = realtime.PairingMethod_FACTOR
	tc.RoundControls[4].Factor = 2

	_, err = tc.SubmitResult(3, player3, player4, 800, 700,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 4
	standings, err := tc.GetStandings(3, false)
	is.NoErr(err)

	expectedstandings := &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player1, Wins: 3, Losses: 1, Draws: 0, Spread: 800},
		{PlayerId: player2, Wins: 3, Losses: 1, Draws: 0, Spread: 600},
		{PlayerId: player3, Wins: 2, Losses: 2, Draws: 0, Spread: -300},
		{PlayerId: player4, Wins: 0, Losses: 4, Draws: 0, Spread: -1100},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	_, err = tc.SubmitResult(4, player1, player3, 900, 800,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Test that using the prohibitive weight will
	// lead to the correct pairings
	tc.RoundControls[6].AllowOverMaxRepeats = false
	tc.RoundControls[6].MaxRepeats = 2

	_, err = tc.SubmitResult(4, player4, player2, 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 5
	standings, err = tc.GetStandings(4, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player1, Wins: 4, Losses: 1, Draws: 0, Spread: 900},
		{PlayerId: player2, Wins: 3, Losses: 2, Draws: 0, Spread: 300},
		{PlayerId: player3, Wins: 2, Losses: 3, Draws: 0, Spread: -400},
		{PlayerId: player4, Wins: 1, Losses: 4, Draws: 0, Spread: -800},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	_, err = tc.SubmitResult(5, player1, player4, 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Once the next round is paired upon the completion
	// of round 6, an error will occur since there is
	// no possible pairing that does not give 3 repeats.
	tc.RoundControls[6].AllowOverMaxRepeats = false
	tc.RoundControls[6].MaxRepeats = 2

	_, err = tc.SubmitResult(5, player2, player3, 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.True(fmt.Sprintf("%s", err) == "prohibitive weight reached, pairings are not possible with these settings")

	tc.RoundControls[6].AllowOverMaxRepeats = true

	_, err = tc.PairRound(6, true)
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

	swissPlayers := makeTournamentPersons(make(map[string]int32))
	for i := 1; i <= numberOfPlayers; i++ {
		swissPlayers.Persons = append(swissPlayers.Persons, &realtime.TournamentPerson{Id: fmt.Sprintf("%d", i), Rating: int32(1000 - i)})
	}

	roundControls[2].PairingMethod = realtime.PairingMethod_SWISS

	tc, err = compactNewClassicDivision(swissPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	err = tc.StartRound()
	is.NoErr(err)

	for i := 0; i < numberOfPlayers; i += 2 {
		_, err = tc.SubmitResult(0, swissPlayers.Persons[i].Id, swissPlayers.Persons[i+1].Id, (numberOfPlayers*100)-i*100, 0,
			realtime.TournamentGameResult_WIN,
			realtime.TournamentGameResult_LOSS,
			realtime.GameEndReason_STANDARD, false, 0, "")
		is.NoErr(err)
	}

	for i := 0; i < numberOfPlayers; i += 4 {
		_, err = tc.SubmitResult(1, swissPlayers.Persons[i].Id, swissPlayers.Persons[i+2].Id, (numberOfPlayers*10)-i*10, 0,
			realtime.TournamentGameResult_WIN,
			realtime.TournamentGameResult_LOSS,
			realtime.GameEndReason_STANDARD, false, 0, "")
		is.NoErr(err)
	}
	for i := 1; i < numberOfPlayers; i += 4 {
		_, err = tc.SubmitResult(1, swissPlayers.Persons[i].Id, swissPlayers.Persons[i+2].Id, 0, (numberOfPlayers*10)-i*10,
			realtime.TournamentGameResult_LOSS,
			realtime.TournamentGameResult_WIN,
			realtime.GameEndReason_STANDARD, false, 0, "")
		is.NoErr(err)
	}

	// Get the standings for round 2
	standings, err = tc.GetStandings(1, false)
	is.NoErr(err)

	for i := 0; i < len(tc.Matrix[2]); i++ {
		pairingKey := tc.Matrix[2][i]
		pairing := tc.PairingMap[pairingKey]
		playerOne := tc.Players.Persons[pairing.Players[0]].Id
		playerTwo := tc.Players.Persons[pairing.Players[1]].Id
		var playerOneIndex int
		var playerTwoIndex int
		for i := 0; i < len(standings.Standings); i++ {
			standingsPlayer := standings.Standings[i].PlayerId
			if playerOne == standingsPlayer {
				playerOneIndex = i
			} else if playerTwo == standingsPlayer {
				playerTwoIndex = i
			}
		}
		// Ensure players only played someone in with the same record
		playerOneStandings := standings.Standings[playerOneIndex]
		playerTwoStandings := standings.Standings[playerTwoIndex]
		is.True(playerOneStandings.Wins == playerTwoStandings.Wins)
		is.True(playerOneStandings.Losses == playerTwoStandings.Losses)
		is.True(playerOneStandings.Draws == playerTwoStandings.Draws)
	}
}

func TestClassicDivisionSwissRemoved(t *testing.T) {
	is := is.New(t)

	roundControls := []*realtime.RoundControl{}
	numberOfRounds := 100
	// This test onlyworks for values of the form 2 ^ n
	numberOfPlayers := 33

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

	swissPlayers := makeTournamentPersons(make(map[string]int32))
	for i := 1; i <= numberOfPlayers; i++ {
		swissPlayers.Persons = append(swissPlayers.Persons, &realtime.TournamentPerson{Id: fmt.Sprintf("%d", i), Rating: int32(1000 - i)})
	}

	tc, err := compactNewClassicDivision(swissPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	err = tc.StartRound()
	is.NoErr(err)

	winLoss := []realtime.TournamentGameResult{realtime.TournamentGameResult_WIN, realtime.TournamentGameResult_LOSS}

	for i := 0; i < numberOfRounds; i++ {
		if i == 16 {
			_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{"1": 50}))
			is.NoErr(err)
		}
		for j := 1; j <= numberOfPlayers; j++ {
			player := fmt.Sprintf("%d", j)
			playerPairing, err := tc.getPairing(player, i)
			opponent, err := tc.opponentOf(player, i)
			is.NoErr(err)
			if player == "1" {
				if i >= 17 {
					passed := playerPairing.Outcomes[0] == realtime.TournamentGameResult_FORFEIT_LOSS &&
						playerPairing.Outcomes[1] == realtime.TournamentGameResult_FORFEIT_LOSS
					is.True(passed)
				} else {
					is.True(playerPairing.Outcomes[0] != realtime.TournamentGameResult_FORFEIT_LOSS &&
						playerPairing.Outcomes[1] != realtime.TournamentGameResult_FORFEIT_LOSS)
				}
			}
			if i >= 17 {
				if player == "1" {
					is.True(opponent == "1")
				} else {
					is.True(opponent != player)
				}
			}
			if playerPairing.Outcomes[0] == realtime.TournamentGameResult_NO_RESULT {
				is.NoErr(err)
				randWin := rand.Intn(2)
				if player == "1" && i < 16 {
					randWin = 0
				}
				_, err = tc.SubmitResult(i, player, opponent, rand.Intn(1000), rand.Intn(1000),
					winLoss[randWin],
					winLoss[1-randWin],
					realtime.GameEndReason_STANDARD, false, 0, "")
				is.NoErr(err)
			}
		}
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

	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)

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
	for _, player := range defaultPlayers.Persons {
		m := make(map[string]int)
		m[player.Id] = 2

		for k := 0; k < len(tc.Matrix); k++ {
			opponent, err := tc.opponentOf(player.Id, k)
			is.NoErr(err)
			m[opponent]++
		}
		for _, opponent := range defaultPlayers.Persons {
			var err error = nil
			if m[opponent.Id] != 2 {
				err = fmt.Errorf("player %s didn't play %s exactly twice: %d", player.Id, opponent.Id, m[opponent.Id])
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

	tc, err = compactNewClassicDivision(defaultPlayersOdd, roundControls, true)
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
	for _, player := range defaultPlayersOdd.Persons {
		m := make(map[string]int)
		// We don't assign the player as having played themselves
		// twice in this case because the bye will do that.

		for k := 0; k < len(tc.Matrix); k++ {
			opponent, err := tc.opponentOf(player.Id, k)
			is.NoErr(err)
			m[opponent]++
		}
		for _, opponent := range defaultPlayersOdd.Persons {
			var err error = nil
			if m[opponent.Id] != 2 {
				err = fmt.Errorf("player %s didn't play %s exactly twice!", player.Id, opponent.Id)
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

	roundControls := defaultRoundControls(3)

	// Make all rounds Initial Fontes
	// This doesn't make sense and shouldn't be done
	// but we can test it anyway
	for i := 0; i < 3; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_INITIAL_FONTES
	}
	_, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)

	roundControls = defaultRoundControls(defaultRounds)

	for i := 1; i < defaultRounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_INITIAL_FONTES
	}

	// InitialFontes can only be used in contiguous defaultRounds
	// starting with round 1
	_, err = compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.True(err != nil)

	roundControls[0].PairingMethod = realtime.PairingMethod_INITIAL_FONTES

	// The number of InitialFontes pairings must be odd
	_, err = compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.True(err != nil)

	numberOfRoundsForInitialFontesTest := 4
	roundControls = defaultRoundControls(numberOfRoundsForInitialFontesTest)

	for i := 0; i < 3; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_INITIAL_FONTES
	}

	// InitialFontes should not be paired if there are more initial fontes rounds
	// than players
	tc, err := compactNewClassicDivision(makeTournamentPersons(map[string]int32{"Will": 10000}), roundControls, true)
	is.NoErr(err)

	// There should be no pairings at all
	for i := 0; i < len(tc.Matrix[0]); i++ {
		for j := 0; j < len(tc.Matrix[0][0]); j++ {
			fmt.Printf("round %d\n", i)
			tc.printPriPairings(i)
			is.True(tc.Matrix[i][j] == "")
		}
	}

	tc, err = compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)

	is.NoErr(validatePairings(tc, 0))
	is.NoErr(validatePairings(tc, 1))
	is.NoErr(validatePairings(tc, 2))
}

func TestClassicDivisionManual(t *testing.T) {
	is := is.New(t)

	roundControls := defaultRoundControls(defaultRounds)

	roundControls[0].PairingMethod = realtime.PairingMethod_MANUAL
	roundControls[1].PairingMethod = realtime.PairingMethod_KING_OF_THE_HILL

	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	player1 := defaultPlayers.Persons[0].Id
	player2 := defaultPlayers.Persons[1].Id
	player3 := defaultPlayers.Persons[2].Id
	player4 := defaultPlayers.Persons[3].Id

	// Check that round 1 is not paired
	for _, pairingKey := range tc.Matrix[0] {
		is.True(pairingKey == "")
	}

	// Pair round 1
	_, err = tc.SetPairing(player1, player2, 0)
	is.NoErr(err)
	_, err = tc.SetPairing(player3, player4, 0)
	is.NoErr(err)

	is.NoErr(validatePairings(tc, 0))

	// Amend a pairing
	_, err = tc.SetPairing(player2, player3, 0)
	is.NoErr(err)

	// Confirm that players 1 and 4 are now unpaired but 2 and 3 are
	is.True(tc.Matrix[0][tc.PlayerIndexMap[player1]] == "")
	is.True(tc.Matrix[0][tc.PlayerIndexMap[player4]] == "")
	is.True(tc.Matrix[0][tc.PlayerIndexMap[player2]] != "")
	is.True(tc.Matrix[0][tc.PlayerIndexMap[player3]] != "")

	// Complete the round 1 pairings
	_, err = tc.SetPairing(player1, player4, 0)
	is.NoErr(err)

	is.NoErr(validatePairings(tc, 0))

	// Submit results for round 1

	err = tc.StartRound()
	is.NoErr(err)

	_, err = tc.SubmitResult(0, player2, player3, 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)
	_, err = tc.SubmitResult(0, player1, player4, 200, 450,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	roundIsComplete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1
	standings, err := tc.GetStandings(0, false)
	is.NoErr(err)

	expectedstandings := &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player4, Wins: 1, Losses: 0, Draws: 0, Spread: 250},
		{PlayerId: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 100},
		{PlayerId: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -100},
		{PlayerId: player1, Wins: 0, Losses: 1, Draws: 0, Spread: -250},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	// Amend a result
	_, err = tc.SubmitResult(0, player1, player4, 500, 450,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, true, 0, "")
	is.NoErr(err)

	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1 again
	standings, err = tc.GetStandings(0, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 100},
		{PlayerId: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 50},
		{PlayerId: player4, Wins: 0, Losses: 1, Draws: 0, Spread: -50},
		{PlayerId: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -100},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))
}

func TestClassicDivisionElimination(t *testing.T) {
	is := is.New(t)

	roundControls := defaultRoundControls(defaultRounds)

	for i := 0; i < defaultRounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_ELIMINATION
		roundControls[i].GamesPerRound = 3
	}

	// Try and make an elimination tournament with the wrong number of defaultRounds
	tc, err := compactNewClassicDivision(defaultPlayers, roundControls[:1], true)
	is.True(err != nil)

	roundControls[0].PairingMethod = realtime.PairingMethod_RANDOM
	// Try and make an elimination tournament with other types
	// of pairings
	tc, err = compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.True(err != nil)

	roundControls = defaultRoundControls(defaultRounds)

	for i := 0; i < defaultRounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_ELIMINATION
		roundControls[i].GamesPerRound = 3
	}

	tc, err = compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	err = tc.StartRound()
	is.NoErr(err)

	is.NoErr(validatePairings(tc, 0))

	player1 := defaultPlayers.Persons[0].Id
	player2 := defaultPlayers.Persons[1].Id
	player3 := defaultPlayers.Persons[2].Id
	player4 := defaultPlayers.Persons[3].Id

	pairing1, err := tc.getPairing(player1, 0)
	is.NoErr(err)
	pairing2, err := tc.getPairing(player3, 0)
	is.NoErr(err)

	expectedpairing1 := newClassicPairing(tc, 0, 1, 0)
	expectedpairing2 := newClassicPairing(tc, 2, 3, 0)

	// Get the initial standings
	standings, err := tc.GetStandings(0, false)
	is.NoErr(err)

	// Ensure standings for Elimination are correct
	expectedstandings := &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player1, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
		{PlayerId: player2, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
		{PlayerId: player3, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
		{PlayerId: player4, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	// The match is decided in two games
	_, err = tc.SubmitResult(0, player1, player2, 500, 490,
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
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	_, err = tc.SubmitResult(0, player1, player2, 50, 0,
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
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	roundIsComplete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(!roundIsComplete)

	// The match is decided in three games
	_, err = tc.SubmitResult(0, player3, player4, 500, 400,
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
	is.NoErr(equalPairings(expectedpairing2, pairing2))

	_, err = tc.SubmitResult(0, player3, player4, 400, 400,
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
	is.NoErr(equalPairings(expectedpairing2, pairing2))

	_, err = tc.SubmitResult(0, player3, player4, 450, 400,
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
	is.NoErr(equalPairings(expectedpairing2, pairing2))

	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1
	standings, err = tc.GetStandings(0, false)
	is.NoErr(err)

	// Elimination standings are based on wins and player order only
	// Losses are not recorded in Elimination standings
	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 60},
		{PlayerId: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 150},
		{PlayerId: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -60},
		{PlayerId: player4, Wins: 0, Losses: 1, Draws: 0, Spread: -150},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	pairing1, err = tc.getPairing(player1, 1)
	is.NoErr(err)
	pairing2, err = tc.getPairing(player4, 1)
	is.NoErr(err)

	expectedpairing1 = newClassicPairing(tc, 0, 2, 1)

	// Half of the field should be eliminated

	// There should be no changes to the PRIs of players still
	// in the tournament. The Record gets carried over from
	// last round in the usual manner.
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	// The usual pri comparison method will fail since the
	// Games and Players are nil for elimianted players
	is.True(pairing2.Outcomes[0] == realtime.TournamentGameResult_ELIMINATED)
	is.True(pairing2.Outcomes[1] == realtime.TournamentGameResult_ELIMINATED)
	is.True(pairing2.Games == nil)
	is.True(pairing2.Players == nil)
	_, err = tc.SubmitResult(1, player1, player3, 500, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	expectedpairing1.Games[0].Results =
		[]realtime.TournamentGameResult{realtime.TournamentGameResult_WIN, realtime.TournamentGameResult_LOSS}
	expectedpairing1.Games[0].Scores[0] = 500
	expectedpairing1.Games[0].Scores[1] = 400
	expectedpairing1.Games[0].GameEndReason = realtime.GameEndReason_STANDARD
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	_, err = tc.SubmitResult(1, player1, player3, 400, 600,
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
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	_, err = tc.SubmitResult(1, player1, player3, 450, 450,
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
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	roundIsComplete, err = tc.IsRoundComplete(1)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Amend a result
	_, err = tc.SubmitResult(1, player1, player3, 451, 450,
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
	is.NoErr(equalPairings(expectedpairing1, pairing1))

	roundIsComplete, err = tc.IsRoundComplete(1)
	is.NoErr(err)
	is.True(roundIsComplete)

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(tournamentIsFinished)

	// Test ties and submitting tiebreaking results
	// Since this test is copied from above, the usual
	// validations are skipped, since they would be redundant.

	tc, err = compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	err = tc.StartRound()
	is.NoErr(err)

	player1 = defaultPlayers.Persons[0].Id
	player2 = defaultPlayers.Persons[1].Id
	player3 = defaultPlayers.Persons[2].Id
	player4 = defaultPlayers.Persons[3].Id

	pairing2, err = tc.getPairing(player3, 0)
	is.NoErr(err)

	// The match is decided in two games
	_, err = tc.SubmitResult(0, player1, player2, 500, 490,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, player1, player2, 50, 0,
		realtime.TournamentGameResult_FORFEIT_WIN,
		realtime.TournamentGameResult_FORFEIT_LOSS,
		realtime.GameEndReason_FORCE_FORFEIT, false, 1, "")
	is.NoErr(err)

	// The next match ends up tied at 1.5 - 1.5
	// with both players having the same spread.
	_, err = tc.SubmitResult(0, player3, player4, 500, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, player3, player4, 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 1, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, player3, player4, 500, 500,
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
	is.NoErr(equalPairings(expectedpairing2, pairing2))

	// Round should not be over
	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(!roundIsComplete)

	// Submit a tiebreaking result, unfortunately, it's another draw
	_, err = tc.SubmitResult(0, player3, player4, 500, 500,
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
	is.NoErr(equalPairings(expectedpairing2, pairing2))

	// Round should still not be over
	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(!roundIsComplete)

	// Attempt to submit a tiebreaking result, unfortunately, the game index is wrong
	_, err = tc.SubmitResult(0, player3, player4, 500, 500,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 5, "")
	is.True(err != nil)

	// Still wrong! Silly director (and definitely not code another layer up)
	_, err = tc.SubmitResult(0, player3, player4, 500, 500,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 2, "")
	is.True(err != nil)

	// Round should still not be over
	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(!roundIsComplete)

	// The players finally reach a decisive result
	_, err = tc.SubmitResult(0, player3, player4, 600, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 4, "")
	is.NoErr(err)

	// Round is finally over
	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1
	standings, err = tc.GetStandings(0, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{
		{PlayerId: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 60},
		{PlayerId: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 300},
		{PlayerId: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -60},
		{PlayerId: player4, Wins: 0, Losses: 1, Draws: 0, Spread: -300},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))
}

func TestClassicDivisionAddLatecomers(t *testing.T) {
	is := is.New(t)

	numberOfRounds := 5

	roundControls := defaultRoundControls(numberOfRounds)

	for i := 0; i < numberOfRounds; i++ {
		roundControls[i].PairingMethod = realtime.PairingMethod_KING_OF_THE_HILL
	}

	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	// Tournament should not be over

	player1 := defaultPlayers.Persons[0].Id
	player2 := defaultPlayers.Persons[1].Id
	player3 := defaultPlayers.Persons[3].Id
	player4 := defaultPlayers.Persons[2].Id

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(!tournamentIsFinished)

	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	_, err = tc.SubmitResult(0, player1, player2, 550, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, player3, player4, 300, 700,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")

	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	_, err = tc.SubmitResult(1, player1, player4, 670, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Add another player before the start of the next round
	_, err = tc.AddPlayers(makeTournamentPersons(map[string]int32{"Bum": 50}))
	is.NoErr(err)

	_, err = tc.SubmitResult(1, player3, player2, 800, 700,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 2
	standings, err := tc.GetStandings(1, false)
	is.NoErr(err)

	expectedstandings := &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{
		{PlayerId: player1, Wins: 2, Losses: 0, Draws: 0, Spread: 420},
		{PlayerId: player4, Wins: 1, Losses: 1, Draws: 0, Spread: 130},
		{PlayerId: player3, Wins: 1, Losses: 1, Draws: 0, Spread: -300},
		{PlayerId: "Bum", Wins: 0, Losses: 2, Draws: 0, Spread: -100},
		{PlayerId: player2, Wins: 0, Losses: 2, Draws: 0, Spread: -250},
	}}
	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	_, err = tc.SubmitResult(2, player1, player4, 400, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(2, player3, "Bum", 700, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// The bye result for player2 should have already been submitted
	standings, err = tc.GetStandings(2, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{
		{PlayerId: player1, Wins: 3, Losses: 0, Draws: 0, Spread: 520},
		{PlayerId: player3, Wins: 2, Losses: 1, Draws: 0, Spread: 100},
		{PlayerId: player4, Wins: 1, Losses: 2, Draws: 0, Spread: 30},
		{PlayerId: player2, Wins: 1, Losses: 2, Draws: 0, Spread: -200},
		{PlayerId: "Bum", Wins: 0, Losses: 3, Draws: 0, Spread: -500},
	}}
	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	_, err = tc.SubmitResult(3, player1, player3, 400, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.AddPlayers(makeTournamentPersons(map[string]int32{"Bummest": 50}))
	is.NoErr(err)

	_, err = tc.SubmitResult(3, player2, player4, 700, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.AddPlayers(makeTournamentPersons(map[string]int32{"Bummer": 50}))
	is.NoErr(err)

	standings, err = tc.GetStandings(3, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player1, Wins: 4, Losses: 0, Draws: 0, Spread: 620},
		{PlayerId: player2, Wins: 2, Losses: 2, Draws: 0, Spread: 200},
		{PlayerId: player3, Wins: 2, Losses: 2, Draws: 0, Spread: 0},
		{PlayerId: player4, Wins: 1, Losses: 3, Draws: 0, Spread: -370},
		{PlayerId: "Bum", Wins: 1, Losses: 3, Draws: 0, Spread: -450},
		{PlayerId: "Bummest", Wins: 0, Losses: 4, Draws: 0, Spread: -200},
		{PlayerId: "Bummer", Wins: 0, Losses: 4, Draws: 0, Spread: -200},
	}}
	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.StartRound()
	is.NoErr(err)

	// Submit results for the round
	_, err = tc.SubmitResult(4, player1, player2, 500, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(4, player3, player4, 300, 700,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")

	// Submit results for the round
	_, err = tc.SubmitResult(4, "Bum", "Bummest", 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	roundIsComplete, err := tc.IsRoundComplete(4)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Check that players cannot be added after the last round has started
	_, err = tc.AddPlayers(makeTournamentPersons(map[string]int32{"Guy": 50}))
	is.True(err != nil)

	standings, err = tc.GetStandings(4, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player1, Wins: 5, Losses: 0, Draws: 0, Spread: 720},
		{PlayerId: player2, Wins: 2, Losses: 3, Draws: 0, Spread: 100},
		{PlayerId: player4, Wins: 2, Losses: 3, Draws: 0, Spread: 30},
		{PlayerId: "Bum", Wins: 2, Losses: 3, Draws: 0, Spread: -250},
		{PlayerId: player3, Wins: 2, Losses: 3, Draws: 0, Spread: -400},
		{PlayerId: "Bummer", Wins: 1, Losses: 4, Draws: 0, Spread: -150},
		{PlayerId: "Bummest", Wins: 0, Losses: 5, Draws: 0, Spread: -400},
	}}
	is.NoErr(equalStandings(expectedstandings, standings))
}

func TestClassicDivisionRemovePlayers(t *testing.T) {

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

	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, true)
	is.NoErr(err)
	is.True(tc != nil)

	err = tc.StartRound()
	is.NoErr(err)

	player1 := defaultPlayers.Persons[0].Id
	player2 := defaultPlayers.Persons[1].Id
	player3 := defaultPlayers.Persons[3].Id
	player4 := defaultPlayers.Persons[2].Id

	_, err = tc.SubmitResult(0, player1, player2, 500, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, player3, player4, 500, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	standings, err := tc.GetStandings(0, false)
	is.NoErr(err)

	expectedstandings := &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 200},
		{PlayerId: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 100},
		{PlayerId: player4, Wins: 0, Losses: 1, Draws: 0, Spread: -100},
		{PlayerId: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -200},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	_, err = tc.SubmitResult(1, player1, player3, 500, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(1, player2, player4, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{player1: 50}))
	is.NoErr(err)
	standings, err = tc.GetStandings(1, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player2, Wins: 1, Losses: 1, Draws: 0, Spread: 0},
		{PlayerId: player3, Wins: 1, Losses: 1, Draws: 0, Spread: -100},
		{PlayerId: player4, Wins: 0, Losses: 2, Draws: 0, Spread: -300},
		// {PlayerId: player1, Wins: 2, Losses: 0, Draws: 0, Spread: 400},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	_, err = tc.SubmitResult(2, player3, player4, 500, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")

	_, err = tc.SubmitResult(2, player1, player2, 400, 400,
		realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_DRAW,
		realtime.GameEndReason_STANDARD, false, 0, "")

	// At this point the removed player is assigned forfeits

	_, err = tc.SubmitResult(3, player2, player4, 500, 300,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(4, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	standings, err = tc.GetStandings(4, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player2, Wins: 3, Losses: 1, Draws: 1, Spread: 400},
		{PlayerId: player3, Wins: 3, Losses: 2, Draws: 0, Spread: -150},
		{PlayerId: player4, Wins: 1, Losses: 4, Draws: 0, Spread: -550},
		// {PlayerId: player1, Wins: 2, Losses: 2, Draws: 1, Spread: 300},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	_, err = tc.SubmitResult(5, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{player4: 40}))
	is.NoErr(err)

	is.True(tc.Players.Persons[tc.PlayerIndexMap[player1]].Suspended)
	is.True(tc.Players.Persons[tc.PlayerIndexMap[player4]].Suspended)

	// Since this round had results, player4's bye against player1 remain unchanged
	pairing, err := tc.getPairing(player4, 5)
	is.NoErr(err)
	is.True(pairing.Games[0].Results[0] == realtime.TournamentGameResult_BYE)
	is.True(pairing.Games[0].Results[1] == realtime.TournamentGameResult_BYE)

	pairing, err = tc.getPairing(player1, 5)
	is.NoErr(err)
	is.True(pairing.Games[0].Results[0] == realtime.TournamentGameResult_FORFEIT_LOSS)
	is.True(pairing.Games[0].Results[1] == realtime.TournamentGameResult_FORFEIT_LOSS)

	standings, err = tc.GetStandings(5, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player2, Wins: 4, Losses: 1, Draws: 1, Spread: 600},
		{PlayerId: player3, Wins: 3, Losses: 3, Draws: 0, Spread: -350},
		// {PlayerId: player1, Wins: 2, Losses: 3, Draws: 1, Spread: 250},
		// {PlayerId: player4, Wins: 2, Losses: 4, Draws: 0, Spread: -500},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	pairing, err = tc.getPairing(player1, 6)
	is.NoErr(err)
	is.True(pairing.Games[0].Results[0] == realtime.TournamentGameResult_FORFEIT_LOSS)
	is.True(pairing.Games[0].Results[1] == realtime.TournamentGameResult_FORFEIT_LOSS)

	_, err = tc.SubmitResult(6, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Starting Round 7 player4 gets forfeit losses

	_, err = tc.SubmitResult(7, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(8, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	standings, err = tc.GetStandings(8, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player2, Wins: 7, Losses: 1, Draws: 1, Spread: 1200},
		{PlayerId: player3, Wins: 3, Losses: 6, Draws: 0, Spread: -950},
		// {PlayerId: player4, Wins: 3, Losses: 6, Draws: 0, Spread: -550},
		// {PlayerId: player1, Wins: 2, Losses: 6, Draws: 1, Spread: 100},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{player2: 10, player3: 60}))
	is.True(fmt.Sprintf("%s", err) == "cannot remove players as tournament would be empty")

	// Idiot director removed all but one player from the tournament
	_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{player2: 10}))

	_, err = tc.SubmitResult(9, player2, player3, 600, 400,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Tournament is now over as remaining player gets byes for the rest of the event

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(tournamentIsFinished)

	standings, err = tc.GetStandings(11, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player3, Wins: 5, Losses: 7, Draws: 0, Spread: -1050}}} // {PlayerId: player2, Wins: 8, Losses: 3, Draws: 1, Spread: 1300},
	// {PlayerId: player4, Wins: 3, Losses: 9, Draws: 0, Spread: -700},
	// {PlayerId: player1, Wins: 2, Losses: 9, Draws: 1, Spread: -50},

	is.NoErr(equalStandings(expectedstandings, standings))
}

func TestClassicDivisionRemovePlayersFactorPair(t *testing.T) {

	is := is.New(t)

	numberOfRounds := 6
	roundControls := []*realtime.RoundControl{}

	for i := 0; i < numberOfRounds; i++ {
		roundControls = append(roundControls, &realtime.RoundControl{FirstMethod: realtime.FirstMethod_MANUAL_FIRST,
			PairingMethod:               realtime.PairingMethod_FACTOR,
			GamesPerRound:               defaultGamesPerRound,
			Round:                       int32(i),
			Factor:                      1,
			MaxRepeats:                  1,
			AllowOverMaxRepeats:         true,
			RepeatRelativeWeight:        1,
			WinDifferenceRelativeWeight: 1})
		if i <= 2 {
			roundControls[i].PairingMethod = realtime.PairingMethod_ROUND_ROBIN
		}
	}

	playerLetterRatings := makeTournamentPersons(map[string]int32{"h": 10000, "g": 3000, "f": 2200, "e": 2100, "d": 100, "c": 43, "b": 40, "a": 2})
	tc, err := compactNewClassicDivision(playerLetterRatings, roundControls, false)
	is.NoErr(err)
	is.True(tc != nil)

	// Remove players

	_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{"f": -2, "c": -1, "b": 4}))
	is.NoErr(err)

	XHRResponse, err := tc.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(makeTournamentPersons(map[string]int32{"h": 10000, "g": 3000, "e": 2100, "d": 100, "a": 2}), XHRResponse.Players))

	// Add players back

	_, err = tc.AddPlayers(makeTournamentPersons(map[string]int32{"c": 43, "f": 2200, "b": 40}))
	is.NoErr(err)

	playerLetterRatings = makeTournamentPersons(map[string]int32{"h": 10000, "g": 3000, "f": 2200, "e": 2100, "d": 100, "c": 43, "b": 40, "a": 2})
	XHRResponse, err = tc.GetXHRResponse()
	is.NoErr(err)
	is.NoErr(equalTournamentPersons(playerLetterRatings, XHRResponse.Players))

	err = tc.StartRound()
	is.NoErr(err)

	_, err = tc.SubmitResult(0, "h", "a", 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, "g", "b", 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, "f", "c", 700, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(0, "e", "d", 600, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Round Robin pairings shouldn't change after removing players
	currentPairings := tc.getPlayerPairings(1)
	is.NoErr(equalPairingStrings(currentPairings, []string{"f", "h", "e", "g", "a", "d", "b", "c"}))

	_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{"c": 43, "f": 2200, "b": 40}))
	is.NoErr(err)

	currentPairings = tc.getPlayerPairings(1)
	is.NoErr(equalPairingStrings(currentPairings, []string{"f", "h", "e", "g", "a", "d", "b", "c"}))

	err = tc.StartRound()
	is.NoErr(err)

	_, err = tc.SubmitResult(1, "g", "e", 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(1, "d", "a", 700, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{"a": -9}))
	is.NoErr(err)

	currentPairings = tc.getPlayerPairings(2)
	is.NoErr(equalPairingStrings(currentPairings, []string{"d", "h", "a", "g", "b", "f", "c", "e"}))

	err = tc.StartRound()
	is.NoErr(err)

	roundControls[3].Factor = 2

	_, err = tc.SubmitResult(2, "h", "d", 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 3
	standings, err := tc.GetStandings(2, false)
	is.NoErr(err)

	expectedstandings := &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: "h", Wins: 3, Losses: 0, Draws: 0, Spread: 850},
		{PlayerId: "g", Wins: 3, Losses: 0, Draws: 0, Spread: 650},
		{PlayerId: "e", Wins: 2, Losses: 1, Draws: 0, Spread: -150},
		{PlayerId: "d", Wins: 1, Losses: 2, Draws: 0, Spread: -300},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	err = tc.StartRound()
	is.NoErr(err)

	_, err = tc.SubmitResult(3, "h", "e", 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	roundControls[4].Factor = 2

	_, err = tc.SubmitResult(3, "g", "d", 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 3
	standings, err = tc.GetStandings(3, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: "h", Wins: 4, Losses: 0, Draws: 0, Spread: 1250},
		{PlayerId: "g", Wins: 4, Losses: 0, Draws: 0, Spread: 950},
		{PlayerId: "e", Wins: 2, Losses: 2, Draws: 0, Spread: -550},
		{PlayerId: "d", Wins: 1, Losses: 3, Draws: 0, Spread: -600},
	}}

	is.NoErr(equalStandings(expectedstandings, standings))

	// Add players back
	currentPairings = tc.getPlayerPairings(4)
	is.NoErr(equalPairingStrings(currentPairings, []string{"e", "h", "d", "g", "f", "c", "b", "a"}))

	_, err = tc.AddPlayers(makeTournamentPersons(map[string]int32{"c": 43, "f": 2200, "b": 40}))
	is.NoErr(err)

	currentPairings = tc.getPlayerPairings(4)
	is.NoErr(equalPairingStrings(currentPairings, []string{"e", "h", "f", "g", "c", "d", "b", "a"}))

	err = tc.StartRound()
	is.NoErr(err)

	_, err = tc.SubmitResult(4, "h", "e", 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(4, "g", "f", 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(4, "d", "c", 700, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	// Get the standings for round 3
	standings, err = tc.GetStandings(4, false)
	is.NoErr(err)

	expectedstandings = &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{
		{PlayerId: "h", Wins: 5, Losses: 0, Draws: 0, Spread: 1650},
		{PlayerId: "g", Wins: 5, Losses: 0, Draws: 0, Spread: 1250},
		{PlayerId: "d", Wins: 2, Losses: 3, Draws: 0, Spread: -400},
		{PlayerId: "e", Wins: 2, Losses: 3, Draws: 0, Spread: -950},
		{PlayerId: "f", Wins: 1, Losses: 4, Draws: 0, Spread: -250},
		{PlayerId: "b", Wins: 1, Losses: 4, Draws: 0, Spread: -400},
		{PlayerId: "c", Wins: 0, Losses: 5, Draws: 0, Spread: -550},
	}}
	is.NoErr(equalStandings(expectedstandings, standings))

	// Check pairings
	currentPairings = tc.getPlayerPairings(5)
	is.NoErr(equalPairingStrings(currentPairings, []string{"g", "h", "d", "f", "b", "e", "c", "a"}))

	_, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{"d": -9}))
	is.NoErr(err)

	currentPairings = tc.getPlayerPairings(5)
	is.NoErr(equalPairingStrings(currentPairings, []string{"g", "h", "e", "f", "b", "c", "d", "a"}))

	err = tc.StartRound()
	is.NoErr(err)

	_, err = tc.SubmitResult(5, "h", "g", 900, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(5, "f", "e", 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	_, err = tc.SubmitResult(5, "b", "c", 800, 500,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	finished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(finished)
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

	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.NoErr(err)
	is.True(tc != nil)

	player1 := defaultPlayers.Persons[0].Id
	player2 := defaultPlayers.Persons[1].Id
	player3 := defaultPlayers.Persons[2].Id
	player4 := defaultPlayers.Persons[3].Id

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

	// Next two defaultRounds should even out firsts and seconds
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

func TestClassicDivisionMessages(t *testing.T) {
	is := is.New(t)

	roundControls := defaultRoundControls(5)

	for idx, control := range roundControls {
		if idx <= 2 {
			control.PairingMethod = realtime.PairingMethod_ROUND_ROBIN
		} else {
			control.PairingMethod = realtime.PairingMethod_KING_OF_THE_HILL
		}
	}

	tc, err := compactNewClassicDivision(defaultPlayers, roundControls, false)
	is.NoErr(err)
	player1 := defaultPlayers.Persons[0].Id
	player2 := defaultPlayers.Persons[1].Id
	player3 := defaultPlayers.Persons[2].Id
	player4 := defaultPlayers.Persons[3].Id

	expectedPairingsRsp := []*realtime.Pairing{
		{
			Players: []int32{0, 3},
			Round:   0,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{1, 2},
			Round:   0,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{0, 2},
			Round:   1,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{1, 3},
			Round:   1,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{0, 1},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{2, 3},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}}}

	pairingsRsp, controlsRsp, err := tc.SetRoundControls(roundControls)
	is.NoErr(err)

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))
	is.NoErr(equalRoundControls(roundControls, controlsRsp))

	controlRsp, err := tc.SetSingleRoundControls(0, roundControls[0])
	is.NoErr(err)
	is.NoErr(equalRoundControls([]*realtime.RoundControl{roundControls[0]}, []*realtime.RoundControl{controlRsp}))

	pairingsRsp, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{player1: -9}))
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{0, 0},
			Round:   0,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 50},
					Results:       []realtime.TournamentGameResult{4, 4},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{4, 4},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{1, 2},
			Round:   0,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{0, 2},
			Round:   1,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{1, 1},
			Round:   1,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 50},
					Results:       []realtime.TournamentGameResult{4, 4},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{4, 4},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{0, 1},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{2, 2},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 50},
					Results:       []realtime.TournamentGameResult{4, 4},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{4, 4},
			ReadyStates: []string{"", ""}}}

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))

	pairingsRsp, err = tc.AddPlayers(makeTournamentPersons(map[string]int32{player1: 10000}))
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{0, 3},
			Round:   0,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{1, 2},
			Round:   0,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{0, 2},
			Round:   1,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{1, 3},
			Round:   1,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{0, 1},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{2, 3},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}}}

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))

	err = tc.StartRound()
	is.NoErr(err)

	pairingsRsp, err = tc.SubmitResult(0, player2, player3, 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{1, 2},
			Round:   0,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{400, 500},
					Results:       []realtime.TournamentGameResult{2, 1},
					GameEndReason: 2}},
			Outcomes:    []realtime.TournamentGameResult{2, 1},
			ReadyStates: []string{"", ""}}}

	expectedStandings, err := tc.GetStandings(0, true)
	is.NoErr(err)
	is.NoErr(equalStandings(expectedStandings, pairingsRsp.DivisionStandings[0]))
	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))

	pairingsRsp, err = tc.SubmitResult(0, player1, player4, 200, 450,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{0, 3},
			Round:   0,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{200, 450},
					Results:       []realtime.TournamentGameResult{2, 1},
					GameEndReason: 2}},
			Outcomes:    []realtime.TournamentGameResult{2, 1},
			ReadyStates: []string{"", ""}}}

	expectedStandings, err = tc.GetStandings(0, true)
	is.NoErr(err)
	is.NoErr(equalStandings(expectedStandings, pairingsRsp.DivisionStandings[0]))

	manualExpectedStandings := &realtime.RoundStandings{Standings: []*realtime.PlayerStanding{{PlayerId: player4, Wins: 1, Losses: 0, Draws: 0, Spread: 250},
		{PlayerId: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 100},
		{PlayerId: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -100},
		{PlayerId: player1, Wins: 0, Losses: 1, Draws: 0, Spread: -250},
	}}
	is.NoErr(equalStandings(expectedStandings, manualExpectedStandings))

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))

	err = tc.StartRound()
	is.NoErr(err)

	pairingsRsp, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{player1: -9}))
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{0, 1},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{-50, 0},
					Results:       []realtime.TournamentGameResult{6, 5},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{6, 5},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{2, 3},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}}}

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))

	pairingsRsp, err = tc.AddPlayers(makeTournamentPersons(map[string]int32{player1: 2200}))
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{0, 1},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{2, 3},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}}}

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))

	pairingsRsp, err = tc.SubmitResult(1, player1, player3, 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{0, 2},
			Round:   1,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{400, 500},
					Results:       []realtime.TournamentGameResult{2, 1},
					GameEndReason: 2}},
			Outcomes:    []realtime.TournamentGameResult{2, 1},
			ReadyStates: []string{"", ""}}}

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))

	pairingsRsp, err = tc.SubmitResult(1, player2, player4, 200, 450,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{1, 3},
			Round:   1,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{200, 450},
					Results:       []realtime.TournamentGameResult{2, 1},
					GameEndReason: 2}},
			Outcomes:    []realtime.TournamentGameResult{2, 1},
			ReadyStates: []string{"", ""}}}

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))

	err = tc.StartRound()
	is.NoErr(err)

	pairingsRsp, err = tc.SubmitResult(2, player4, player3, 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{2, 3},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{500, 400},
					Results:       []realtime.TournamentGameResult{1, 2},
					GameEndReason: 2}},
			Outcomes:    []realtime.TournamentGameResult{1, 2},
			ReadyStates: []string{"", ""}}}

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))

	pairingsRsp, err = tc.SubmitResult(2, player1, player2, 200, 450,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{0, 1},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{200, 450},
					Results:       []realtime.TournamentGameResult{2, 1},
					GameEndReason: 2}},
			Outcomes:    []realtime.TournamentGameResult{2, 1},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{2, 3},
			Round:   3,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{1, 0},
			Round:   3,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}}}

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))

	pairingsRsp, err = tc.RemovePlayers(makeTournamentPersons(map[string]int32{player2: -9}))
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{2, 3},
			Round:   3,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{0, 0},
			Round:   3,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 50},
					Results:       []realtime.TournamentGameResult{4, 4},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{4, 4},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{1, 1},
			Round:   3,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, -50},
					Results:       []realtime.TournamentGameResult{6, 6},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{6, 6},
			ReadyStates: []string{"", ""}}}

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))

	pairingsRsp, err = tc.AddPlayers(makeTournamentPersons(map[string]int32{"Guy": 4200}))
	is.NoErr(err)

	expectedPairingsRsp = []*realtime.Pairing{
		{
			Players: []int32{4, 4},
			Round:   0,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, -50},
					Results:       []realtime.TournamentGameResult{6, 6},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{6, 6},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{4, 4},
			Round:   1,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, -50},
					Results:       []realtime.TournamentGameResult{6, 6},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{6, 6},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{4, 4},
			Round:   2,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, -50},
					Results:       []realtime.TournamentGameResult{6, 6},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{6, 6},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{2, 3},
			Round:   3,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{4, 0},
			Round:   3,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, 0},
					Results:       []realtime.TournamentGameResult{0, 0},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{0, 0},
			ReadyStates: []string{"", ""}},
		{
			Players: []int32{1, 1},
			Round:   3,
			Games: []*realtime.TournamentGame{
				{Scores: []int32{0, -50},
					Results:       []realtime.TournamentGameResult{6, 6},
					GameEndReason: 0}},
			Outcomes:    []realtime.TournamentGameResult{6, 6},
			ReadyStates: []string{"", ""}}}

	is.NoErr(equalPairingsResponses(expectedPairingsRsp, pairingsRsp.DivisionPairings))
}

/* func formatPairingsResponse(pairings []*realtime.Pairing) string {
	s := "	expectedPairingsRsp = []*realtime.Pairing {\n"
	for idx, pairing := range pairings {
		s += formatPairing(pairing)
		if idx != len(pairings)-1 {
			s += ",\n"
		}
	}
	s += "}"
	return s
}

func formatPairing(pairing *realtime.Pairing) string {
	s := "  &realtime.Pairing {\n"
	if pairing.Players != nil {
		s += fmt.Sprintf("    Players: []int32{%d, %d},\n", pairing.Players[0], pairing.Players[1])
	}
	s += fmt.Sprintf("    Round: %d", pairing.Round)
	if pairing.Games != nil {
		s += ",\n    Games: []*realtime.TournamentGame {\n"
		for idx, game := range pairing.Games {
			s += formatTournamentGame(game)
			if idx != len(pairing.Games)-1 {
				s += ","
			}
		}
		s += "}"
	}
	if pairing.Outcomes != nil {
		s += fmt.Sprintf(",\n    Outcomes: []realtime.TournamentGameResult{%d, %d}", pairing.Outcomes[0], pairing.Outcomes[1])
	}
	if pairing.ReadyStates != nil {
		s += fmt.Sprintf(",\n    ReadyStates: []string{\"%s\", \"%s\"}", pairing.ReadyStates[0], pairing.ReadyStates[1])
	}
	s += "}"
	return s
}

func formatTournamentGame(tg *realtime.TournamentGame) string {
	s := "      &realtime.TournamentGame {\n"
	s += fmt.Sprintf("      Scores: []int32{%d, %d},\n", tg.Scores[0], tg.Scores[1])
	s += fmt.Sprintf("      Results: []realtime.TournamentGameResult{%d, %d},\n", tg.Results[0], tg.Results[1])
	s += fmt.Sprintf("      GameEndReason: %d}", tg.GameEndReason)
	return s
}
*/
func runFirstMethodRound(tc *ClassicDivision, playerOrder []string, fs []int, round int, useByes bool) error {
	_, err := tc.SetPairing(playerOrder[0], playerOrder[1], round)

	if err != nil {
		return err
	}

	if useByes {
		_, err = tc.SetPairing(playerOrder[2], playerOrder[2], round)

		if err != nil {
			return err
		}
		_, err = tc.SetPairing(playerOrder[3], playerOrder[3], round)

		if err != nil {
			return err
		}
	} else {
		_, err = tc.SetPairing(playerOrder[2], playerOrder[3], round)

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

	_, err := tc.SubmitResult(round, player1, player2, 400, 500,
		realtime.TournamentGameResult_LOSS,
		realtime.TournamentGameResult_WIN,
		realtime.GameEndReason_STANDARD, false, 0, "")

	if err != nil {
		return err
	}

	// Results for byes are automatically submitted
	if !useByes {
		_, err = tc.SubmitResult(round, player3, player4, 200, 450,
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
		playerfs := getPlayerFirstsAndSeconds(tc, tc.PlayerIndexMap[players[i]], round)
		actualfs = append(actualfs, playerfs...)
	}

	for i := 0; i < len(actualfs); i++ {
		if actualfs[i] != fs[i] {
			actualfsString := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(actualfs)), ", "), "[]")
			fsString := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(fs)), ", "), "[]")
			return fmt.Errorf("firsts and seconds are not equal in round %d:\n%s\n%s", round, fsString, actualfsString)
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

		playersRandom := makeTournamentPersons(map[string]int32{})
		for i := 0; i < numberOfPlayers; i++ {
			playersRandom.Persons = append(playersRandom.Persons, &realtime.TournamentPerson{Id: fmt.Sprintf("%d", i), Rating: int32(i)})
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
			// Even-numbered games per defaultRounds was leading to ties
			// which give inconclusive results, therefor not ending the round
			for i := 0; i < numberOfRounds; i++ {
				roundControls[i].GamesPerRound = int32((rand.Intn(5) * 2) + 1)
			}
		}

		tc, err := compactNewClassicDivision(playersRandom, roundControls, true)
		if err != nil {
			return err
		}

		err = tc.StartRound()
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

					_, err = tc.SubmitResult(round,
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
					_, err = tc.SubmitResult(round,
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

			_, err = tc.GetStandings(round, false)
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
			standings, err := tc.GetStandings(numberOfRounds-1, false)
			if err != nil {
				return err
			}
			bottomHalfSize := numberOfPlayers / 2
			eliminationPlayerIndex := numberOfPlayers - 1
			eliminatedInRound := 0
			for bottomHalfSize > 0 {
				for i := 0; i < bottomHalfSize; i++ {
					if int(standings.Standings[eliminationPlayerIndex].Wins) != eliminatedInRound {
						return fmt.Errorf("player has incorrect number of wins (%d, %d, %d)",
							eliminationPlayerIndex,
							eliminatedInRound,
							standings.Standings[eliminationPlayerIndex].Wins)
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

func equalStandings(rs1 *realtime.RoundStandings, rs2 *realtime.RoundStandings) error {

	sa1 := rs1.Standings
	sa2 := rs2.Standings

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
	if s1.PlayerId != s2.PlayerId ||
		s1.Wins != s2.Wins ||
		s1.Losses != s2.Losses ||
		s1.Draws != s2.Draws ||
		s1.Spread != s2.Spread {
		return fmt.Errorf("standings do not match: (%s, %d, %d, %d, %d) != (%s, %d, %d, %d, %d)",
			s1.PlayerId, s1.Wins, s1.Losses, s1.Draws, s1.Spread,
			s2.PlayerId, s2.Wins, s2.Losses, s2.Draws, s2.Spread)
	}
	return nil
}

func (tc *ClassicDivision) getPlayerPairings(round int) []string {
	players := tc.Players.Persons

	pairingKeys := []string{}
	for i := 0; i < len(tc.Matrix[round]); i++ {
		pairingKeys = append(pairingKeys, tc.Matrix[round][i])
	}
	sort.Strings(pairingKeys)
	m := make(map[string]int)
	for _, player := range players {
		m[player.Id] = 0
	}

	playerPairings := []string{}
	for _, pk := range pairingKeys {
		// An eliminated player could have nil for Players, skip them
		pairing := tc.PairingMap[pk]
		p0 := tc.Players.Persons[pairing.Players[0]].Id
		p1 := tc.Players.Persons[pairing.Players[1]].Id
		if p0 > p1 {
			p0, p1 = p1, p0
		}
		if pairing.Players != nil && m[p0] == 0 {
			playerPairings = append(playerPairings, p0)
			if p0 != p1 {
				playerPairings = append(playerPairings, p1)
			}
			m[p0] = 1
			m[p1] = 1
		}
	}
	return playerPairings
}

func equalPairings(p1 *realtime.Pairing, p2 *realtime.Pairing) error {
	// We are not concerned with ordering
	// Firsts and seconds are tested independently
	if (p1.Players[0] != p2.Players[0] && p1.Players[0] != p2.Players[1]) ||
		(p1.Players[1] != p2.Players[0] && p1.Players[1] != p2.Players[1]) {
		return fmt.Errorf("players are not the same: (%d, %d) != (%d, %d)",
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

func equalRoundControls(rc1 []*realtime.RoundControl, rc2 []*realtime.RoundControl) error {
	if len(rc1) != len(rc2) {
		return fmt.Errorf("round controls are not the same length: %d != %d", len(rc1), len(rc2))
	}
	for i := 0; i < len(rc1); i++ {
		if !equalSingleRoundControls(rc1[i], rc2[i]) {
			return fmt.Errorf("round controls are not equal for round %d", i)
		}
	}
	return nil
}

func equalSingleRoundControls(rc1 *realtime.RoundControl, rc2 *realtime.RoundControl) bool {
	return rc1.PairingMethod == rc2.PairingMethod &&
		rc1.FirstMethod == rc2.FirstMethod &&
		rc1.GamesPerRound == rc2.GamesPerRound &&
		rc1.Round == rc2.Round &&
		rc1.Factor == rc2.Factor &&
		rc1.InitialFontes == rc2.InitialFontes &&
		rc1.MaxRepeats == rc2.MaxRepeats &&
		rc1.AllowOverMaxRepeats == rc2.AllowOverMaxRepeats &&
		rc1.RepeatRelativeWeight == rc2.RepeatRelativeWeight &&
		rc1.WinDifferenceRelativeWeight == rc2.WinDifferenceRelativeWeight
}

func equalPairingsResponses(pr1 []*realtime.Pairing, pr2 []*realtime.Pairing) error {
	if len(pr1) != len(pr2) {
		return fmt.Errorf("pairing responses are not the same length: %d != %d", len(pr1), len(pr2))
	}
	for i := 0; i < len(pr1); i++ {
		err := equalPairings(pr1[i], pr2[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func pairingToString(p *realtime.Pairing) string {
	return fmt.Sprintf("(%d, %d)", p.Players[0], p.Players[1])
}

func equalTournamentPersons(tp1 *realtime.TournamentPersons, tp2 *realtime.TournamentPersons) error {
	p1 := tp1.Persons
	p2 := tp2.Persons
	if p1 == nil && p2 == nil {
		return nil
	}
	if p1 == nil && p2 != nil {
		return fmt.Errorf("persons one is nil but persons two is not")
	}
	if p1 != nil && p2 == nil {
		return fmt.Errorf("persons two is nil but persons one is not")
	}
	if len(p1) != len(p2) {
		return fmt.Errorf("persons length one does not equal persons length two: %d != %d", len(p1), len(p2))
	}
	for i := 0; i < len(p1); i++ {
		if !equalTournamentPerson(p1[i], p2[i]) {
			return fmt.Errorf("persons are not equal: (%s, %d, %t) != (%s, %d, %t)", p1[i].Id, p1[i].Rating, p1[i].Suspended, p2[i].Id, p2[i].Rating, p2[i].Suspended)
		}
	}
	return nil
}

func equalTournamentPerson(p1 *realtime.TournamentPerson, p2 *realtime.TournamentPerson) bool {
	return p1.Id == p2.Id && p1.Rating == p2.Rating && p1.Suspended == p2.Suspended
}

func equalPairingStrings(s1 []string, s2 []string) error {
	if len(s1) != len(s2) {
		return fmt.Errorf("pairing lengths do not match: %d != %d", len(s1), len(s2))
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return fmt.Errorf("pairings are not equal:\n%s\n%s", utilities.StringArrayToString(s1), utilities.StringArrayToString(s2))
		}
	}
	return nil
}

func compactNewClassicDivision(players *realtime.TournamentPersons, roundControls []*realtime.RoundControl, autostart bool) (*ClassicDivision, error) {
	t := NewClassicDivision()

	for _, player := range players.Persons {
		player.Suspended = false
	}

	_, err := t.AddPlayers(players)
	if err != nil {
		return nil, err
	}

	_, _, err = t.SetRoundControls(roundControls)
	if err != nil {
		return nil, err
	}

	t.DivisionControls.AutoStart = autostart
	t.DivisionControls.SuspendedSpread = -50
	t.DivisionControls.SuspendedResult = realtime.TournamentGameResult_FORFEIT_LOSS
	return t, nil
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
		fmt.Printf("%s ", pk)
		fmt.Println(tc.PairingMap[pk])
	}
}

func printPairingMap(tc *ClassicDivision) {
	for key, value := range tc.PairingMap {
		fmt.Printf("key: %s", key)
		fmt.Print(", ")
		fmt.Println(value)
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

func makeTournamentPersons(persons map[string]int32) *realtime.TournamentPersons {
	tp := &realtime.TournamentPersons{Persons: []*realtime.TournamentPerson{}}
	for key, value := range persons {
		tp.Persons = append(tp.Persons, &realtime.TournamentPerson{Id: key, Rating: value})
	}
	sort.Sort(PlayerSorter(tp.Persons))
	return tp
}
