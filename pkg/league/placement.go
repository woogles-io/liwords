package league

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// PlacementManager handles initial division placement for a season
type PlacementManager struct {
	store league.Store
}

// NewPlacementManager creates a new placement manager
func NewPlacementManager(store league.Store) *PlacementManager {
	return &PlacementManager{
		store: store,
	}
}

// PlacementResult tracks the outcome of placing players into divisions
type PlacementResult struct {
	// Successfully placed returning players (assigned to their previous divisions)
	PlacedReturning []PlacedPlayer

	// New players (rookies) that need division assignment
	NeedingRookiePlacement []CategorizedPlayer

	// Returning players placed in the lowest division because their previous division doesn't exist
	PlacedInLowestDivision []PlacedPlayer
}

// PlacedPlayer represents a player that has been assigned to a division
type PlacedPlayer struct {
	CategorizedPlayer
	DivisionID   uuid.UUID
	DivisionName string
}

// findLowestDivision finds the division with the highest division number (lowest rank)
func findLowestDivision(divisions []models.LeagueDivision) (uuid.UUID, string, error) {
	if len(divisions) == 0 {
		return uuid.UUID{}, "", fmt.Errorf("no divisions available")
	}

	lowestDiv := divisions[0]
	for _, div := range divisions {
		if div.DivisionNumber > lowestDiv.DivisionNumber {
			lowestDiv = div
		}
	}

	divName := fmt.Sprintf("Division %d", lowestDiv.DivisionNumber)
	if lowestDiv.DivisionName.Valid {
		divName = lowestDiv.DivisionName.String
	}

	return lowestDiv.Uuid, divName, nil
}

// calculateAndSetPlacementStatus determines placement_status based on hiatus length
// and sets it in the database
func (pm *PlacementManager) calculateAndSetPlacementStatus(
	ctx context.Context,
	userID string,
	seasonID uuid.UUID,
	currentSeasonNumber int32,
	lastPlayedSeasonNumber int32,
	previousPlacementStatus string,
	previousRank int32,
) error {
	// Calculate seasons away: current_season_number - last_played_season_number - 1
	seasonsAway := currentSeasonNumber - lastPlayedSeasonNumber - 1

	var newPlacementStatus string

	if seasonsAway == 0 {
		// Consecutive season - use existing placement_status from previous season
		if previousPlacementStatus != "" {
			newPlacementStatus = previousPlacementStatus
		} else {
			// No previous status, default to STAYED
			newPlacementStatus = "STAYED"
		}
	} else if seasonsAway >= 1 && seasonsAway <= 3 {
		// Short hiatus (1-3 seasons away)
		newPlacementStatus = "SHORT_HIATUS_RETURNING"
	} else {
		// Long hiatus (4+ seasons away)
		newPlacementStatus = "LONG_HIATUS_RETURNING"
	}

	// Update placement status in database
	return pm.store.UpdatePlacementStatusWithSeasonsAway(ctx, models.UpdatePlacementStatusWithSeasonsAwayParams{
		UserID:               userID,
		PlacementStatus:      pgtype.Text{String: newPlacementStatus, Valid: true},
		PreviousDivisionRank: pgtype.Int4{Int32: previousRank, Valid: previousRank != 0},
		SeasonsAway:          pgtype.Int4{Int32: seasonsAway, Valid: true},
		SeasonID:             seasonID,
	})
}

// PlaceReturningPlayers attempts to place returning players back into their
// previous divisions. Returns a result indicating which players were placed
// and which still need placement.
//
// This function also detects hiatus and sets placement_status:
//   - 0 seasons away (consecutive): Keeps existing placement_status from previous season
//   - 1-3 seasons away: Sets placement_status = SHORT_HIATUS_RETURNING
//   - 4+ seasons away: Sets placement_status = LONG_HIATUS_RETURNING
func (pm *PlacementManager) PlaceReturningPlayers(
	ctx context.Context,
	leagueID uuid.UUID,
	currentSeasonID uuid.UUID,
	currentSeasonNumber int32,
	categorizedPlayers []CategorizedPlayer,
) (*PlacementResult, error) {
	result := &PlacementResult{
		PlacedReturning:        []PlacedPlayer{},
		NeedingRookiePlacement: []CategorizedPlayer{},
		PlacedInLowestDivision: []PlacedPlayer{},
	}

	// Get all divisions for the current season
	divisions, err := pm.store.GetDivisionsBySeason(ctx, currentSeasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get divisions for season: %w", err)
	}

	// Create a map of division IDs for quick lookup
	divisionMap := make(map[uuid.UUID]models.LeagueDivision)
	for _, div := range divisions {
		divisionMap[div.Uuid] = div
	}

	// Process each player
	for _, player := range categorizedPlayers {
		if player.Category == PlayerCategoryNew {
			// New players go into rookie placement queue
			result.NeedingRookiePlacement = append(result.NeedingRookiePlacement, player)
			continue
		}

		// For returning players, find their previous division
		history, err := pm.store.GetPlayerSeasonHistory(ctx, models.GetPlayerSeasonHistoryParams{
			UserID:   player.Registration.UserID,
			LeagueID: leagueID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get player history for %s: %w", player.Registration.UserID, err)
		}

		// Find the most recent completed season (not the current one)
		var previousDivisionID uuid.UUID
		var lastPlayedSeasonNumber int32
		var placementStatus string
		var previousRank int32
		found := false
		for _, season := range history {
			if season.SeasonID != currentSeasonID && season.DivisionID.Valid {
				previousDivisionID = season.DivisionID.Bytes
				lastPlayedSeasonNumber = season.SeasonNumber
				// Store the previous placement status and rank for hiatus calculation
				if season.PlacementStatus.Valid {
					placementStatus = season.PlacementStatus.String
				}
				if season.PreviousDivisionRank.Valid {
					previousRank = season.PreviousDivisionRank.Int32
				}
				found = true
				break // Take the first one (most recent)
			}
		}

		if !found {
			// Returning player but no previous division found - place in lowest division
			lowestDivID, lowestDivName, err := findLowestDivision(divisions)
			if err != nil {
				return nil, fmt.Errorf("no divisions available to place player %s: %w", player.Registration.UserID, err)
			}

			err = pm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
				UserID:      player.Registration.UserID,
				SeasonID:    currentSeasonID,
				DivisionID:  pgtype.UUID{Bytes: lowestDivID, Valid: true},
				FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to assign player %s to lowest division: %w", player.Registration.UserID, err)
			}

			result.PlacedInLowestDivision = append(result.PlacedInLowestDivision, PlacedPlayer{
				CategorizedPlayer: player,
				DivisionID:        lowestDivID,
				DivisionName:      lowestDivName,
			})
			continue
		}

		// Check if that division exists in the current season
		// Divisions are identified by division_number, not UUID
		// We need to find a division with the same division_number
		var targetDivisionID uuid.UUID
		var targetDivisionName string
		divisionFound := false

		// First, get the previous division's number
		previousDiv, err := pm.store.GetDivision(ctx, previousDivisionID)
		if err != nil {
			// Previous division doesn't exist anymore - place in lowest division
			lowestDivID, lowestDivName, err := findLowestDivision(divisions)
			if err != nil {
				return nil, fmt.Errorf("no divisions available to place player %s: %w", player.Registration.UserID, err)
			}

			err = pm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
				UserID:      player.Registration.UserID,
				SeasonID:    currentSeasonID,
				DivisionID:  pgtype.UUID{Bytes: lowestDivID, Valid: true},
				FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to assign player %s to lowest division: %w", player.Registration.UserID, err)
			}

			result.PlacedInLowestDivision = append(result.PlacedInLowestDivision, PlacedPlayer{
				CategorizedPlayer: player,
				DivisionID:        lowestDivID,
				DivisionName:      lowestDivName,
			})
			continue
		}

		// Find a division in the current season with the same division_number
		for _, div := range divisions {
			if div.DivisionNumber == previousDiv.DivisionNumber {
				targetDivisionID = div.Uuid
				if div.DivisionName.Valid {
					targetDivisionName = div.DivisionName.String
				} else {
					targetDivisionName = fmt.Sprintf("Division %d", div.DivisionNumber)
				}
				divisionFound = true
				break
			}
		}

		if !divisionFound {
			// Division number doesn't exist in current season - place in lowest division
			lowestDivID, lowestDivName, err := findLowestDivision(divisions)
			if err != nil {
				return nil, fmt.Errorf("no divisions available to place player %s: %w", player.Registration.UserID, err)
			}

			err = pm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
				UserID:      player.Registration.UserID,
				SeasonID:    currentSeasonID,
				DivisionID:  pgtype.UUID{Bytes: lowestDivID, Valid: true},
				FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to assign player %s to lowest division: %w", player.Registration.UserID, err)
			}

			result.PlacedInLowestDivision = append(result.PlacedInLowestDivision, PlacedPlayer{
				CategorizedPlayer: player,
				DivisionID:        lowestDivID,
				DivisionName:      lowestDivName,
			})
			continue
		}

		// Place the player in their previous division
		err = pm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      player.Registration.UserID,
			SeasonID:    currentSeasonID,
			DivisionID:  pgtype.UUID{Bytes: targetDivisionID, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true}, // Will be calculated later when pairings are generated
		})
		if err != nil {
			return nil, fmt.Errorf("failed to assign player %s to division: %w", player.Registration.UserID, err)
		}

		// Calculate and set placement status based on hiatus length
		err = pm.calculateAndSetPlacementStatus(
			ctx,
			player.Registration.UserID,
			currentSeasonID,
			currentSeasonNumber,
			lastPlayedSeasonNumber,
			placementStatus,
			previousRank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to set placement status for %s: %w", player.Registration.UserID, err)
		}

		result.PlacedReturning = append(result.PlacedReturning, PlacedPlayer{
			CategorizedPlayer: player,
			DivisionID:        targetDivisionID,
			DivisionName:      targetDivisionName,
		})
	}

	return result, nil
}
