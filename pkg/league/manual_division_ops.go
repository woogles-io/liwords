package league

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// ManualDivisionManager provides tools for manual division management
type ManualDivisionManager struct {
	store league.Store
}

// NewManualDivisionManager creates a new manual division manager
func NewManualDivisionManager(store league.Store) *ManualDivisionManager {
	return &ManualDivisionManager{
		store: store,
	}
}

// MergeResult tracks the outcome of merging two divisions
type MergeResult struct {
	PlayersAffected      int
	DivisionsRenumbered  int
	NewDivisionNumbers   map[uuid.UUID]int32 // Maps division UUID to new number
	DeletedDivisionID    uuid.UUID
	ReceivingDivisionID  uuid.UUID
}

// MoveResult tracks the outcome of moving a player
type MoveResult struct {
	Success             bool
	UserID              string
	PreviousDivisionID  uuid.UUID
	NewDivisionID       uuid.UUID
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
	receivingDiv, err := mdm.store.GetDivision(ctx, receivingDivID)
	if err != nil {
		return nil, fmt.Errorf("failed to get receiving division: %w", err)
	}

	mergingDiv, err := mdm.store.GetDivision(ctx, mergingDivID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merging division: %w", err)
	}

	// Verify both divisions are in the same season
	if receivingDiv.SeasonID != seasonID || mergingDiv.SeasonID != seasonID {
		return nil, fmt.Errorf("divisions must be in the specified season")
	}

	// Get all players from the merging division
	mergingPlayers, err := mdm.store.GetDivisionRegistrations(ctx, mergingDivID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merging division players: %w", err)
	}

	// Move all players to receiving division
	for _, player := range mergingPlayers {
		err := mdm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      player.UserID,
			SeasonID:    seasonID,
			DivisionID:  pgtype.UUID{Bytes: receivingDivID, Valid: true},
			FirstsCount: player.FirstsCount,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to move player %s: %w", player.UserID, err)
		}
	}

	// Delete the merging division
	err = mdm.store.DeleteDivision(ctx, mergingDivID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete merging division: %w", err)
	}

	// Get all remaining divisions in the season (excluding rookie divisions)
	allDivisions, err := mdm.store.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get season divisions: %w", err)
	}

	// Filter to regular divisions only (< 100)
	regularDivisions := []models.LeagueDivision{}
	for _, div := range allDivisions {
		if div.DivisionNumber < RookieDivisionNumberBase {
			regularDivisions = append(regularDivisions, div)
		}
	}

	// Sort by current division number
	sort.Slice(regularDivisions, func(i, j int) bool {
		return regularDivisions[i].DivisionNumber < regularDivisions[j].DivisionNumber
	})

	// Renumber sequentially: 1, 2, 3, ..., N
	newNumbers := make(map[uuid.UUID]int32)
	divisionsRenumbered := 0

	for i, div := range regularDivisions {
		newNumber := int32(i + 1)
		if div.DivisionNumber != newNumber {
			// Need to renumber this division
			divName := fmt.Sprintf("Division %d", newNumber)
			err := mdm.store.UpdateDivisionNumber(ctx, models.UpdateDivisionNumberParams{
				Uuid:         div.Uuid,
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
func (mdm *ManualDivisionManager) MovePlayer(
	ctx context.Context,
	userID string,
	seasonID uuid.UUID,
	fromDivID uuid.UUID,
	toDivID uuid.UUID,
) (*MoveResult, error) {
	if fromDivID == toDivID {
		return nil, fmt.Errorf("cannot move player to the same division")
	}

	// Verify player exists in the from division
	reg, err := mdm.store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
		SeasonID: seasonID,
		UserID:   userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get player registration: %w", err)
	}

	if !reg.DivisionID.Valid || reg.DivisionID.Bytes != fromDivID {
		return nil, fmt.Errorf("player %s is not in division %s", userID, fromDivID)
	}

	// Verify target division exists and is in the same season
	toDiv, err := mdm.store.GetDivision(ctx, toDivID)
	if err != nil {
		return nil, fmt.Errorf("failed to get target division: %w", err)
	}

	if toDiv.SeasonID != seasonID {
		return nil, fmt.Errorf("target division is not in the same season")
	}

	// Move the player
	err = mdm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
		UserID:      userID,
		SeasonID:    seasonID,
		DivisionID:  pgtype.UUID{Bytes: toDivID, Valid: true},
		FirstsCount: reg.FirstsCount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to move player: %w", err)
	}

	return &MoveResult{
		Success:            true,
		UserID:             userID,
		PreviousDivisionID: fromDivID,
		NewDivisionID:      toDivID,
	}, nil
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

	// Check if this is a rookie division number
	if divisionNumber >= RookieDivisionNumberBase {
		return models.LeagueDivision{}, fmt.Errorf("cannot create regular division with number >= %d (reserved for rookie divisions)", RookieDivisionNumberBase)
	}

	// Get all regular divisions in the season
	allDivisions, err := mdm.store.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return models.LeagueDivision{}, fmt.Errorf("failed to get season divisions: %w", err)
	}

	// Filter to regular divisions only
	regularDivisions := []models.LeagueDivision{}
	for _, div := range allDivisions {
		if div.DivisionNumber < RookieDivisionNumberBase {
			regularDivisions = append(regularDivisions, div)
		}
	}

	// Sort by division number (descending) to shift from bottom up
	sort.Slice(regularDivisions, func(i, j int) bool {
		return regularDivisions[i].DivisionNumber > regularDivisions[j].DivisionNumber
	})

	// Shift existing divisions that are >= divisionNumber
	for _, div := range regularDivisions {
		if div.DivisionNumber >= divisionNumber {
			newNumber := div.DivisionNumber + 1
			newName := fmt.Sprintf("Division %d", newNumber)
			err := mdm.store.UpdateDivisionNumber(ctx, models.UpdateDivisionNumberParams{
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

	newDiv, err := mdm.store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       seasonID,
		DivisionNumber: divisionNumber,
		DivisionName:   pgtype.Text{String: divisionName, Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
		IsComplete:     pgtype.Bool{Bool: false, Valid: true},
	})
	if err != nil {
		return models.LeagueDivision{}, fmt.Errorf("failed to create division: %w", err)
	}

	return newDiv, nil
}
