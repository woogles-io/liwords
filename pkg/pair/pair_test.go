package pair

import (
	"fmt"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/utilities"
)

// The vast majority of pairing tests are in the tournament package

func TestRoundRobin(t *testing.T) {
	// This test is used to ensure that round robin
	// pairings work correctly.

	is := is.New(t)

	// Test Round Robin with only two players.
	// This should obviously never be used
	// but it shouldn't throw any errors.
	pairings, err := getRoundRobinPairings(2, 0)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings, err = getRoundRobinPairings(2, 1)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings, err = getRoundRobinPairings(2, 2)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	numberOfPlayers := 8

	roundToPairingsMap := map[int][]int{
		getRoundRobinRotation(numberOfPlayers, 0): {7, 6, 5, 4, 3, 2, 1, 0},
		getRoundRobinRotation(numberOfPlayers, 1): {2, 3, 0, 1, 7, 6, 5, 4},
		getRoundRobinRotation(numberOfPlayers, 2): {4, 7, 6, 5, 0, 3, 2, 1},
		getRoundRobinRotation(numberOfPlayers, 3): {6, 4, 3, 2, 1, 7, 0, 5},
		getRoundRobinRotation(numberOfPlayers, 4): {1, 0, 7, 6, 5, 4, 3, 2},
		getRoundRobinRotation(numberOfPlayers, 5): {3, 5, 4, 0, 2, 1, 7, 6},
		getRoundRobinRotation(numberOfPlayers, 6): {5, 2, 1, 7, 6, 0, 4, 3}}

	// Modulus operation should repeat the pairings past the first round robin
	// Test first pairing of third round robin just to be sure

	for i := 0; i < 15; i++ {
		pairings, err := getRoundRobinPairings(numberOfPlayers, i)
		is.NoErr(err)
		is.NoErr(equalPairings(roundToPairingsMap[getRoundRobinRotation(numberOfPlayers, i)], pairings))
	}
}

func TestTeamRoundRobin(t *testing.T) {
	// This test is used to ensure that team round robin
	// pairings work correctly.

	is := is.New(t)

	// Test two teams of 5, triple round robin (15 rounds), team A vs team B.
	// Both teams are in the same division but they only play the other team.

	numberOfPlayers := 10

	roundToPairingsMap := map[int][]int{
		getTeamRoundRobinRotation(numberOfPlayers, 0, 3):  {9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
		getTeamRoundRobinRotation(numberOfPlayers, 1, 3):  {9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
		getTeamRoundRobinRotation(numberOfPlayers, 2, 3):  {9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
		getTeamRoundRobinRotation(numberOfPlayers, 3, 3):  {5, 4, 3, 2, 1, 0, 9, 8, 7, 6},
		getTeamRoundRobinRotation(numberOfPlayers, 4, 3):  {5, 4, 3, 2, 1, 0, 9, 8, 7, 6},
		getTeamRoundRobinRotation(numberOfPlayers, 5, 3):  {5, 4, 3, 2, 1, 0, 9, 8, 7, 6},
		getTeamRoundRobinRotation(numberOfPlayers, 6, 3):  {1, 0, 9, 8, 7, 6, 5, 4, 3, 2},
		getTeamRoundRobinRotation(numberOfPlayers, 7, 3):  {1, 0, 9, 8, 7, 6, 5, 4, 3, 2},
		getTeamRoundRobinRotation(numberOfPlayers, 8, 3):  {1, 0, 9, 8, 7, 6, 5, 4, 3, 2},
		getTeamRoundRobinRotation(numberOfPlayers, 9, 3):  {7, 6, 5, 4, 3, 2, 1, 0, 9, 8},
		getTeamRoundRobinRotation(numberOfPlayers, 10, 3): {7, 6, 5, 4, 3, 2, 1, 0, 9, 8},
		getTeamRoundRobinRotation(numberOfPlayers, 11, 3): {7, 6, 5, 4, 3, 2, 1, 0, 9, 8},
		getTeamRoundRobinRotation(numberOfPlayers, 12, 3): {3, 2, 1, 0, 9, 8, 7, 6, 5, 4},
		getTeamRoundRobinRotation(numberOfPlayers, 13, 3): {3, 2, 1, 0, 9, 8, 7, 6, 5, 4},
		getTeamRoundRobinRotation(numberOfPlayers, 14, 3): {3, 2, 1, 0, 9, 8, 7, 6, 5, 4},
	}

	for i := 0; i < 15; i++ {
		pairings, err := getTeamRoundRobinPairings(numberOfPlayers, i, 3)
		is.NoErr(err)
		is.NoErr(equalPairings(
			roundToPairingsMap[getTeamRoundRobinRotation(numberOfPlayers, i, 3)],
			pairings))
	}
}

func TestInitialFontes(t *testing.T) {
	is := is.New(t)

	allByes, err := getInitialFontesPairings(6, 7, 0)
	is.NoErr(err)
	for i := 0; i < len(allByes); i++ {
		is.True(allByes[i] == -1)
	}

	_, err = getInitialFontesPairings(9, 4, 0)
	is.NoErr(err)
	_, err = getInitialFontesPairings(9, 4, 1)
	is.NoErr(err)
	_, err = getInitialFontesPairings(9, 4, 2)
	is.NoErr(err)

	initialNumberOfPlayers := 16
	roundToPairingsMap := map[string][]int{
		"3:16:0": []int{12, 13, 14, 15, 8, 9, 10, 11, 4, 5, 6, 7, 0, 1, 2, 3},
		"3:16:1": []int{8, 9, 10, 11, 12, 13, 14, 15, 0, 1, 2, 3, 4, 5, 6, 7},
		"3:16:2": []int{4, 5, 6, 7, 0, 1, 2, 3, 12, 13, 14, 15, 8, 9, 10, 11},
		"3:17:0": []int{13, 14, 16, -1, 9, 10, 15, 11, 12, 4, 5, 7, 8, 0, 1, 6, 2},
		"3:17:1": []int{9, 10, 11, 8, 13, 14, 12, 16, 3, 0, 1, 2, 6, 4, 5, -1, 7},
		"3:17:2": []int{4, 5, 7, 15, 0, 1, 8, 2, 6, 13, 14, 16, -1, 9, 10, 3, 11},
		"3:18:0": []int{13, 14, 16, 17, 9, 10, 15, 11, 12, 4, 5, 7, 8, 0, 1, 6, 2, 3},
		"3:18:1": []int{9, 10, 11, 8, 13, 14, 12, 16, 3, 0, 1, 2, 6, 4, 5, 17, 7, 15},
		"3:18:2": []int{4, 5, 7, 15, 0, 1, 8, 2, 6, 13, 14, 16, 17, 9, 10, 3, 11, 12},
		"3:19:0": []int{15, 16, 17, 18, -1, 10, 11, 12, 13, 14, 5, 6, 7, 8, 9, 0, 1, 2, 3},
		"3:19:1": []int{10, 11, 12, 13, 14, 15, 16, 17, 18, -1, 0, 1, 2, 3, 4, 5, 6, 7, 8},
		"3:19:2": []int{5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 15, 16, 17, 18, -1, 10, 11, 12, 13},
		"5:16:0": []int{13, 15, 10, 14, 12, 8, 11, 9, 5, 7, 2, 6, 4, 0, 3, 1},
		"5:16:1": []int{5, 4, 8, 6, 1, 0, 3, 15, 2, 14, 13, 12, 11, 10, 9, 7},
		"5:16:2": []int{10, 7, 5, 12, 11, 2, 9, 1, 13, 6, 0, 4, 3, 8, 15, 14},
		"5:16:3": []int{2, 11, 0, 4, 3, 13, 15, 14, 10, 12, 8, 1, 9, 5, 7, 6},
		"5:16:4": []int{8, 14, 13, 11, 9, 10, 7, 6, 0, 4, 5, 3, 15, 2, 1, 12},
		"5:17:0": []int{15, 16, -1, 12, 13, 14, 9, 10, 11, 6, 7, 8, 3, 4, 5, 0, 1},
		"5:17:1": []int{6, 7, 8, 9, 10, 11, 0, 1, 2, 3, 4, 5, 15, 16, -1, 12, 13},
		"5:17:2": []int{12, 13, 14, 6, 7, 8, 3, 4, 5, 15, 16, -1, 0, 1, 2, 9, 10},
		"5:17:3": []int{3, 4, 5, 0, 1, 2, 15, 16, -1, 12, 13, 14, 9, 10, 11, 6, 7},
		"5:17:4": []int{9, 10, 11, 15, 16, -1, 12, 13, 14, 0, 1, 2, 6, 7, 8, 3, 4},
		"5:18:0": []int{15, 16, 17, 12, 13, 14, 9, 10, 11, 6, 7, 8, 3, 4, 5, 0, 1, 2},
		"5:18:1": []int{6, 7, 8, 9, 10, 11, 0, 1, 2, 3, 4, 5, 15, 16, 17, 12, 13, 14},
		"5:18:2": []int{12, 13, 14, 6, 7, 8, 3, 4, 5, 15, 16, 17, 0, 1, 2, 9, 10, 11},
		"5:18:3": []int{3, 4, 5, 0, 1, 2, 15, 16, 17, 12, 13, 14, 9, 10, 11, 6, 7, 8},
		"5:18:4": []int{9, 10, 11, 15, 16, 17, 12, 13, 14, 0, 1, 2, 6, 7, 8, 3, 4, 5},
		"5:19:0": []int{17, 18, -1, 13, 14, 16, 15, 10, 11, 12, 7, 8, 9, 3, 4, 6, 5, 0, 1},
		"5:19:1": []int{7, 8, 6, 10, 11, 9, 2, 0, 1, 5, 3, 4, -1, 17, 18, 16, 15, 13, 14},
		"5:19:2": []int{13, 14, 12, 7, 8, -1, 16, 3, 4, 15, 17, 18, 2, 0, 1, 9, 6, 10, 11},
		"5:19:3": []int{3, 4, 16, 0, 1, 12, 9, 17, 18, 6, 13, 14, 5, 10, 11, -1, 2, 7, 8},
		"5:19:4": []int{10, 11, 5, 17, 18, 2, -1, 13, 14, 16, 0, 1, 15, 7, 8, 12, 9, 3, 4}}

	for i := 0; i < 2; i++ {
		initialNumberOfFontesRounds := 3 + (i * 2)
		for j := 0; j < 4; j++ {
			numberOfPlayers := initialNumberOfPlayers + j

			for k := 0; k < initialNumberOfFontesRounds; k++ {
				pairings, err := getInitialFontesPairings(numberOfPlayers, initialNumberOfFontesRounds+1, k)
				is.NoErr(err)
				pairingsString := ""
				for _, value := range pairings {
					pairingsString += fmt.Sprintf("%2d, ", value)
				}
				key := fmt.Sprintf("%d:%d:%d", initialNumberOfFontesRounds, numberOfPlayers, k)
				is.NoErr(equalPairings(roundToPairingsMap[key], pairings))
			}
		}
	}
}

func equalPairings(s1 []int, s2 []int) error {
	if len(s1) != len(s2) {
		return fmt.Errorf("pairing lengths do not match: %d != %d", len(s1), len(s2))
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return fmt.Errorf("pairings are not equal:\n%s\n%s", utilities.IntArrayToString(s1), utilities.IntArrayToString(s2))
		}
	}
	return nil
}
