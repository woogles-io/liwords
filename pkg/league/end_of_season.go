package league

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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

// MarkSeasonOutcomes calculates final standings and updates previous_division_rank
// for all players in a season. This should be called when a season completes.
//
// For each player in the season:
//   - Sets previous_division_rank based on their final rank
//   - Does NOT modify placement_status (preserves entry status like NEW, RETURNING)
//
// The outcome (PROMOTED/RELEGATED/STAYED) is stored in the standings table's result
// field and should be read from there when determining next season placement.
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

	// For each division, update previous_division_rank in league_registrations
	for _, division := range divisions {
		err := em.updateRegistrationRanks(ctx, seasonID, division.Uuid)
		if err != nil {
			return fmt.Errorf("failed to update registration ranks for division %d: %w",
				division.DivisionNumber, err)
		}
	}

	return nil
}

// updateRegistrationRanks updates previous_division_rank in league_registrations
// based on the standings. Does NOT update placement_status to preserve entry status.
func (em *EndOfSeasonManager) updateRegistrationRanks(
	ctx context.Context,
	seasonID uuid.UUID,
	divisionID uuid.UUID,
) error {
	// Get standings for this division
	standings, err := em.store.GetStandings(ctx, divisionID)
	if err != nil {
		return fmt.Errorf("failed to get standings: %w", err)
	}

	// Sort standings to determine rank (rank is calculated from position, not stored)
	SortStandingsByRank(standings)

	// Update each player's registration with their rank only
	for i, standing := range standings {
		// Rank is position in sorted array (1-based)
		rank := int32(i + 1)

		// Update the registration's rank only (preserve placement_status)
		err := em.store.UpdatePreviousDivisionRank(ctx, models.UpdatePreviousDivisionRankParams{
			UserID:               standing.UserID,
			PreviousDivisionRank: pgtype.Int4{Int32: rank, Valid: true},
			SeasonID:             seasonID,
		})
		if err != nil {
			return fmt.Errorf("failed to update rank for user %d: %w",
				standing.UserID, err)
		}
	}

	return nil
}
