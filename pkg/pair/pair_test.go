package pair

import (
	"errors"
	"fmt"
	"math/rand/v2"
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

// previousTeamRoundRobinPairings is getTeamRoundRobinPairings as it was before
// the snaked round robin, kept to check that the team and interleaved round
// robins still pair exactly as they did for tournaments already under way.
func previousTeamRoundRobinPairings(numberOfPlayers, round, gamesPerMatchup int, interleave bool, seed uint64) ([]int, error) {
	// A team round robin contains two teams: A and B
	// Everyone in A plays everyone in B gamesPerMatchup times (in a row for speed).
	if numberOfPlayers%2 == 1 && !interleave {
		return nil, errors.New("cannot have an odd number of players without interleaving team round robin pairings")
	}

	players := []int{}
	if !interleave {
		for i := 0; i < numberOfPlayers; i++ {
			players = append(players, i)
		}
	} else {
		for i := 0; i < numberOfPlayers; i += 2 {
			players = append(players, i)
		}
		for i := 1; i < numberOfPlayers; i += 2 {
			players = append(players, i)
		}
	}

	bye := numberOfPlayers%2 == 1
	// If there are an odd number of players add a bye
	if bye {
		players = append(players, numberOfPlayers)
	}

	l := len(players)

	lbh := l / 2
	rotatedBottomPlayers := players[lbh:l]

	source := rand.NewPCG(seed, 0)
	rng := rand.New(source)
	rng.Shuffle(len(rotatedBottomPlayers),
		func(i, j int) {
			rotatedBottomPlayers[i], rotatedBottomPlayers[j] = rotatedBottomPlayers[j], rotatedBottomPlayers[i]
		})

	rotationIndex := (round / gamesPerMatchup) % lbh
	rotatedBottomPlayers = append(rotatedBottomPlayers[rotationIndex:lbh], rotatedBottomPlayers[0:rotationIndex]...)

	firstSeg := rotatedBottomPlayers[0 : lbh/2]
	secondSeg := rotatedBottomPlayers[lbh/2 : lbh]

	topHalf := players[0:lbh]
	bottomHalf := append(firstSeg, secondSeg...)
	utilities.Reverse(bottomHalf)

	pairings := []int{}

	// Assign -2 as default so if it doesn't
	// get overwritten we know something is wrong
	for i := 0; i < len(players); i++ {
		pairings = append(pairings, -2)
	}

	// 	Interface("players", players).Int("numPlayers", numberOfPlayers).Int("round", round).
	// 	Interface("topHalf", topHalf).Interface("bottomHalf", bottomHalf).Msg("debug-pairings")

	for i := 0; i < len(players)/2; i++ {
		pairings[topHalf[i]] = bottomHalf[i]
		pairings[bottomHalf[i]] = topHalf[i]
	}

	// Convert the bye from l to -1 because it's easier
	// on the Round Robin pairing algorithm
	if bye {
		for i := 0; i < len(pairings); i++ {
			if pairings[i] == numberOfPlayers {
				pairings[i] = -1
			}
		}
		pairings = pairings[0 : len(pairings)-1]
	}

	return pairings, nil
}

func TestTeamRoundRobinPairsAsBefore(t *testing.T) {
	is := is.New(t)
	for _, interleave := range []bool{false, true} {
		method := pb.PairingMethod_TEAM_ROUND_ROBIN
		if interleave {
			method = pb.PairingMethod_INTERLEAVED_ROUND_ROBIN
		}
		for seed := uint64(10); seed < 14; seed++ {
			for numberOfPlayers := 2; numberOfPlayers <= 25; numberOfPlayers++ {
				if numberOfPlayers%2 == 1 && !interleave {
					continue
				}
				for gamesPerMatchup := 1; gamesPerMatchup <= 3; gamesPerMatchup++ {
					for round := 0; round < 3*numberOfPlayers*gamesPerMatchup; round++ {
						want, err := previousTeamRoundRobinPairings(numberOfPlayers, round, gamesPerMatchup, interleave, seed)
						is.NoErr(err)
						got, err := getTeamRoundRobinPairings(numberOfPlayers, round, gamesPerMatchup, method, false, seed)
						is.NoErr(err)
						is.Equal(got, want)
					}
				}
			}
		}
	}
}

func TestSnakedTeams(t *testing.T) {
	is := is.New(t)
	teamA, teamB, err := getTeams(10, pb.PairingMethod_SNAKED_ROUND_ROBIN)
	is.NoErr(err)
	is.Equal(teamA, []int{0, 3, 4, 7, 8})
	is.Equal(teamB, []int{1, 2, 5, 6, 9})

	// With seven players, team A is the short one, so it gets the bye seat:
	// seeds 1, 4 and 5 never meet each other, and each sits out once.
	byes := map[int]int{}
	for round := 0; round < 4; round++ {
		pairings, err := getTeamRoundRobinPairings(7, round, 1, pb.PairingMethod_SNAKED_ROUND_ROBIN, false, 3)
		is.NoErr(err)
		for player, opponent := range pairings {
			if opponent == -1 {
				byes[player]++
			} else {
				is.True((player%4 == 0 || player%4 == 3) != (opponent%4 == 0 || opponent%4 == 3))
			}
		}
	}
	is.Equal(byes, map[int]int{1: 1, 2: 1, 5: 1, 6: 1})
}

// teamRoundRobinCycle is how many rounds it takes for every team in a
// team-style round robin to finish a whole number of cycles.
func teamRoundRobinCycle(numberOfPlayers, gamesPerMatchup int, method pb.PairingMethod, playWithinTeam bool) int {
	if !playWithinTeam {
		return (numberOfPlayers + 1) / 2 * gamesPerMatchup
	}
	teamA, teamB, _ := getTeams(numberOfPlayers, method)
	cycle := 1
	for _, team := range [][]int{teamA, teamB} {
		// A team plays its round robin in an odd number of rounds: one per
		// opponent, plus a bye round if it has an odd number of players.
		rounds := max(len(team)-1+len(team)%2, 1)
		lcm := cycle
		for lcm%rounds != 0 {
			lcm += cycle
		}
		cycle = lcm
	}
	return cycle * gamesPerMatchup
}

func TestTeamRoundRobinFirsts(t *testing.T) {
	type mode struct {
		method         pb.PairingMethod
		playWithinTeam bool
		// The most firsts or seconds in a row anyone may get.
		maxStreak int
	}
	modes := []mode{
		{pb.PairingMethod_TEAM_ROUND_ROBIN, false, 2},
		// Its pairings are kept as they were, which costs an occasional third
		// first or second in a row in odd fields over more than one cycle.
		{pb.PairingMethod_INTERLEAVED_ROUND_ROBIN, false, 3},
		{pb.PairingMethod_INTERLEAVED_ROUND_ROBIN, true, 2},
		{pb.PairingMethod_SNAKED_ROUND_ROBIN, false, 2},
		{pb.PairingMethod_SNAKED_ROUND_ROBIN, true, 2},
	}
	for _, md := range modes {
		for gamesPerMatchup := 1; gamesPerMatchup <= 3; gamesPerMatchup++ {
			if gamesPerMatchup > 1 && md.method != pb.PairingMethod_TEAM_ROUND_ROBIN {
				continue
			}
			for numberOfPlayers := 2; numberOfPlayers <= 30; numberOfPlayers++ {
				if numberOfPlayers%2 == 1 && md.method == pb.PairingMethod_TEAM_ROUND_ROBIN {
					continue
				}
				for seed := uint64(1); seed < 6; seed++ {
					name := fmt.Sprintf("%s within=%v games=%d players=%d seed=%d",
						md.method, md.playWithinTeam, gamesPerMatchup, numberOfPlayers, seed)
					checkTeamRoundRobinFirsts(t, name, numberOfPlayers, gamesPerMatchup, md.method,
						md.playWithinTeam, md.maxStreak, seed)
				}
			}
		}
	}
}

func checkTeamRoundRobinFirsts(t *testing.T, name string, numberOfPlayers, gamesPerMatchup int,
	method pb.PairingMethod, playWithinTeam bool, maxStreak int, seed uint64) {

	teamA, _, err := getTeams(numberOfPlayers, method)
	if err != nil {
		t.Fatal(name, err)
	}
	onTeamA := map[int]bool{}
	for _, p := range teamA {
		onTeamA[p] = true
	}
	cycle := teamRoundRobinCycle(numberOfPlayers, gamesPerMatchup, method, playWithinTeam)

	firsts := make([]int, numberOfPlayers)
	games := make([]int, numberOfPlayers)
	streaks := make([]int, numberOfPlayers)
	lastFirst := make([]bool, numberOfPlayers)
	// Firsts minus seconds between each two players.
	pairFirsts := map[[2]int]int{}
	meetings := map[[2]int]int{}

	for round := 0; round < 4*cycle; round++ {
		pairings, goesFirst, err := getTeamRoundRobinSchedule(numberOfPlayers, round, gamesPerMatchup,
			method, playWithinTeam, seed)
		if err != nil {
			t.Fatal(name, err)
		}
		byes := 0
		for player, opponent := range pairings {
			if opponent == -1 {
				byes++
				continue
			}
			if pairings[opponent] != player {
				t.Fatalf("%s round %d: %d plays %d, who plays %d", name, round, player, opponent, pairings[opponent])
			}
			if goesFirst[player] == goesFirst[opponent] {
				t.Fatalf("%s round %d: %d and %d both go first or both second", name, round, player, opponent)
			}
			sameTeam := onTeamA[player] == onTeamA[opponent]
			if playWithinTeam != sameTeam {
				// Within teams of odd size, the two bye receivers play across.
				if !playWithinTeam || len(teamA)%2 == 0 || (numberOfPlayers-len(teamA))%2 == 0 {
					t.Fatalf("%s round %d: %d plays %d across the wrong teams", name, round, player, opponent)
				}
			}
			key := [2]int{min(player, opponent), max(player, opponent)}
			if player < opponent {
				meetings[key]++
			}
			if goesFirst[player] {
				firsts[player]++
				if player == key[0] {
					pairFirsts[key]++
				} else {
					pairFirsts[key]--
				}
			}
			if games[player] > 0 && goesFirst[player] == lastFirst[player] {
				streaks[player]++
			} else {
				streaks[player] = 1
			}
			if streaks[player] > maxStreak {
				t.Fatalf("%s round %d: player %d has %d firsts or seconds in a row", name, round, player, streaks[player])
			}
			lastFirst[player] = goesFirst[player]
			games[player]++
			// A bye can leave a player two firsts or seconds up for a while.
			if diff := 2*firsts[player] - games[player]; diff > 2 || diff < -2 {
				t.Fatalf("%s round %d: player %d has %d firsts in %d games", name, round, player, firsts[player], games[player])
			}
		}
		if byes > numberOfPlayers%2 {
			t.Fatalf("%s round %d: %d byes", name, round, byes)
		}

		if (round+1)%cycle == 0 && !playWithinTeam {
			// After each round robin, everyone is within one of even.
			for player := range firsts {
				if diff := 2*firsts[player] - games[player]; diff > 1 || diff < -1 {
					t.Fatalf("%s round %d: player %d has %d firsts in %d games", name, round, player, firsts[player], games[player])
				}
			}
		}
		if (round+1)%(2*cycle) == 0 {
			// After a whole number of double round robins, everyone has gone
			// first exactly as often as second, and so has every pair.
			for player := range firsts {
				if 2*firsts[player] != games[player] {
					t.Fatalf("%s round %d: player %d has %d firsts in %d games", name, round, player, firsts[player], games[player])
				}
			}
			for key, diff := range pairFirsts {
				if diff != 0 {
					t.Fatalf("%s round %d: %d went first %d more times than %d", name, round, key[0], diff, key[1])
				}
			}
		}
	}
	// Everyone met every opponent they should, equally often: within teams,
	// a smaller team gets round more often, so only pairs from the same team
	// (or both across) need meet as often as each other.
	want := map[int]int{}
	for key, n := range meetings {
		group := 2
		if onTeamA[key[0]] == onTeamA[key[1]] && onTeamA[key[0]] {
			group = 0
		} else if onTeamA[key[0]] == onTeamA[key[1]] {
			group = 1
		}
		if _, ok := want[group]; !ok {
			want[group] = n
		}
		if n != want[group] {
			t.Fatalf("%s: uneven meetings %v", name, meetings)
		}
	}
	pairs := 0
	for range meetings {
		pairs++
	}
	teamB := numberOfPlayers - len(teamA)
	wantPairs := len(teamA) * teamB
	if playWithinTeam {
		wantPairs = len(teamA)*(len(teamA)-1)/2 + teamB*(teamB-1)/2
	}
	if pairs < wantPairs {
		t.Fatalf("%s: only %d of %d pairs met", name, pairs, wantPairs)
	}
}
