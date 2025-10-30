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

const (
	// RookieDivisionNumberBase is the starting division number for rookie divisions
	// Rookie divisions use numbers 100, 101, 102, etc. to distinguish from regular divisions
	RookieDivisionNumberBase = 100

	// MinPlayersForRookieDivision is the minimum number of rookies needed to create separate rookie divisions
	MinPlayersForRookieDivision = 10

	// Regular division size constraints
	// Regular divisions have higher minimums to maintain competitive balance

	// MinRegularDivisionSize is the minimum number of players in a regular division
	// (unless the league has collapsed and there aren't enough players)
	MinRegularDivisionSize = 13

	// TargetRegularDivisionSize is the ideal size for a regular division
	TargetRegularDivisionSize = 15

	// MaxRegularDivisionSize is the maximum number of players in a regular division
	MaxRegularDivisionSize = 20

	// Rookie division size constraints
	// Rookie divisions are more flexible to accommodate varying numbers of new players

	// MinRookieDivisionSize is the minimum number of players in a rookie division
	MinRookieDivisionSize = 10

	// TargetRookieDivisionSize is the ideal size for a rookie division
	TargetRookieDivisionSize = 15

	// MaxRookieDivisionSize is the maximum number of players in a rookie division
	MaxRookieDivisionSize = 20
)

// RookieManager handles rookie division creation and placement
type RookieManager struct {
	store league.Store
}

// NewRookieManager creates a new rookie manager
func NewRookieManager(store league.Store) *RookieManager {
	return &RookieManager{
		store: store,
	}
}

// RookiePlacementResult tracks the outcome of placing rookies
type RookiePlacementResult struct {
	// Rookie divisions that were created
	CreatedDivisions []models.LeagueDivision

	// Rookies placed in newly created rookie divisions
	PlacedInRookieDivisions []PlacedPlayer

	// Rookies placed in existing regular divisions (when < 10 rookies)
	PlacedInRegularDivisions []PlacedPlayer
}

// PlaceRookies handles rookie placement based on the number of new players
// If < 10 rookies: splits them by rating into bottom 2 regular divisions
// If >= 10 rookies: creates separate rookie divisions (10-15 players each)
func (rm *RookieManager) PlaceRookies(
	ctx context.Context,
	seasonID uuid.UUID,
	rookies []CategorizedPlayer,
) (*RookiePlacementResult, error) {
	result := &RookiePlacementResult{
		CreatedDivisions:         []models.LeagueDivision{},
		PlacedInRookieDivisions:  []PlacedPlayer{},
		PlacedInRegularDivisions: []PlacedPlayer{},
	}

	if len(rookies) == 0 {
		return result, nil
	}

	// Sort rookies by rating (highest to lowest)
	sortedRookies := make([]CategorizedPlayer, len(rookies))
	copy(sortedRookies, rookies)
	sort.Slice(sortedRookies, func(i, j int) bool {
		return sortedRookies[i].Rating > sortedRookies[j].Rating
	})

	if len(sortedRookies) < MinPlayersForRookieDivision {
		// Place in existing regular divisions
		return rm.placeInRegularDivisions(ctx, seasonID, sortedRookies)
	}

	// Create rookie divisions
	return rm.createRookieDivisions(ctx, seasonID, sortedRookies)
}

// placeInRegularDivisions places rookies into the bottom 2 regular divisions by rating
// This is used when there are fewer than MinPlayersForRookieDivision rookies.
// Note: This function adds rookies to existing divisions that already contain returning players.
// Division size limits (MinDivisionSize to HardMaxDivisionSize) should be enforced during
// the initial division creation and rebalancing phases, not here.
func (rm *RookieManager) placeInRegularDivisions(
	ctx context.Context,
	seasonID uuid.UUID,
	sortedRookies []CategorizedPlayer,
) (*RookiePlacementResult, error) {
	result := &RookiePlacementResult{
		CreatedDivisions:         []models.LeagueDivision{},
		PlacedInRookieDivisions:  []PlacedPlayer{},
		PlacedInRegularDivisions: []PlacedPlayer{},
	}

	// Get existing divisions for this season
	divisions, err := rm.store.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get divisions: %w", err)
	}

	// Filter to only regular divisions (division_number < RookieDivisionNumberBase)
	regularDivisions := []models.LeagueDivision{}
	for _, div := range divisions {
		if div.DivisionNumber < RookieDivisionNumberBase {
			regularDivisions = append(regularDivisions, div)
		}
	}

	if len(regularDivisions) == 0 {
		return nil, fmt.Errorf("no regular divisions available to place %d rookies", len(sortedRookies))
	}

	// Sort regular divisions by division number (descending) to get bottom divisions
	sort.Slice(regularDivisions, func(i, j int) bool {
		return regularDivisions[i].DivisionNumber > regularDivisions[j].DivisionNumber
	})

	bottomDiv := regularDivisions[0] // Lowest division (highest number)

	// If only 1 division, place all rookies there
	if len(regularDivisions) == 1 {
		for _, rookie := range sortedRookies {
			err := rm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
				UserID:      rookie.Registration.UserID,
				SeasonID:    seasonID,
				DivisionID:  pgtype.UUID{Bytes: bottomDiv.Uuid, Valid: true},
				FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to assign rookie to division: %w", err)
			}

			divName := fmt.Sprintf("Division %d", bottomDiv.DivisionNumber)
			if bottomDiv.DivisionName.Valid {
				divName = bottomDiv.DivisionName.String
			}

			result.PlacedInRegularDivisions = append(result.PlacedInRegularDivisions, PlacedPlayer{
				CategorizedPlayer: rookie,
				DivisionID:        bottomDiv.Uuid,
				DivisionName:      divName,
			})
		}
		return result, nil
	}

	// If 2+ divisions, split rookies between bottom 2
	secondBottomDiv := regularDivisions[1] // Second lowest

	// Split rookies: top half goes to second-bottom division, bottom half to bottom division
	midpoint := len(sortedRookies) / 2

	// Top half (higher rated) -> second-bottom division
	for i := 0; i < midpoint; i++ {
		rookie := sortedRookies[i]
		err := rm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      rookie.Registration.UserID,
			SeasonID:    seasonID,
			DivisionID:  pgtype.UUID{Bytes: secondBottomDiv.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to assign rookie to division: %w", err)
		}

		divName := fmt.Sprintf("Division %d", secondBottomDiv.DivisionNumber)
		if secondBottomDiv.DivisionName.Valid {
			divName = secondBottomDiv.DivisionName.String
		}

		result.PlacedInRegularDivisions = append(result.PlacedInRegularDivisions, PlacedPlayer{
			CategorizedPlayer: rookie,
			DivisionID:        secondBottomDiv.Uuid,
			DivisionName:      divName,
		})
	}

	// Bottom half (lower rated) -> bottom division
	for i := midpoint; i < len(sortedRookies); i++ {
		rookie := sortedRookies[i]
		err := rm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      rookie.Registration.UserID,
			SeasonID:    seasonID,
			DivisionID:  pgtype.UUID{Bytes: bottomDiv.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to assign rookie to division: %w", err)
		}

		divName := fmt.Sprintf("Division %d", bottomDiv.DivisionNumber)
		if bottomDiv.DivisionName.Valid {
			divName = bottomDiv.DivisionName.String
		}

		result.PlacedInRegularDivisions = append(result.PlacedInRegularDivisions, PlacedPlayer{
			CategorizedPlayer: rookie,
			DivisionID:        bottomDiv.Uuid,
			DivisionName:      divName,
		})
	}

	return result, nil
}

// createRookieDivisions creates separate rookie divisions with balanced sizes (10-15 players each)
func (rm *RookieManager) createRookieDivisions(
	ctx context.Context,
	seasonID uuid.UUID,
	sortedRookies []CategorizedPlayer,
) (*RookiePlacementResult, error) {
	result := &RookiePlacementResult{
		CreatedDivisions:         []models.LeagueDivision{},
		PlacedInRookieDivisions:  []PlacedPlayer{},
		PlacedInRegularDivisions: []PlacedPlayer{},
	}

	// Calculate optimal division sizes
	divisionSizes := calculateRookieDivisionSizes(len(sortedRookies))

	// Create divisions and assign players
	playerIndex := 0
	for divIndex, size := range divisionSizes {
		// Create the division
		divNumber := RookieDivisionNumberBase + divIndex
		divName := fmt.Sprintf("Rookie Division %d", divIndex+1)

		division, err := rm.store.CreateDivision(ctx, models.CreateDivisionParams{
			Uuid:           uuid.New(),
			SeasonID:       seasonID,
			DivisionNumber: int32(divNumber),
			DivisionName:   pgtype.Text{String: divName, Valid: true},
			PlayerCount:    pgtype.Int4{Int32: int32(size), Valid: true},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create rookie division: %w", err)
		}

		result.CreatedDivisions = append(result.CreatedDivisions, division)

		// Assign players to this division
		for i := 0; i < size && playerIndex < len(sortedRookies); i++ {
			rookie := sortedRookies[playerIndex]
			playerIndex++

			err := rm.store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
				UserID:      rookie.Registration.UserID,
				SeasonID:    seasonID,
				DivisionID:  pgtype.UUID{Bytes: division.Uuid, Valid: true},
				FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to assign rookie to division: %w", err)
			}

			result.PlacedInRookieDivisions = append(result.PlacedInRookieDivisions, PlacedPlayer{
				CategorizedPlayer: rookie,
				DivisionID:        division.Uuid,
				DivisionName:      divName,
			})
		}
	}

	return result, nil
}

// calculateRookieDivisionSizes determines the optimal sizes for rookie divisions
// Aims to keep divisions between MinRookieDivisionSize and TargetRookieDivisionSize (10-15)
// but will allow up to MaxRookieDivisionSize (20) to avoid divisions that are too small
func calculateRookieDivisionSizes(numRookies int) []int {
	if numRookies < MinRookieDivisionSize {
		return []int{}
	}

	// For 10-20 rookies, use one division (up to max)
	if numRookies <= MaxRookieDivisionSize {
		return []int{numRookies}
	}

	// For more than MaxRookieDivisionSize, we need multiple divisions
	// Start by trying to use the target (15) as the goal
	numDivisions := (numRookies + TargetRookieDivisionSize - 1) / TargetRookieDivisionSize

	// Calculate sizes with this number of divisions
	baseSize := numRookies / numDivisions
	remainder := numRookies % numDivisions
	maxSize := baseSize
	if remainder > 0 {
		maxSize = baseSize + 1
	}

	// If the minimum size is too small, reduce number of divisions
	// This will make divisions larger but still respect the max
	for baseSize < MinRookieDivisionSize && numDivisions > 1 {
		numDivisions--
		baseSize = numRookies / numDivisions
		remainder = numRookies % numDivisions
		maxSize = baseSize
		if remainder > 0 {
			maxSize = baseSize + 1
		}
	}

	// Verify we don't exceed the max
	if maxSize > MaxRookieDivisionSize {
		// Need more divisions to stay under max
		numDivisions = (numRookies + MaxRookieDivisionSize - 1) / MaxRookieDivisionSize
		baseSize = numRookies / numDivisions
		remainder = numRookies % numDivisions
	}

	// Calculate actual sizes, distributing remainder across first divisions
	sizes := make([]int, numDivisions)
	for i := 0; i < numDivisions; i++ {
		sizes[i] = baseSize
		if i < remainder {
			sizes[i]++
		}
	}

	return sizes
}
