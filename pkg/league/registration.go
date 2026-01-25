package league

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

const (
	// StableRatingDeviation is the threshold for considering a rating "stable"
	// Ratings with RD below this value will be included in the average
	StableRatingDeviation = entity.RatingDeviationConfidence // 90
)

// RegistrationManager handles player registration for league seasons
type RegistrationManager struct {
	store     league.Store
	clock     Clock
	allStores *stores.Stores
}

// NewRegistrationManager creates a new registration manager
func NewRegistrationManager(store league.Store, clock Clock, allStores *stores.Stores) *RegistrationManager {
	return &RegistrationManager{
		store:     store,
		clock:     clock,
		allStores: allStores,
	}
}

// RegisterPlayer registers a player for a season
// Returns error if already registered or if validation fails
func (rm *RegistrationManager) RegisterPlayer(
	ctx context.Context,
	userID int32,
	seasonID uuid.UUID,
) error {
	_, err := rm.store.RegisterPlayer(ctx, models.RegisterPlayerParams{
		UserID:           userID,
		SeasonID:         seasonID,
		DivisionID:       pgtype.UUID{Valid: false}, // No division assigned yet
		RegistrationDate: pgtype.Timestamptz{Time: rm.clock.Now(), Valid: true},
		FirstsCount:      pgtype.Int4{Int32: 0, Valid: true},
		Status:           pgtype.Text{String: "REGISTERED", Valid: true},
		SeasonsAway:      pgtype.Int4{Int32: 0, Valid: true}, // Will be calculated during placement
	})

	return err
}

// GetSeasonRegistrations returns all registrations for a season
func (rm *RegistrationManager) GetSeasonRegistrations(
	ctx context.Context,
	seasonID uuid.UUID,
) ([]models.GetSeasonRegistrationsRow, error) {
	return rm.store.GetSeasonRegistrations(ctx, seasonID)
}

// PlayerCategory indicates whether a player is new or returning
type PlayerCategory string

const (
	PlayerCategoryNew       PlayerCategory = "NEW"       // First time in this league
	PlayerCategoryReturning PlayerCategory = "RETURNING" // Has played in previous seasons
)

// CategorizedPlayer contains a registration and its category
type CategorizedPlayer struct {
	Registration models.GetSeasonRegistrationsRow
	Category     PlayerCategory
	Rating       int32
}

// calculateAverageRating calculates the average Woogles rating for a user
// following these rules:
// 1. Use average of all ratings where RD < 90 (stable ratings)
// 2. If no stable ratings but has unstable ratings, use average of unstable ratings
// 3. If no rated games at all, return 0 (not 1500)
func (rm *RegistrationManager) calculateAverageRating(ctx context.Context, userUUID string) (int32, error) {
	// Get user with ratings
	user, err := rm.allStores.UserStore.GetByUUID(ctx, userUUID)
	if err != nil {
		return 0, err
	}

	if user.Profile == nil || user.Profile.Ratings.Data == nil || len(user.Profile.Ratings.Data) == 0 {
		// No ratings at all - return 0 instead of 1500
		return 0, nil
	}

	// Collect stable and unstable ratings
	var stableRatings []float64
	var unstableRatings []float64

	for _, rating := range user.Profile.Ratings.Data {
		if rating.RatingDeviation < StableRatingDeviation {
			stableRatings = append(stableRatings, rating.Rating)
		} else {
			unstableRatings = append(unstableRatings, rating.Rating)
		}
	}

	// Calculate average based on what's available
	if len(stableRatings) > 0 {
		// Use stable ratings
		sum := 0.0
		for _, r := range stableRatings {
			sum += r
		}
		return int32(sum / float64(len(stableRatings))), nil
	} else if len(unstableRatings) > 0 {
		// No stable ratings, use unstable ratings
		sum := 0.0
		for _, r := range unstableRatings {
			sum += r
		}
		return int32(sum / float64(len(unstableRatings))), nil
	}

	// No ratings at all
	return 0, nil
}

// CategorizeRegistrations determines which players are new vs returning
// for a given season. A player is "new" (rookie) if they have never participated
// in any previous season of this league.
func (rm *RegistrationManager) CategorizeRegistrations(
	ctx context.Context,
	leagueID uuid.UUID,
	seasonID uuid.UUID,
	registrations []models.GetSeasonRegistrationsRow,
) ([]CategorizedPlayer, error) {
	categorized := make([]CategorizedPlayer, 0, len(registrations))

	for _, reg := range registrations {
		// Check if player has history in this league (excluding current season)
		history, err := rm.store.GetPlayerSeasonHistory(ctx, models.GetPlayerSeasonHistoryParams{
			UserID:   reg.UserID,
			LeagueID: leagueID,
		})
		if err != nil {
			return nil, err
		}

		// Filter out the current season from history
		previousSeasons := 0
		for _, season := range history {
			if season.SeasonID != seasonID {
				previousSeasons++
			}
		}

		category := PlayerCategoryNew
		if previousSeasons > 0 {
			category = PlayerCategoryReturning
		}

		// Calculate average rating for this player
		avgRating := int32(0)
		if reg.UserUuid.Valid {
			avgRating, err = rm.calculateAverageRating(ctx, reg.UserUuid.String)
			if err != nil {
				// If we can't get the rating, default to 0
				avgRating = 0
			}
		}

		categorized = append(categorized, CategorizedPlayer{
			Registration: reg,
			Category:     category,
			Rating:       avgRating,
		})
	}

	return categorized, nil
}
