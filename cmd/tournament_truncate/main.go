package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/tournament"
	tl "github.com/woogles-io/liwords/pkg/tournament"
)

// Truncates tournament rounds by removing the last N rounds of data.
// This tool:
// 1. Empties out the last N rows in the "matrix" (keeps them as [])
// 2. Removes pairings from pairingMap that were in those rows
// 3. Removes standings for those rounds
// 4. Updates currentRound to reflect the truncation

func main() {
	if len(os.Args) < 4 {
		log.Fatal("Usage: tournament_truncate <tournament-slug> <division-name> <num-rounds-to-truncate>")
	}

	slug := os.Args[1]
	divisionName := os.Args[2]
	numRoundsStr := os.Args[3]

	numRounds, err := strconv.ParseInt(numRoundsStr, 10, 32)
	if err != nil {
		log.Fatal("num-rounds-to-truncate must be a number")
	}
	if numRounds < 1 {
		log.Fatal("num-rounds-to-truncate must be positive")
	}

	// Initialize config
	cfg := &config.Config{}
	cfg.Load(nil)

	// Initialize tournament store
	// We pass nil for game store since we don't need it for this operation
	tournamentStore, err := tournament.NewDBStore(cfg, nil)
	if err != nil {
		log.Fatalf("Failed to initialize tournament store: %v", err)
	}

	ctx := context.Background()

	// Get tournament by slug
	tourney, err := tournamentStore.GetBySlug(ctx, slug)
	if err != nil {
		log.Fatalf("Failed to get tournament %s: %v", slug, err)
	}

	fmt.Printf("Found tournament: %s (UUID: %s)\n", tourney.Name, tourney.UUID)

	// Get the division
	division, ok := tourney.Divisions[divisionName]
	if !ok {
		log.Fatalf("Division %s not found in tournament. Available divisions:", divisionName)
	}

	// Cast to ClassicDivision
	classicDiv, ok := division.DivisionManager.(*tl.ClassicDivision)
	if !ok {
		log.Fatalf("Division %s is not a ClassicDivision", divisionName)
	}

	fmt.Printf("Division: %s\n", classicDiv.DivisionName)
	fmt.Printf("Current round: %d\n", classicDiv.CurrentRound)
	fmt.Printf("Total rounds in matrix: %d\n", len(classicDiv.Matrix))

	if int32(numRounds) > classicDiv.CurrentRound {
		log.Fatalf("Cannot truncate %d rounds when current round is only %d", numRounds, classicDiv.CurrentRound)
	}

	// Calculate which rounds to remove
	totalRounds := len(classicDiv.Matrix)
	roundsToRemoveStart := totalRounds - int(numRounds)

	if roundsToRemoveStart < 0 {
		log.Fatalf("Cannot truncate %d rounds when matrix only has %d rounds", numRounds, totalRounds)
	}

	fmt.Printf("\nTruncating last %d rounds (rounds %d-%d, 0-indexed)...\n",
		numRounds, roundsToRemoveStart, totalRounds-1)

	// Collect all pairing keys that need to be removed
	pairingKeysToRemove := make(map[string]bool)
	numPlayers := len(classicDiv.Players.Persons)
	for i := roundsToRemoveStart; i < totalRounds; i++ {
		for _, key := range classicDiv.Matrix[i] {
			if key != "" {
				pairingKeysToRemove[key] = true
			}
		}
		// Empty out the row but maintain structure (keep length = numPlayers)
		classicDiv.Matrix[i] = make([]string, numPlayers)
	}

	// Remove pairings from pairingMap
	fmt.Printf("Removing %d pairings from pairingMap...\n", len(pairingKeysToRemove))
	for key := range pairingKeysToRemove {
		delete(classicDiv.PairingMap, key)
	}

	// Remove standings for the truncated rounds
	// Standings are 0-indexed by round
	fmt.Printf("Removing standings for rounds %d-%d...\n", roundsToRemoveStart, totalRounds-1)
	for i := roundsToRemoveStart; i < totalRounds; i++ {
		delete(classicDiv.Standings, int32(i))
	}

	// Update currentRound
	// If current round was 15 (0-indexed) and we truncate 2 rounds,
	// new current round should be 13 (15 - 2)
	oldCurrentRound := classicDiv.CurrentRound
	classicDiv.CurrentRound = classicDiv.CurrentRound - int32(numRounds)

	fmt.Printf("Updated currentRound: %d -> %d\n", oldCurrentRound, classicDiv.CurrentRound)

	// Save the tournament back
	fmt.Printf("\nSaving tournament...\n")
	tourney.Lock()
	err = tournamentStore.Set(ctx, tourney)
	tourney.Unlock()
	if err != nil {
		log.Fatalf("Failed to save tournament: %v", err)
	}

	fmt.Printf("\n✓ Successfully truncated %d rounds from division %s\n", numRounds, divisionName)
	fmt.Printf("  - Emptied %d rows in matrix\n", numRounds)
	fmt.Printf("  - Removed %d pairings\n", len(pairingKeysToRemove))
	fmt.Printf("  - Removed %d rounds of standings\n", numRounds)
	fmt.Printf("  - Updated current round: %d -> %d\n", oldCurrentRound, classicDiv.CurrentRound)
}
