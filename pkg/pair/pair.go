package pair

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/matching"
	"github.com/woogles-io/liwords/pkg/utilities"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func Pair(members *entity.UnpairedPoolMembers) ([]int, error) {

	pm := members.RoundControls.PairingMethod
	if pm == pb.PairingMethod_MANUAL {
		return nil, errors.New("cannot pair with the given pairing method")
	}
	// This way of dispatching is slightly clunky and will
	// remain until we can think of a better way to do it.
	var pairings []int
	var err error
	if pm == pb.PairingMethod_RANDOM {
		pairings, err = pairRandom(members)
	} else if pm == pb.PairingMethod_ROUND_ROBIN {
		pairings, err = pairRoundRobin(members)
	} else if pm == pb.PairingMethod_KING_OF_THE_HILL || pm == pb.PairingMethod_ELIMINATION {
		pairings, err = pairKingOfTheHill(members)
	} else if pm == pb.PairingMethod_FACTOR {
		pairings, err = pairFactor(members)
	} else if pm == pb.PairingMethod_INITIAL_FONTES {
		pairings, err = pairInitialFontes(members)
	} else if pm == pb.PairingMethod_TEAM_ROUND_ROBIN {
		pairings, err = pairTeamRoundRobin(members)
	} else if pm == pb.PairingMethod_INTERLEAVED_ROUND_ROBIN {
		pairings, err = pairInterleavedRoundRobin(members)
	} else if pm == pb.PairingMethod_AUSTRALIAN_DRAW {
		pairings, err = pairAustralianDraw(members)
	} else if pm == pb.PairingMethod_WIN_GROUPS {
		pairings, err = pairWinGroups(members)
	} else if pm == pb.PairingMethod_SNAKED_ROUND_ROBIN {
		pairings, err = pairSnakedRoundRobin(members)
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
	// Unlike round robin (which derives each round's pairings by rotating a
	// single seed-derived order, so it doesn't need the seed itself to vary)
	// random pairing has no such structure to guarantee variety across
	// rounds - it must fold the round number into the seed itself, or every
	// round pairs identically whenever called with the same base seed (as
	// happens when the same request is reused round after round).
	source := rand.NewPCG(members.Seed, uint64(members.RoundControls.Round))
	rng := rand.New(source)
	rng.Shuffle(len(playerIndexes),
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
	return GetRoundRobinPairings(len(members.PoolMembers), int(members.RoundControls.Round), members.Seed)
}

func pairTeamRoundRobin(members *entity.UnpairedPoolMembers) ([]int, error) {
	return getTeamRoundRobinPairings(len(members.PoolMembers), int(members.RoundControls.Round),
		int(members.RoundControls.GamesPerRound), pb.PairingMethod_TEAM_ROUND_ROBIN, false, members.Seed)
}

func pairInterleavedRoundRobin(members *entity.UnpairedPoolMembers) ([]int, error) {
	// This is a special case of the team round robin function. Perhaps the function
	// should be renamed slightly.
	return getTeamRoundRobinPairings(len(members.PoolMembers), int(members.RoundControls.Round),
		1, pb.PairingMethod_INTERLEAVED_ROUND_ROBIN, members.RoundControls.PlayWithinTeam, members.Seed)

}

func pairSnakedRoundRobin(members *entity.UnpairedPoolMembers) ([]int, error) {
	return getTeamRoundRobinPairings(len(members.PoolMembers), int(members.RoundControls.Round),
		int(members.RoundControls.GamesPerRound), pb.PairingMethod_SNAKED_ROUND_ROBIN,
		members.RoundControls.PlayWithinTeam, members.Seed)
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
			" have %d instead (precondition violation)", members.RoundControls.InitialFontes)
	}

	numberOfPlayers := len(members.PoolMembers)
	numberOfNtiles := int(members.RoundControls.InitialFontes) + 1
	round := int(members.RoundControls.Round)
	// This function was created to make testing easier
	return getInitialFontesPairings(numberOfPlayers, numberOfNtiles, round, members.Seed)
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
		newPairing := -1
		if lowerPairings[i] != -1 {
			newPairing = lowerPairings[i] + numberOfUpperPlayers
		}
		upperPairings = append(upperPairings, newPairing)
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
		log.Debug().Msgf("matching incomplete: %v, %v", pairings, edges)
		return nil, errors.New("pairings and members are not the same length")
	}

	if weight >= entity.ProhibitiveWeight {
		return nil, errors.New("prohibitive weight reached, pairings are not possible with these settings")
	}

	if members.RoundControls.PairingMethod == pb.PairingMethod_QUICKPAIR {
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
	if pm == pb.PairingMethod_SWISS || pm == pb.PairingMethod_FACTOR {
		weight = weighSwiss(members, i, j)
	} else if pm == pb.PairingMethod_QUICKPAIR {
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

	// Add marginal penalties to make the weighing function nonlinear
	// This will discourage larger maximum win differences over a round
	unscaledWinDiffWeight += (unscaledWinDiffWeight / (entity.DifferencePenaltyMargin * 2)) * (entity.DifferencePenalty * 2)

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

func getInitialFontesPairings(numberOfPlayers int, numberOfNtiles int, round int, seed uint64) ([]int, error) {
	log.Info().Int("players", numberOfPlayers).Int("ntiles", numberOfNtiles).Int("round", round).Msg("initial-fontes-params")
	// If there are more Ntiles than players, InitialFontes is not valid
	// Return all byes
	// In practice this should never be true, since it is usually undesirable
	// behavior. This extra check is here to avoid a panic.
	if numberOfPlayers < numberOfNtiles {
		pairings := make([]int, numberOfPlayers)
		for i := 0; i < numberOfPlayers; i++ {
			pairings[i] = -1
		}
		return pairings, nil
	}

	addBye := numberOfPlayers%2 == 1

	if addBye {
		numberOfPlayers++
	}

	sizeOfNtiles := numberOfPlayers / numberOfNtiles

	if sizeOfNtiles < 2 {
		return nil, fmt.Errorf("number of initial fontes rounds (%d) should be less than half the number of players (%d)", numberOfNtiles-1, numberOfPlayers)
	}

	numberOfRemainingPlayers := numberOfPlayers - (sizeOfNtiles * numberOfNtiles)

	pairings := make([]int, numberOfPlayers)
	for i := range pairings {
		pairings[i] = -2
	}

	groupings := [][]int{}
	for i := 0; i < sizeOfNtiles; i++ {
		groupings = append(groupings, []int{})
	}

	if numberOfRemainingPlayers == 0 {
		// Evenly divides into sizeOfNtiles standard-size groups; distribute
		// players across them round robin.
		currentGroup := 0
		for i := 0; i < numberOfPlayers; i++ {
			groupings[currentGroup] = append(groupings[currentGroup], i)
			currentGroup = (currentGroup + 1) % sizeOfNtiles
		}
	} else {
		// There's a remainder, so one group (the last) will be oversized by
		// numberOfRemainingPlayers to absorb it, e.g. quartiles (numberOfNtiles=4)
		// with a remainder of 2 produce a single oversized sextile (size 6).
		//
		// Rather than draw that oversized group's extra players from just the
		// tail of the standings, spread them across the whole field: split the
		// ranked player list into oversizedGroupSize evenly-sized mini-groups
		// (any leftover from an uneven split is distributed across the first
		// few mini-groups), and take the single worst-ranked player from each
		// mini-group into the oversized group. The rest of each mini-group is
		// distributed round robin across the other sizeOfNtiles-1 standard-size
		// groups. E.g. 18 players, numberOfNtiles=4: 6 mini-groups of 3 ranks
		// each (1-3, 4-6, ..., 16-18); the oversized group ends up with ranks
		// 3, 6, 9, 12, 15, 18 - the bottom player of each mini-group.
		oversizedGroupIdx := sizeOfNtiles - 1
		oversizedGroupSize := numberOfNtiles + numberOfRemainingPlayers
		numberOfMiniGroups := oversizedGroupSize
		baseMiniGroupSize := numberOfPlayers / numberOfMiniGroups
		extraMiniGroups := numberOfPlayers % numberOfMiniGroups

		currentGroup := 0
		playerIdx := 0
		for m := 0; m < numberOfMiniGroups; m++ {
			miniGroupSize := baseMiniGroupSize
			if m < extraMiniGroups {
				miniGroupSize++
			}
			for k := 0; k < miniGroupSize; k++ {
				if k == miniGroupSize-1 {
					groupings[oversizedGroupIdx] = append(groupings[oversizedGroupIdx], playerIdx)
				} else {
					groupings[currentGroup] = append(groupings[currentGroup], playerIdx)
					currentGroup = (currentGroup + 1) % oversizedGroupIdx
				}
				playerIdx++
			}
		}
	}

	log.Info().Str("groupings", fmt.Sprint(groupings)).Msg("initial-fontes-groupings")

	for i := 0; i < len(groupings); i++ {
		groupSize := len(groupings[i])
		if groupSize%2 == 1 {
			return nil, fmt.Errorf("initial fontes pairing failure for %d players, "+
				"have odd group size of %d", numberOfPlayers, groupSize)
		}

		var groupPairings []int
		var err error
		if groupSize == numberOfNtiles {
			// Standard-size N-tile group: pair with a normal round robin, as before.
			groupPairings, err = GetRoundRobinPairings(groupSize, round, seed)
		} else {
			// Oversized remainder group (the leftover players folded in above via
			// remainderOffset/remainderSpacing). Rather than a full round robin,
			// use a fixed "shirts vs skins" cross pattern: split the group into
			// two teams by local rank parity (odds vs evens) and rotate one team
			// against the other each round. This only avoids repeats across all
			// numberOfNtiles-1 fontes rounds if the group is big enough
			// (half its size >= rounds needed); otherwise fall back to round
			// robin, which is always safe.
			k := groupSize / 2
			roundsNeeded := numberOfNtiles - 1
			if k >= roundsNeeded {
				groupPairings = getShirtsVsSkinsPairings(groupSize, round)
			} else {
				groupPairings, err = GetRoundRobinPairings(groupSize, round, seed)
			}
		}
		if err != nil {
			return nil, err
		}
		log.Info().Str("groupPairings", fmt.Sprint(groupPairings)).Msg("initial-fontes-group-pairings")
		for j := 0; j < groupSize; j++ {
			pairings[groupings[i][j]] = groupings[i][groupPairings[j]]
		}
	}

	// Convert the "bye player" to the bye representation
	// which is an opponent index value of -1

	log.Info().Str("pairings", fmt.Sprint(pairings)).Msg("initial-fontes-pairings")

	if addBye {
		pairings[pairings[numberOfPlayers-1]] = -1
		pairings = pairings[:numberOfPlayers-1]
	}

	log.Info().Str("pairings", fmt.Sprint(pairings)).Msg("initial-fontes-pairings-bye-correction")

	for i := 0; i < len(pairings); i++ {
		if pairings[i] < -1 {
			return nil, fmt.Errorf("initial fontes pairing failure for %d players", numberOfPlayers)
		}
	}
	return pairings, nil
}

// getShirtsVsSkinsPairings pairs an oversized Initial Fontes remainder group
// using a fixed cross pattern instead of a round robin. Local player indices
// are assumed to already be in rank order (0 = best local seed). Team A is
// the even local indices (local ranks 1, 3, 5, ...) and Team B is the odd
// local indices (local ranks 2, 4, 6, ...). Each round, Team A is paired
// against Team B rotated by `round` positions, e.g. for a group of 6
// (local ranks 1-6):
//
//	Round 0: 1v2  3v4  5v6
//	Round 1: 1v4  3v6  5v2
//	Round 2: 1v6  3v2  5v4
//
// This only avoids repeating a pairing across all of the fontes rounds if
// groupSize/2 >= the number of rounds needed; the caller is responsible for
// checking that before calling this function.
func getShirtsVsSkinsPairings(groupSize int, round int) []int {
	k := groupSize / 2
	pairings := make([]int, groupSize)
	for x := 0; x < k; x++ {
		a := 2 * x
		b := 2*((x+round)%k) + 1
		pairings[a] = b
		pairings[b] = a
	}
	return pairings
}

// GetRoundRobinPairings generates round-robin pairings for a given number of players and round
func GetRoundRobinPairings(numberOfPlayers int, round int, seed uint64) ([]int, error) {

	/* Round Robin pairing algorithm:

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

	players := roundRobinSeats(numberOfPlayers, seed)
	bye := numberOfPlayers%2 == 1

	rotatedPlayers := players[1:]

	l := len(rotatedPlayers)

	rotationIndex := round % l

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

// roundRobinSeats shuffles the players, and the bye (numberOfPlayers) if
// there are an odd number of them, into GetRoundRobinPairings' seats: the
// first seat is fixed and the rest rotate past it.
func roundRobinSeats(numberOfPlayers int, seed uint64) []int {
	players := []int{}
	for i := 0; i < numberOfPlayers; i++ {
		players = append(players, i)
	}

	// If there are an odd number of players add a bye
	if numberOfPlayers%2 == 1 {
		players = append(players, numberOfPlayers)
	}

	source := rand.NewPCG(seed, 0)
	rng := rand.New(source)
	rng.Shuffle(len(players),
		func(i, j int) {
			players[i], players[j] = players[j], players[i]
		})
	return players
}

// RoundRobinGoesFirst says whether player goes first in round of the round
// robin GetRoundRobinPairings pairs with the same seed.
//
// In each round the fixed seat plays the rotating seat at the center of the
// circle, and the seats k ahead of and behind the center play each other.
// The player ahead goes first when k is odd, so as the circle turns each
// player on it alternates, apart from around their game against the fixed
// seat; that game is free to go either way, and is used as follows.
//
// With an even number of players, the fixed seat alternates, starting
// second, except that its last two rounds are swapped. That puts every
// player's one double (which every round robin must give all but two
// players) away from the ends of the cycle.
//
// With an odd number of players and the bye on the circle, each player
// sitting out skips a first or a second, and the game against the fixed
// seat makes it up, so that everyone has as many firsts as seconds.
//
// The pairings repeat in the same order every cycle, so every other cycle is
// flipped: each pair splits their games.
func RoundRobinGoesFirst(numberOfPlayers, round, player int, seed uint64) (bool, error) {
	if player < 0 || player >= numberOfPlayers {
		return false, fmt.Errorf("player %d out of range for %d players", player, numberOfPlayers)
	}
	players := roundRobinSeats(numberOfPlayers, seed)
	circle := players[1:]
	l := len(circle) // always odd
	rotation := round % l
	center := (rotation + l - 1) % l
	flip := (round / l) % 2
	var fixedFirst bool
	if byeSeat := slices.Index(circle, numberOfPlayers); byeSeat >= 0 {
		// The player at seat x sits out the round the bye is (x-b)/2 seats
		// behind them, where b is the bye's seat, and so misses a first if
		// that's odd. The center player goes first against the fixed seat
		// exactly then.
		missed := (center - byeSeat + l) % l * ((l + 1) / 2) % l
		fixedFirst = missed%2 == 0
	} else {
		fixedFirst = rotation%2 == 1
		if rotation >= l-2 {
			fixedFirst = !fixedFirst
		}
	}
	fixedFirst = fixedFirst != (flip == 1)
	if player == players[0] {
		return fixedFirst, nil
	}
	d := (slices.Index(circle, player) - center + l) % l
	if d == 0 {
		return !fixedFirst, nil
	}
	return (d+flip)%2 == 1, nil
}

func getTeamRoundRobinPairings(numberOfPlayers, round, gamesPerMatchup int, method pb.PairingMethod, playWithinTeam bool,
	seed uint64) ([]int, error) {
	pairings, _, err := getTeamRoundRobinSchedule(numberOfPlayers, round, gamesPerMatchup, method, playWithinTeam, seed)
	return pairings, err
}

// TeamRoundRobinGamesPerMatchup is how many rounds in a row the team-style
// round robin of rc pairs the same two players.
func TeamRoundRobinGamesPerMatchup(rc *pb.RoundControl) int {
	if rc.PairingMethod == pb.PairingMethod_TEAM_ROUND_ROBIN {
		return int(rc.GamesPerRound)
	}
	return 1
}

// TeamRoundRobinGoesFirst says whether player goes first in round of the
// team-style round robin (team, interleaved or snaked) that rc describes.
// The schedule decides firsts along with opponents, so that over the round
// robin every player's firsts and seconds stay even; see
// getTeamRoundRobinSchedule.
func TeamRoundRobinGoesFirst(rc *pb.RoundControl, numberOfPlayers, round, player int, seed uint64) (bool, error) {
	_, firsts, err := getTeamRoundRobinSchedule(numberOfPlayers, round, TeamRoundRobinGamesPerMatchup(rc),
		rc.PairingMethod, rc.PlayWithinTeam, seed)
	if err != nil {
		return false, err
	}
	if player < 0 || player >= len(firsts) {
		return false, fmt.Errorf("player %d out of range for %d players", player, len(firsts))
	}
	return firsts[player], nil
}

// getTeams splits players 0 through numberOfPlayers-1, who are in seed
// order, into the two teams of a team-style round robin.
func getTeams(numberOfPlayers int, method pb.PairingMethod) ([]int, []int, error) {
	teamA, teamB := []int{}, []int{}
	switch method {
	case pb.PairingMethod_TEAM_ROUND_ROBIN:
		// Top half against bottom half.
		if numberOfPlayers%2 == 1 {
			return nil, nil, errors.New("cannot have an odd number of players without interleaving team round robin pairings")
		}
		for i := 0; i < numberOfPlayers; i++ {
			if i < numberOfPlayers/2 {
				teamA = append(teamA, i)
			} else {
				teamB = append(teamB, i)
			}
		}
	case pb.PairingMethod_INTERLEAVED_ROUND_ROBIN:
		// Shirts and skins: seeds 1, 3, 5, ... against seeds 2, 4, 6, ...
		for i := 0; i < numberOfPlayers; i++ {
			if i%2 == 0 {
				teamA = append(teamA, i)
			} else {
				teamB = append(teamB, i)
			}
		}
	case pb.PairingMethod_SNAKED_ROUND_ROBIN:
		// A snake draft: seeds 1, 4, 5, 8, 9, ... against seeds 2, 3, 6, 7, ...
		for i := 0; i < numberOfPlayers; i++ {
			if i%4 == 0 || i%4 == 3 {
				teamA = append(teamA, i)
			} else {
				teamB = append(teamB, i)
			}
		}
	default:
		return nil, nil, fmt.Errorf("pairing method %s is not a team round robin pairing method", method.String())
	}
	return teamA, teamB, nil
}

// getTeamRoundRobinSchedule pairs round of a team-style round robin, and says
// who goes first in each game. pairings[i] is player i's opponent, or -1 for
// a bye; firsts[i] says whether player i goes first.
//
// The players are split into two teams (see getTeams). Across teams, everyone
// on one team plays everyone on the other, gamesPerMatchup rounds in a row.
// Within teams (playWithinTeam), each team plays a round robin of its own.
//
// Firsts are decided by where each game sits in the schedule, not by player
// numbers, so they come out even however the teams were drawn:
//   - Across teams, every game is between the two teams, so one whole team
//     goes first each round, alternating, and so does everyone. A team with an
//     odd number of seats facing the bye is the exception; see below.
//   - Within a team, the round robin is the circle method: one player stays
//     fixed and the rest rotate around a circle. The two players k places
//     either side of the circle's center play, and the one ahead goes first
//     when k is odd, so everyone alternates all the way round the circle. The
//     fixed player plays the center, alternating each round; when a team has
//     an odd number of players the fixed seat is a bye instead.
//   - Repeated round robins swap who goes first, so that two players who
//     meet twice go first once each, and every two cycles everyone has gone
//     first exactly as often as second.
//   - Repeated games between the same two players (gamesPerMatchup) swap who
//     goes first each time.
func getTeamRoundRobinSchedule(numberOfPlayers, round, gamesPerMatchup int, method pb.PairingMethod,
	playWithinTeam bool, seed uint64) ([]int, []bool, error) {

	if method == pb.PairingMethod_TEAM_ROUND_ROBIN && playWithinTeam {
		return nil, nil, errors.New("cannot play within team in team round robin pairings")
	}
	if gamesPerMatchup < 1 {
		return nil, nil, fmt.Errorf("games per matchup must be at least 1, not %d", gamesPerMatchup)
	}
	teamA, teamB, err := getTeams(numberOfPlayers, method)
	if err != nil {
		return nil, nil, err
	}

	pairings := make([]int, numberOfPlayers)
	firsts := make([]bool, numberOfPlayers)
	for i := range pairings {
		// Overwritten for everyone below; left over, it is an error.
		pairings[i] = -2
	}
	matchupRound := round / gamesPerMatchup
	// Which of the repeated games between the same two players this is.
	game := round % gamesPerMatchup
	rng := rand.New(rand.NewPCG(seed, 0))
	bye := numberOfPlayers

	// play records a game, or a bye if either player is the bye.
	play := func(first, second int) {
		if first == bye {
			pairings[second] = -1
		} else if second == bye {
			pairings[first] = -1
		} else {
			pairings[first] = second
			pairings[second] = first
			firsts[first] = true
		}
	}

	if !playWithinTeam {
		// An odd number of players means one team is a player short; the bye
		// fills that team's last seat.
		if numberOfPlayers%2 == 1 {
			if len(teamA) < len(teamB) {
				teamA = append(teamA, bye)
			} else {
				teamB = append(teamB, bye)
			}
		}
		m := len(teamA)
		shuffledB := slices.Clone(teamB)
		rng.Shuffle(m, func(i, j int) {
			shuffledB[i], shuffledB[j] = shuffledB[j], shuffledB[i]
		})
		byeSeat := slices.Index(shuffledB, bye)
		// The seed picks which team starts, so that the top seed doesn't
		// always go first in the first round.
		bStarts := rng.IntN(2) == 1
		cycle := matchupRound / m
		rotation := matchupRound % m
		if method == pb.PairingMethod_SNAKED_ROUND_ROBIN && cycle%2 == 1 {
			// Every other cycle runs backwards, so that each player's second
			// cycle is their first read backwards: two players who meet in
			// both go first once each, and nobody gets three firsts or
			// seconds in a row across the turn. The team and interleaved
			// round robins' pairings predate this and are kept as they were.
			rotation = m - 1 - rotation
		}
		for i := 0; i < m; i++ {
			s := (rotation + m - 1 - i) % m
			var aFirst bool
			if byeSeat >= 0 && m%2 == 1 {
				// Team B has the bye seat and an odd number of seats, so team
				// A's players each play one round fewer than there are, and
				// alternating by round would leave a player who sits out a
				// round they would have gone second in two firsts up.
				// Instead, team A's i-th player goes first against team B's
				// s-th when i+s is even, counting seats from the bye's: each
				// player's opponent moves one seat along each round, so they
				// still alternate, and team A comes out even every cycle.
				// Team B's players are a first or second off, so every other
				// cycle is flipped.
				aFirst = (i+(s-byeSeat+m)%m+game)%2 == 0
				if cycle%2 == 1 {
					aFirst = !aFirst
				}
			} else {
				// Otherwise one whole team goes first each round, team A in
				// even rounds and team B in odd ones, and everyone alternates.
				aFirst = round%2 == 0
				if method != pb.PairingMethod_SNAKED_ROUND_ROBIN && (m*gamesPerMatchup)%2 == 0 && cycle%2 == 1 {
					// A pair meets again an even number of rounds later, on
					// the same parity, so every other cycle is flipped. (The
					// snaked round robin's backwards cycles land on the
					// opposite parity already.)
					aFirst = !aFirst
				}
			}
			if aFirst != bStarts {
				play(teamA[i], shuffledB[s])
			} else {
				play(shuffledB[s], teamA[i])
			}
		}
	} else {
		// Each team's bye receiver, if it has a bye seat this round.
		byeReceivers := []int{}
		for _, team := range [][]int{teamA, teamB} {
			if len(team) == 0 {
				continue
			}
			seats := slices.Clone(team)
			rng.Shuffle(len(seats), func(i, j int) {
				seats[i], seats[j] = seats[j], seats[i]
			})
			fixed := bye
			circle := seats
			if len(seats)%2 == 0 {
				fixed = seats[0]
				circle = seats[1:]
			}
			n := len(circle) // always odd
			cycle := matchupRound / n
			center := matchupRound % n
			if cycle%2 == 1 {
				// Every other cycle runs backwards with firsts and seconds
				// swapped: two players who meet in both go first once each,
				// and each player's games read the same either way across
				// the turn, so nobody gets three firsts or seconds in a row.
				center = n - 1 - center
			}
			flip := (cycle + game) % 2
			if (center+flip)%2 == 0 {
				play(fixed, circle[center])
			} else {
				play(circle[center], fixed)
			}
			if fixed == bye {
				byeReceivers = append(byeReceivers, circle[center])
			}
			for k := 1; k <= n/2; k++ {
				ahead := circle[(center+k)%n]
				behind := circle[(center-k+n)%n]
				if (k+flip)%2 == 1 {
					play(ahead, behind)
				} else {
					play(behind, ahead)
				}
			}
		}
		// When both teams have an odd number of players, both have a bye every
		// round; rather than sit out, the two bye receivers play each other.
		if len(byeReceivers) == 2 {
			if (matchupRound/len(teamA)+game)%2 == 0 {
				play(byeReceivers[0], byeReceivers[1])
			} else {
				play(byeReceivers[1], byeReceivers[0])
			}
		}
	}

	for i := range pairings {
		if pairings[i] < -1 {
			return nil, nil, fmt.Errorf("team round robin pairing failure for %d players", numberOfPlayers)
		}
	}
	log.Debug().Interface("pairings", pairings).Int("numPlayers", numberOfPlayers).Int("round", round).
		Int("gamesPerMatchup", gamesPerMatchup).Bool("playWithinTeam", playWithinTeam).Msg("final team round robin pairings")
	return pairings, firsts, nil
}

func IsStandingsIndependent(pm pb.PairingMethod) bool {
	return pm == pb.PairingMethod_ROUND_ROBIN ||
		pm == pb.PairingMethod_TEAM_ROUND_ROBIN ||
		pm == pb.PairingMethod_INTERLEAVED_ROUND_ROBIN ||
		pm == pb.PairingMethod_SNAKED_ROUND_ROBIN ||
		pm == pb.PairingMethod_RANDOM ||
		pm == pb.PairingMethod_INITIAL_FONTES ||
		pm == pb.PairingMethod_MANUAL
}
