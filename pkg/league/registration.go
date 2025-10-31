package league

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// RegistrationManager handles player registration for league seasons
type RegistrationManager struct {
	store league.Store
}

// NewRegistrationManager creates a new registration manager
func NewRegistrationManager(store league.Store) *RegistrationManager {
	return &RegistrationManager{
		store: store,
	}
}

// RegisterPlayer registers a player for a season
// Returns error if already registered or if validation fails
func (rm *RegistrationManager) RegisterPlayer(
	ctx context.Context,
	userID string,
	seasonID uuid.UUID,
	rating int32,
) error {
	_, err := rm.store.RegisterPlayer(ctx, models.RegisterPlayerParams{
		UserID:           userID,
		SeasonID:         seasonID,
		DivisionID:       pgtype.UUID{Valid: false}, // No division assigned yet
		RegistrationDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		StartingRating:   pgtype.Int4{Int32: rating, Valid: true},
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
) ([]models.LeagueRegistration, error) {
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
	Registration models.LeagueRegistration
	Category     PlayerCategory
	Rating       int32
}

// CategorizeRegistrations determines which players are new vs returning
// for a given season. A player is "new" (rookie) if they have never participated
// in any previous season of this league.
func (rm *RegistrationManager) CategorizeRegistrations(
	ctx context.Context,
	leagueID uuid.UUID,
	seasonID uuid.UUID,
	registrations []models.LeagueRegistration,
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

		rating := int32(0)
		if reg.StartingRating.Valid {
			rating = reg.StartingRating.Int32
		}

		categorized = append(categorized, CategorizedPlayer{
			Registration: reg,
			Category:     category,
			Rating:       rating,
		})
	}

	return categorized, nil
}
