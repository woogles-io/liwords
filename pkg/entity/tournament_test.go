package entity

import (
	"errors"
	"fmt"
	"github.com/matryer/is"
	"strings"
	"testing"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

var players []string = []string{"Josh", "Will", "Cesar", "Jesse"}
var playersOdd []string = []string{"Josh", "Will", "Cesar", "Jesse", "Conrad"}
var rounds int = 2

func TestTournamentClassicRandom(t *testing.T) {
	// This test attempts to cover the basic
	// functions of a Classic Tournament

	is := is.New(t)

	// Tournaments must have at least two players
	tc, err := NewTournamentClassic([]string{"Sad"}, rounds, Random, 1)
	is.True(err != nil)

	// Tournaments must have at least 1 round
	tc, err = NewTournamentClassic(players, 0, Random, 1)
	is.True(err != nil)

	tc, err = NewTournamentClassic(players, rounds, Random, 1)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	// Test getting a nonexistent round
	_, err = tc.GetPlayerRoundInfo("Josh", 9)
	is.True(err != nil)

	// Test getting a nonexistent player
	_, err = tc.GetPlayerRoundInfo("No one", 1)
	is.True(err != nil)

	playerPairings := getPlayerPairings(tc.Players, tc.Matrix[0])
	player1 := playerPairings[0]
	player2 := playerPairings[1]
	player3 := playerPairings[2]
	player4 := playerPairings[3]

	pri1, err := tc.GetPlayerRoundInfo(player1, 0)
	is.NoErr(err)
	pri2, err := tc.GetPlayerRoundInfo(player3, 0)
	is.NoErr(err)

	expectedpri1 := newPlayerRoundInfo(player1, player2, tc.GamesPerRound)
	expectedpri2 := newPlayerRoundInfo(player3, player4, tc.GamesPerRound)

	// Submit result for an unpaired round
	err = tc.SubmitResult(1, player1, player2, 10000, -40, Win, Loss, realtime.GameEndReason_STANDARD, false, 0)
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPRI(expectedpri1, pri1))

	// Submit result for players that didn't player each other
	err = tc.SubmitResult(1, player1, player3, 10000, -40, Win, Loss, realtime.GameEndReason_STANDARD, false, 0)
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPRI(expectedpri1, pri1))

	// Submit a result for a paired round
	err = tc.SubmitResult(0, player1, player2, 10000, -40, Win, Loss, realtime.GameEndReason_STANDARD, false, 0)
	is.NoErr(err)

	// The result and record should have changed
	expectedpri1.Pairing.Games[0].Results = []Result{Win, Loss}
	expectedpri1.Pairing.Games[0].Scores[0] = 10000
	expectedpri1.Pairing.Games[0].Scores[1] = -40
	expectedpri1.Pairing.Games[0].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpri1.Pairing.Outcomes[0] = Win
	expectedpri1.Pairing.Outcomes[1] = Loss
	expectedpri1.Record[Win]++
	expectedpri1.Spread = 10040
	is.NoErr(equalPRI(expectedpri1, pri1))

	// Attempt to submit the same result
	err = tc.SubmitResult(0, player1, player2, 10000, -40, Win, Loss, realtime.GameEndReason_STANDARD, false, 0)
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPRI(expectedpri1, pri1))

	// Round 2 should not have been paired,
	// so attempting to submit a result for
	// it will throw an error.
	err = tc.SubmitResult(1, player1, player2, 10000, -40, Win, Loss, realtime.GameEndReason_STANDARD, false, 0)
	is.True(err != nil)

	// The result and record should remain unchanged
	is.NoErr(equalPRI(expectedpri1, pri1))

	// Amend the result
	err = tc.SubmitResult(0, player1, player2, 30, 900, Loss, Win, realtime.GameEndReason_STANDARD, true, 0)
	is.NoErr(err)

	// The result and record should be amended
	expectedpri1.Pairing.Games[0].Results = []Result{Loss, Win}
	expectedpri1.Pairing.Games[0].Scores[0] = 30
	expectedpri1.Pairing.Games[0].Scores[1] = 900
	expectedpri1.Pairing.Outcomes[0] = Loss
	expectedpri1.Pairing.Outcomes[1] = Win
	expectedpri1.Record[Win]--
	expectedpri1.Record[Loss]++
	expectedpri1.Spread = -870
	is.NoErr(equalPRI(expectedpri1, pri1))

	// Submit the final result for round 1
	err = tc.SubmitResult(0, player3, player4, 1, 1, Draw, Draw, realtime.GameEndReason_ABANDONED, false, 0)
	is.NoErr(err)

	expectedpri2.Pairing.Games[0].Results = []Result{Draw, Draw}
	expectedpri2.Spread = 0
	expectedpri2.Pairing.Games[0].Scores[0] = 1
	expectedpri2.Pairing.Games[0].Scores[1] = 1
	expectedpri2.Pairing.Outcomes[0] = Draw
	expectedpri2.Pairing.Outcomes[1] = Draw
	expectedpri2.Spread = 0
	expectedpri2.Pairing.Games[0].GameEndReason = realtime.GameEndReason_ABANDONED
	expectedpri2.Record[Draw]++
	is.NoErr(equalPRI(expectedpri2, pri2))

	roundIsComplete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Set pairings to test more easily
	err = tc.SetPairing(player1, player2, 1)
	is.NoErr(err)
	err = tc.SetPairing(player3, player4, 1)
	is.NoErr(err)

	pri1, err = tc.GetPlayerRoundInfo(player1, 1)
	is.NoErr(err)
	pri2, err = tc.GetPlayerRoundInfo(player3, 1)
	is.NoErr(err)

	expectedpri1 = newPlayerRoundInfo(player1, player2, tc.GamesPerRound)
	expectedpri2 = newPlayerRoundInfo(player3, player4, tc.GamesPerRound)

	// Round 2 should have been paired,
	// submit a result

	err = tc.SubmitResult(1, player1, player2, 0, 0, ForfeitLoss, ForfeitLoss, realtime.GameEndReason_ABANDONED, false, 0)
	is.NoErr(err)

	expectedpri1.Pairing.Games[0].Scores[0] = 0
	expectedpri1.Pairing.Games[0].Scores[1] = 0
	expectedpri1.Pairing.Games[0].GameEndReason = realtime.GameEndReason_ABANDONED
	expectedpri1.Pairing.Outcomes[0] = ForfeitLoss
	expectedpri1.Pairing.Outcomes[1] = ForfeitLoss
	expectedpri1.Pairing.Games[0].Results = []Result{ForfeitLoss, ForfeitLoss}
	expectedpri1.Spread = -870
	expectedpri1.Record[ForfeitLoss]++
	expectedpri1.Record[Loss]++
	is.NoErr(equalPRI(expectedpri1, pri1))

	// Submit the final tournament results
	err = tc.SubmitResult(1, player3, player4, 50, 50, Bye, Bye, realtime.GameEndReason_ABANDONED, false, 0)
	is.NoErr(err)

	expectedpri2.Pairing.Games[0].Results = []Result{Bye, Bye}
	expectedpri2.Pairing.Games[0].Scores[0] = 50
	expectedpri2.Pairing.Games[0].Scores[1] = 50
	expectedpri2.Pairing.Games[0].GameEndReason = realtime.GameEndReason_ABANDONED
	expectedpri2.Pairing.Outcomes[0] = Bye
	expectedpri2.Pairing.Outcomes[1] = Bye
	expectedpri2.Spread = 0
	expectedpri2.Record[Draw]++
	expectedpri2.Record[Bye]++
	is.NoErr(equalPRI(expectedpri2, pri2))

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
	tc, err = NewTournamentClassic(playersOdd, rounds, Random, 1)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))
}

func TestTournamentClassicKingOfTheHill(t *testing.T) {
	// This test is used to ensure that the standings are
	// calculated correctly and that King of the Hill
	// pairings are correct

	is := is.New(t)

	tc, err := NewTournamentClassic(players, rounds, KingOfTheHill, 1)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	// Tournament should not be over

	player1 := players[0]
	player2 := players[1]
	player3 := players[2]
	player4 := players[3]

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(!tournamentIsFinished)

	playerPairings := getPlayerPairings(tc.Players, tc.Matrix[0])
	for i := 0; i < len(playerPairings); i++ {
		is.True(playerPairings[i] == players[i])
	}

	// Submit results for the round
	err = tc.SubmitResult(0, player1, player2, 550, 400, Win, Loss, realtime.GameEndReason_STANDARD, false, 0)
	is.NoErr(err)

	err = tc.SubmitResult(0, player3, player4, 300, 700, Loss, Win, realtime.GameEndReason_STANDARD, false, 0)
	is.NoErr(err)

	// Get the standings for round 1
	standings, err := tc.GetStandings(0)
	is.NoErr(err)

	expectedstandings := []*Standing{&Standing{Player: player4, Wins: 1, Losses: 0, Draws: 0, Spread: 400},
		&Standing{Player: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 150},
		&Standing{Player: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -150},
		&Standing{Player: player3, Wins: 0, Losses: 1, Draws: 0, Spread: -400},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	// The next round should have been paired

	// Tournament should not be over

	tournamentIsFinished, err = tc.IsFinished()
	is.NoErr(err)
	is.True(!tournamentIsFinished)

	// Submit results for the round
	err = tc.SubmitResult(1, player1, player4, 670, 400, Win, Loss, realtime.GameEndReason_STANDARD, false, 0)
	is.NoErr(err)

	err = tc.SubmitResult(1, player3, player2, 700, 700, Draw, Draw, realtime.GameEndReason_STANDARD, false, 0)
	is.NoErr(err)

	// Get the standings for round 2
	standings, err = tc.GetStandings(1)
	is.NoErr(err)

	expectedstandings = []*Standing{&Standing{Player: player1, Wins: 2, Losses: 0, Draws: 0, Spread: 420},
		&Standing{Player: player4, Wins: 1, Losses: 1, Draws: 0, Spread: 130},
		&Standing{Player: player2, Wins: 0, Losses: 1, Draws: 1, Spread: -150},
		&Standing{Player: player3, Wins: 0, Losses: 1, Draws: 1, Spread: -400},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	// Tournament should be over

	tournamentIsFinished, err = tc.IsFinished()
	is.NoErr(err)
	is.True(tournamentIsFinished)

	// Check that pairings are correct with an odd number of players
	tc, err = NewTournamentClassic(playersOdd, rounds, KingOfTheHill, 1)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	// The last player should have a bye
	l := len(tc.Players) - 1
	lastPlayer := tc.Players[l]
	is.True(tc.Matrix[0][l].Pairing.Players[0] == lastPlayer)
	is.True(tc.Matrix[0][l].Pairing.Players[1] == lastPlayer)
}

func TestTournamentClassicRoundRobinAlgorithm(t *testing.T) {
	// This test is used to ensure that round robin
	// pairings work correctly. See the function in tournament.go
	// for more details about the algorithm.

	is := is.New(t)

	roundRobinPlayers := []string{"1", "2", "3", "4", "5", "6", "7", "8"}

	is.NoErr(equalRoundRobinPairings([]string{"1", "8", "2", "7", "3", "6", "4", "5"},
		getRoundRobinPairings(roundRobinPlayers, 0)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "7", "8", "6", "2", "5", "3", "4"},
		getRoundRobinPairings(roundRobinPlayers, 1)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "6", "7", "5", "8", "4", "2", "3"},
		getRoundRobinPairings(roundRobinPlayers, 2)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "5", "6", "4", "7", "3", "8", "2"},
		getRoundRobinPairings(roundRobinPlayers, 3)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "4", "5", "3", "6", "2", "7", "8"},
		getRoundRobinPairings(roundRobinPlayers, 4)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "3", "4", "2", "5", "8", "6", "7"},
		getRoundRobinPairings(roundRobinPlayers, 5)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "2", "3", "8", "4", "7", "5", "6"},
		getRoundRobinPairings(roundRobinPlayers, 6)))

	// Modulus operation should repeat the pairings past the first round robin

	is.NoErr(equalRoundRobinPairings([]string{"1", "8", "2", "7", "3", "6", "4", "5"},
		getRoundRobinPairings(roundRobinPlayers, 7)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "7", "8", "6", "2", "5", "3", "4"},
		getRoundRobinPairings(roundRobinPlayers, 8)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "6", "7", "5", "8", "4", "2", "3"},
		getRoundRobinPairings(roundRobinPlayers, 9)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "5", "6", "4", "7", "3", "8", "2"},
		getRoundRobinPairings(roundRobinPlayers, 10)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "4", "5", "3", "6", "2", "7", "8"},
		getRoundRobinPairings(roundRobinPlayers, 11)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "3", "4", "2", "5", "8", "6", "7"},
		getRoundRobinPairings(roundRobinPlayers, 12)))
	is.NoErr(equalRoundRobinPairings([]string{"1", "2", "3", "8", "4", "7", "5", "6"},
		getRoundRobinPairings(roundRobinPlayers, 13)))

	// Test first pairing of third round robin just to be sure

	is.NoErr(equalRoundRobinPairings([]string{"1", "8", "2", "7", "3", "6", "4", "5"},
		getRoundRobinPairings(roundRobinPlayers, 14)))
}

func TestTournamentClassicRoundRobin(t *testing.T) {
	// This test is used to ensure that round robin
	// pairings work correctly

	is := is.New(t)

	tc, err := NewTournamentClassic(players, 6, RoundRobin, 1)
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
	for i, player := range players {
		m := make(map[string]int)
		m[player] = 2

		for k := 0; k < len(tc.Matrix); k++ {
			opponent, err := opponentOf(tc.Matrix[k][i].Pairing, player)
			is.NoErr(err)
			m[opponent]++
		}
		for _, opponent := range players {
			var err error = nil
			if m[opponent] != 2 {
				err = errors.New(fmt.Sprintf("Player %s didn't play %s exactly twice!", player, opponent))
			}
			is.NoErr(err)
		}
	}

	// Test Round Robin with an odd number of players (a bye)

	tc, err = NewTournamentClassic(playersOdd, 10, RoundRobin, 1)
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
	for i, player := range players {
		m := make(map[string]int)
		// We don't assign the player as having played themselves
		// twice in this case because the bye will do that.

		for k := 0; k < len(tc.Matrix); k++ {
			opponent, err := opponentOf(tc.Matrix[k][i].Pairing, player)
			is.NoErr(err)
			m[opponent]++
		}
		for _, opponent := range players {
			var err error = nil
			if m[opponent] != 2 {
				err = errors.New(fmt.Sprintf("Player %s didn't play %s exactly twice!", player, opponent))
			}
			is.NoErr(err)
		}
	}
}

func TestTournamentClassicManual(t *testing.T) {
	is := is.New(t)

	tc, err := NewTournamentClassic(players, rounds, Manual, 1)
	is.NoErr(err)
	is.True(tc != nil)

	player1 := players[0]
	player2 := players[1]
	player3 := players[2]
	player4 := players[3]

	// Check that round 1 is not paired
	for _, pri := range tc.Matrix[0] {
		pairing := pri.Pairing
		is.True(pairing == nil)
	}

	// Pair round 1
	err = tc.SetPairing(player1, player2, 0)
	is.NoErr(err)
	err = tc.SetPairing(player3, player4, 0)
	is.NoErr(err)

	is.NoErr(validatePairings(tc, 0))

	// Amend a pairing
	err = tc.SetPairing(player2, player3, 0)
	is.NoErr(err)

	// Confirm that players 1 and 4 are now unpaired
	is.True(tc.Matrix[0][tc.PlayerIndexMap[player1]].Pairing == nil)
	is.True(tc.Matrix[0][tc.PlayerIndexMap[player4]].Pairing == nil)

	// Complete the round 1 pairings
	err = tc.SetPairing(player1, player4, 0)
	is.NoErr(err)

	is.NoErr(validatePairings(tc, 0))

	// Submit results for round 1

	// Here, amend is set to true, but it should matter.
	// Maybe at some point we want to be stricter and
	// reject submissions that think they are amending
	// when really there is no result.
	err = tc.SubmitResult(0, player2, player3, 400, 500, Loss, Win, realtime.GameEndReason_STANDARD, true, 0)
	is.NoErr(err)
	err = tc.SubmitResult(0, player1, player4, 200, 450, Loss, Win, realtime.GameEndReason_STANDARD, true, 0)
	is.NoErr(err)

	roundIsComplete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1
	standings, err := tc.GetStandings(0)
	is.NoErr(err)

	expectedstandings := []*Standing{&Standing{Player: player4, Wins: 1, Losses: 0, Draws: 0, Spread: 250},
		&Standing{Player: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 100},
		&Standing{Player: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -100},
		&Standing{Player: player1, Wins: 0, Losses: 1, Draws: 0, Spread: -250},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	// Amend a result
	err = tc.SubmitResult(0, player1, player4, 500, 450, Win, Loss, realtime.GameEndReason_STANDARD, true, 0)
	is.NoErr(err)

	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1 again
	standings, err = tc.GetStandings(0)
	is.NoErr(err)

	expectedstandings = []*Standing{&Standing{Player: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 100},
		&Standing{Player: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 50},
		&Standing{Player: player4, Wins: 0, Losses: 1, Draws: 0, Spread: -50},
		&Standing{Player: player2, Wins: 0, Losses: 1, Draws: 0, Spread: -100},
	}

	is.NoErr(equalStandings(expectedstandings, standings))
}

func TestTournamentClassicElimination(t *testing.T) {
	is := is.New(t)

	// Try and make an elimination tournament with too many rounds
	tc, err := NewTournamentClassic(players, 3, Elimination, 3)
	is.True(err != nil)

	tc, err = NewTournamentClassic(players, 2, Elimination, 3)
	is.NoErr(err)
	is.True(tc != nil)

	is.NoErr(validatePairings(tc, 0))

	player1 := players[0]
	player2 := players[1]
	player3 := players[2]
	player4 := players[3]

	pri1, err := tc.GetPlayerRoundInfo(player1, 0)
	is.NoErr(err)
	pri2, err := tc.GetPlayerRoundInfo(player3, 0)
	is.NoErr(err)

	expectedpri1 := newPlayerRoundInfo(player1, player2, tc.GamesPerRound)
	expectedpri2 := newPlayerRoundInfo(player3, player4, tc.GamesPerRound)

	// Get the initial standings
	standings, err := tc.GetStandings(0)
	is.NoErr(err)

	// Ensure standings for Elimination are correct
	expectedstandings := []*Standing{&Standing{Player: player1, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
		&Standing{Player: player2, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
		&Standing{Player: player3, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
		&Standing{Player: player4, Wins: 0, Losses: 0, Draws: 0, Spread: 0},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	// The match is decided in two games
	err = tc.SubmitResult(0, player1, player2, 500, 490, Win, Loss, realtime.GameEndReason_STANDARD, false, 0)
	is.NoErr(err)

	// The spread and games should have changed
	expectedpri1.Pairing.Games[0].Results = []Result{Win, Loss}
	expectedpri1.Pairing.Games[0].Scores[0] = 500
	expectedpri1.Pairing.Games[0].Scores[1] = 490
	expectedpri1.Pairing.Games[0].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpri1.Spread = 10
	is.NoErr(equalPRI(expectedpri1, pri1))

	err = tc.SubmitResult(0, player1, player2, 50, 0, ForfeitWin, ForfeitLoss, realtime.GameEndReason_ABANDONED, false, 1)
	is.NoErr(err)

	// The outcomes should now be set
	expectedpri1.Pairing.Games[1].Results = []Result{ForfeitWin, ForfeitLoss}
	expectedpri1.Pairing.Games[1].Scores[0] = 50
	expectedpri1.Pairing.Games[1].Scores[1] = 0
	expectedpri1.Pairing.Games[1].GameEndReason = realtime.GameEndReason_ABANDONED
	expectedpri1.Spread = 60
	expectedpri1.Pairing.Outcomes[0] = Win
	expectedpri1.Pairing.Outcomes[1] = Eliminated
	expectedpri1.Record[Win]++
	is.NoErr(equalPRI(expectedpri1, pri1))

	roundIsComplete, err := tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(!roundIsComplete)

	// The match is decided in three games
	err = tc.SubmitResult(0, player3, player4, 500, 400, Win, Loss, realtime.GameEndReason_STANDARD, false, 0)
	is.NoErr(err)

	// The spread and games should have changed
	expectedpri2.Pairing.Games[0].Results = []Result{Win, Loss}
	expectedpri2.Pairing.Games[0].Scores[0] = 500
	expectedpri2.Pairing.Games[0].Scores[1] = 400
	expectedpri2.Pairing.Games[0].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpri2.Spread = 100
	is.NoErr(equalPRI(expectedpri2, pri2))

	err = tc.SubmitResult(0, player3, player4, 400, 400, Draw, Draw, realtime.GameEndReason_STANDARD, false, 1)
	is.NoErr(err)

	// The spread and games should have changed
	expectedpri2.Pairing.Games[1].Results = []Result{Draw, Draw}
	expectedpri2.Pairing.Games[1].Scores[0] = 400
	expectedpri2.Pairing.Games[1].Scores[1] = 400
	expectedpri2.Pairing.Games[1].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpri2.Spread = 100
	is.NoErr(equalPRI(expectedpri2, pri2))

	err = tc.SubmitResult(0, player3, player4, 450, 400, Win, Loss, realtime.GameEndReason_STANDARD, false, 2)
	is.NoErr(err)

	// The spread and games should have changed
	// The outcome and record should have changed
	expectedpri2.Pairing.Games[2].Results = []Result{Win, Loss}
	expectedpri2.Pairing.Games[2].Scores[0] = 450
	expectedpri2.Pairing.Games[2].Scores[1] = 400
	expectedpri2.Pairing.Games[2].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpri2.Spread = 150
	expectedpri2.Pairing.Outcomes[0] = Win
	expectedpri2.Pairing.Outcomes[1] = Eliminated
	expectedpri2.Record[Win]++
	is.NoErr(equalPRI(expectedpri2, pri2))

	roundIsComplete, err = tc.IsRoundComplete(0)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Get the standings for round 1
	standings, err = tc.GetStandings(0)
	is.NoErr(err)

	// Elimination standings are based on wins and player order only
	// Losses are not recorded in Elimination standings
	expectedstandings = []*Standing{&Standing{Player: player1, Wins: 1, Losses: 0, Draws: 0, Spread: 60},
		&Standing{Player: player3, Wins: 1, Losses: 0, Draws: 0, Spread: 150},
		&Standing{Player: player2, Wins: 0, Losses: 0, Draws: 0, Spread: -60},
		&Standing{Player: player4, Wins: 0, Losses: 0, Draws: 0, Spread: -150},
	}

	is.NoErr(equalStandings(expectedstandings, standings))

	pri1, err = tc.GetPlayerRoundInfo(player1, 1)
	is.NoErr(err)
	pri2, err = tc.GetPlayerRoundInfo(player4, 1)
	is.NoErr(err)

	expectedpri1 = newPlayerRoundInfo(player1, player3, tc.GamesPerRound)

	// Half of the field should be eliminated

	// There should be no changes to the PRIs of players still
	// in the tournament. The Record gets carried over from
	// last round in the usual manner.
	expectedpri1.Record[Win]++
	expectedpri1.Spread = 60
	is.NoErr(equalPRI(expectedpri1, pri1))

	// The usual pri comparison methd will fail since the
	// Games and Players are nil for elimianted players
	is.True(pri2.Pairing.Outcomes[0] == Eliminated)
	is.True(pri2.Pairing.Outcomes[1] == Eliminated)
	is.True(pri2.Pairing.Games == nil)
	is.True(pri2.Pairing.Players == nil)

	// The match is decided in three games
	err = tc.SubmitResult(1, player1, player3, 500, 400, Win, Loss, realtime.GameEndReason_STANDARD, false, 0)
	is.NoErr(err)

	expectedpri1.Pairing.Games[0].Results = []Result{Win, Loss}
	expectedpri1.Pairing.Games[0].Scores[0] = 500
	expectedpri1.Pairing.Games[0].Scores[1] = 400
	expectedpri1.Pairing.Games[0].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpri1.Spread = 160
	is.NoErr(equalPRI(expectedpri1, pri1))

	err = tc.SubmitResult(1, player1, player3, 400, 600, Loss, Win, realtime.GameEndReason_STANDARD, false, 1)
	is.NoErr(err)

	expectedpri1.Pairing.Games[1].Results = []Result{Loss, Win}
	expectedpri1.Pairing.Games[1].Scores[0] = 400
	expectedpri1.Pairing.Games[1].Scores[1] = 600
	expectedpri1.Pairing.Games[1].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpri1.Spread = -40
	is.NoErr(equalPRI(expectedpri1, pri1))

	err = tc.SubmitResult(1, player1, player3, 450, 450, Draw, Draw, realtime.GameEndReason_STANDARD, false, 2)
	is.NoErr(err)

	expectedpri1.Pairing.Games[2].Results = []Result{Draw, Draw}
	expectedpri1.Pairing.Games[2].Scores[0] = 450
	expectedpri1.Pairing.Games[2].Scores[1] = 450
	expectedpri1.Pairing.Games[2].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpri1.Spread = -40
	expectedpri1.Pairing.Outcomes[0] = Eliminated
	expectedpri1.Pairing.Outcomes[1] = Win
	expectedpri1.Record[Eliminated]++
	is.NoErr(equalPRI(expectedpri1, pri1))

	roundIsComplete, err = tc.IsRoundComplete(1)
	is.NoErr(err)
	is.True(roundIsComplete)

	// Amend a result
	err = tc.SubmitResult(1, player1, player3, 451, 450, Win, Loss, realtime.GameEndReason_STANDARD, true, 2)
	is.NoErr(err)

	expectedpri1.Pairing.Games[2].Results = []Result{Win, Loss}
	expectedpri1.Pairing.Games[2].Scores[0] = 451
	expectedpri1.Pairing.Games[2].Scores[1] = 450
	expectedpri1.Pairing.Games[2].GameEndReason = realtime.GameEndReason_STANDARD
	expectedpri1.Spread = -39
	expectedpri1.Pairing.Outcomes[0] = Win
	expectedpri1.Pairing.Outcomes[1] = Eliminated
	expectedpri1.Record[Eliminated]--
	expectedpri1.Record[Win]++
	is.NoErr(equalPRI(expectedpri1, pri1))

	roundIsComplete, err = tc.IsRoundComplete(1)
	is.NoErr(err)
	is.True(roundIsComplete)

	tournamentIsFinished, err := tc.IsFinished()
	is.NoErr(err)
	is.True(tournamentIsFinished)
}

func validatePairings(tc *TournamentClassic, round int) error {
	// For each pairing, check that
	//   - Player's opponent is nonnull
	//   - Player's opponent's opponent is the player

	if round < 0 || round >= len(tc.Matrix) {
		return errors.New(fmt.Sprintf("Round number out of range: %d\n", round))
	}

	for i, pri := range tc.Matrix[round] {
		pairing := pri.Pairing
		if pri.Pairing == nil {
			return errors.New(fmt.Sprintf("Player %s is unpaired", players[i]))
		}
		// Check that the pairing refs are correct
		opponent, err := opponentOf(pairing, tc.Players[i])
		if err != nil {
			return err
		}
		opponentOpponent, err := opponentOf(tc.Matrix[round][tc.PlayerIndexMap[opponent]].Pairing, opponent)
		if err != nil {
			return err
		}
		if tc.Players[i] != opponentOpponent {
			return errors.New(
				fmt.Sprintf("Player %s's opponent's (%s) opponent (%s) is not themself.",
					tc.Players[i],
					opponent,
					opponentOpponent))
		}
	}
	return nil
}

func equalStandings(sa1 []*Standing, sa2 []*Standing) error {

	if len(sa1) != len(sa2) {
		return errors.New(fmt.Sprintf("Length of the standings are not equal: %d != %d\n", len(sa1), len(sa2)))
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

func equalStandingsRecord(s1 *Standing, s2 *Standing) error {
	if s1.Player != s2.Player ||
		s1.Wins != s2.Wins ||
		s1.Losses != s2.Losses ||
		s1.Draws != s2.Draws ||
		s1.Spread != s2.Spread {
		return errors.New(fmt.Sprintf("Standings do not match: (%s, %d, %d, %d, %d) != (%s, %d, %d, %d, %d)",
			s1.Player, s1.Wins, s1.Losses, s1.Draws, s1.Spread,
			s2.Player, s2.Wins, s2.Losses, s2.Draws, s2.Spread))
	}
	return nil
}

func getPlayerPairings(players []string, pris []*PlayerRoundInfo) []string {
	m := make(map[string]int)
	for _, player := range players {
		m[player] = 0
	}

	playerPairings := []string{}
	for _, pri := range pris {
		if m[pri.Pairing.Players[0]] == 0 {
			playerPairings = append(playerPairings, pri.Pairing.Players[0])
			playerPairings = append(playerPairings, pri.Pairing.Players[1])
			m[pri.Pairing.Players[0]] = 1
			m[pri.Pairing.Players[1]] = 1
		}
	}
	return playerPairings
}

func newPlayerRoundInfo(playerOne string, playerTwo string, gamesPerRound int) *PlayerRoundInfo {
	return &PlayerRoundInfo{Pairing: newClassicPairing(playerOne, playerTwo, gamesPerRound),
		Record: emptyRecord(),
		Spread: 0}
}

func equalPRI(pri1 *PlayerRoundInfo, pri2 *PlayerRoundInfo) error {
	err := equalPairing(pri1.Pairing, pri2.Pairing)
	if err != nil {
		return err
	}
	err = equalRecord(pri1.Record, pri2.Record)
	if err != nil {
		return err
	}
	if pri1.Spread != pri2.Spread {
		return errors.New(fmt.Sprintf("Spreads are not equal: %d != %d", pri1.Spread, pri2.Spread))
	}
	return nil
}

func equalPairing(p1 *Pairing, p2 *Pairing) error {
	if p1.Players[0] != p2.Players[0] || p1.Players[1] != p2.Players[1] {
		return errors.New(fmt.Sprintf("Players are not the same: (%s, %s) != (%s, %s)",
			p1.Players[0],
			p1.Players[1],
			p2.Players[0],
			p2.Players[1]))
	}
	if p1.Outcomes[0] != p2.Outcomes[0] || p1.Outcomes[1] != p2.Outcomes[1] {
		return errors.New(fmt.Sprintf("Outcomes are not the same: (%d, %d) != (%d, %d)",
			p1.Outcomes[0],
			p1.Outcomes[1],
			p2.Outcomes[0],
			p2.Outcomes[1]))
	}
	if len(p1.Games) != len(p2.Games) {
		return errors.New(fmt.Sprintf("Number of games are not the same: %d != %d", len(p1.Games), len(p2.Games)))
	}
	for i := 0; i < len(p1.Games); i++ {
		err := equalTournamentGame(p1.Games[i], p2.Games[i], i)
		if err != nil {
			return err
		}
	}
	return nil
}

func equalRecord(r1 []int, r2 []int) error {
	if len(r1) != len(r2) {
		return errors.New(fmt.Sprintf("Records are not the same length: %d != %d", len(r1), len(r2)))
	}
	for i := 0; i < len(r1); i++ {
		if r1[i] != r2[i] {
			return errors.New(fmt.Sprintf("Records are not equal at index %d: %d != %d\n", i, r1[i], r2[i]))
		}
	}
	return nil
}

func equalTournamentGame(t1 *TournamentGame, t2 *TournamentGame, i int) error {
	if t1.Scores[0] != t2.Scores[0] || t1.Scores[1] != t2.Scores[1] {
		return errors.New(fmt.Sprintf("Scores are not the same at game %d: (%d, %d) != (%d, %d)",
			i,
			t1.Scores[0],
			t1.Scores[1],
			t2.Scores[0],
			t2.Scores[1]))
	}
	if t1.Results[0] != t2.Results[0] || t1.Results[1] != t2.Results[1] {
		return errors.New(fmt.Sprintf("Results are not the same at game %d: (%d, %d) != (%d, %d)",
			i,
			t1.Results[0],
			t1.Results[1],
			t2.Results[0],
			t2.Results[1]))
	}
	if t1.GameEndReason != t2.GameEndReason {
		return errors.New(fmt.Sprintf("Game end reasons are not the same for game %d: %d != %d", i, t1.GameEndReason, t2.GameEndReason))
	}
	return nil
}

func equalRoundRobinPairings(s1 []string, s2 []string) error {
	if len(s1) != len(s2) {
		return errors.New(fmt.Sprintf("Pairing lengths do not match: %d != %d\n", len(s1), len(s2)))
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return errors.New(fmt.Sprintf("Pairings are not equal:\n%s\n%s\n", strings.Join(s1, ", "), strings.Join(s2, ", ")))
		}
	}
	return nil
}

func printPriPairings(pris []*PlayerRoundInfo) {
	for _, pri := range pris {
		fmt.Println(pri.Pairing)
	}
}

func printStandings(standings []*Standing) {
	for _, standing := range standings {
		fmt.Println(standing)
	}
}
