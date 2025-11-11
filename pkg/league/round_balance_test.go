package league

import (
	"fmt"
	"testing"
)

// TestGameDistributionAcrossAllDivisionSizes verifies that all players
// play the same number of games for all division sizes
func TestGameDistributionAcrossAllDivisionSizes(t *testing.T) {
	// Test all division sizes from 10 to 25 players
	divisionSizes := []int{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25}
	seed := uint64(12345)

	for _, numPlayers := range divisionSizes {
		t.Run(fmt.Sprintf("%d_players", numPlayers), func(t *testing.T) {
			// Calculate max rounds based on current logic
			maxRounds := calculateMaxRoundsForTest(numPlayers)

			// Generate pairings
			pairings, err := GenerateAllLeaguePairings(numPlayers, seed, maxRounds)
			if err != nil {
				t.Fatalf("Failed to generate pairings: %v", err)
			}

			// Count games per player
			gamesPerPlayer := make([]int, numPlayers)
			for _, pairing := range pairings {
				gamesPerPlayer[pairing.Player1Index]++
				gamesPerPlayer[pairing.Player2Index]++
			}

			// Analyze distribution
			min, max := minMaxGames(gamesPerPlayer)
			isBalanced := (max - min) == 0

			// Log distribution
			t.Logf("%d players: rounds=%d, games=%d, distribution=%v, balanced=%v",
				numPlayers, len(getRounds(pairings)), len(pairings), gamesPerPlayer, isBalanced)

			// Check if balanced
			if !isBalanced {
				t.Errorf("IMBALANCED: %d players have %d-%d games (diff=%d)",
					numPlayers, min, max, max-min)
			}

			// Check if within 14-game target
			if max > 14 {
				t.Logf("WARNING: %d players have %d games (exceeds 14-game target)",
					numPlayers, max)
			}

			// Report ideal state
			if min == 14 && max == 14 {
				t.Logf("✓ IDEAL: All %d players play exactly 14 games", numPlayers)
			} else if isBalanced && min <= 14 {
				t.Logf("✓ GOOD: All %d players play %d games", numPlayers, min)
			}
		})
	}
}

// calculateMaxRoundsForTest mimics the logic in season_start.go
func calculateMaxRoundsForTest(numPlayers int) int {
	if numPlayers >= 16 {
		return 14
	}
	return 0 // No limit
}

// minMaxGames returns the min and max games across all players
func minMaxGames(gamesPerPlayer []int) (int, int) {
	if len(gamesPerPlayer) == 0 {
		return 0, 0
	}

	min := gamesPerPlayer[0]
	max := gamesPerPlayer[0]

	for _, count := range gamesPerPlayer[1:] {
		if count < min {
			min = count
		}
		if count > max {
			max = count
		}
	}

	return min, max
}

// getRounds returns the unique rounds from pairings
func getRounds(pairings []*GamePairing) map[int]bool {
	rounds := make(map[int]bool)
	for _, pairing := range pairings {
		rounds[pairing.Round] = true
	}
	return rounds
}
