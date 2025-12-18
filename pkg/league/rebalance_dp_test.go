package league

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// TestCalculateSizePenalty tests the size penalty function
func TestCalculateSizePenalty(t *testing.T) {
	tests := []struct {
		name          string
		size          int
		idealSize     int
		expectedCost  float64
		description   string
	}{
		{
			name:         "Perfect size (15)",
			size:         15,
			idealSize:    15,
			expectedCost: 0,
			description:  "No penalty for ideal size",
		},
		{
			name:         "Size 13 (within ideal range)",
			size:         13,
			idealSize:    15,
			expectedCost: 40, // (13-15)^2 * 10 = 40
			description:  "Small penalty within ideal range",
		},
		{
			name:         "Size 17 (within ideal range)",
			size:         17,
			idealSize:    15,
			expectedCost: 40, // (17-15)^2 * 10 = 40
			description:  "Small penalty within ideal range",
		},
		{
			name:         "Size 12 (at ideal range boundary)",
			size:         12,
			idealSize:    15,
			expectedCost: 90, // (12-15)^2 * 10 = 90
			description:  "Penalty at ideal range boundary",
		},
		{
			name:         "Size 18 (at ideal range boundary)",
			size:         18,
			idealSize:    15,
			expectedCost: 90, // (18-15)^2 * 10 = 90
			description:  "Penalty at ideal range boundary",
		},
		{
			name:         "Size 10 (outside ideal range)",
			size:         10,
			idealSize:    15,
			expectedCost: 350, // (10-15)^2 * 10 + (12-10) * 50 = 250 + 100 = 350
			description:  "Large penalty outside ideal range",
		},
		{
			name:         "Size 22 (outside ideal range)",
			size:         22,
			idealSize:    15,
			expectedCost: 690, // (22-15)^2 * 10 + (22-18) * 50 = 490 + 200 = 690
			description:  "Large penalty outside ideal range",
		},
		{
			name:         "Size 8 (emergency minimum)",
			size:         8,
			idealSize:    15,
			expectedCost: 690, // (8-15)^2 * 10 + (12-8) * 50 = 490 + 200 = 690
			description:  "Very large penalty at emergency minimum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := calculateSizePenalty(tt.size, tt.idealSize)
			assert.Equal(t, tt.expectedCost, cost, tt.description)
		})
	}
}

// TestCalculatePlacementPenalty tests placement penalties for RETURNING players
func TestCalculatePlacementPenalty(t *testing.T) {
	tests := []struct {
		name         string
		player       DPPlayer
		divNum       int
		expectedCost float64
		description  string
	}{
		{
			name: "Perfect placement",
			player: DPPlayer{
				TargetDivision:  3,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
				WasRelegated:    false,
				SeasonsAway:     0,
			},
			divNum:       3,
			expectedCost: 0,
			description:  "Player in correct division, no penalty",
		},
		{
			name: "Forced relegation (no hiatus)",
			player: DPPlayer{
				TargetDivision:  2,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
				WasRelegated:    false,
				SeasonsAway:     0,
			},
			divNum:       3,
			expectedCost: 1000, // W_FORCED_REL * 1 division
			description:  "Safe player pushed down 1 division",
		},
		{
			name: "Forced relegation (1 season hiatus)",
			player: DPPlayer{
				TargetDivision:  2,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_SHORT_HIATUS_RETURNING,
				WasRelegated:    false,
				SeasonsAway:     1,
			},
			divNum:       3,
			expectedCost: 500, // W_FORCED_REL / 2 * 1 division
			description:  "Hiatus player (1 season) pushed down, penalty decayed",
		},
		{
			name: "Forced relegation (2 season hiatus)",
			player: DPPlayer{
				TargetDivision:  2,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_LONG_HIATUS_RETURNING,
				WasRelegated:    false,
				SeasonsAway:     2,
			},
			divNum:       3,
			expectedCost: 333.3333, // W_FORCED_REL / 3 * 1 division
			description:  "Hiatus player (2 seasons) pushed down, penalty more decayed",
		},
		{
			name: "Double relegation",
			player: DPPlayer{
				TargetDivision:  3,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_RELEGATED,
				WasRelegated:    true,
				SeasonsAway:     0,
			},
			divNum:       4,
			expectedCost: 100000, // W_DOUBLE_REL * 1 division
			description:  "Relegated player pushed down again - nuclear penalty",
		},
		{
			name: "Bad keep (relegated player kept up)",
			player: DPPlayer{
				TargetDivision:  3,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_RELEGATED,
				WasRelegated:    true,
				SeasonsAway:     0,
			},
			divNum:       2,
			expectedCost: 500, // W_BAD_KEEP * 1 division
			description:  "Relegated player kept in higher division",
		},
		{
			name: "Lucky promotion (safe player bumped up)",
			player: DPPlayer{
				TargetDivision:  3,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
				WasRelegated:    false,
				SeasonsAway:     0,
			},
			divNum:       2,
			expectedCost: 50, // W_LUCKY_PRO * 1 division
			description:  "Safe player promoted to fill gap",
		},
		{
			name: "Multi-division forced relegation",
			player: DPPlayer{
				TargetDivision:  1,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
				WasRelegated:    false,
				SeasonsAway:     0,
			},
			divNum:       3,
			expectedCost: 2000, // W_FORCED_REL * 2 divisions
			description:  "Safe player pushed down 2 divisions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := calculatePlacementPenalty(tt.player, tt.divNum)
			assert.InDelta(t, tt.expectedCost, cost, 0.01, tt.description)
		})
	}
}

// TestCalculateNewPlayerPenalty tests deviation penalties for NEW players
func TestCalculateNewPlayerPenalty(t *testing.T) {
	tests := []struct {
		name         string
		player       DPPlayer
		divNum       int
		expectedCost float64
	}{
		{
			name: "NEW player in correct division",
			player: DPPlayer{
				TargetDivision:  5,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW,
			},
			divNum:       5,
			expectedCost: 0,
		},
		{
			name: "NEW player pushed down 1 division",
			player: DPPlayer{
				TargetDivision:  5,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW,
			},
			divNum:       6,
			expectedCost: 100,
		},
		{
			name: "NEW player promoted up 1 division",
			player: DPPlayer{
				TargetDivision:  5,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW,
			},
			divNum:       4,
			expectedCost: 100,
		},
		{
			name: "NEW player pushed down 2 divisions",
			player: DPPlayer{
				TargetDivision:  5,
				PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW,
			},
			divNum:       7,
			expectedCost: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := calculateNewPlayerPenalty(tt.player, tt.divNum)
			assert.Equal(t, tt.expectedCost, cost)
		})
	}
}

// TestSortPlayersByPriorityScore tests that players are sorted correctly
func TestSortPlayersByPriorityScore(t *testing.T) {
	players := []DPPlayer{
		{UserID: "alice", PriorityScore: 540000},
		{UserID: "bob", PriorityScore: 530000},
		{UserID: "charlie", PriorityScore: 450000},
		{UserID: "dave", PriorityScore: 405000}, // Long hiatus
		{UserID: "eve", PriorityScore: 740000},
	}

	sortPlayersByPriorityScore(players)

	// Verify descending order
	expected := []string{"eve", "alice", "bob", "charlie", "dave"}
	for i, expectedUserID := range expected {
		assert.Equal(t, expectedUserID, players[i].UserID,
			"Player at position %d should be %s", i, expectedUserID)
	}
}

// TestDPSolver_SmallLeague tests DP with a small league scenario
func TestDPSolver_SmallLeague(t *testing.T) {
	// 22 players total - set targets for 2 divisions to test the solver
	players := make([]DPPlayer, 22)
	for i := 0; i < 22; i++ {
		// Set targets: first 11 want div 1, next 11 want div 2
		targetDiv := int32((i / 11) + 1)
		players[i] = DPPlayer{
			UserID:          "player" + string(rune('A'+i)),
			TargetDivision:  targetDiv,
			PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
			PriorityScore:   float64(1000000 - i*1000), // Descending priority
		}
	}

	rm := &RebalanceManager{}
	solution, err := rm.SolveDivisionsDP(nil, players, 15)

	assert.NoError(t, err)
	assert.NotNil(t, solution)

	// With targets properly set, should prefer 2 divisions of 11 each
	// Cost of 1 div size 22: (22-15)^2 * 10 + (22-18)*50 = 490 + 200 = 690
	//   Plus placement penalties for 11 players at wrong division
	// Cost of 2 divs size 11: 2 * ((11-15)^2 * 10 + (12-11)*50) = 2 * (160 + 50) = 420
	assert.Equal(t, 2, solution.NumDivisions, "Should create 2 divisions")

	// Verify all players were assigned
	totalAssigned := 0
	for _, div := range solution.Divisions {
		totalAssigned += len(div.Players)
	}
	assert.Equal(t, 22, totalAssigned)
}

// TestDPSolver_PerfectFit tests DP with perfectly sized league
func TestDPSolver_PerfectFit(t *testing.T) {
	// 45 players = perfect for 3 divisions of 15
	players := make([]DPPlayer, 45)
	for i := 0; i < 45; i++ {
		targetDiv := int32((i / 15) + 1) // 15 per division
		players[i] = DPPlayer{
			UserID:          "player" + string(rune('A'+i)),
			TargetDivision:  targetDiv,
			PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
			PriorityScore:   float64(1000000 - i*1000),
		}
	}

	rm := &RebalanceManager{}
	solution, err := rm.SolveDivisionsDP(nil, players, 15)

	assert.NoError(t, err)
	assert.Equal(t, 3, solution.NumDivisions)

	// Verify all players were assigned
	totalAssigned := 0
	for _, div := range solution.Divisions {
		totalAssigned += len(div.Players)
	}
	assert.Equal(t, 45, totalAssigned)
	assert.Equal(t, 0.0, solution.TotalCost, "Perfect fit should have 0 cost")

	// Verify each division has 15 players
	for _, div := range solution.Divisions {
		assert.Equal(t, 15, len(div.Players))
	}
}

// TestDPSolver_DoubleRelegationPrevention tests nuclear penalty
func TestDPSolver_DoubleRelegationPrevention(t *testing.T) {
	// 32 players: 15 for div 1, 15 for div 2, 2 relegated from div 1
	players := make([]DPPlayer, 32)

	// Div 1 players (high priority)
	for i := 0; i < 15; i++ {
		players[i] = DPPlayer{
			UserID:          "div1_player" + string(rune('A'+i)),
			TargetDivision:  1,
			PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
			PriorityScore:   float64(900000 - i*1000),
			WasRelegated:    false,
		}
	}

	// Div 2 target players (medium priority)
	for i := 0; i < 15; i++ {
		players[15+i] = DPPlayer{
			UserID:          "div2_player" + string(rune('A'+i)),
			TargetDivision:  2,
			PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
			PriorityScore:   float64(700000 - i*1000),
			WasRelegated:    false,
		}
	}

	// 2 relegated players (target div 2, but were already relegated once)
	players[30] = DPPlayer{
		UserID:          "relegated1",
		TargetDivision:  2,
		PlacementStatus: ipc.PlacementStatus_PLACEMENT_RELEGATED,
		PriorityScore:   680000,
		WasRelegated:    true,
	}
	players[31] = DPPlayer{
		UserID:          "relegated2",
		TargetDivision:  2,
		PlacementStatus: ipc.PlacementStatus_PLACEMENT_RELEGATED,
		PriorityScore:   679000,
		WasRelegated:    true,
	}

	rm := &RebalanceManager{}
	solution, err := rm.SolveDivisionsDP(nil, players, 15)

	assert.NoError(t, err)

	// Should create 2 divisions, possibly oversized, to avoid double relegation
	// The relegated players MUST be in division 2, not pushed to division 3
	relegatedInDiv2 := 0
	for _, div := range solution.Divisions {
		if div.DivisionNumber == 2 {
			for _, p := range div.Players {
				if p.UserID == "relegated1" || p.UserID == "relegated2" {
					relegatedInDiv2++
				}
			}
		}
	}

	assert.Equal(t, 2, relegatedInDiv2, "Both relegated players must be in Div 2 to avoid double relegation")
}

// TestDPSolver_HiatusDecay tests that hiatus players can be bumped easier
func TestDPSolver_HiatusDecay(t *testing.T) {
	// 31 players targeting div 1, but only want ~15 in div 1
	// Include hiatus player who should get bumped
	players := make([]DPPlayer, 31)

	// 14 recent div 1 players (high priority, no hiatus)
	for i := 0; i < 14; i++ {
		players[i] = DPPlayer{
			UserID:          "recent" + string(rune('A'+i)),
			TargetDivision:  1,
			PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
			PriorityScore:   float64(900000 - i*1000),
			SeasonsAway:     0,
		}
	}

	// 1 hiatus player (10 seasons away, lower priority)
	players[14] = DPPlayer{
		UserID:          "hiatus_player",
		TargetDivision:  1,
		PlacementStatus: ipc.PlacementStatus_PLACEMENT_LONG_HIATUS_RETURNING,
		PriorityScore:   405000, // Low due to hiatus decay
		SeasonsAway:     10,
	}

	// 16 div 2 players
	for i := 0; i < 16; i++ {
		players[15+i] = DPPlayer{
			UserID:          "div2" + string(rune('A'+i)),
			TargetDivision:  2,
			PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
			PriorityScore:   float64(700000 - i*1000),
			SeasonsAway:     0,
		}
	}

	rm := &RebalanceManager{}
	solution, err := rm.SolveDivisionsDP(nil, players, 15)

	assert.NoError(t, err)

	// Hiatus player should likely be in Div 2, not Div 1
	hiatusPlayerDiv := int32(0)
	for _, div := range solution.Divisions {
		for _, p := range div.Players {
			if p.UserID == "hiatus_player" {
				hiatusPlayerDiv = div.DivisionNumber
			}
		}
	}

	assert.Equal(t, int32(2), hiatusPlayerDiv, "Hiatus player should be bumped to Div 2 due to decay")
}

// TestDPSolver_NewPlayerPlacement tests rating-based NEW player targets
func TestDPSolver_NewPlayerPlacement(t *testing.T) {
	// Mix of returning players and NEW players with various ratings
	players := make([]DPPlayer, 30)

	// 12 returning Div 1 players (avg rating ~1800)
	for i := 0; i < 12; i++ {
		players[i] = DPPlayer{
			UserID:          "div1_vet" + string(rune('A'+i)),
			TargetDivision:  1,
			PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
			PriorityScore:   float64(900000 - i*1000),
			Rating:          1800 + (i * 10), // 1800-1910
		}
	}

	// 12 returning Div 2 players (avg rating ~1400)
	for i := 0; i < 12; i++ {
		players[12+i] = DPPlayer{
			UserID:          "div2_vet" + string(rune('A'+i)),
			TargetDivision:  2,
			PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
			PriorityScore:   float64(700000 - i*1000),
			Rating:          1400 + (i * 10), // 1400-1510
		}
	}

	// NEW players with different ratings and targets
	// High-rated rookie (1750) - targets Div 2 (closest to 1800/1400, excluded from Div 1)
	players[24] = DPPlayer{
		UserID:          "new_high",
		TargetDivision:  2, // Should target Div 2
		PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW,
		PriorityScore:   105000 + 1750, // Low base + rating
		Rating:          1750,
	}

	// Medium-rated rookie (1450) - targets Div 2
	players[25] = DPPlayer{
		UserID:          "new_medium",
		TargetDivision:  2,
		PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW,
		PriorityScore:   105000 + 1450,
		Rating:          1450,
	}

	// Low-rated rookie (1200) - targets Div 2 (bottom division)
	players[26] = DPPlayer{
		UserID:          "new_low",
		TargetDivision:  2,
		PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW,
		PriorityScore:   105000 + 1200,
		Rating:          1200,
	}

	// Unrated rookie (0) - targets Div 2 (bottom division)
	players[27] = DPPlayer{
		UserID:          "new_unrated",
		TargetDivision:  2,
		PlacementStatus: ipc.PlacementStatus_PLACEMENT_NEW,
		PriorityScore:   105000,
		Rating:          0,
	}

	// Fill remaining spots
	players[28] = DPPlayer{
		UserID:          "div2_vet_extra1",
		TargetDivision:  2,
		PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
		PriorityScore:   650000,
		Rating:          1420,
	}
	players[29] = DPPlayer{
		UserID:          "div2_vet_extra2",
		TargetDivision:  2,
		PlacementStatus: ipc.PlacementStatus_PLACEMENT_STAYED,
		PriorityScore:   640000,
		Rating:          1410,
	}

	rm := &RebalanceManager{}
	solution, err := rm.SolveDivisionsDP(nil, players, 15)

	assert.NoError(t, err)
	assert.Equal(t, 2, solution.NumDivisions)

	// Verify no NEW players made it into Div 1
	newInDiv1 := 0
	for _, div := range solution.Divisions {
		if div.DivisionNumber == 1 {
			for _, p := range div.Players {
				if p.PlacementStatus == ipc.PlacementStatus_PLACEMENT_NEW {
					newInDiv1++
				}
			}
		}
	}

	assert.Equal(t, 0, newInDiv1, "No NEW players should be in Division 1 (kept exclusive)")
}
