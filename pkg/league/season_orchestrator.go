package league

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/woogles-io/liwords/pkg/stores"
)

// SeasonOrchestrator coordinates all phases of season setup
type SeasonOrchestrator struct {
	stores          *stores.Stores
	registrationMgr *RegistrationManager
	placementMgr    *PlacementManager
}

// NewSeasonOrchestrator creates a new season orchestrator
func NewSeasonOrchestrator(allStores *stores.Stores) *SeasonOrchestrator {
	return &SeasonOrchestrator{
		stores:          allStores,
		registrationMgr: NewRegistrationManager(allStores.LeagueStore, RealClock{}, allStores),
		placementMgr:    NewPlacementManager(allStores.LeagueStore),
	}
}

// DivisionPreparationResult tracks the outcome of preparing divisions for a new season
type DivisionPreparationResult struct {
	// Summary counts
	TotalRegistrations int
	NewPlayers         int
	ReturningPlayers   int

	// Placement results
	PlacedPlayers int

	// Division counts
	RegularDivisionsUsed int

	// Detailed results
	PlacementResult *PlacementResult
}

// PrepareNextSeasonDivisions orchestrates the complete process of preparing divisions
// for a new season. This function should be called on Day 20 at midnight when the
// current season is force-closed.
//
// Process:
// 1. Get all registrations for the new season
// 2. Categorize players (NEW vs RETURNING)
// 3. Rebalance ALL players (new and returning together)
//    - Updates placement statuses
//    - Assigns virtual divisions
//    - Calculates priority scores (NEW players have lowest priority)
//    - Creates real divisions (round(count/idealDivisionSize))
//    - Assigns players by priority
//
// Note: NEW players are placed naturally via rebalancing with lowest priority.
func (so *SeasonOrchestrator) PrepareNextSeasonDivisions(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	newSeasonID uuid.UUID,
	newSeasonNumber int32,
	idealDivisionSize int32,
) (*DivisionPreparationResult, error) {
	result := &DivisionPreparationResult{}

	// Step 1: Get all registrations for the new season
	registrations, err := so.registrationMgr.GetSeasonRegistrations(ctx, newSeasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season registrations: %w", err)
	}
	result.TotalRegistrations = len(registrations)

	if len(registrations) == 0 {
		// No registrations, nothing to do
		return result, nil
	}

	// Step 2: Categorize players (NEW vs RETURNING)
	categorized, err := so.registrationMgr.CategorizeRegistrations(ctx, leagueID, newSeasonID, registrations)
	if err != nil {
		return nil, fmt.Errorf("failed to categorize registrations: %w", err)
	}

	// Count categories
	for _, cp := range categorized {
		if cp.Category == PlayerCategoryNew {
			result.NewPlayers++
		} else {
			result.ReturningPlayers++
		}
	}

	// Step 3: Rebalance ALL players (new and returning together)
	// New players have lowest priority and will be naturally placed in lower divisions
	if len(categorized) > 0 {
		rebalanceMgr := NewRebalanceManager(so.stores)
		rebalanceResult, err := rebalanceMgr.RebalanceDivisions(
			ctx,
			leagueID,
			previousSeasonID,
			newSeasonID,
			newSeasonNumber,
			categorized,
			idealDivisionSize,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to rebalance divisions: %w", err)
		}

		result.RegularDivisionsUsed = rebalanceResult.DivisionsCreated
		result.PlacedPlayers = len(categorized)
	}

	return result, nil
}
