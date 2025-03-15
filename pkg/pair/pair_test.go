package pair

import (
	"fmt"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"

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
	pairings, err := getRoundRobinPairings(2, 0, 10)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings, err = getRoundRobinPairings(2, 1, 10)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings, err = getRoundRobinPairings(2, 2, 10)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	zerolog.SetGlobalLevel(zerolog.Disabled)
	for seed := int64(10); seed < 14; seed++ {
		for numberOfPlayers := 2; numberOfPlayers <= 25; numberOfPlayers++ {
			phasePairings := map[string]bool{}
			pairingsStr := ""
			prevPairingsStr := ""
			for round := 0; round < 60; round++ {
				pairings, err := getRoundRobinPairings(numberOfPlayers, round, seed)
				is.NoErr(err)
				pairingsStr += fmt.Sprintf(">%v<\n", pairings)
				for player, opponent := range pairings {
					if opponent == -1 {
						opponent = player
					} else if player > opponent {
						continue
					}
					key := fmt.Sprintf("%d-%d", player, opponent)
					is.True(!phasePairings[key])
					phasePairings[key] = true
				}
				if (round+1)%(numberOfPlayers-(1-(numberOfPlayers%2))) == 0 {
					if prevPairingsStr != "" {
						is.Equal(prevPairingsStr, pairingsStr)
					}
					n := numberOfPlayers + (numberOfPlayers % 2)
					is.Equal(len(phasePairings), (n*(n-1))/2)
					prevPairingsStr = pairingsStr
					pairingsStr = ""
					phasePairings = map[string]bool{}
				}
			}
		}
	}
}

func getRRPairingsOrDie(numberOfPlayers, round, gamesPerMatchup int, interleave bool) []int {
	pairings, err := getTeamRoundRobinPairings(numberOfPlayers, round, gamesPerMatchup, interleave)
	if err != nil {
		panic(err)
	}
	// pairings are index vs value
	// so e.g. [1, 0, 3, 2] means 0v1, 1v0, 2v3, 3v2
	return pairings
}

func TestTeamRoundRobin(t *testing.T) {
	// This test is used to ensure that team round robin
	// pairings work correctly.

	is := is.New(t)

	numberOfPlayers := 2
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 0, 1, false), []int{1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 1, 1, false), []int{1, 0}))

	numberOfPlayers = 4
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 0, 1, false), []int{3, 2, 1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 1, 1, false), []int{2, 3, 0, 1}))
	// Ensure the pattern restarts
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 2, 1, false), []int{3, 2, 1, 0}))
	numberOfPlayers = 10
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 0, 1, false), []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 1, 1, false), []int{5, 9, 8, 7, 6, 0, 4, 3, 2, 1}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 2, 1, false), []int{6, 5, 9, 8, 7, 1, 0, 4, 3, 2}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 3, 1, false), []int{7, 6, 5, 9, 8, 2, 1, 0, 4, 3}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 4, 1, false), []int{8, 7, 6, 5, 9, 3, 2, 1, 0, 4}))
	// Ensure the pattern restarts
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 5, 1, false), []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 6, 1, false), []int{5, 9, 8, 7, 6, 0, 4, 3, 2, 1}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 7, 1, false), []int{6, 5, 9, 8, 7, 1, 0, 4, 3, 2}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 8, 1, false), []int{7, 6, 5, 9, 8, 2, 1, 0, 4, 3}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 9, 1, false), []int{8, 7, 6, 5, 9, 3, 2, 1, 0, 4}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 10, 1, false), []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}))

	numberOfPlayers = 8
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 0, 1, false), []int{7, 6, 5, 4, 3, 2, 1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 1, 1, false), []int{4, 7, 6, 5, 0, 3, 2, 1}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 2, 1, false), []int{5, 4, 7, 6, 1, 0, 3, 2}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 3, 1, false), []int{6, 5, 4, 7, 2, 1, 0, 3}))
	// Ensure the pattern restarts
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 4, 1, false), []int{7, 6, 5, 4, 3, 2, 1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 5, 1, false), []int{4, 7, 6, 5, 0, 3, 2, 1}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 6, 1, false), []int{5, 4, 7, 6, 1, 0, 3, 2}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 7, 1, false), []int{6, 5, 4, 7, 2, 1, 0, 3}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 8, 1, false), []int{7, 6, 5, 4, 3, 2, 1, 0}))

}

func TestTeamRoundRobinInterleave(t *testing.T) {
	is := is.New(t)

	numberOfPlayers := 2
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 0, 1, true), []int{1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 1, 1, true), []int{1, 0}))

	numberOfPlayers = 4
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 0, 1, true), []int{3, 2, 1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 1, 1, true), []int{1, 0, 3, 2}))
	// Ensure the pattern restarts
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 2, 1, true), []int{3, 2, 1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 3, 1, true), []int{1, 0, 3, 2}))

	numberOfPlayers = 10
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 0, 1, true), []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 1, 1, true), []int{1, 0, 9, 8, 7, 6, 5, 4, 3, 2}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 2, 1, true), []int{3, 2, 1, 0, 9, 8, 7, 6, 5, 4}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 3, 1, true), []int{5, 4, 3, 2, 1, 0, 9, 8, 7, 6}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 4, 1, true), []int{7, 6, 5, 4, 3, 2, 1, 0, 9, 8}))
	// Ensure the pattern restarts
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 5, 1, true), []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 6, 1, true), []int{1, 0, 9, 8, 7, 6, 5, 4, 3, 2}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 7, 1, true), []int{3, 2, 1, 0, 9, 8, 7, 6, 5, 4}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 8, 1, true), []int{5, 4, 3, 2, 1, 0, 9, 8, 7, 6}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 9, 1, true), []int{7, 6, 5, 4, 3, 2, 1, 0, 9, 8}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 10, 1, true), []int{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}))

	numberOfPlayers = 8
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 0, 1, true), []int{7, 6, 5, 4, 3, 2, 1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 1, 1, true), []int{1, 0, 7, 6, 5, 4, 3, 2}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 2, 1, true), []int{3, 2, 1, 0, 7, 6, 5, 4}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 3, 1, true), []int{5, 4, 3, 2, 1, 0, 7, 6}))
	// Ensure the pattern restarts
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 4, 1, true), []int{7, 6, 5, 4, 3, 2, 1, 0}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 5, 1, true), []int{1, 0, 7, 6, 5, 4, 3, 2}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 6, 1, true), []int{3, 2, 1, 0, 7, 6, 5, 4}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 7, 1, true), []int{5, 4, 3, 2, 1, 0, 7, 6}))
	is.NoErr(equalPairings(getRRPairingsOrDie(numberOfPlayers, 8, 1, true), []int{7, 6, 5, 4, 3, 2, 1, 0}))
}

func TestInitialFontes(t *testing.T) {
	is := is.New(t)

	allByes, err := getInitialFontesPairings(6, 7, 0, 0)
	is.NoErr(err)
	for i := 0; i < len(allByes); i++ {
		is.True(allByes[i] == -1)
	}

	_, err = getInitialFontesPairings(18, 8, 0, 0)
	is.NoErr(err)

	_, err = getInitialFontesPairings(18, 3, 0, 0)
	is.Equal(err.Error(), "initial fontes pairing failure for 18 players, have odd group size of 3")

	_, err = getInitialFontesPairings(18, 10, 0, 0)
	is.Equal(err.Error(), "number of initial fontes rounds (9) should be less than half the number of players (18)")

	_, err = getInitialFontesPairings(9, 4, 0, 0)
	is.NoErr(err)
	_, err = getInitialFontesPairings(9, 4, 1, 0)
	is.NoErr(err)
	_, err = getInitialFontesPairings(9, 4, 2, 0)
	is.NoErr(err)

	zerolog.SetGlobalLevel(zerolog.Disabled)

	initialNumberOfPlayers := 16
	for i := 0; i < 2; i++ {
		initialNumberOfFontesRounds := 3 + (i * 2)
		for j := 0; j < 4; j++ {
			numberOfPlayers := initialNumberOfPlayers + j
			allPairings := map[string]bool{}
			for k := 0; k < initialNumberOfFontesRounds; k++ {
				pairings, err := getInitialFontesPairings(numberOfPlayers, initialNumberOfFontesRounds+1, k, 10)
				is.NoErr(err)
				fmt.Printf("pairings: %v\n", pairings)
				for player, opponent := range pairings {
					if opponent == -1 {
						opponent = player
					} else if player > opponent {
						continue
					}
					key := fmt.Sprintf("%d-%d", player, opponent)
					is.True(!allPairings[key])
					allPairings[key] = true
				}
			}
			fmt.Printf("numberOfPlayers: %d, initialNumberOfFontesRounds: %d, allPairings: %d\n", numberOfPlayers, initialNumberOfFontesRounds, len(allPairings))
			is.Equal(len(allPairings), ((numberOfPlayers+numberOfPlayers%2)*initialNumberOfFontesRounds)/2)
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
