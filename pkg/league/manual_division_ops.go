package league

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// ManualDivisionManager provides tools for manual division management
type ManualDivisionManager struct {
	stores *stores.Stores
}

// NewManualDivisionManager creates a new manual division manager
func NewManualDivisionManager(allStores *stores.Stores) *ManualDivisionManager {
	return &ManualDivisionManager{
		stores: allStores,
	}
}

// MergeResult tracks the outcome of merging two divisions
type MergeResult struct {
	PlayersAffected     int
	DivisionsRenumbered int
	NewDivisionNumbers  map[uuid.UUID]int32 // Maps division UUID to new number
	DeletedDivisionID   uuid.UUID
	ReceivingDivisionID uuid.UUID
}

// MoveResult tracks the outcome of moving a player
type MoveResult struct {
	Success            bool
	UserID             string
	PreviousDivisionID uuid.UUID
	NewDivisionID      uuid.UUID
}

// MergeDivisions merges all players from mergingDiv into receivingDiv,
// deletes mergingDiv, and renumbers remaining divisions sequentially.
//
// This is useful for manually handling undersized divisions.
func (mdm *ManualDivisionManager) MergeDivisions(
	ctx context.Context,
	seasonID uuid.UUID,
	receivingDivID uuid.UUID,
	mergingDivID uuid.UUID,
) (*MergeResult, error) {
	if receivingDivID == mergingDivID {
		return nil, fmt.Errorf("cannot merge a division into itself")
	}

	// Get both divisions to validate they exist and are in the same season
	receivingDiv, err := mdm.stores.LeagueStore.GetDivision(ctx, receivingDivID)
	if err != nil {
		return nil, fmt.Errorf("failed to get receiving division: %w", err)
	}

	mergingDiv, err := mdm.stores.LeagueStore.GetDivision(ctx, mergingDivID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merging division: %w", err)
	}

	// Verify both divisions are in the same season
	if receivingDiv.SeasonID != seasonID || mergingDiv.SeasonID != seasonID {
		return nil, fmt.Errorf("divisions must be in the specified season")
	}

	// Get all players from the merging division
	mergingPlayers, err := mdm.stores.LeagueStore.GetDivisionRegistrations(ctx, mergingDivID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merging division players: %w", err)
	}

	// Move all players to receiving division
	for _, player := range mergingPlayers {
		err := mdm.stores.LeagueStore.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      player.UserID,
			SeasonID:    seasonID,
			DivisionID:  pgtype.UUID{Bytes: receivingDivID, Valid: true},
			FirstsCount: player.FirstsCount,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to move player %d: %w", player.UserID, err)
		}
	}

	// Delete the merging division
	err = mdm.stores.LeagueStore.DeleteDivision(ctx, mergingDivID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete merging division: %w", err)
	}

	// Get all remaining divisions in the season (excluding rookie divisions)
	allDivisions, err := mdm.stores.LeagueStore.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season divisions: %w", err)
	}

	// Sort by current division number
	sort.Slice(allDivisions, func(i, j int) bool {
		return allDivisions[i].DivisionNumber < allDivisions[j].DivisionNumber
	})

	// Renumber sequentially: 1, 2, 3, ..., N
	newNumbers := make(map[uuid.UUID]int32)
	divisionsRenumbered := 0

	for i, div := range allDivisions {
		newNumber := int32(i + 1)
		if div.DivisionNumber != newNumber {
			// Need to renumber this division
			divName := fmt.Sprintf("Division %d", newNumber)
			err := mdm.stores.LeagueStore.UpdateDivisionNumber(ctx, models.UpdateDivisionNumberParams{
				Uuid:           div.Uuid,
				DivisionNumber: newNumber,
				DivisionName:   pgtype.Text{String: divName, Valid: true},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to renumber division %s: %w", div.Uuid, err)
			}
			divisionsRenumbered++
		}
		newNumbers[div.Uuid] = newNumber
	}

	return &MergeResult{
		PlayersAffected:     len(mergingPlayers),
		DivisionsRenumbered: divisionsRenumbered,
		NewDivisionNumbers:  newNumbers,
		DeletedDivisionID:   mergingDivID,
		ReceivingDivisionID: receivingDivID,
	}, nil
}

// MovePlayer moves a single player from one division to another.
// This is useful for manual corrections and balancing.
// After moving, placement_status is recomputed based on actual movement vs previous season.
func (mdm *ManualDivisionManager) MovePlayer(
	ctx context.Context,
	userID string, // UUID string
	leagueID uuid.UUID,
	seasonID uuid.UUID,
	fromDivID uuid.UUID,
	toDivID uuid.UUID,
) (*MoveResult, error) {
	if fromDivID == toDivID {
		return nil, fmt.Errorf("cannot move player to the same division")
	}

	// Look up user database ID from UUID
	dbUser, err := mdm.stores.UserStore.GetByUUID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	userDBID := int32(dbUser.ID)

	// Verify player exists in the from division
	reg, err := mdm.stores.LeagueStore.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
		SeasonID: seasonID,
		UserID:   userDBID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get player registration: %w", err)
	}

	if !reg.DivisionID.Valid || reg.DivisionID.Bytes != fromDivID {
		return nil, fmt.Errorf("player %s is not in division %s", userID, fromDivID)
	}

	// Verify target division exists and is in the same season
	toDiv, err := mdm.stores.LeagueStore.GetDivision(ctx, toDivID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target division: %w", err)
	}

	if toDiv.SeasonID != seasonID {
		return nil, fmt.Errorf("target division is not in the same season")
	}

	// Move the player
	err = mdm.stores.LeagueStore.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
		UserID:      userDBID,
		SeasonID:    seasonID,
		DivisionID:  pgtype.UUID{Bytes: toDivID, Valid: true},
		FirstsCount: reg.FirstsCount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to move player: %w", err)
	}

	// Recompute placement_status based on actual movement vs previous season.
	mdm.recomputePlacementStatus(ctx, userID, userDBID, leagueID, seasonID, reg, toDiv.DivisionNumber)

	return &MoveResult{
		Success:            true,
		UserID:             userID,
		PreviousDivisionID: fromDivID,
		NewDivisionID:      toDivID,
	}, nil
}

// recomputePlacementStatus recalculates and persists the placement_status after a
// manual move.  NEW / HIATUS statuses are preserved (they describe entry, not movement).
// Errors are logged and swallowed so a status-update hiccup does not fail the move.
func (mdm *ManualDivisionManager) recomputePlacementStatus(
	ctx context.Context,
	username string,
	userDBID int32,
	leagueID uuid.UUID,
	seasonID uuid.UUID,
	reg models.LeagueRegistration,
	newDivNumber int32,
) {
	oldStatus := ipc.PlacementStatus_PLACEMENT_NONE
	if reg.PlacementStatus.Valid {
		oldStatus = ipc.PlacementStatus(reg.PlacementStatus.Int32)
	}

	// Find the player's division number from their previous season.
	history, err := mdm.stores.LeagueStore.GetPlayerSeasonHistory(ctx, models.GetPlayerSeasonHistoryParams{
		UserID:   userDBID,
		LeagueID: leagueID,
	})
	if err != nil {
		log.Warn().Err(err).Str("userID", username).
			Msg("recomputePlacementStatus: failed to get season history, skipping")
		return
	}

	prevDivNumber := int32(0)
	for _, h := range history {
		if h.SeasonID == seasonID {
			continue // Skip the current season.
		}
		if !h.DivisionID.Valid {
			continue
		}
		div, err := mdm.stores.LeagueStore.GetDivision(ctx, h.DivisionID.Bytes)
		if err != nil {
			continue
		}
		prevDivNumber = div.DivisionNumber
		break
	}

	if prevDivNumber == 0 {
		// No previous season found — player is new; status stays as-is.
		return
	}

	newStatus := CorrectPlacementStatus(oldStatus, prevDivNumber, newDivNumber)
	if newStatus == oldStatus {
		return
	}

	err = mdm.stores.LeagueStore.UpdatePlacementStatus(ctx, models.UpdatePlacementStatusParams{
		UserID:               userDBID,
		PlacementStatus:      pgtype.Int4{Int32: int32(newStatus), Valid: true},
		PreviousDivisionRank: reg.PreviousDivisionRank,
		SeasonID:             seasonID,
	})
	if err != nil {
		log.Warn().Err(err).Str("userID", username).
			Str("old_status", oldStatus.String()).
			Str("new_status", newStatus.String()).
			Msg("recomputePlacementStatus: failed to update placement status")
		return
	}

	log.Info().
		Str("username", username).
		Str("old_status", oldStatus.String()).
		Str("new_status", newStatus.String()).
		Int32("previous_div", prevDivNumber).
		Int32("new_div", newDivNumber).
		Msg("recomputed placement status after manual move")
}

// CreateDivision creates a new division at the specified number.
// If a division already exists at that number, existing divisions are shifted up.
//
// Example: Creating Division 3 when divisions [1, 2, 3, 4] exist:
//   - Division 3 becomes Division 4
//   - Division 4 becomes Division 5
//   - New Division 3 is created
func (mdm *ManualDivisionManager) CreateDivision(
	ctx context.Context,
	seasonID uuid.UUID,
	divisionNumber int32,
	divisionName string,
) (models.LeagueDivision, error) {
	if divisionNumber < 1 {
		return models.LeagueDivision{}, fmt.Errorf("division number must be >= 1")
	}

	// Get all divisions in the season
	allDivisions, err := mdm.stores.LeagueStore.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return models.LeagueDivision{}, fmt.Errorf("failed to get season divisions: %w", err)
	}

	// Sort by division number (descending) to shift from bottom up
	sort.Slice(allDivisions, func(i, j int) bool {
		return allDivisions[i].DivisionNumber > allDivisions[j].DivisionNumber
	})

	// Shift existing divisions that are >= divisionNumber
	for _, div := range allDivisions {
		if div.DivisionNumber >= divisionNumber {
			newNumber := div.DivisionNumber + 1
			newName := fmt.Sprintf("Division %d", newNumber)
			err := mdm.stores.LeagueStore.UpdateDivisionNumber(ctx, models.UpdateDivisionNumberParams{
				Uuid:           div.Uuid,
				DivisionNumber: newNumber,
				DivisionName:   pgtype.Text{String: newName, Valid: true},
			})
			if err != nil {
				return models.LeagueDivision{}, fmt.Errorf("failed to shift division %d: %w", div.DivisionNumber, err)
			}
		}
	}

	// Create the new division
	if divisionName == "" {
		divisionName = fmt.Sprintf("Division %d", divisionNumber)
	}

	newDiv, err := mdm.stores.LeagueStore.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       seasonID,
		DivisionNumber: divisionNumber,
		DivisionName:   pgtype.Text{String: divisionName, Valid: true},
		IsComplete:     pgtype.Bool{Bool: false, Valid: true},
	})
	if err != nil {
		return models.LeagueDivision{}, fmt.Errorf("failed to create division: %w", err)
	}

	return newDiv, nil
}
