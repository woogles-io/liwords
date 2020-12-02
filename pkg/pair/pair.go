package pair

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"math/rand"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/matching"
	"github.com/domino14/liwords/pkg/utilities"
)

func Pair(members *entity.UnpairedPoolMembers) ([]int, error) {

	pm := members.RoundControls.PairingMethod
	if pm == entity.Manual {
		return nil, errors.New("cannot pair with the given pairing method")
	}
	// This way of dispatching is slightly clunky and will
	// remain until we can think of a better way to do it.
	var pairings []int
	var err error
	if pm == entity.Random {
		pairings, err = pairRandom(members)
	} else if pm == entity.RoundRobin {
		pairings, err = pairRoundRobin(members)
	} else if pm == entity.KingOfTheHill || pm == entity.Elimination {
		pairings, err = pairKingOfTheHill(members)
	} else if pm == entity.Factor {
		pairings, err = pairFactor(members)
	} else {
		// The remaining pairing methods are solved by
		// reduction to minimum weight matching
		pairings, err = minWeightMatching(members)
	}
	return pairings, err
}

func GetRepeatKey(playerOne string, playerTwo string) string {
	if playerTwo < playerOne {
		playerOne, playerTwo = playerTwo, playerOne
	}

	return playerOne + "::" + playerTwo
}

func pairRandom(members *entity.UnpairedPoolMembers) ([]int, error) {

	poolMembers := members.PoolMembers
	playerIndexes := []int{}
	for i, _ := range members.PoolMembers {
		playerIndexes = append(playerIndexes, i)
	}
	// rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(playerIndexes),
		func(i, j int) {
			playerIndexes[i], playerIndexes[j] = playerIndexes[j], playerIndexes[i]
		})

	pairings := []int{}
	for i := 0; i < len(poolMembers); i++ {
		pairings = append(pairings, -1)
	}

	for i := 0; i < len(playerIndexes)-1; i += 2 {
		pairings[playerIndexes[i]] = playerIndexes[i+1]
		pairings[playerIndexes[i+1]] = playerIndexes[i]
	}
	return pairings, nil
}

func pairRoundRobin(members *entity.UnpairedPoolMembers) ([]int, error) {
	players := []int{}
	l := len(members.PoolMembers)
	for i := 0; i < l; i++ {
		players = append(players, i)
	}

	bye := l%2 == 1
	// If there are an odd number of players add a bye
	if bye {
		players = append(players, l)
	}

	pairings := getRoundRobinPairings(players, members.RoundControls.Round)

	// Convert byes from l to -1 because it's easier
	// on the Round Robin pairing algorithm
	if bye {
		for i := 0; i < len(pairings); i++ {
			if pairings[i] == l {
				pairings[i] = -1
			}
		}
		pairings = pairings[0 : len(pairings)-1]
	}

	return pairings, nil
}

func pairKingOfTheHill(members *entity.UnpairedPoolMembers) ([]int, error) {
	pairings := []int{}
	for i := 0; i < len(members.PoolMembers); i++ {
		pairings = append(pairings, -1)
	}

	for i := 0; i < len(members.PoolMembers)-1; i += 2 {
		pairings[i] = i + 1
		pairings[i+1] = i
	}

	return pairings, nil
}

func pairFactor(members *entity.UnpairedPoolMembers) ([]int, error) {

	// First pair everyone with KOH then overwrite with Factor
	pairings, err := pairKingOfTheHill(members)
	if err != nil {
		return nil, err
	}
	numberOfPlayers := len(pairings)
	for i := 0; i < members.RoundControls.Factor; i += 1 {
		factor := i + members.RoundControls.Factor
		if factor >= numberOfPlayers {
			return nil, fmt.Errorf("cannot pair with factor %d on %d players", factor, numberOfPlayers)
		}
		pairings[i] = factor
		pairings[factor] = i
	}

	return pairings, nil
}

func minWeightMatching(members *entity.UnpairedPoolMembers) ([]int, error) {
	numberOfMembers := len(members.PoolMembers)
	edges := []*matching.Edge{}

	if members.RoundControls.RepeatRelativeWeight > entity.MaxRelativeWeight {
		members.RoundControls.RepeatRelativeWeight = entity.MaxRelativeWeight
	}

	if members.RoundControls.WinDifferenceRelativeWeight > entity.MaxRelativeWeight {
		members.RoundControls.WinDifferenceRelativeWeight = entity.MaxRelativeWeight
	}

	for i := 0; i < numberOfMembers; i++ {
		for j := i + 1; j < numberOfMembers; j++ {
			if pairable(members, i, j) {
				weight, err := weigh(members, i, j)
				if err != nil {
					return nil, err
				}
				edges = append(edges, matching.NewEdge(i, j, weight))
			}
		}
	}

	pairings, weight, err := matching.MinWeightMatching(edges, true)

	if err != nil {
		log.Debug().Msgf("matching failed: %v", edges)
		return nil, err
	}

	if len(pairings) != numberOfMembers {
		log.Debug().Msgf("matching incomplete: %v", edges)
		return nil, errors.New("pairings and members are not the same length")
	}

	if weight >= entity.ProhibitiveWeight {
		return nil, errors.New("prohibitive weight reached, pairings are not possible with these settings")
	}

	if members.RoundControls.PairingMethod == entity.Quickpair {
		for index, pairing := range pairings {
			if pairing == -1 {
				members.PoolMembers[index].Misses++
			}
		}
	}

	return pairings, nil
}

func pairable(members *entity.UnpairedPoolMembers, i int, j int) bool {
	// There is probably a better way to do this, but for now:
	PoolMemberA := members.PoolMembers[i]
	PoolMemberB := members.PoolMembers[j]
	for _, blockedId := range PoolMemberA.Blocking {
		if PoolMemberB.Id == blockedId {
			return false
		}
	}
	for _, blockedId := range PoolMemberB.Blocking {
		if PoolMemberA.Id == blockedId {
			return false
		}
	}
	return true
}

func weigh(members *entity.UnpairedPoolMembers, i int, j int) (int64, error) {
	// This way of dispatching is slightly clunky and will
	// remain until we can think of a better way to do it.
	var weight int64
	pm := members.RoundControls.PairingMethod
	if pm == entity.Swiss {
		weight = weighSwiss(members, i, j)
	} else if pm == entity.Quickpair {
		weight = weighQuickpair(members, i, j)
	} else {
		return 0, errors.New("pairing method is either unimplemented or is not a reduction to minimum weight matching")
	}
	return weight, nil
}

func weighSwiss(members *entity.UnpairedPoolMembers, i int, j int) int64 {
	p1 := members.PoolMembers[i]
	p2 := members.PoolMembers[j]
	// Scale up wins to ensure any single edge win difference weight
	// outweighs the sum of all of the edge's possible spread weight

	// The unscaled weight difference where a win is worth 2 and a draw is worth 1
	unscaledWinDiffWeight := utilities.Abs(((p1.Wins - p2.Wins) * 2) + (p1.Draws - p2.Draws))

	// Now scale the weight difference by WinWeightScaling, but divide by 2 because
	// we have already multitplied by 2 in the previous step. This was so that
	// all arithmetic can stay in integer form.
	winDiffWeight := int64(unscaledWinDiffWeight) * (entity.WinWeightScaling / 2) *
		int64(members.RoundControls.WinDifferenceRelativeWeight)

	// Subtract the spread difference for swiss, since we would like to pair players
	// that have similar records but large differences in spread.
	spreadDiffWeight := -int64(utilities.Abs(p1.Spread - p2.Spread))

	// Add one to account for the pairing of p1 and p2 for this round
	repeatsOverMax := utilities.Max(0, members.Repeats[GetRepeatKey(p1.Id, p2.Id)]+1-members.RoundControls.MaxRepeats)
	var repeatWeight int64 = 0
	if members.RoundControls.AllowOverMaxRepeats {
		// Since wins were scaled, repeats have to be scaled up
		// proportionally since we want them to have the same weight.
		repeatWeight = int64(repeatsOverMax) * entity.WinWeightScaling * int64(members.RoundControls.RepeatRelativeWeight)
	} else if repeatsOverMax > 0 {
		repeatWeight = entity.ProhibitiveWeight
	}
	return winDiffWeight + spreadDiffWeight + repeatWeight
}

func weighQuickpair(members *entity.UnpairedPoolMembers, i int, j int) int64 {
	PoolMemberA := members.PoolMembers[i]
	PoolMemberB := members.PoolMembers[j]
	ratingDiff := utilities.Abs(PoolMemberA.Rating - PoolMemberB.Rating)
	missBonus := utilities.Min(missBonus(PoolMemberA), missBonus(PoolMemberB))
	rangeBonus := rangeBonus(PoolMemberA, PoolMemberB)
	return int64(ratingDiff - missBonus - rangeBonus)
}

func missBonus(p *entity.PoolMember) int {
	return utilities.Min(p.Misses*12, 400)
}

func rangeBonus(PoolMemberA *entity.PoolMember, PoolMemberB *entity.PoolMember) int {
	rangeBonus := 0
	if PoolMemberA.RatingRange[0] <= PoolMemberB.Rating &&
		PoolMemberA.RatingRange[1] >= PoolMemberB.Rating &&
		PoolMemberB.RatingRange[0] >= PoolMemberA.Rating &&
		PoolMemberB.RatingRange[1] >= PoolMemberA.Rating {
		rangeBonus = 200
	}
	return rangeBonus
}

func getRoundRobinPairings(players []int, round int) []int {

	/* Round Robin pairing algorithm (from stackoverflow, where else?):

	Players are numbered 1..n. In this example, there are 8 players

	Write all the players in two rows.

	1 2 3 4
	8 7 6 5

	The columns show which players will play in that round (1 vs 8, 2 vs 7, ...).

	Now, keep 1 fixed, but rotate all the other players. In round 2, you get

	1 8 2 3
	7 6 5 4

	and in round 3, you get

	1 7 8 2
	6 5 4 3

	This continues through round n-1, in this case,

	1 3 4 5
	2 8 7 6

	The following algorithm captures the pairings for a certain rotation
	based on the round. The length of players will always be even
	since a bye will be added for any odd length players.

	*/

	rotatedPlayers := players[1:len(players)]

	l := len(rotatedPlayers)
	rotationIndex := l - (round % l)

	rotatedPlayers = append(rotatedPlayers[rotationIndex:l], rotatedPlayers[0:rotationIndex]...)
	rotatedPlayers = append([]int{players[0]}, rotatedPlayers...)

	l = len(rotatedPlayers)
	topHalf := rotatedPlayers[0 : l/2]
	bottomHalf := rotatedPlayers[l/2 : l]
	utilities.Reverse(bottomHalf)

	pairings := []int{}

	// Assign -2 as default so if it doesn't
	// get overwritten we know something is wrong
	for i := 0; i < len(players); i++ {
		pairings = append(pairings, -2)
	}

	for i := 0; i < len(players)/2; i++ {
		pairings[topHalf[i]] = bottomHalf[i]
		pairings[bottomHalf[i]] = topHalf[i]
	}
	return pairings
}
