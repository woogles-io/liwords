package league

import (
	"crypto/sha256"
	"encoding/binary"
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

	// Special case: Odd divisions >= 17 need subset selection to limit to 14 games
	// For these divisions, we generate a full round-robin then select a balanced subset
	needsSubsetSelection := (numPlayers%2 == 1) && (numPlayers >= 17)

	// Check if this is a complete round-robin or incomplete (capped)
	isCompleteRoundRobin := true

	// Apply max rounds cap if specified (but not for subset selection cases)
	if !needsSubsetSelection && maxRounds > 0 && numRounds > maxRounds {
		numRounds = maxRounds
		isCompleteRoundRobin = false
	}

	// Generate all pairings
	allPairings := []*GamePairing{}

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

			// For complete round-robins, use the phase-based algorithm
			// For incomplete round-robins, we'll use greedy balancing (set below)
			var isPlayer1First bool
			if isCompleteRoundRobin {
				isPlayer1First = determineFirstPlayer(i, opponent, round, numPlayers)
			}

			allPairings = append(allPairings, &GamePairing{
				Player1Index:   i,
				Player2Index:   opponent,
				IsPlayer1First: isPlayer1First,
				Round:          round,
			})
		}
	}

	// If subset selection is needed (odd divisions >= 17), select exactly 14 games per player
	useSmartFirstAssignment := false
	if needsSubsetSelection {
		targetGames := 14
		subset, err := selectBalancedSubset(allPairings, numPlayers, targetGames, seed)
		if err != nil {
			return nil, fmt.Errorf("failed to select balanced subset: %w", err)
		}
		allPairings = subset
		// Subset selection creates an incomplete round-robin
		isCompleteRoundRobin = false
		useSmartFirstAssignment = true
	}

	// For incomplete round-robins, use greedy balancing to assign first-player
	if !isCompleteRoundRobin {
		// For subset-selected divisions, use smarter assignment
		if useSmartFirstAssignment {
			assignFirstPlayerSmart(allPairings, numPlayers, seed)
		} else {
			// For capped round-robins, use regular greedy
			assignFirstPlayerGreedy(allPairings, numPlayers, seed)
		}
	}

	return allPairings, nil
}

// assignFirstPlayerGreedy assigns first-player using simple greedy algorithm
func assignFirstPlayerGreedy(pairings []*GamePairing, numPlayers int, seed uint64) {
	firstsCounts := make([]int, numPlayers)

	for _, pairing := range pairings {
		p1 := pairing.Player1Index
		p2 := pairing.Player2Index

		// Assign first to whoever has fewer firsts so far
		if firstsCounts[p1] < firstsCounts[p2] {
			pairing.IsPlayer1First = true
			firstsCounts[p1]++
		} else if firstsCounts[p2] < firstsCounts[p1] {
			pairing.IsPlayer1First = false
			firstsCounts[p2]++
		} else {
			// Tie: use seed-based deterministic tiebreaker
			pairing.IsPlayer1First = tiebreakFirstPlayer(p1, p2, pairing.Round, seed)
			if pairing.IsPlayer1First {
				firstsCounts[p1]++
			} else {
				firstsCounts[p2]++
			}
		}
	}
}

// assignFirstPlayerSmart assigns first-player using a smarter algorithm for subset selections
// Sorts pairings before applying greedy to improve balance
func assignFirstPlayerSmart(pairings []*GamePairing, numPlayers int, seed uint64) {
	// Sort pairings by player index sum to give structure to greedy algorithm
	// This helps the greedy approach make better global decisions
	sortPairingsByStructure(pairings, seed)

	// Now apply greedy algorithm on sorted pairings
	assignFirstPlayerGreedy(pairings, numPlayers, seed)
}

// sortPairingsByStructure sorts pairings to optimize first-player assignment
// Uses a combination of round number and player indices for deterministic ordering
func sortPairingsByStructure(pairings []*GamePairing, seed uint64) {
	// Sort by: round first, then by player index sum
	// This creates structure that helps greedy algorithm balance better
	for i := 0; i < len(pairings)-1; i++ {
		for j := i + 1; j < len(pairings); j++ {
			// Primary sort: round number
			if pairings[i].Round > pairings[j].Round {
				pairings[i], pairings[j] = pairings[j], pairings[i]
			} else if pairings[i].Round == pairings[j].Round {
				// Secondary sort: player index sum (deterministic)
				sum_i := pairings[i].Player1Index + pairings[i].Player2Index
				sum_j := pairings[j].Player1Index + pairings[j].Player2Index
				if sum_i > sum_j {
					pairings[i], pairings[j] = pairings[j], pairings[i]
				}
			}
		}
	}
}

// selectBalancedSubset selects a subset of pairings where each player plays exactly targetGames
// This is used for large odd divisions (17+) where a full round-robin exceeds the 14-game target
// Returns the selected pairings and any error
func selectBalancedSubset(allPairings []*GamePairing, numPlayers int, targetGames int, seed uint64) ([]*GamePairing, error) {
	// Strategy: Start with all pairings, remove some to get to targetGames per player
	// Each player plays (numPlayers-1) games in full round-robin
	// We need to remove (numPlayers-1-targetGames) games per player

	gamesToRemovePerPlayer := numPlayers - 1 - targetGames
	if gamesToRemovePerPlayer <= 0 {
		return allPairings, nil // Already at or below target
	}

	// Try multiple times with different shuffle orders if greedy approach fails
	maxAttempts := 100
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Use a different seed for each attempt
		attemptSeed := seed + uint64(attempt)

		result, err := trySelectBalancedSubset(allPairings, numPlayers, gamesToRemovePerPlayer, attemptSeed)
		if err == nil {
			return result, nil
		}
		// If failed, try again with different shuffle
	}

	return nil, fmt.Errorf("failed to find balanced subset after %d attempts", maxAttempts)
}

// trySelectBalancedSubset attempts to select a balanced subset with a given seed
func trySelectBalancedSubset(allPairings []*GamePairing, numPlayers int, gamesToRemovePerPlayer int, seed uint64) ([]*GamePairing, error) {
	// Shuffle pairings deterministically for fair removal
	shuffled := make([]*GamePairing, len(allPairings))
	copy(shuffled, allPairings)
	shufflePairings(shuffled, seed)

	// Track how many more games each player needs to lose
	gamesToLose := make([]int, numPlayers)
	for i := range gamesToLose {
		gamesToLose[i] = gamesToRemovePerPlayer
	}

	// Mark pairings to keep (true = keep, false = remove)
	keep := make([]bool, len(shuffled))
	for i := range keep {
		keep[i] = true // Start with all pairings
	}

	// Remove pairings strategically
	for i, pairing := range shuffled {
		p1 := pairing.Player1Index
		p2 := pairing.Player2Index

		// Can we remove this pairing? (both players still need to lose games)
		if gamesToLose[p1] > 0 && gamesToLose[p2] > 0 {
			keep[i] = false
			gamesToLose[p1]--
			gamesToLose[p2]--
		}
	}

	// Verify all players lost exactly the right number of games
	for i, count := range gamesToLose {
		if count != 0 {
			return nil, fmt.Errorf("player %d needs to lose %d more games", i, count)
		}
	}

	// Build result with kept pairings
	selectedPairings := []*GamePairing{}
	for i, pairing := range shuffled {
		if keep[i] {
			selectedPairings = append(selectedPairings, pairing)
		}
	}

	return selectedPairings, nil
}

// shufflePairings shuffles pairings deterministically based on seed
// Uses Fisher-Yates shuffle with seed-based random number generation
func shufflePairings(pairings []*GamePairing, seed uint64) {
	// Simple deterministic shuffle using seed
	for i := len(pairings) - 1; i > 0; i-- {
		// Generate deterministic "random" index based on seed and position
		h := sha256.New()
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], seed)
		h.Write(buf[:])
		binary.LittleEndian.PutUint64(buf[:], uint64(i))
		h.Write(buf[:])
		hash := h.Sum(nil)

		// Take modulo before converting to int to avoid overflow issues
		randVal := binary.LittleEndian.Uint64(hash[:8])
		j := int(randVal % uint64(i+1))

		// Swap
		pairings[i], pairings[j] = pairings[j], pairings[i]
	}
}

// determineFirstPlayer uses the round-robin balancing algorithm to decide who goes first
// This ensures balanced first/second distribution across all games in a complete round-robin
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

// tiebreakFirstPlayer provides a deterministic but pseudo-random tiebreaker
// when both players have the same number of first-player assignments.
// Uses a hash of player indices, round, and seed to ensure reproducibility.
func tiebreakFirstPlayer(playerIndex1, playerIndex2, round int, seed uint64) bool {
	// Create a deterministic hash from the inputs
	h := sha256.New()

	// Write all inputs to the hash
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], seed)
	h.Write(buf[:])

	binary.LittleEndian.PutUint64(buf[:], uint64(playerIndex1))
	h.Write(buf[:])

	binary.LittleEndian.PutUint64(buf[:], uint64(playerIndex2))
	h.Write(buf[:])

	binary.LittleEndian.PutUint64(buf[:], uint64(round))
	h.Write(buf[:])

	hash := h.Sum(nil)

	// Use the first byte to determine who goes first
	return (hash[0] % 2) == 0
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
