package pair

import (
	"fmt"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"

	"github.com/woogles-io/liwords/pkg/utilities"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
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

	pairings0_1, err := getRoundRobinPairings(10, 0, 3)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings0_2, err := getRoundRobinPairings(10, 0, 3)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings9, err := getRoundRobinPairings(10, 9, 3)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings18, err := getRoundRobinPairings(10, 18, 3)
	is.NoErr(err)
	is.NoErr(equalPairings([]int{1, 0}, pairings))

	pairings20, err := getRoundRobinPairings(10, 20, 3)
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

func TestTeamRoundRobin(t *testing.T) {
	is := is.New(t)
	for _, pmethod := range []pb.PairingMethod{pb.PairingMethod_TEAM_ROUND_ROBIN, pb.PairingMethod_INTERLEAVED_ROUND_ROBIN} {
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
							pairings, err := getTeamRoundRobinPairings(numberOfPlayers, round, gamesPerMatchup, pmethod, false, seed)
							if numberOfPlayers%2 == 1 && pmethod == pb.PairingMethod_TEAM_ROUND_ROBIN {
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
								if pmethod == pb.PairingMethod_INTERLEAVED_ROUND_ROBIN {
									if opponent != player {
										is.Equal((opponent-player)%2, 1)
									}
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
