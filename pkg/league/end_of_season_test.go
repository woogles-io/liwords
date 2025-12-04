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
// sets previous_division_rank for all players and calculates standings with outcomes
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

	// Verify Division 1 registrations have previous_division_rank set
	div1Regs, err := store.GetDivisionRegistrations(ctx, div1ID)
	is.NoErr(err)
	is.Equal(len(div1Regs), 3)

	for _, reg := range div1Regs {
		is.True(reg.PreviousDivisionRank.Valid)
		// Rank should be between 1 and 3
		is.True(reg.PreviousDivisionRank.Int32 >= 1 && reg.PreviousDivisionRank.Int32 <= 3)
	}

	// Verify Division 2 registrations have previous_division_rank set
	div2Regs, err := store.GetDivisionRegistrations(ctx, div2ID)
	is.NoErr(err)
	is.Equal(len(div2Regs), 3)

	for _, reg := range div2Regs {
		is.True(reg.PreviousDivisionRank.Valid)
		// Rank should be between 1 and 3
		is.True(reg.PreviousDivisionRank.Int32 >= 1 && reg.PreviousDivisionRank.Int32 <= 3)
	}

	// Verify Division 1 standings have correct outcomes
	// Division 1 is the highest division, so:
	// - Top player(s) should be STAYED (can't promote from top)
	// - Bottom player should be RELEGATED
	div1Standings, err := store.GetStandings(ctx, div1ID)
	is.NoErr(err)
	is.Equal(len(div1Standings), 3)

	var stayed, relegated int
	for _, standing := range div1Standings {
		is.True(standing.Result.Valid)
		if ipc.StandingResult(standing.Result.Int32) == ipc.StandingResult_RESULT_STAYED {
			stayed++
		} else if ipc.StandingResult(standing.Result.Int32) == ipc.StandingResult_RESULT_RELEGATED {
			relegated++
		}
	}
	is.Equal(stayed, 2)
	is.Equal(relegated, 1)

	// Verify Division 2 standings have correct outcomes
	// Division 2 is the lowest division, so:
	// - Top player should be PROMOTED
	// - Bottom player(s) should be STAYED (can't relegate from bottom)
	div2Standings, err := store.GetStandings(ctx, div2ID)
	is.NoErr(err)
	is.Equal(len(div2Standings), 3)

	var promoted int
	stayed = 0
	for _, standing := range div2Standings {
		is.True(standing.Result.Valid)
		if ipc.StandingResult(standing.Result.Int32) == ipc.StandingResult_RESULT_PROMOTED {
			promoted++
		} else if ipc.StandingResult(standing.Result.Int32) == ipc.StandingResult_RESULT_STAYED {
			stayed++
		}
	}
	is.Equal(promoted, 1)
	is.Equal(stayed, 2)
}
