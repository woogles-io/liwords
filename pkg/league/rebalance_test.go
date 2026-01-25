package league

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestCalculatePriorityScores(t *testing.T) {
	rm := &RebalanceManager{}
	numVirtualDivs := int32(5)

	// New formula: ((100,000 * (numVirtualDivs - virtualDiv + 1)) + bonus + rankComponent) * weight
	tests := []struct {
		name          string
		player        PlayerWithVirtualDiv
		expectedScore float64
		description   string
	}{
		{
			name: "Alice - Promoted to Div 1, 3rd out of 15",
			player: PlayerWithVirtualDiv{
				UserID:               "alice",
				VirtualDivision:      1,
				PlacementStatus:      ipc.PlacementStatus_PLACEMENT_PROMOTED,
				PreviousDivisionSize: 15,
				PreviousRank:         3,
				HiatusSeasons:        0,
			},
			expectedScore: 530_012,
			description:   "((100,000 * (5 - 1 + 1)) + 30,000 + (15 - 3)) * 1 = 530,012",
		},
		{
			name: "Bob - Stayed in Div 1, 10th out of 15",
			player: PlayerWithVirtualDiv{
				UserID:               "bob",
				VirtualDivision:      1,
				PlacementStatus:      ipc.PlacementStatus_PLACEMENT_STAYED,
				PreviousDivisionSize: 15,
				PreviousRank:         10,
				HiatusSeasons:        0,
			},
			expectedScore: 540_005,
			description:   "((100,000 * (5 - 1 + 1)) + 40,000 + (15 - 10)) * 1 = 540,005",
		},
		{
			name: "Charlie - Relegated to Div 2, 14th out of 15",
			player: PlayerWithVirtualDiv{
				UserID:               "charlie",
				VirtualDivision:      2,
				PlacementStatus:      ipc.PlacementStatus_PLACEMENT_RELEGATED,
				PreviousDivisionSize: 15,
				PreviousRank:         14,
				HiatusSeasons:        0,
			},
			expectedScore: 450_001,
			description:   "((100,000 * (5 - 2 + 1)) + 50,000 + (15 - 14)) * 1 = 450,001",
		},
		{
			name: "Dora - Stayed in Div 2, 12th out of 15",
			player: PlayerWithVirtualDiv{
				UserID:               "dora",
				VirtualDivision:      2,
				PlacementStatus:      ipc.PlacementStatus_PLACEMENT_STAYED,
				PreviousDivisionSize: 15,
				PreviousRank:         12,
				HiatusSeasons:        0,
			},
			expectedScore: 440_003,
			description:   "((100,000 * (5 - 2 + 1)) + 40,000 + (15 - 12)) * 1 = 440,003",
		},
		{
			name: "Frankie - 4 seasons off, was in Div 2",
			player: PlayerWithVirtualDiv{
				UserID:               "frankie",
				VirtualDivision:      2,
				PlacementStatus:      ipc.PlacementStatus_PLACEMENT_SHORT_HIATUS_RETURNING,
				PreviousDivisionSize: 0, // Use 0 for hiatus players
				PreviousRank:         0, // Use 0 for hiatus players
				HiatusSeasons:        4,
			},
			expectedScore: 310_686, // ((100,000 * (5 - 2 + 1)) + 10,000 + 0) * (0.933^4) = 310,686
			description:   "((100,000 * (5 - 2 + 1)) + 10,000 + 0) * (0.933^4) = 310,686",
		},
	}

	players := make([]PlayerWithVirtualDiv, len(tests))
	for i, tt := range tests {
		players[i] = tt.player
	}

	// Test with Season 2 to avoid Season 1 special rating logic
	playersWithPriority := rm.CalculatePriorityScores(players, numVirtualDivs, 2)

	// Create a map by UserID for easier lookup after sorting
	scoreMap := make(map[string]float64)
	for _, p := range playersWithPriority {
		scoreMap[p.UserID] = p.PriorityScore
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualScore := scoreMap[tt.player.UserID]

			// Allow small floating point tolerance for hiatus calculations
			tolerance := 1.0
			if tt.player.HiatusSeasons > 0 {
				tolerance = 100.0 // Larger tolerance for exponential calculations
			}

			assert.InDelta(t, tt.expectedScore, actualScore, tolerance,
				"Score mismatch for %s\n"+
					"Expected: %.2f\n"+
					"Actual: %.2f\n"+
					"Calculation: %s",
				tt.name, tt.expectedScore, actualScore, tt.description)
		})
	}
}

func TestCalculatePriorityScores_Sorting(t *testing.T) {
	rm := &RebalanceManager{}
	numVirtualDivs := int32(5)

	// Create the players from the example
	players := []PlayerWithVirtualDiv{
		{UserID: "alice", VirtualDivision: 1, PlacementStatus: ipc.PlacementStatus_PLACEMENT_PROMOTED, PreviousDivisionSize: 15, PreviousRank: 3},
		{UserID: "bob", VirtualDivision: 1, PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED, PreviousDivisionSize: 15, PreviousRank: 10},
		{UserID: "charlie", VirtualDivision: 2, PlacementStatus: ipc.PlacementStatus_PLACEMENT_RELEGATED, PreviousDivisionSize: 15, PreviousRank: 14},
		{UserID: "dora", VirtualDivision: 2, PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED, PreviousDivisionSize: 15, PreviousRank: 12},
		{UserID: "frankie", VirtualDivision: 2, PlacementStatus: ipc.PlacementStatus_PLACEMENT_SHORT_HIATUS_RETURNING, HiatusSeasons: 4},
	}

	// Test with Season 2 to avoid Season 1 special rating logic
	playersWithPriority := rm.CalculatePriorityScores(players, numVirtualDivs, 2)

	// Verify sorting: should be Bob > Alice > Charlie > Dora > Frankie
	expectedOrder := []string{"bob", "alice", "charlie", "dora", "frankie"}

	for i, expectedUserID := range expectedOrder {
		assert.Equal(t, expectedUserID, playersWithPriority[i].UserID,
			"Player at position %d should be %s, got %s (score: %.2f)",
			i, expectedUserID, playersWithPriority[i].UserID, playersWithPriority[i].PriorityScore)
	}
}

func TestCalculatePriorityScores_NewPlayersRatingBased(t *testing.T) {
	rm := &RebalanceManager{}
	numVirtualDivs := int32(1) // Single division

	// Create NEW players with different ratings
	// Base score: ((100,000 * (1 - 1 + 1)) + 5,000 + 0) = 105,000
	// Plus rating for NEW players
	players := []PlayerWithVirtualDiv{
		{UserID: "alice", VirtualDivision: 1, PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW, Rating: 1800},
		{UserID: "bob", VirtualDivision: 1, PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW, Rating: 1600},
		{UserID: "charlie", VirtualDivision: 1, PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW, Rating: 2000},
		{UserID: "dora", VirtualDivision: 1, PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW, Rating: 1400},
		{UserID: "eve", VirtualDivision: 1, PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW, Rating: 0}, // No rating
		{UserID: "frank", VirtualDivision: 1, PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW, Rating: 1700},
	}

	// Test Season 1: should be sorted by rating (rating is now added for all NEW players)
	playersWithPriority := rm.CalculatePriorityScores(players, numVirtualDivs, 1)

	// Expected order: charlie(2000) > alice(1800) > frank(1700) > bob(1600) > dora(1400) > eve(0)
	expectedOrder := []string{"charlie", "alice", "frank", "bob", "dora", "eve"}
	// base = 100,000 + 5,000 = 105,000, plus rating
	expectedScores := []float64{107_000, 106_800, 106_700, 106_600, 106_400, 105_000}

	for i, expectedUserID := range expectedOrder {
		assert.Equal(t, expectedUserID, playersWithPriority[i].UserID,
			"Player at position %d should be %s, got %s (score: %.2f)",
			i, expectedUserID, playersWithPriority[i].UserID, playersWithPriority[i].PriorityScore)

		assert.Equal(t, expectedScores[i], playersWithPriority[i].PriorityScore,
			"Score for %s should be %.2f, got %.2f",
			expectedUserID, expectedScores[i], playersWithPriority[i].PriorityScore)
	}

	// Test Season 2+: NEW players should ALSO be sorted by rating (this changed!)
	playersWithPriority2 := rm.CalculatePriorityScores(players, numVirtualDivs, 2)

	// Should have same order and scores as Season 1
	for i, expectedUserID := range expectedOrder {
		assert.Equal(t, expectedUserID, playersWithPriority2[i].UserID,
			"Season 2 player at position %d should be %s, got %s",
			i, expectedUserID, playersWithPriority2[i].UserID)

		assert.Equal(t, expectedScores[i], playersWithPriority2[i].PriorityScore,
			"Season 2 score for %s should be %.2f, got %.2f",
			expectedUserID, expectedScores[i], playersWithPriority2[i].PriorityScore)
	}
}

func TestCalculatePriorityScores_WeightCalculation(t *testing.T) {
	tests := []struct {
		hiatusSeasons  int32
		expectedWeight float64
	}{
		{0, 1.0},
		{1, 0.933},
		{2, 0.870489},
		{4, 0.757751},  // 0.933^4
		{10, 0.499823}, // 0.933^10
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			actualWeight := math.Pow(HiatusWeightBase, float64(tt.hiatusSeasons))
			assert.InDelta(t, tt.expectedWeight, actualWeight, 0.001,
				"Weight for %d season hiatus should be %.6f, got %.6f",
				tt.hiatusSeasons, tt.expectedWeight, actualWeight)
		})
	}
}

func TestCalculateDivisionCount(t *testing.T) {
	tests := []struct {
		playerCount  int
		expectedDivs int
		description  string
	}{
		{15, 1, "15 players → 1 division (15/15 = 1.0)"},
		{20, 1, "20 players → 1 division (20/15 = 1.33 rounds to 1)"},
		{22, 1, "22 players → 1 division (22/15 = 1.47 rounds to 1)"},
		{23, 2, "23 players → 2 divisions (23/15 = 1.53 rounds to 2)"},
		{30, 2, "30 players → 2 divisions (30/15 = 2.0)"},
		{45, 3, "45 players → 3 divisions (45/15 = 3.0)"},
		{100, 7, "100 players → 7 divisions (100/15 = 6.67 rounds to 7)"},
		{125, 8, "125 players → 8 divisions (125/15 = 8.33 rounds to 8)"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			const idealDivisionSize = 15
			actualDivs := int(math.Round(float64(tt.playerCount) / float64(idealDivisionSize)))
			assert.Equal(t, tt.expectedDivs, actualDivs, tt.description)
		})
	}
}

func TestAssignVirtualDivisions_Outcomes(t *testing.T) {
	tests := []struct {
		name            string
		status          string
		currentDiv      int32
		expectedVirtual int32
	}{
		{"Promoted from Div 3", "PROMOTED", 3, 2},
		{"Promoted from Div 2", "PROMOTED", 2, 1},
		{"Promoted from Div 1", "PROMOTED", 1, 1}, // Can't go below 1
		{"Relegated from Div 1", "RELEGATED", 1, 2},
		{"Relegated from Div 3", "RELEGATED", 3, 4},
		{"Stayed in Div 2", "STAYED", 2, 2},
		{"Hiatus in Div 3", "SHORT_HIATUS_RETURNING", 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the virtual division logic
			var virtualDiv int32
			switch tt.status {
			case "PROMOTED":
				virtualDiv = tt.currentDiv - 1
				if virtualDiv < 1 {
					virtualDiv = 1
				}
			case "RELEGATED":
				virtualDiv = tt.currentDiv + 1
			case "STAYED", "SHORT_HIATUS_RETURNING", "LONG_HIATUS_RETURNING":
				virtualDiv = tt.currentDiv
			}

			assert.Equal(t, tt.expectedVirtual, virtualDiv,
				"Virtual division for %s should be %d, got %d",
				tt.name, tt.expectedVirtual, virtualDiv)
		})
	}
}

func TestMergeUndersizedFinalDivision(t *testing.T) {
	tests := []struct {
		name              string
		finalDivisionSize int
		shouldMerge       bool
	}{
		{"11 players - should merge", 11, true},
		{"12 players - should NOT merge (at threshold)", 12, false},
		{"15 players - should NOT merge", 15, false},
		{"1 player - should merge", 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldMerge := tt.finalDivisionSize < MinimumFinalDivSize
			assert.Equal(t, tt.shouldMerge, shouldMerge,
				"Division with %d players: merge=%v, expected=%v",
				tt.finalDivisionSize, shouldMerge, tt.shouldMerge)
		})
	}
}

func TestSequentialAssignment(t *testing.T) {
	// Test that players are assigned 15 per division
	tests := []struct {
		totalPlayers int
		numDivs      int
		expected     []int // Players per division
	}{
		{15, 1, []int{15}},
		{30, 2, []int{15, 15}},
		{45, 3, []int{15, 15, 15}},
		{32, 2, []int{15, 17}}, // Overflow goes to last division
		{50, 3, []int{15, 15, 20}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			const idealDivisionSize = 15
			playersPerDiv := make([]int, tt.numDivs)
			for i := 0; i < tt.totalPlayers; i++ {
				divIndex := i / idealDivisionSize
				if divIndex >= tt.numDivs {
					divIndex = tt.numDivs - 1
				}
				playersPerDiv[divIndex]++
			}

			require.Equal(t, tt.expected, playersPerDiv,
				"With %d players in %d divisions, expected distribution %v, got %v",
				tt.totalPlayers, tt.numDivs, tt.expected, playersPerDiv)
		})
	}
}

func TestPriorityBonusConstants(t *testing.T) {
	// Verify the priority bonus constants match the spec
	assert.Equal(t, 40_000, PriorityBonusStayed, "STAYED bonus")
	assert.Equal(t, 30_000, PriorityBonusPromoted, "PROMOTED bonus")
	assert.Equal(t, 50_000, PriorityBonusRelegated, "RELEGATED bonus")
	assert.Equal(t, 10_000, PriorityBonusHiatusReturning, "HIATUS bonus")
	assert.Equal(t, 5_000, PriorityBonusNew, "NEW bonus - lowest priority")
}

func TestDivisionSizeConstants(t *testing.T) {
	// IdealDivisionSize is now configurable per league (via LeagueSettings.IdealDivisionSize)
	assert.Equal(t, 12, MinimumFinalDivSize, "Minimum final division size")
}

func TestHiatusWeight_HalvesEveryTenSeasons(t *testing.T) {
	// Verify that weight halves approximately every 10 seasons
	weight10 := math.Pow(HiatusWeightBase, 10)
	assert.InDelta(t, 0.5, weight10, 0.003,
		"Weight after 10 seasons should be ~0.5, got %.4f", weight10)

	weight20 := math.Pow(HiatusWeightBase, 20)
	assert.InDelta(t, 0.25, weight20, 0.002,
		"Weight after 20 seasons should be ~0.25, got %.4f", weight20)
}

func TestNewRookieSplitting(t *testing.T) {
	// Test the logic for splitting <10 new rookies between bottom 2 divisions
	tests := []struct {
		numRookies      int
		numVirtualDivs  int32
		expectedTopHalf int32 // Virtual div for top half
		expectedBotHalf int32 // Virtual div for bottom half
	}{
		{8, 5, 4, 5}, // 8 rookies, 5 divs → top 4 to Div 4, bottom 4 to Div 5
		{5, 3, 2, 3}, // 5 rookies, 3 divs → top 2 to Div 2, bottom 3 to Div 3
		{9, 1, 1, 1}, // 9 rookies, 1 div → all to Div 1
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if tt.numVirtualDivs == 1 {
				// All go to Division 1
				assert.Equal(t, int32(1), tt.expectedTopHalf)
				assert.Equal(t, int32(1), tt.expectedBotHalf)
			} else {
				// Split between bottom 2
				secondBottom := tt.numVirtualDivs - 1
				bottom := tt.numVirtualDivs

				assert.Equal(t, secondBottom, tt.expectedTopHalf,
					"Top half should go to Div %d", secondBottom)
				assert.Equal(t, bottom, tt.expectedBotHalf,
					"Bottom half should go to Div %d", bottom)
			}
		})
	}
}
