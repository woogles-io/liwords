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

// initialFontesGroups returns the pairings of every Initial Fontes round and
// the groups they were drawn from: two players are in the same group if they
// are connected through those rounds' pairings. Each group is in rank order,
// and groups are ordered by their best-ranked player.
func initialFontesGroups(t *testing.T, numberOfPlayers, numberOfNtiles int, seed uint64) ([][]int, [][]int) {
	t.Helper()
	parent := make([]int, numberOfPlayers)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	rounds := [][]int{}
	for round := 0; round < numberOfNtiles-1; round++ {
		pairings, err := getInitialFontesPairings(numberOfPlayers, numberOfNtiles, round, seed)
		if err != nil {
			t.Fatal(err)
		}
		rounds = append(rounds, pairings)
		for player, opponent := range pairings {
			if opponent >= 0 {
				parent[find(player)] = find(opponent)
			}
		}
	}
	groupIdxByRoot := map[int]int{}
	groups := [][]int{}
	for player := 0; player < numberOfPlayers; player++ {
		root := find(player)
		groupIdx, ok := groupIdxByRoot[root]
		if !ok {
			groupIdx = len(groups)
			groupIdxByRoot[root] = groupIdx
			groups = append(groups, []int{})
		}
		groups[groupIdx] = append(groups[groupIdx], player)
	}
	return rounds, groups
}

func TestInitialFontesRandomizesNtileSelection(t *testing.T) {
	is := is.New(t)

	// 16 players, InitialFontes = 3 (numberOfNtiles = 4, i.e. quartiles):
	// 4 groups of 4, each with one player from each of ranks 1-4, 5-8, 9-12,
	// and 13-16. Which player from each quartile is randomized by the seed,
	// rather than always grouping ranks 1, 5, 9, 13 and so on.
	distinctGroupings := map[string]bool{}
	for seed := uint64(0); seed < 20; seed++ {
		_, groups := initialFontesGroups(t, 16, 4, seed)
		is.Equal(len(groups), 4)
		for _, group := range groups {
			is.Equal(len(group), 4)
			for quartile, player := range group {
				is.Equal(player/4, quartile)
			}
		}
		distinctGroupings[fmt.Sprint(groups)] = true

		// The same seed always produces the same groups.
		_, again := initialFontesGroups(t, 16, 4, seed)
		is.Equal(fmt.Sprint(again), fmt.Sprint(groups))
	}
	is.True(len(distinctGroupings) > 1)
}

func TestInitialFontesRemainderGroupUsesShirtsVsSkins(t *testing.T) {
	is := is.New(t)

	// 18 players, InitialFontes = 3 (numberOfNtiles = 4, i.e. quartiles).
	// With a remainder of 2, the oversized remainder group has size 6
	// (numberOfNtiles + remainder), built from 6 mini-groups of 3 ranks each
	// (1-3, 4-6, ..., 16-18): one random player of each mini-group is pulled
	// into the remainder group. The other 12 players, in rank order, form
	// 4 N-tiles of 3, and each of the 3 standard-size groups takes one random
	// player from each of those N-tiles.
	distinctGroupings := map[string]bool{}
	for seed := uint64(0); seed < 20; seed++ {
		rounds, groups := initialFontesGroups(t, 18, 4, seed)
		is.Equal(len(groups), 4)
		distinctGroupings[fmt.Sprint(groups)] = true

		var remainder []int
		standardPlayers := []int{}
		standardGroups := [][]int{}
		for _, group := range groups {
			if len(group) == 6 {
				is.True(remainder == nil)
				remainder = group
			} else {
				is.Equal(len(group), 4)
				standardGroups = append(standardGroups, group)
			}
		}
		is.True(remainder != nil)
		for miniGroup, player := range remainder {
			is.Equal(player/3, miniGroup)
		}
		inRemainder := map[int]bool{}
		for _, player := range remainder {
			inRemainder[player] = true
		}
		for player := 0; player < 18; player++ {
			if !inRemainder[player] {
				standardPlayers = append(standardPlayers, player)
			}
		}
		ntileOf := map[int]int{}
		for i, player := range standardPlayers {
			ntileOf[player] = i / 3
		}
		for _, group := range standardGroups {
			for ntile, player := range group {
				is.Equal(ntileOf[player], ntile)
			}
		}

		// The remainder group (size 6, k=3) has enough capacity for all 3
		// fontes rounds, so it is paired via the shirts-vs-skins cross pattern
		// instead of round robin: Team A = local ranks 1,3,5 and Team B = local
		// ranks 2,4,6.
		r := remainder
		is.Equal(rounds[0][r[0]], r[1])
		is.Equal(rounds[0][r[2]], r[3])
		is.Equal(rounds[0][r[4]], r[5])
		is.Equal(rounds[1][r[0]], r[3])
		is.Equal(rounds[1][r[2]], r[5])
		is.Equal(rounds[1][r[1]], r[4])
		is.Equal(rounds[2][r[0]], r[5])
		is.Equal(rounds[2][r[1]], r[2])
		is.Equal(rounds[2][r[3]], r[4])

		// No pairing among the remainder group should repeat across the 3 rounds.
		seen := map[string]bool{}
		for _, pairings := range rounds {
			for _, player := range remainder {
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
	is.True(len(distinctGroupings) > 1)
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
