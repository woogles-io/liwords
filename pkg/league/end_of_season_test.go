package league

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// TestMarkSeasonOutcomes tests that end-of-season processing correctly
// sets placement_status and previous_division_rank for all players
func TestMarkSeasonOutcomes(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	store, cleanup := setupTest(t)
	defer cleanup()

	em := NewEndOfSeasonManager(store)

	// Create league
	league := uuid.New()
	_, err := store.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid:        league,
		Name:        "Test League",
		Description: pgtype.Text{String: "Test", Valid: true},
		Slug:        "test",
		Settings:    []byte(`{}`),
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedBy:   pgtype.Int8{Int64: 1, Valid: true},
	})
	is.NoErr(err)

	// Create season
	seasonID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     league,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 1, 0), Valid: true},
		Status:       "SEASON_ACTIVE",
	})
	is.NoErr(err)

	// Create two divisions
	div1ID := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div1ID,
		SeasonID:       seasonID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 3, Valid: true},
	})
	is.NoErr(err)

	div2ID := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div2ID,
		SeasonID:       seasonID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 3, Valid: true},
	})
	is.NoErr(err)

	// Register 3 players in Division 1
	for i := 1; i <= 3; i++ {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:           uuid.New().String(),
			SeasonID:         seasonID,
			DivisionID:       pgtype.UUID{Bytes: div1ID, Valid: true},
			RegistrationDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			StartingRating:   pgtype.Int4{Int32: 1500, Valid: true},
			FirstsCount:      pgtype.Int4{Int32: 0, Valid: true},
			Status:           pgtype.Text{String: "ACTIVE", Valid: true},
			SeasonsAway:      pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Register 3 players in Division 2
	for i := 1; i <= 3; i++ {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:           uuid.New().String(),
			SeasonID:         seasonID,
			DivisionID:       pgtype.UUID{Bytes: div2ID, Valid: true},
			RegistrationDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			StartingRating:   pgtype.Int4{Int32: 1500, Valid: true},
			FirstsCount:      pgtype.Int4{Int32: 0, Valid: true},
			Status:           pgtype.Text{String: "ACTIVE", Valid: true},
			SeasonsAway:      pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Mark season outcomes
	err = em.MarkSeasonOutcomes(ctx, seasonID)
	is.NoErr(err)

	// Verify Division 1 registrations
	div1Regs, err := store.GetDivisionRegistrations(ctx, div1ID)
	is.NoErr(err)
	is.Equal(len(div1Regs), 3)

	// Division 1 is the highest division, so:
	// - Top player (rank 1) should be STAYED (can't promote)
	// - Middle player (rank 2) should be STAYED
	// - Bottom player (rank 3) should be RELEGATED
	var stayed, relegated int
	for _, reg := range div1Regs {
		is.True(reg.PlacementStatus.Valid)
		is.True(reg.PreviousDivisionRank.Valid)

		if reg.PlacementStatus.String == "STAYED" {
			stayed++
		} else if reg.PlacementStatus.String == "RELEGATED" {
			relegated++
		}
	}
	is.Equal(stayed, 2)
	is.Equal(relegated, 1)

	// Verify Division 2 registrations
	div2Regs, err := store.GetDivisionRegistrations(ctx, div2ID)
	is.NoErr(err)
	is.Equal(len(div2Regs), 3)

	// Division 2 is the lowest division, so:
	// - Top player (rank 1) should be PROMOTED
	// - Middle player (rank 2) should be STAYED
	// - Bottom player (rank 3) should be STAYED (can't relegate)
	var promoted int
	stayed = 0
	for _, reg := range div2Regs {
		is.True(reg.PlacementStatus.Valid)
		is.True(reg.PreviousDivisionRank.Valid)

		if reg.PlacementStatus.String == "PROMOTED" {
			promoted++
		} else if reg.PlacementStatus.String == "STAYED" {
			stayed++
		}
	}
	is.Equal(promoted, 1)
	is.Equal(stayed, 2)
}

// TestMarkSeasonOutcomesWithRookieDivision tests that rookie divisions
// don't get promotion/relegation outcomes
func TestMarkSeasonOutcomesWithRookieDivision(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	store, cleanup := setupTest(t)
	defer cleanup()

	em := NewEndOfSeasonManager(store)

	// Create league
	league := uuid.New()
	_, err := store.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid:        league,
		Name:        "Test League",
		Description: pgtype.Text{String: "Test", Valid: true},
		Slug:        "test",
		Settings:    []byte(`{}`),
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedBy:   pgtype.Int8{Int64: 1, Valid: true},
	})
	is.NoErr(err)

	// Create season
	seasonID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     league,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 1, 0), Valid: true},
		Status:       "SEASON_ACTIVE",
	})
	is.NoErr(err)

	// Create a rookie division (100+)
	rookieDivID := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           rookieDivID,
		SeasonID:       seasonID,
		DivisionNumber: RookieDivisionNumberBase, // 100
		DivisionName:   pgtype.Text{String: "Rookie Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 10, Valid: true},
	})
	is.NoErr(err)

	// Register 10 rookies
	for i := 1; i <= 10; i++ {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:           uuid.New().String(),
			SeasonID:         seasonID,
			DivisionID:       pgtype.UUID{Bytes: rookieDivID, Valid: true},
			RegistrationDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			StartingRating:   pgtype.Int4{Int32: 1200, Valid: true},
			FirstsCount:      pgtype.Int4{Int32: 0, Valid: true},
			Status:           pgtype.Text{String: "ACTIVE", Valid: true},
			SeasonsAway:      pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Mark season outcomes
	err = em.MarkSeasonOutcomes(ctx, seasonID)
	is.NoErr(err)

	// Verify rookie division
	// Rookie divisions are treated as lowest division for marking purposes
	// Top performers (ceil(10/6) = 2) get PROMOTED
	// Middle/Bottom performers (8) get STAYED (can't relegate from lowest)
	rookieRegs, err := store.GetDivisionRegistrations(ctx, rookieDivID)
	is.NoErr(err)
	is.Equal(len(rookieRegs), 10)

	var promoted, stayed int
	for _, reg := range rookieRegs {
		is.True(reg.PlacementStatus.Valid)
		if reg.PlacementStatus.String == "PROMOTED" {
			promoted++
		} else if reg.PlacementStatus.String == "STAYED" {
			stayed++
		}
	}
	is.Equal(promoted, 2) // ceil(10/6) = 2
	is.Equal(stayed, 8)   // Rest stay
}
