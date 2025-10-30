package league

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// StandingsManager handles calculating and marking player standings
type StandingsManager struct {
	store league.Store
}

// NewStandingsManager creates a new standings manager
func NewStandingsManager(store league.Store) *StandingsManager {
	return &StandingsManager{
		store: store,
	}
}

// PlayerStanding represents a player's standing within a division
type PlayerStanding struct {
	UserID      string
	DivisionID  uuid.UUID
	Wins        float64 // includes ties as 0.5
	Losses      int
	Draws       int
	Spread      int
	GamesPlayed int
	Rank        int
	Outcome     pb.StandingResult
}

// standingResultToString converts a StandingResult enum to its string representation for database storage
func standingResultToString(result pb.StandingResult) string {
	switch result {
	case pb.StandingResult_RESULT_PROMOTED:
		return "PROMOTED"
	case pb.StandingResult_RESULT_RELEGATED:
		return "RELEGATED"
	case pb.StandingResult_RESULT_STAYED:
		return "STAYED"
	case pb.StandingResult_RESULT_CHAMPION:
		return "CHAMPION"
	default:
		return ""
	}
}

// CalculateAndSaveStandings calculates final standings for all divisions in a season
// and marks players with their outcomes (promoted/relegated/stayed)
func (sm *StandingsManager) CalculateAndSaveStandings(
	ctx context.Context,
	seasonID uuid.UUID,
) error {
	// Get all divisions for this season
	divisions, err := sm.store.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get divisions: %w", err)
	}

	// Sort divisions by division number to know which is highest/lowest
	sort.Slice(divisions, func(i, j int) bool {
		return divisions[i].DivisionNumber < divisions[j].DivisionNumber
	})

	// Find the highest regular division number (excluding rookie divisions)
	highestRegularDivision := int32(0)
	for _, div := range divisions {
		if div.DivisionNumber < RookieDivisionNumberBase && div.DivisionNumber > highestRegularDivision {
			highestRegularDivision = div.DivisionNumber
		}
	}

	// Calculate standings for each division
	for _, division := range divisions {
		err := sm.calculateDivisionStandings(ctx, division, highestRegularDivision)
		if err != nil {
			return fmt.Errorf("failed to calculate standings for division %d: %w", division.DivisionNumber, err)
		}
	}

	return nil
}

// calculateDivisionStandings calculates standings for a single division
func (sm *StandingsManager) calculateDivisionStandings(
	ctx context.Context,
	division models.LeagueDivision,
	highestRegularDivision int32,
) error {
	// Get all registrations for this division
	registrations, err := sm.store.GetDivisionRegistrations(ctx, division.Uuid)
	if err != nil {
		return fmt.Errorf("failed to get division registrations: %w", err)
	}

	if len(registrations) == 0 {
		return nil // Nothing to do
	}

	// TODO: Get game results and calculate wins/losses/draws/spread
	// For now, we'll create placeholder standings
	// This will be implemented when game results are integrated

	standings := make([]PlayerStanding, len(registrations))
	for i, reg := range registrations {
		standings[i] = PlayerStanding{
			UserID:      reg.UserID,
			DivisionID:  division.Uuid,
			Wins:        0, // TODO: Calculate from game results
			Losses:      0,
			Draws:       0,
			Spread:      0,
			GamesPlayed: 0,
		}
	}

	// Sort standings by wins (desc), then spread (desc), then randomly for ties
	sm.sortStandings(standings)

	// Assign ranks
	for i := range standings {
		standings[i].Rank = i + 1
	}

	// Mark outcomes based on rank
	sm.markOutcomes(standings, division.DivisionNumber, highestRegularDivision)

	// Save to database
	for _, standing := range standings {
		err := sm.store.UpsertStanding(ctx, models.UpsertStandingParams{
			DivisionID:     division.Uuid,
			UserID:         standing.UserID,
			Rank:           pgtype.Int4{Int32: int32(standing.Rank), Valid: true},
			Wins:           pgtype.Int4{Int32: int32(standing.Wins), Valid: true},
			Losses:         pgtype.Int4{Int32: int32(standing.Losses), Valid: true},
			Draws:          pgtype.Int4{Int32: int32(standing.Draws), Valid: true},
			Spread:         pgtype.Int4{Int32: int32(standing.Spread), Valid: true},
			GamesPlayed:    pgtype.Int4{Int32: int32(standing.GamesPlayed), Valid: true},
			GamesRemaining: pgtype.Int4{Int32: 0, Valid: true},
			Result:         pgtype.Text{String: standingResultToString(standing.Outcome), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to save standing: %w", err)
		}
	}

	return nil
}

// sortStandings sorts standings by wins (desc), spread (desc), then randomly
func (sm *StandingsManager) sortStandings(standings []PlayerStanding) {
	// Assign random tiebreaker values
	for i := range standings {
		standings[i].Rank = rand.Intn(1000000)
	}

	sort.Slice(standings, func(i, j int) bool {
		// First by wins (descending)
		if standings[i].Wins != standings[j].Wins {
			return standings[i].Wins > standings[j].Wins
		}
		// Then by spread (descending)
		if standings[i].Spread != standings[j].Spread {
			return standings[i].Spread > standings[j].Spread
		}
		// Finally by random rank (for true ties)
		return standings[i].Rank < standings[j].Rank
	})
}

// markOutcomes assigns promotion/relegation/stayed outcomes based on rank
func (sm *StandingsManager) markOutcomes(
	standings []PlayerStanding,
	divisionNumber int32,
	highestRegularDivision int32,
) {
	divSize := len(standings)
	if divSize == 0 {
		return
	}

	// Calculate number of promoted and relegated: ceil(div_size / 6)
	promotionCount := int(math.Ceil(float64(divSize) / 6.0))
	relegationCount := promotionCount

	isHighestDivision := divisionNumber == 1
	isLowestDivision := divisionNumber >= highestRegularDivision || divisionNumber >= RookieDivisionNumberBase

	for i := range standings {
		rank := i + 1

		if rank <= promotionCount && !isHighestDivision {
			// Top performers get promoted (unless already in Division 1)
			standings[i].Outcome = pb.StandingResult_RESULT_PROMOTED
		} else if rank > divSize-relegationCount && !isLowestDivision {
			// Bottom performers get relegated (unless in lowest division)
			standings[i].Outcome = pb.StandingResult_RESULT_RELEGATED
		} else {
			// Everyone else stays
			standings[i].Outcome = pb.StandingResult_RESULT_STAYED
		}
	}
}
