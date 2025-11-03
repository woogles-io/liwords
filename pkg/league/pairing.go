package league

import (
	"fmt"

	"github.com/woogles-io/liwords/pkg/pair"
)

// GamePairing represents a single game in the league
type GamePairing struct {
	Player1Index   int
	Player2Index   int
	IsPlayer1First bool
	Round          int // Which round (0-12 for 14 players)
}

// GenerateAllLeaguePairings generates all pairings for a full round-robin league season
// For N players, this generates N-1 rounds with N/2 games per round (or (N-1)/2 if odd)
// If maxRounds > 0, limits the number of rounds to min(calculated, maxRounds)
// Returns a slice of all game pairings that should be created when the season starts
func GenerateAllLeaguePairings(numPlayers int, seed uint64, maxRounds int) ([]*GamePairing, error) {
	if numPlayers < 2 {
		return nil, fmt.Errorf("need at least 2 players for league pairings")
	}

	// Calculate number of rounds needed for full round-robin
	numRounds := numPlayers
	if numPlayers%2 == 0 {
		numRounds = numPlayers - 1
	}

	// Apply max rounds cap if specified
	if maxRounds > 0 && numRounds > maxRounds {
		numRounds = maxRounds
	}

	allPairings := []*GamePairing{}

	// Generate pairings for each round
	for round := 0; round < numRounds; round++ {
		roundPairings, err := pair.GetRoundRobinPairings(numPlayers, round, seed)
		if err != nil {
			return nil, fmt.Errorf("failed to generate round %d pairings: %w", round, err)
		}

		// Convert the pairing array into GamePairing structs
		// The array is indexed by player number, and the value is their opponent
		// We only process each pairing once (when i < opponent)
		for i := 0; i < len(roundPairings); i++ {
			opponent := roundPairings[i]

			// Skip if no opponent (bye) or if we already processed this pairing
			if opponent < 0 || opponent < i {
				continue
			}

			// Determine who goes first using the round-robin balancing algorithm
			isPlayer1First := determineFirstPlayer(i, opponent, round, numPlayers)

			allPairings = append(allPairings, &GamePairing{
				Player1Index:   i,
				Player2Index:   opponent,
				IsPlayer1First: isPlayer1First,
				Round:          round,
			})
		}
	}

	return allPairings, nil
}

// determineFirstPlayer uses the round-robin balancing algorithm to decide who goes first
// This ensures balanced first/second distribution across all games
// Adapted from pkg/tournament/classic_division.go newClassicPairing
func determineFirstPlayer(playerIndex1, playerIndex2, round, numPlayers int) bool {
	// Use the round robin phase to consistently switch who is going
	// first between different phases of the round robin.
	// Use the playersIndexSum to determine who is going first initially
	// to give some initial variety to the pairings so that a given player
	// doesn't go first every game in the first phase and second every
	// game in the second phase.

	playerIndexSum := playerIndex1 + playerIndex2

	// For round-robin: round/(numPlayers+(numPlayers%2)-1) + playerIndexSum
	// numPlayers+(numPlayers%2)-1 = total rounds in a complete round-robin
	totalRounds := numPlayers + (numPlayers % 2) - 1

	sum := round/totalRounds + playerIndexSum
	switchFirst := (sum % 2) == 1

	// If switchFirst is true, player2 goes first (so return false)
	// If switchFirst is false, player1 goes first (so return true)
	return !switchFirst
}

// CalculateFirstsCounts calculates how many games each player should go first
// This is useful for validation and for storing in the database
func CalculateFirstsCounts(pairings []*GamePairing, numPlayers int) []int {
	firstsCounts := make([]int, numPlayers)

	for _, pairing := range pairings {
		if pairing.IsPlayer1First {
			firstsCounts[pairing.Player1Index]++
		} else {
			firstsCounts[pairing.Player2Index]++
		}
	}

	return firstsCounts
}
