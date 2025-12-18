package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/league"
)

// TestDPRebalance tests the DP rebalancing algorithm with real data from an existing league
func TestDPRebalance(ctx context.Context, leagueSlugOrUUID string, seasonNumber int32) error {
	log.Info().
		Str("league", leagueSlugOrUUID).
		Int32("seasonNumber", seasonNumber).
		Msg("Testing DP rebalancing with real data")

	// Initialize stores
	allStores, err := initStores(ctx)
	if err != nil {
		return err
	}

	// Get league
	leagueUUID, err := getLeagueUUID(ctx, allStores, leagueSlugOrUUID)
	if err != nil {
		return err
	}

	dbLeague, err := allStores.LeagueStore.GetLeagueByUUID(ctx, leagueUUID)
	if err != nil {
		return fmt.Errorf("failed to get league: %w", err)
	}

	// Get season
	season, err := allStores.LeagueStore.GetSeasonByLeagueAndNumber(ctx, leagueUUID, seasonNumber)
	if err != nil {
		return fmt.Errorf("failed to get season %d: %w", seasonNumber, err)
	}

	// Get previous season (if exists)
	previousSeasonID := uuid.Nil
	if seasonNumber > 1 {
		prevSeason, err := allStores.LeagueStore.GetSeasonByLeagueAndNumber(ctx, leagueUUID, seasonNumber-1)
		if err == nil {
			previousSeasonID = prevSeason.Uuid
		}
	}

	// Parse league settings
	leagueSettings, err := parseLeagueSettings(dbLeague.Settings)
	if err != nil {
		return fmt.Errorf("failed to parse league settings: %w", err)
	}

	idealDivisionSize := int32(15) // Default
	if leagueSettings.IdealDivisionSize > 0 {
		idealDivisionSize = leagueSettings.IdealDivisionSize
	}

	log.Info().
		Str("seasonID", season.Uuid.String()).
		Str("previousSeasonID", previousSeasonID.String()).
		Int32("idealDivisionSize", idealDivisionSize).
		Msg("Loaded season data")

	// Get registrations for this season
	registrations, err := allStores.LeagueStore.GetSeasonRegistrations(ctx, season.Uuid)
	if err != nil {
		return fmt.Errorf("failed to get registrations: %w", err)
	}

	log.Info().Int("registrationCount", len(registrations)).Msg("Loaded registrations")

	// Categorize players (NEW vs RETURNING)
	regMgr := league.NewRegistrationManager(allStores.LeagueStore, &league.RealClock{}, allStores)
	categorizedPlayers, err := regMgr.CategorizeRegistrations(ctx, leagueUUID, season.Uuid, registrations)
	if err != nil {
		return fmt.Errorf("failed to categorize players: %w", err)
	}

	newCount := 0
	returningCount := 0
	for _, cp := range categorizedPlayers {
		if cp.Category == league.PlayerCategoryNew {
			newCount++
		} else {
			returningCount++
		}
	}

	log.Info().
		Int("newPlayers", newCount).
		Int("returningPlayers", returningCount).
		Msg("Categorized players")

	// Run the rebalancing algorithm (READ-ONLY - no database writes)
	rebalanceMgr := league.NewRebalanceManager(allStores)

	// Compute placement status and virtual divisions WITHOUT writing to database
	playersWithVirtualDivs, err := rebalanceMgr.ComputePlayersWithPlacementStatus(ctx, leagueUUID, previousSeasonID, season.Uuid, seasonNumber, categorizedPlayers)
	if err != nil {
		return fmt.Errorf("failed to compute player data: %w", err)
	}

	// Calculate number of virtual divisions
	numVirtualDivs := int32(0)
	for _, p := range playersWithVirtualDivs {
		if p.VirtualDivision > numVirtualDivs {
			numVirtualDivs = p.VirtualDivision
		}
	}

	log.Info().Int32("numVirtualDivs", numVirtualDivs).Msg("Assigned virtual divisions")

	// Step 3: Calculate priority scores
	playersWithPriority := rebalanceMgr.CalculatePriorityScores(playersWithVirtualDivs, numVirtualDivs, seasonNumber)

	// Step 4: Convert to DP format
	dpPlayers := league.ConvertToDPPlayers(playersWithPriority)

	// Step 5: Run DP solver
	log.Info().
		Int("totalPlayers", len(dpPlayers)).
		Int32("idealDivisionSize", idealDivisionSize).
		Msg("Starting DP rebalancing algorithm")
	dpSolution, err := rebalanceMgr.SolveDivisionsDP(ctx, dpPlayers, idealDivisionSize)
	if err != nil {
		return fmt.Errorf("DP solver failed: %w", err)
	}

	log.Info().
		Int("numDivisions", dpSolution.NumDivisions).
		Float64("totalCost", dpSolution.TotalCost).
		Msg("DP solver completed")

	// Print results
	printDPResults(dpSolution, playersWithPriority)

	// Compare with old sequential method
	compareWithSequential(playersWithPriority, dpSolution, int(idealDivisionSize))

	log.Info().
		Int("numDivisions", dpSolution.NumDivisions).
		Float64("totalCost", dpSolution.TotalCost).
		Str("league", leagueSlugOrUUID).
		Int32("season", seasonNumber).
		Msg("DP rebalancing test completed successfully")

	return nil
}

func printDPResults(solution *league.DPSolution, allPlayers []league.PlayerWithPriority) {
	fmt.Println("\n========================================")
	fmt.Println("DP REBALANCING RESULTS")
	fmt.Println("========================================")
	fmt.Printf("Total Divisions: %d\n", solution.NumDivisions)
	fmt.Printf("Total Cost: %.2f\n", solution.TotalCost)

	// Calculate total players assigned
	totalPlayersAssigned := 0
	for _, div := range solution.Divisions {
		totalPlayersAssigned += len(div.Players)
	}
	fmt.Printf("Players Assigned: %d\n\n", totalPlayersAssigned)

	for _, div := range solution.Divisions {
		fmt.Printf("DIVISION %d (%d players, cost: %.2f)\n", div.DivisionNumber, len(div.Players), div.Cost)
		fmt.Println("├─ Player Breakdown:")

		// Count by status
		statusCounts := make(map[string]int)
		newCount := 0
		for _, p := range div.Players {
			statusCounts[p.PlacementStatus.String()]++
			if p.PlacementStatus == 3 { // PLACEMENT_NEW
				newCount++
			}
		}

		for status, count := range statusCounts {
			fmt.Printf("│  %s: %d\n", status, count)
		}

		// Show size penalty
		sizeDiff := len(div.Players) - 15
		fmt.Printf("│  Size deviation: %+d\n", sizeDiff)

		// Show all players
		fmt.Println("├─ All Players:")
		for _, p := range div.Players {
			fmt.Printf("│  %s (target: div %d, priority: %.0f)\n", p.Username, p.TargetDivision, p.PriorityScore)
		}
		fmt.Println("└─")
		fmt.Println()
	}
}

func compareWithSequential(players []league.PlayerWithPriority, dpSolution *league.DPSolution, idealSize int) {
	fmt.Println("\n========================================")
	fmt.Println("COMPARISON: DP vs SEQUENTIAL")
	fmt.Println("========================================")

	// Simulate old sequential assignment
	numDivsOld := len(players) / idealSize
	if len(players)%idealSize != 0 {
		numDivsOld++
	}

	sequentialDivs := make([][]league.PlayerWithPriority, numDivsOld)
	for i, p := range players {
		divIdx := i / idealSize
		if divIdx >= numDivsOld {
			divIdx = numDivsOld - 1
		}
		sequentialDivs[divIdx] = append(sequentialDivs[divIdx], p)
	}

	fmt.Printf("DP Solution: %d divisions\n", dpSolution.NumDivisions)
	fmt.Printf("Sequential: %d divisions\n\n", numDivsOld)

	fmt.Println("Division Sizes:")
	fmt.Printf("%-15s %-15s\n", "DP", "Sequential")
	fmt.Println("──────────────────────────────")

	maxDivs := max(dpSolution.NumDivisions, numDivsOld)
	for i := 0; i < maxDivs; i++ {
		dpSize := 0
		if i < len(dpSolution.Divisions) {
			dpSize = len(dpSolution.Divisions[i].Players)
		}

		seqSize := 0
		if i < len(sequentialDivs) {
			seqSize = len(sequentialDivs[i])
		}

		fmt.Printf("Div %d: %-6d      %-6d\n", i+1, dpSize, seqSize)
	}

	// Check for violations (double relegations, etc.)
	fmt.Println("\n========================================")
	fmt.Println("VIOLATION CHECKS")
	fmt.Println("========================================")

	doubleRelegations := 0
	for _, div := range dpSolution.Divisions {
		for _, p := range div.Players {
			if p.WasRelegated && int32(div.DivisionNumber) > p.TargetDivision {
				doubleRelegations++
				fmt.Printf("⚠️  Double relegation: %s (target: div %d, placed: div %d)\n",
					p.Username, p.TargetDivision, div.DivisionNumber)
			}
		}
	}

	if doubleRelegations == 0 {
		fmt.Println("✓ No double relegations detected")
	}

	// Check NEW players in Division 1
	newInDiv1 := 0
	if len(dpSolution.Divisions) > 0 {
		for _, p := range dpSolution.Divisions[0].Players {
			if p.PlacementStatus == 3 { // PLACEMENT_NEW
				newInDiv1++
				fmt.Printf("⚠️  NEW player in Division 1: %s (rating: %d)\n", p.Username, p.Rating)
			}
		}
	}

	if newInDiv1 == 0 {
		fmt.Println("✓ No NEW players in Division 1 (exclusive)")
	}
}

// Helper to export results to JSON
func exportResultsToJSON(solution *league.DPSolution, filename string) error {
	data, err := json.MarshalIndent(solution, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
