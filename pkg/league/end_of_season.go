package league

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// EndOfSeasonManager handles end-of-season processing to mark player outcomes
type EndOfSeasonManager struct {
	store            league.Store
	standingsManager *StandingsManager
}

// NewEndOfSeasonManager creates a new end-of-season manager
func NewEndOfSeasonManager(store league.Store) *EndOfSeasonManager {
	return &EndOfSeasonManager{
		store:            store,
		standingsManager: NewStandingsManager(store),
	}
}

// MarkSeasonOutcomes calculates final standings and updates placement_status
// for all players in a season. This should be called when a season completes.
//
// For each player in the season:
//   - Sets placement_status based on their outcome (PROMOTED/RELEGATED/STAYED)
//   - Sets previous_division_rank based on their final rank
//
// This data is then used when placing players into the next season.
func (em *EndOfSeasonManager) MarkSeasonOutcomes(
	ctx context.Context,
	seasonID uuid.UUID,
) error {
	// First, calculate and save standings to league_standings table
	// This determines each player's outcome (PROMOTED/RELEGATED/STAYED)
	err := em.standingsManager.CalculateAndSaveStandings(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to calculate standings: %w", err)
	}

	// Get all divisions for this season
	divisions, err := em.store.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get divisions: %w", err)
	}

	// For each division, update placement_status in league_registrations
	for _, division := range divisions {
		err := em.updateRegistrationOutcomes(ctx, seasonID, division.Uuid)
		if err != nil {
			return fmt.Errorf("failed to update registration outcomes for division %d: %w",
				division.DivisionNumber, err)
		}
	}

	return nil
}

// updateRegistrationOutcomes updates placement_status in league_registrations
// based on the outcomes calculated in league_standings
func (em *EndOfSeasonManager) updateRegistrationOutcomes(
	ctx context.Context,
	seasonID uuid.UUID,
	divisionID uuid.UUID,
) error {
	// Get standings for this division
	standings, err := em.store.GetStandings(ctx, divisionID)
	if err != nil {
		return fmt.Errorf("failed to get standings: %w", err)
	}

	// Update each player's registration with their outcome
	for _, standing := range standings {
		// Map StandingResult to PlacementStatus
		placementStatus := pgtype.Int4{}
		if standing.Result.Valid {
			// Convert StandingResult to PlacementStatus
			var ps ipc.PlacementStatus
			switch ipc.StandingResult(standing.Result.Int32) {
			case ipc.StandingResult_RESULT_PROMOTED:
				ps = ipc.PlacementStatus_PLACEMENT_PROMOTED
			case ipc.StandingResult_RESULT_RELEGATED:
				ps = ipc.PlacementStatus_PLACEMENT_RELEGATED
			case ipc.StandingResult_RESULT_STAYED:
				ps = ipc.PlacementStatus_PLACEMENT_STAYED
			default:
				ps = ipc.PlacementStatus_PLACEMENT_STAYED
			}
			placementStatus = pgtype.Int4{Int32: int32(ps), Valid: true}
		} else {
			// Default to STAYED if not set
			placementStatus = pgtype.Int4{Int32: int32(ipc.PlacementStatus_PLACEMENT_STAYED), Valid: true}
		}

		// Get the rank (1-based)
		rank := int32(0)
		if standing.Rank.Valid {
			rank = standing.Rank.Int32
		}

		// Update the registration
		err := em.store.UpdatePlacementStatus(ctx, models.UpdatePlacementStatusParams{
			UserID:               standing.UserID,
			PlacementStatus:      placementStatus,
			PreviousDivisionRank: pgtype.Int4{Int32: rank, Valid: true},
			SeasonID:             seasonID,
		})
		if err != nil {
			return fmt.Errorf("failed to update placement status for user %s: %w",
				standing.UserID, err)
		}
	}

	return nil
}
