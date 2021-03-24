package pair

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/matching"
	"github.com/domino14/liwords/pkg/utilities"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

func Pair(members *entity.UnpairedPoolMembers) ([]int, error) {

	pm := members.RoundControls.PairingMethod
	if pm == realtime.PairingMethod_MANUAL {
		return nil, errors.New("cannot pair with the given pairing method")
	}
	// This way of dispatching is slightly clunky and will
	// remain until we can think of a better way to do it.
	var pairings []int
	var err error
	if pm == realtime.PairingMethod_RANDOM {
		pairings, err = pairRandom(members)
	} else if pm == realtime.PairingMethod_ROUND_ROBIN {
		pairings, err = pairRoundRobin(members)
	} else if pm == realtime.PairingMethod_KING_OF_THE_HILL || pm == realtime.PairingMethod_ELIMINATION {
		pairings, err = pairKingOfTheHill(members)
	} else if pm == realtime.PairingMethod_FACTOR {
		pairings, err = pairFactor(members)
	} else if pm == realtime.PairingMethod_INITIAL_FONTES {
		pairings, err = pairInitialFontes(members)
	} else if pm == realtime.PairingMethod_TEAM_ROUND_ROBIN {
		pairings, err = pairTeamRoundRobin(members)
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
	return getRoundRobinPairings(len(members.PoolMembers), int(members.RoundControls.Round))
}

func pairTeamRoundRobin(members *entity.UnpairedPoolMembers) ([]int, error) {
	return getTeamRoundRobinPairings(len(members.PoolMembers), int(members.RoundControls.Round), int(members.RoundControls.GamesPerRound))
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

	// Remaining players are paired using the swiss-like min weight matching

	factorMembers, swissMembers, err := splitMembers(members, int(members.RoundControls.Factor)*2)
	if err != nil {
		return nil, err
	}

	factorPairings := []int{}
	swissPairings := []int{}

	if len(factorMembers.PoolMembers) > 0 {
		factorPairings, err = getFactorPairings(len(factorMembers.PoolMembers), int(members.RoundControls.Factor))
		if err != nil {
			return nil, err
		}
	}

	if len(swissMembers.PoolMembers) > 0 {
		swissPairings, err = minWeightMatching(swissMembers)
		if err != nil {
			return nil, err
		}
	}

	return combinePairings(factorPairings, swissPairings), nil
}

func pairInitialFontes(members *entity.UnpairedPoolMembers) ([]int, error) {
	if members.RoundControls.InitialFontes%2 == 0 {
		return nil, fmt.Errorf("number of rounds paired with Initial Fontes must be odd,"+
			" have %d instead (preconditiion violation)", members.RoundControls.InitialFontes)
	}

	numberOfPlayers := len(members.PoolMembers)
	numberOfNtiles := int(members.RoundControls.InitialFontes) + 1
	round := int(members.RoundControls.Round)
	// This function was created to make testing easier
	return getInitialFontesPairings(numberOfPlayers, numberOfNtiles, round)
}

func splitMembers(members *entity.UnpairedPoolMembers, i int) (*entity.UnpairedPoolMembers, *entity.UnpairedPoolMembers, error) {
	// Let i be the index of the first player in the standings
	// to be in the second group of pool members
	if i <= 0 {
		emptyMembers := &entity.UnpairedPoolMembers{PoolMembers: []*entity.PoolMember{},
			RoundControls: members.RoundControls,
			Repeats:       members.Repeats}
		return emptyMembers, members, nil
	}
	if i >= len(members.PoolMembers) {
		emptyMembers := &entity.UnpairedPoolMembers{PoolMembers: []*entity.PoolMember{},
			RoundControls: members.RoundControls,
			Repeats:       members.Repeats}
		return members, emptyMembers, nil
	}

	splitMembers := []*entity.PoolMember{}

	for j := i; j < len(members.PoolMembers); j++ {
		splitMembers = append(splitMembers, members.PoolMembers[j])
	}

	upm1 := &entity.UnpairedPoolMembers{PoolMembers: members.PoolMembers[:i],
		RoundControls: members.RoundControls,
		Repeats:       members.Repeats}

	upm2 := &entity.UnpairedPoolMembers{PoolMembers: splitMembers,
		RoundControls: members.RoundControls,
		Repeats:       members.Repeats}

	return upm1, upm2, nil
}

func combinePairings(upperPairings []int, lowerPairings []int) []int {
	numberOfUpperPlayers := len(upperPairings)
	for i := 0; i < len(lowerPairings); i++ {
		upperPairings = append(upperPairings, lowerPairings[i]+numberOfUpperPlayers)
	}
	return upperPairings
}

func minWeightMatching(members *entity.UnpairedPoolMembers) ([]int, error) {
	numberOfMembers := len(members.PoolMembers)
	edges := []*matching.Edge{}

	if int(members.RoundControls.RepeatRelativeWeight) > entity.MaxRelativeWeight {
		members.RoundControls.RepeatRelativeWeight = int32(entity.MaxRelativeWeight)
	}

	if int(members.RoundControls.WinDifferenceRelativeWeight) > entity.MaxRelativeWeight {
		members.RoundControls.WinDifferenceRelativeWeight = int32(entity.MaxRelativeWeight)
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

	if members.RoundControls.PairingMethod == realtime.PairingMethod_QUICKPAIR {
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
	if pm == realtime.PairingMethod_SWISS || pm == realtime.PairingMethod_FACTOR {
		weight = weighSwiss(members, i, j)
	} else if pm == realtime.PairingMethod_QUICKPAIR {
		weight = weighQuickpair(members, i, j)
	} else {
		return 0, errors.New("pairing method is either unimplemented or is not a reduction to minimum weight matching")
	}
	return weight, nil
}

func weighSwiss(members *entity.UnpairedPoolMembers, i int, j int) int64 {
	p1 := members.PoolMembers[i]
	p2 := members.PoolMembers[j]

	// Egregious, temporary hack
	var removedWeight int64 = 0
	if p1.Removed || p2.Removed {
		removedWeight = entity.WinWeightScaling * 100
	}

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
	repeatsOverMax := utilities.Max(0, members.Repeats[GetRepeatKey(p1.Id, p2.Id)]+1-int(members.RoundControls.MaxRepeats))
	var repeatWeight int64 = 0
	if members.RoundControls.AllowOverMaxRepeats {
		// Since wins were scaled, repeats have to be scaled up
		// proportionally since we want them to have the same weight.
		repeatWeight = int64(repeatsOverMax) * entity.WinWeightScaling * int64(members.RoundControls.RepeatRelativeWeight)
	} else if repeatsOverMax > 0 {
		repeatWeight = entity.ProhibitiveWeight
	}
	return winDiffWeight + spreadDiffWeight + repeatWeight + removedWeight
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

func getFactorPairings(numberOfPlayers int, factor int) ([]int, error) {
	factorPairings := []int{}
	for i := 0; i < numberOfPlayers; i++ {
		factorPairings = append(factorPairings, -1)
	}

	for i := 0; i < factor; i += 1 {
		factor := i + factor
		if factor >= numberOfPlayers {
			return nil, fmt.Errorf("cannot pair with factor %d on %d players", factor, numberOfPlayers)
		}
		factorPairings[i] = factor
		factorPairings[factor] = i
	}
	return factorPairings, nil
}

func getInitialFontesPairings(numberOfPlayers int, numberOfNtiles int, round int) ([]int, error) {
	addBye := numberOfPlayers%2 == 1

	if addBye {
		numberOfPlayers++
	}

	sizeOfNtiles := numberOfPlayers / numberOfNtiles
	numberOfRemainingPlayers := numberOfPlayers - (sizeOfNtiles * numberOfNtiles)
	remainderOffset := 0
	remainderSpacing := 0

	if numberOfRemainingPlayers != 0 {
		remainderOffset = numberOfPlayers / (numberOfRemainingPlayers + 1)
		remainderSpacing = numberOfPlayers / numberOfRemainingPlayers
	}

	pairings := []int{}
	groupings := [][]int{}

	for i := 0; i < sizeOfNtiles; i++ {
		groupings = append(groupings, []int{})
	}

	currentGroup := 0
	for i := 0; i < numberOfPlayers; i++ {
		if numberOfRemainingPlayers != 0 &&
			i >= remainderOffset &&
			(i-remainderOffset)%remainderSpacing == 0 {
			groupings[sizeOfNtiles-1] = append(groupings[sizeOfNtiles-1], i)
		} else {
			groupings[currentGroup] = append(groupings[currentGroup], i)
			currentGroup = (currentGroup + 1) % sizeOfNtiles
		}
		pairings = append(pairings, -2)
	}

	for i := 0; i < len(groupings); i++ {
		groupSize := len(groupings[i])
		if groupSize%2 == 1 {
			return nil, fmt.Errorf("initial fontes pairing failure for %d players, "+
				"have odd group size of %d", numberOfPlayers, groupSize)
		}
		groupPairings, err := getRoundRobinPairings(groupSize, round)
		if err != nil {
			return nil, err
		}
		for j := 0; j < groupSize; j++ {
			pairings[groupings[i][j]] = groupings[i][groupPairings[j]]
		}
	}

	// Convert the "bye player" to the bye representation
	// which is an opponent index value of -1

	if addBye {
		pairings[pairings[numberOfPlayers-1]] = -1
		pairings = pairings[:numberOfPlayers-1]
	}

	for i := 0; i < len(pairings); i++ {
		if pairings[i] < -1 {

			return nil, fmt.Errorf("initial fontes pairing failure for %d players", numberOfPlayers)
		}
	}
	return pairings, nil
}

func getRoundRobinPairings(numberOfPlayers int, round int) ([]int, error) {

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

	players := []int{}
	for i := 0; i < numberOfPlayers; i++ {
		players = append(players, i)
	}

	bye := numberOfPlayers%2 == 1
	// If there are an odd number of players add a bye
	if bye {
		players = append(players, numberOfPlayers)
	}

	rotatedPlayers := players[1:len(players)]

	l := len(rotatedPlayers)

	rotationIndex := l - getRoundRobinRotation(len(players), round)

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

	for i := 0; i < len(pairings); i++ {
		if pairings[i] < -1 {
			return nil, fmt.Errorf("round robin pairing failure for %d players", l)
		}
	}
	return pairings, nil
}

func getTeamRoundRobinRotation(numberOfPlayers int, round int, gamesPerMatchup int) int {
	return ((round / gamesPerMatchup) * 2) % (numberOfPlayers - 1)
}

func getRoundRobinRotation(numberOfPlayers int, round int) int {
	// This has been made a separate function
	// for ease of testing and future improvements

	// We rotate by different value instead of just the round here
	// so that incomplete round robins will have slightly more
	// equitable pairings. If we rotated by round only, the
	// rotationIndex would only change by 1 each round, which means
	// player 1 in the round robin would play significantly weaker
	// players than the rest of the field. This solution is
	// relatively weak as there are probably stronger algorithmic
	// methods that could be used to address the problem. However,
	// this way is by far the most simple.

	// This solution of the form round * x requires that x and (numberOfPlayers - 1)
	// be coprime, or, in other words, that:
	//
	// GCD(x, numberOfPlayers - 1) = 1
	//
	// If this does not hold, the round robin will not be complete.
	//
	// In this case, it is guaranteed to hold since
	//
	// GCD(x, numberOfPlayers - 1) = GCD(2n - 3, 2n - 1) = 1
	//
	// We can use 2n for the above equation since the number of players is
	// always even
	return (round * (numberOfPlayers - 3)) % (numberOfPlayers - 1)
}

func getTeamRoundRobinPairings(numberOfPlayers, round, gamesPerMatchup int) ([]int, error) {
	// A team round robin contains two teams: A and B
	// Everyone in A plays everyone in B gamesPerMatchup times (in a row for speed).
	if numberOfPlayers%2 == 1 {
		return nil, errors.New("cannot have an odd number of players with team round robin pairings")
	}

	players := []int{}
	for i := 0; i < numberOfPlayers; i++ {
		players = append(players, i)
	}

	rotatedPlayers := players[:]

	l := len(rotatedPlayers)
	rotationIndex := l - getTeamRoundRobinRotation(len(players), round, gamesPerMatchup)
	rotatedPlayers = append(rotatedPlayers[rotationIndex:l], rotatedPlayers[0:rotationIndex]...)

	topHalf := rotatedPlayers[0 : l/2]
	bottomHalf := rotatedPlayers[l/2 : l]
	utilities.Reverse(bottomHalf)

	pairings := []int{}

	// Assign -2 as default so if it doesn't
	// get overwritten we know something is wrong
	for i := 0; i < len(players); i++ {
		pairings = append(pairings, -2)
	}

	// log.Debug().Interface("pairings", pairings).Interface("rotatedPlayers", rotatedPlayers).
	// 	Interface("players", players).Int("numPlayers", numberOfPlayers).Int("round", round).
	// 	Int("gamesPerMatchup", gamesPerMatchup).Int("rotationIndex", rotationIndex).
	// 	Interface("topHalf", topHalf).Interface("bottomHalf", bottomHalf).Msg("debug-pairings")

	for i := 0; i < len(players)/2; i++ {
		pairings[topHalf[i]] = bottomHalf[i]
		pairings[bottomHalf[i]] = topHalf[i]
	}

	for i := 0; i < len(pairings); i++ {
		if pairings[i] < -1 {
			return nil, fmt.Errorf("team round robin pairing failure for %d players", l)
		}
	}
	log.Debug().Interface("pairings", pairings).Int("numPlayers", numberOfPlayers).Int("round", round).
		Int("gamesPerMatchup", gamesPerMatchup).Msg("final pairings")
	return pairings, nil
}

func IsStandingsIndependent(pm realtime.PairingMethod) bool {
	return pm == realtime.PairingMethod_ROUND_ROBIN ||
		pm == realtime.PairingMethod_TEAM_ROUND_ROBIN ||
		pm == realtime.PairingMethod_RANDOM ||
		pm == realtime.PairingMethod_INITIAL_FONTES ||
		pm == realtime.PairingMethod_MANUAL
}
