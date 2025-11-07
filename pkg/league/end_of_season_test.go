package league

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// TestMarkSeasonOutcomes tests that end-of-season processing correctly
// sets placement_status and previous_division_rank for all players
func TestMarkSeasonOutcomes(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	_, store, cleanup := setupTest(t)
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
		Status:       int32(ipc.SeasonStatus_SEASON_ACTIVE),
	})
	is.NoErr(err)

	// Create two divisions
	div1ID := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div1ID,
		SeasonID:       seasonID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
	})
	is.NoErr(err)

	div2ID := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div2ID,
		SeasonID:       seasonID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
	})
	is.NoErr(err)

	// Register 3 players in Division 1
	for i := 1; i <= 3; i++ {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:           int32(i),
			SeasonID:         seasonID,
			DivisionID:       pgtype.UUID{Bytes: div1ID, Valid: true},
			RegistrationDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			FirstsCount:      pgtype.Int4{Int32: 0, Valid: true},
			Status:           pgtype.Text{String: "ACTIVE", Valid: true},
			SeasonsAway:      pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Register 3 different players in Division 2 (players 4, 5, 6)
	for i := 4; i <= 6; i++ {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:           int32(i),
			SeasonID:         seasonID,
			DivisionID:       pgtype.UUID{Bytes: div2ID, Valid: true},
			RegistrationDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
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

		if reg.PlacementStatus.Int32 == int32(ipc.PlacementStatus_PLACEMENT_STAYED) {
			stayed++
		} else if reg.PlacementStatus.Int32 == int32(ipc.PlacementStatus_PLACEMENT_RELEGATED) {
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

		if reg.PlacementStatus.Int32 == int32(ipc.PlacementStatus_PLACEMENT_PROMOTED) {
			promoted++
		} else if reg.PlacementStatus.Int32 == int32(ipc.PlacementStatus_PLACEMENT_STAYED) {
			stayed++
		}
	}
	is.Equal(promoted, 1)
	is.Equal(stayed, 2)
}
