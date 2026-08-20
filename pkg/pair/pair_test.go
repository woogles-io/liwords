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
	pairings, err := GetRoundRobinPairings(2, 0, 10)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings, err = GetRoundRobinPairings(2, 1, 10)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings, err = GetRoundRobinPairings(2, 2, 10)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings0_1, err := GetRoundRobinPairings(10, 0, 3)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings0_2, err := GetRoundRobinPairings(10, 0, 3)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings9, err := GetRoundRobinPairings(10, 9, 3)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings18, err := GetRoundRobinPairings(10, 18, 3)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings20, err := GetRoundRobinPairings(10, 20, 3)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	is.NoErr(equalPairings(pairings0_1, pairings0_2))
	is.NoErr(equalPairings(pairings0_1, pairings9))
	is.NoErr(equalPairings(pairings9, pairings18))
	is.True(equalPairings(pairings20, pairings9) != nil)
	is.True(equalPairings(pairings20, pairings18) != nil)

	zerolog.SetGlobalLevel(zerolog.Disabled)
	for seed := uint64(10); seed < 14; seed++ {
		for numberOfPlayers := 2; numberOfPlayers <= 25; numberOfPlayers++ {
			phasePairings := map[string]bool{}
			pairingsStr := ""
			prevPairingsStr := ""
			for round := 0; round < 60; round++ {
				pairings, err := GetRoundRobinPairings(numberOfPlayers, round, seed)
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

func TestTeamRoundRobin(t *testing.T) {
	is := is.New(t)
	for interleave := false; interleave != true; interleave = !interleave {
		for seed := uint64(10); seed < 14; seed++ {
			for numberOfPlayers := 2; numberOfPlayers <= 25; numberOfPlayers++ {
				halfNoP := numberOfPlayers / 2
				evenedNoP := numberOfPlayers + (numberOfPlayers % 2)
				halfEvenedNoP := evenedNoP / 2
				for gamesPerMatchup := 1; gamesPerMatchup <= 4; gamesPerMatchup++ {
					for matchups := halfEvenedNoP; matchups <= halfEvenedNoP*3; matchups++ {
						maxRound := matchups * gamesPerMatchup
						pairingsStr := ""
						prevPairingsStr := ""
						phasePairingsStr := ""
						prevPhasePairingsStr := ""
						phasePairings := map[string]int{}
						for round := range maxRound {
							pairings, err := getTeamRoundRobinPairings(numberOfPlayers, round, gamesPerMatchup, interleave, seed)
							if numberOfPlayers%2 == 1 && !interleave {
								is.True(err != nil)
								continue
							}
							is.NoErr(err)
							pairingsStr = fmt.Sprintf(">%v<", pairings)
							if round > 0 && numberOfPlayers > 2 {
								if round%gamesPerMatchup == 0 {
									is.True(pairingsStr != prevPairingsStr)
								} else {
									is.Equal(pairingsStr, prevPairingsStr)
								}
							}
							prevPairingsStr = pairingsStr
							phasePairingsStr += pairingsStr
							for player, opponent := range pairings {
								if opponent == -1 {
									opponent = player
								} else if player > opponent {
									continue
								}
								key := fmt.Sprintf("%d-%d", player, opponent)
								_, exists := phasePairings[key]
								if exists {
									phasePairings[key]++
								} else {
									phasePairings[key] = 1
								}
								if interleave {
									is.Equal((opponent-player)%2, 0)
								} else {
									is.True((opponent < halfNoP && player >= halfNoP) || (player < halfNoP && opponent >= halfNoP))
								}
							}
							if (round+1)%(halfEvenedNoP*gamesPerMatchup) == 0 {
								if prevPhasePairingsStr != "" {
									is.Equal(prevPhasePairingsStr, phasePairingsStr)
								}
								phasePairingsSum := 0
								for _, v := range phasePairings {
									phasePairingsSum += v
								}
								is.Equal(phasePairingsSum, halfEvenedNoP*halfEvenedNoP*gamesPerMatchup)
								prevPhasePairingsStr = phasePairingsStr
								phasePairingsStr = ""
								phasePairings = map[string]int{}
							}
						}
					}
				}
			}
		}
	}
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

	initialNumberOfPlayers := 16
	for i := 0; i < 2; i++ {
		initialNumberOfFontesRounds := 3 + (i * 2)
		for j := 0; j < 4; j++ {
			numberOfPlayers := initialNumberOfPlayers + j
			allPairings := map[string]bool{}
			for k := 0; k < initialNumberOfFontesRounds; k++ {
				pairings, err := getInitialFontesPairings(numberOfPlayers, initialNumberOfFontesRounds+1, k, 10)
				is.NoErr(err)
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
			is.Equal(len(allPairings), ((numberOfPlayers+numberOfPlayers%2)*initialNumberOfFontesRounds)/2)
		}
	}
}

func TestGetShirtsVsSkinsPairings(t *testing.T) {
	is := is.New(t)

	// Group of 6, local ranks 1-6 (0-indexed 0-5).
	// Team A = local indices 0,2,4 (ranks 1,3,5)
	// Team B = local indices 1,3,5 (ranks 2,4,6)
	is.NoErr(equalPairings([]int{1, 0, 3, 2, 5, 4}, getShirtsVsSkinsPairings(6, 0))) // 1v2 3v4 5v6
	is.NoErr(equalPairings([]int{3, 4, 5, 0, 1, 2}, getShirtsVsSkinsPairings(6, 1))) // 1v4 3v6 5v2
	is.NoErr(equalPairings([]int{5, 2, 1, 4, 3, 0}, getShirtsVsSkinsPairings(6, 2))) // 1v6 3v2 5v4

	// No pairing should repeat across the 3 available rounds.
	seen := map[string]bool{}
	for round := 0; round < 3; round++ {
		pairings := getShirtsVsSkinsPairings(6, round)
		for player, opponent := range pairings {
			if player > opponent {
				continue
			}
			key := fmt.Sprintf("%d-%d", player, opponent)
			is.True(!seen[key])
			seen[key] = true
		}
	}
}

func TestInitialFontesRemainderGroupUsesShirtsVsSkins(t *testing.T) {
	is := is.New(t)

	// 18 players, InitialFontes = 3 (numberOfNtiles = 4, i.e. quartiles).
	// With a remainder of 2, the oversized remainder group has size 6
	// (numberOfNtiles + remainder), built from 6 mini-groups of 3 ranks each
	// (1-3, 4-6, ..., 16-18): the worst-ranked (last) player of each mini-group
	// is pulled into the remainder group, and the rest are distributed round
	// robin across the other 3 standard-size groups:
	//   group 0: 0,4,9,13
	//   group 1: 1,6,10,15
	//   group 2: 3,7,12,16
	//   group 3 (remainder): 2,5,8,11,14,17
	// Group 3 (size 6, k=3) has enough capacity for all 3 fontes rounds, so it
	// should be paired via the shirts-vs-skins cross pattern instead of round
	// robin: Team A = {2,8,14} (local ranks 1,3,5), Team B = {5,11,17} (local
	// ranks 2,4,6).
	round0, err := getInitialFontesPairings(18, 4, 0, 0)
	is.NoErr(err)
	is.Equal(round0[2], 5)
	is.Equal(round0[8], 11)
	is.Equal(round0[14], 17)

	round1, err := getInitialFontesPairings(18, 4, 1, 0)
	is.NoErr(err)
	is.Equal(round1[2], 11)
	is.Equal(round1[8], 17)
	is.Equal(round1[5], 14)

	round2, err := getInitialFontesPairings(18, 4, 2, 0)
	is.NoErr(err)
	is.Equal(round2[2], 17)
	is.Equal(round2[5], 8)
	is.Equal(round2[11], 14)

	// No pairing among the remainder group should repeat across the 3 rounds.
	seen := map[string]bool{}
	for _, pairings := range [][]int{round0, round1, round2} {
		for _, player := range []int{2, 5, 8, 11, 14, 17} {
			opponent := pairings[player]
			if player > opponent {
				continue
			}
			key := fmt.Sprintf("%d-%d", player, opponent)
			is.True(!seen[key])
			seen[key] = true
		}
	}
}

func TestInitialFontesGeneralizedNtiles(t *testing.T) {
	is := is.New(t)

	// numberOfNtiles = 6 (InitialFontes = 5 rounds), roundsNeeded = 5.
	//
	// 20 players -> remainder group of size 8, k=4 < roundsNeeded(5),
	// so this falls back to round robin for the remainder group.
	//
	// 22 players -> remainder group of size 10, k=5 >= roundsNeeded(5),
	// so this uses the shirts-vs-skins cross pattern for the remainder group.
	for _, numberOfPlayers := range []int{20, 22} {
		allPairings := map[string]bool{}
		for round := 0; round < 5; round++ {
			pairings, err := getInitialFontesPairings(numberOfPlayers, 6, round, 10)
			is.NoErr(err)
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
		is.Equal(len(allPairings), (numberOfPlayers*5)/2)
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
