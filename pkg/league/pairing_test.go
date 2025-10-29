package league

import (
	"testing"
)

func TestGenerateAllLeaguePairings(t *testing.T) {
	tests := []struct {
		name            string
		numPlayers      int
		expectedGames   int
		expectedRounds  int
		minFirsts       int
		maxFirsts       int
	}{
		{
			name:           "14 players (even)",
			numPlayers:     14,
			expectedGames:  91, // 14*13/2 = 91 total games
			expectedRounds: 13, // 14-1 = 13 rounds
			minFirsts:      6,  // Each player should get 6 or 7 firsts
			maxFirsts:      7,
		},
		{
			name:           "12 players (even)",
			numPlayers:     12,
			expectedGames:  66, // 12*11/2 = 66 total games
			expectedRounds: 11, // 12-1 = 11 rounds
			minFirsts:      5,  // Each player should get 5 or 6 firsts
			maxFirsts:      6,
		},
		{
			name:           "13 players (odd)",
			numPlayers:     13,
			expectedGames:  78, // 13*12/2 = 78 total games
			expectedRounds: 13, // Same as numPlayers when odd
			minFirsts:      6,  // Each player should get 6 firsts
			maxFirsts:      6,
		},
		{
			name:           "4 players (small even)",
			numPlayers:     4,
			expectedGames:  6, // 4*3/2 = 6 total games
			expectedRounds: 3, // 4-1 = 3 rounds
			minFirsts:      1, // Each player should get 1 or 2 firsts
			maxFirsts:      2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairings, err := GenerateAllLeaguePairings(tt.numPlayers, 12345)
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
	_, err := GenerateAllLeaguePairings(1, 12345)
	if err == nil {
		t.Error("Expected error for 1 player, got nil")
	}

	_, err = GenerateAllLeaguePairings(0, 12345)
	if err == nil {
		t.Error("Expected error for 0 players, got nil")
	}
}

func TestGenerateAllLeaguePairings_Deterministic(t *testing.T) {
	// Same seed should produce same results
	pairings1, _ := GenerateAllLeaguePairings(14, 99999)
	pairings2, _ := GenerateAllLeaguePairings(14, 99999)

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
	pairings1, _ := GenerateAllLeaguePairings(14, 11111)
	pairings2, _ := GenerateAllLeaguePairings(14, 22222)

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
	// Test that the algorithm produces consistent results
	numPlayers := 14

	// Test a few specific cases
	tests := []struct {
		player1 int
		player2 int
		round   int
		// We don't specify expected result because the algorithm is complex,
		// but we verify it's deterministic
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
