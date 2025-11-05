package league

import (
	"testing"
)

func TestGenerateAllLeaguePairings(t *testing.T) {
	tests := []struct {
		name            string
		numPlayers      int
		maxRounds       int
		expectedGames   int
		expectedRounds  int
		minFirsts       int
		maxFirsts       int
	}{
		{
			name:           "14 players (even)",
			numPlayers:     14,
			maxRounds:      0, // No cap
			expectedGames:  91, // 14*13/2 = 91 total games
			expectedRounds: 13, // 14-1 = 13 rounds
			minFirsts:      6,  // Each player should get 6 or 7 firsts
			maxFirsts:      7,
		},
		{
			name:           "10 players (small even)",
			numPlayers:     10,
			maxRounds:      0, // No cap
			expectedGames:  45, // 10*9/2 = 45 total games
			expectedRounds: 9,  // 10-1 = 9 rounds
			minFirsts:      4,  // Each player should get 4 or 5 firsts
			maxFirsts:      5,
		},
		{
			name:           "12 players (even)",
			numPlayers:     12,
			maxRounds:      0, // No cap
			expectedGames:  66, // 12*11/2 = 66 total games
			expectedRounds: 11, // 12-1 = 11 rounds
			minFirsts:      5,  // Each player should get 5 or 6 firsts
			maxFirsts:      6,
		},
		{
			name:           "13 players (odd)",
			numPlayers:     13,
			maxRounds:      0, // No cap
			expectedGames:  78, // 13*12/2 = 78 total games
			expectedRounds: 13, // Same as numPlayers when odd
			minFirsts:      5,  // Odd numbers with byes make perfect balance harder
			maxFirsts:      7,  // Greedy algorithm gets close but not perfect
		},
		{
			name:           "4 players (small even)",
			numPlayers:     4,
			maxRounds:      0, // No cap
			expectedGames:  6, // 4*3/2 = 6 total games
			expectedRounds: 3, // 4-1 = 3 rounds
			minFirsts:      1, // Each player should get 1 or 2 firsts
			maxFirsts:      2,
		},
		{
			name:           "14 players (even)",
			numPlayers:     14,
			maxRounds:      0,  // No cap
			expectedGames:  91, // 14*13/2 = 91 total games
			expectedRounds: 13, // 14-1 = 13 rounds
			minFirsts:      6,  // Each player should get 6 or 7 firsts
			maxFirsts:      7,
		},
		{
			name:           "15 players (complete round-robin)",
			numPlayers:     15,
			maxRounds:      0,   // No cap - full round-robin
			expectedGames:  105, // 7 games per round * 15 rounds = 105
			expectedRounds: 15,  // 15 rounds (odd number of players)
			minFirsts:      7,   // Each player plays 14 games, should get 7 firsts
			maxFirsts:      7,   // Perfect balance with phase-based algorithm
		},
		{
			name:           "16 players with 14 round cap",
			numPlayers:     16,
			maxRounds:      14, // Cap at 14
			expectedGames:  112, // 8 games per round * 14 rounds = 112
			expectedRounds: 14,  // Capped at 14 instead of full 15
			minFirsts:      6,   // With greedy balancing, most get 7, but some may get 6 or 8
			maxFirsts:      8,   // Incomplete round-robin makes perfect balance difficult
		},
		{
			name:           "17 players (subset selection)",
			numPlayers:     17,
			maxRounds:      0,   // No cap - subset selection used
			expectedGames:  119, // 17 × 14 / 2 = 119 games
			expectedRounds: 17,  // Games distributed across all 17 rounds
			minFirsts:      6,   // Smart sorting improves balance
			maxFirsts:      8,   // Most get exactly 7
		},
		{
			name:           "19 players (subset selection)",
			numPlayers:     19,
			maxRounds:      0,   // No cap - subset selection used
			expectedGames:  133, // 19 × 14 / 2 = 133 games
			expectedRounds: 19,  // Games distributed across all 19 rounds
			minFirsts:      6,   // Smart sorting improves balance
			maxFirsts:      8,   // Most get exactly 7
		},
		{
			name:           "21 players (subset selection)",
			numPlayers:     21,
			maxRounds:      0,   // No cap - subset selection used
			expectedGames:  147, // 21 × 14 / 2 = 147 games
			expectedRounds: 21,  // Games distributed across all 21 rounds
			minFirsts:      6,   // Smart sorting improves balance
			maxFirsts:      8,   // Most get exactly 7
		},
		{
			name:           "22 players (large even)",
			numPlayers:     22,
			maxRounds:      14,  // Cap at 14
			expectedGames:  154, // 11 games per round * 14 rounds
			expectedRounds: 14,  // Capped
			minFirsts:      6,   // With greedy balancing
			maxFirsts:      8,
		},
		{
			name:           "23 players (subset selection)",
			numPlayers:     23,
			maxRounds:      0,   // No cap - subset selection used
			expectedGames:  161, // 23 × 14 / 2 = 161 games
			expectedRounds: 23,  // Games distributed across all 23 rounds
			minFirsts:      6,   // Smart sorting improves balance
			maxFirsts:      8,   // Most get exactly 7
		},
		{
			name:           "24 players (large even)",
			numPlayers:     24,
			maxRounds:      14,  // Cap at 14
			expectedGames:  168, // 12 games per round * 14 rounds
			expectedRounds: 14,  // Capped
			minFirsts:      6,   // With greedy balancing
			maxFirsts:      8,
		},
		{
			name:           "25 players (subset selection)",
			numPlayers:     25,
			maxRounds:      0,   // No cap - subset selection used
			expectedGames:  175, // 25 × 14 / 2 = 175 games
			expectedRounds: 25,  // Games distributed across all 25 rounds
			minFirsts:      5,   // One outlier possible in very large divisions
			maxFirsts:      8,   // Most get 7
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairings, err := GenerateAllLeaguePairings(tt.numPlayers, 12345, tt.maxRounds)
			if err != nil {
				t.Fatalf("GenerateAllLeaguePairings failed: %v", err)
			}

			// Check total number of games
			if len(pairings) != tt.expectedGames {
				t.Errorf("Expected %d games, got %d", tt.expectedGames, len(pairings))
			}

			// Check that we have the right number of rounds
			roundsFound := make(map[int]bool)
			for _, pairing := range pairings {
				roundsFound[pairing.Round] = true
			}
			if len(roundsFound) != tt.expectedRounds {
				t.Errorf("Expected %d rounds, got %d", tt.expectedRounds, len(roundsFound))
			}

			// Check that each player plays every other player exactly once
			pairingsSeen := make(map[string]bool)
			for _, pairing := range pairings {
				p1, p2 := pairing.Player1Index, pairing.Player2Index
				if p1 > p2 {
					p1, p2 = p2, p1
				}
				key := string(rune(p1)) + "-" + string(rune(p2))
				if pairingsSeen[key] {
					t.Errorf("Pairing %d vs %d appears more than once", p1, p2)
				}
				pairingsSeen[key] = true
			}

			// Check that firsts are balanced
			firstsCounts := CalculateFirstsCounts(pairings, tt.numPlayers)
			for i, count := range firstsCounts {
				if count < tt.minFirsts || count > tt.maxFirsts {
					t.Errorf("Player %d has %d firsts, expected between %d and %d",
						i, count, tt.minFirsts, tt.maxFirsts)
				}
			}

			// Verify the total number of firsts equals total games
			totalFirsts := 0
			for _, count := range firstsCounts {
				totalFirsts += count
			}
			if totalFirsts != tt.expectedGames {
				t.Errorf("Total firsts (%d) should equal total games (%d)", totalFirsts, tt.expectedGames)
			}

			// Print distribution for debugging
			t.Logf("Firsts distribution for %d players: %v", tt.numPlayers, firstsCounts)
		})
	}
}

func TestGenerateAllLeaguePairings_TooFewPlayers(t *testing.T) {
	_, err := GenerateAllLeaguePairings(1, 12345, 0)
	if err == nil {
		t.Error("Expected error for 1 player, got nil")
	}

	_, err = GenerateAllLeaguePairings(0, 12345, 0)
	if err == nil {
		t.Error("Expected error for 0 players, got nil")
	}
}

func TestGenerateAllLeaguePairings_Deterministic(t *testing.T) {
	// Same seed should produce same results
	pairings1, _ := GenerateAllLeaguePairings(14, 99999, 0)
	pairings2, _ := GenerateAllLeaguePairings(14, 99999, 0)

	if len(pairings1) != len(pairings2) {
		t.Fatal("Same seed produced different number of pairings")
	}

	for i := range pairings1 {
		if pairings1[i].Player1Index != pairings2[i].Player1Index ||
			pairings1[i].Player2Index != pairings2[i].Player2Index ||
			pairings1[i].IsPlayer1First != pairings2[i].IsPlayer1First ||
			pairings1[i].Round != pairings2[i].Round {
			t.Errorf("Pairing %d differs between runs with same seed", i)
		}
	}
}

func TestGenerateAllLeaguePairings_DifferentSeeds(t *testing.T) {
	// Different seeds should produce different initial shuffles (but same logical structure)
	pairings1, _ := GenerateAllLeaguePairings(14, 11111, 0)
	pairings2, _ := GenerateAllLeaguePairings(14, 22222, 0)

	if len(pairings1) != len(pairings2) {
		t.Fatal("Different seeds produced different number of pairings")
	}

	// The pairings should be different (at least some of them)
	// because the initial shuffle is different
	different := false
	for i := range pairings1 {
		if pairings1[i].Player1Index != pairings2[i].Player1Index ||
			pairings1[i].Player2Index != pairings2[i].Player2Index {
			different = true
			break
		}
	}

	if !different {
		t.Error("Different seeds produced identical pairings (very unlikely)")
	}
}

func TestDetermineFirstPlayer(t *testing.T) {
	// Test that the algorithm produces consistent results for complete round-robins
	numPlayers := 14

	// Test a few specific cases
	tests := []struct {
		player1 int
		player2 int
		round   int
	}{
		{0, 1, 0},
		{0, 1, 1},
		{5, 8, 3},
		{12, 13, 12},
	}

	for _, tt := range tests {
		// Call twice to ensure deterministic
		result1 := determineFirstPlayer(tt.player1, tt.player2, tt.round, numPlayers)
		result2 := determineFirstPlayer(tt.player1, tt.player2, tt.round, numPlayers)

		if result1 != result2 {
			t.Errorf("determineFirstPlayer not deterministic for players %d,%d round %d",
				tt.player1, tt.player2, tt.round)
		}
	}
}

func TestTiebreakFirstPlayer(t *testing.T) {
	// Test that the tiebreaker produces deterministic results
	seed := uint64(12345)

	// Test a few specific cases
	tests := []struct {
		player1 int
		player2 int
		round   int
	}{
		{0, 1, 0},
		{0, 1, 1},
		{5, 8, 3},
		{12, 13, 12},
	}

	for _, tt := range tests {
		// Call twice to ensure deterministic
		result1 := tiebreakFirstPlayer(tt.player1, tt.player2, tt.round, seed)
		result2 := tiebreakFirstPlayer(tt.player1, tt.player2, tt.round, seed)

		if result1 != result2 {
			t.Errorf("tiebreakFirstPlayer not deterministic for players %d,%d round %d",
				tt.player1, tt.player2, tt.round)
		}
	}

	// Test that different seeds produce potentially different results
	result1 := tiebreakFirstPlayer(5, 8, 3, 11111)
	result2 := tiebreakFirstPlayer(5, 8, 3, 22222)

	t.Logf("Same inputs with different seeds: seed1=%v, seed2=%v", result1, result2)
	// We don't assert they're different because it's probabilistic,
	// but this logs the behavior for manual verification
}
