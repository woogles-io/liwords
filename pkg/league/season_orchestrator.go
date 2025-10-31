package league

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"github.com/woogles-io/liwords/pkg/stores/league"
)

// SeasonOrchestrator coordinates all phases of season setup
type SeasonOrchestrator struct {
	store              league.Store
	registrationMgr    *RegistrationManager
	placementMgr       *PlacementManager
	graduationMgr      *GraduationManager
	rookieMgr          *RookieManager
}

// NewSeasonOrchestrator creates a new season orchestrator
func NewSeasonOrchestrator(store league.Store) *SeasonOrchestrator {
	return &SeasonOrchestrator{
		store:              store,
		registrationMgr:    NewRegistrationManager(store),
		placementMgr:       NewPlacementManager(store),
		graduationMgr:      NewGraduationManager(store),
		rookieMgr:          NewRookieManager(store),
	}
}

// DivisionPreparationResult tracks the outcome of preparing divisions for a new season
type DivisionPreparationResult struct {
	// Summary counts
	TotalRegistrations   int
	NewPlayers           int
	ReturningPlayers     int

	// Placement results
	PlacedReturning      int
	GraduatedRookies     int
	PlacedInRookieDivs   int
	PlacedInRegularDivs  int

	// Division counts
	RegularDivisionsUsed int
	RookieDivisionsCreated int

	// Detailed results
	PlacementResult  *PlacementResult
	GraduationResult *GraduationResult
	RookieResult     *RookiePlacementResult
}

// PrepareNextSeasonDivisions orchestrates the complete process of preparing divisions
// for a new season. This function should be called on Day 20 at midnight when the
// current season is force-closed.
//
// Process:
// 1. Get all registrations for the new season
// 2. Categorize players (NEW vs RETURNING)
// 3. Separate new rookies (≥10 → rookie divisions, <10 → include in rebalancing)
// 4. Rebalance all regular division players (includes <10 new rookies if applicable)
//    - Updates placement statuses
//    - Assigns virtual divisions
//    - Calculates priority scores
//    - Creates real divisions (round(count/15))
//    - Assigns players by priority
// 5. Create rookie divisions for ≥10 new rookies
//
// Note: This function NOW creates regular divisions automatically based on player count.
func (so *SeasonOrchestrator) PrepareNextSeasonDivisions(
	ctx context.Context,
	leagueID uuid.UUID,
	previousSeasonID uuid.UUID,
	newSeasonID uuid.UUID,
	newSeasonNumber int32,
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

	// Step 3: Separate new rookies based on count
	newRookies := []CategorizedPlayer{}
	regularPlayers := []CategorizedPlayer{}

	for _, cp := range categorized {
		if cp.Category == PlayerCategoryNew {
			newRookies = append(newRookies, cp)
		} else {
			regularPlayers = append(regularPlayers, cp)
		}
	}

	// Determine if we create rookie divisions or include rookies in rebalancing
	playersForRebalancing := regularPlayers
	if len(newRookies) < MinPlayersForRookieDivision {
		// Include new rookies in regular division rebalancing
		playersForRebalancing = append(playersForRebalancing, newRookies...)
	}

	// Step 4: Rebalance divisions (creates divisions + assigns players)
	if len(playersForRebalancing) > 0 {
		rebalanceMgr := NewRebalanceManager(so.store)
		rebalanceResult, err := rebalanceMgr.RebalanceDivisions(
			ctx,
			leagueID,
			previousSeasonID,
			newSeasonID,
			newSeasonNumber,
			playersForRebalancing,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to rebalance divisions: %w", err)
		}

		result.RegularDivisionsUsed = rebalanceResult.DivisionsCreated
		result.PlacedReturning = len(playersForRebalancing) - len(newRookies)
		if len(newRookies) < MinPlayersForRookieDivision {
			result.PlacedInRegularDivs = len(newRookies)
		}
	}

	// Step 5: Create rookie divisions for ≥10 new rookies
	if len(newRookies) >= MinPlayersForRookieDivision {
		// Sort rookies by rating (highest first) before creating divisions
		sort.Slice(newRookies, func(i, j int) bool {
			return newRookies[i].Rating > newRookies[j].Rating
		})

		rookieResult, err := so.rookieMgr.createRookieDivisions(ctx, newSeasonID, newRookies)
		if err != nil {
			return nil, fmt.Errorf("failed to create rookie divisions: %w", err)
		}
		result.RookieResult = rookieResult
		result.PlacedInRookieDivs = len(rookieResult.PlacedInRookieDivisions)
		result.RookieDivisionsCreated = len(rookieResult.CreatedDivisions)
	}

	return result, nil
}
