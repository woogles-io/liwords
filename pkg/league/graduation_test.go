package league

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestCalculateGraduationGroups_Standard(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 19 rookie standings
	rookies := make([]models.LeagueStanding, 19)
	for i := 0; i < 19; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 19 rookies, 12 divisions -> [4,4,4,4,3] into [8,9,10,11,12]
	groups := gm.calculateGraduationGroups(rookies, 12)

	is.Equal(len(groups), 5)

	// Check group sizes
	is.Equal(len(groups[0].Rookies), 4)
	is.Equal(len(groups[1].Rookies), 4)
	is.Equal(len(groups[2].Rookies), 4)
	is.Equal(len(groups[3].Rookies), 4)
	is.Equal(len(groups[4].Rookies), 3)

	// Check target divisions
	is.Equal(groups[0].TargetDivision, int32(8))
	is.Equal(groups[1].TargetDivision, int32(9))
	is.Equal(groups[2].TargetDivision, int32(10))
	is.Equal(groups[3].TargetDivision, int32(11))
	is.Equal(groups[4].TargetDivision, int32(12))

	// Check ranks
	is.Equal(groups[0].StartRank, 1)
	is.Equal(groups[0].EndRank, 4)
	is.Equal(groups[4].StartRank, 17)
	is.Equal(groups[4].EndRank, 19)
}

func TestCalculateGraduationGroups_SkipDivision1(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 15 rookie standings
	rookies := make([]models.LeagueStanding, 15)
	for i := 0; i < 15; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 15 rookies, 5 divisions -> [3,3,3,3,3] into [2,3,4,5] (skip Div 1)
	groups := gm.calculateGraduationGroups(rookies, 5)

	is.Equal(len(groups), 5)

	// Check that we skip Division 1
	is.Equal(groups[0].TargetDivision, int32(2))
	is.Equal(groups[1].TargetDivision, int32(3))
	is.Equal(groups[2].TargetDivision, int32(4))
	is.Equal(groups[3].TargetDivision, int32(5))
	is.Equal(groups[4].TargetDivision, int32(5)) // Overflow to last division

	// Check group sizes (all should be 3)
	for _, group := range groups {
		is.Equal(len(group.Rookies), 3)
	}
}

func TestCalculateGraduationGroups_SingleDivision(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 20 rookie standings
	rookies := make([]models.LeagueStanding, 20)
	for i := 0; i < 20; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 20 rookies, 1 division -> [20] into [1]
	groups := gm.calculateGraduationGroups(rookies, 1)

	is.Equal(len(groups), 1)
	is.Equal(len(groups[0].Rookies), 20)
	is.Equal(groups[0].TargetDivision, int32(1))
	is.Equal(groups[0].StartRank, 1)
	is.Equal(groups[0].EndRank, 20)
}

func TestCalculateGraduationGroups_TwoDivisions(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 12 rookie standings
	rookies := make([]models.LeagueStanding, 12)
	for i := 0; i < 12; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 12 rookies, 2 divisions -> all to Division 2
	groups := gm.calculateGraduationGroups(rookies, 2)

	// groupSize = ceil(12/6) = 2, numGroups = ceil(12/2) = 6 groups
	// But only 2 divisions exist, so all overflow to Div 2
	is.Equal(len(groups), 6)
	for _, group := range groups {
		is.Equal(group.TargetDivision, int32(2)) // All go to Div 2
	}
}

func TestCalculateGraduationGroups_ExactGroups(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 6 rookie standings
	rookies := make([]models.LeagueStanding, 6)
	for i := 0; i < 6; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 6 rookies, 5 divisions
	// groupSize = ceil(6/6) = 1, numGroups = ceil(6/1) = 6 groups
	// Starting div = max(2, 5 - 6 + 1) = max(2, 0) = 2
	groups := gm.calculateGraduationGroups(rookies, 5)

	is.Equal(len(groups), 6)
	// Groups should be distributed [2,3,4,5,5,5] (overflow to div 5)
	is.Equal(groups[0].TargetDivision, int32(2))
	is.Equal(groups[1].TargetDivision, int32(3))
	is.Equal(groups[2].TargetDivision, int32(4))
	is.Equal(groups[3].TargetDivision, int32(5))
	is.Equal(groups[4].TargetDivision, int32(5))
	is.Equal(groups[5].TargetDivision, int32(5))
}

func TestCalculateGraduationGroups_ManyRookies(t *testing.T) {
	is := is.New(t)

	gm := &GraduationManager{}

	// Create 100 rookie standings
	rookies := make([]models.LeagueStanding, 100)
	for i := 0; i < 100; i++ {
		rookies[i] = models.LeagueStanding{
			UserID: uuid.NewString(),
			Rank:   pgtype.Int4{Int32: int32(i + 1), Valid: true},
		}
	}

	// 100 rookies, 3 divisions -> overflow case
	// ceil(100/6) = 17 per group, ceil(100/17) = 6 groups
	// Starting div = max(2, 3 - 6 + 1) = max(2, -2) = 2
	groups := gm.calculateGraduationGroups(rookies, 3)

	is.Equal(len(groups), 6)

	// All should target divisions 2 or 3 (capped at highest)
	for _, group := range groups {
		is.True(group.TargetDivision >= 2 && group.TargetDivision <= 3)
	}

	// Check total rookies
	totalRookies := 0
	for _, group := range groups {
		totalRookies += len(group.Rookies)
	}
	is.Equal(totalRookies, 100)
}

func TestGraduateRookies_Integration(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	store, cleanup := setupTest(t)
	defer cleanup()

	// Create league
	leagueID := uuid.New()
	_, err := store.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid:        leagueID,
		Name:        "Test League",
		Description: pgtype.Text{String: "Test", Valid: true},
		Slug:        "test",
		Settings:    []byte(`{}`),
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedBy:   pgtype.Int8{Int64: 1, Valid: true},
	})
	is.NoErr(err)

	// Season 1: Create rookie division with 15 players
	season1ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season1ID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -30), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -16), Valid: true},
		Status:       "COMPLETED",
	})
	is.NoErr(err)

	rookieDiv, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season1ID,
		DivisionNumber: 100,
		DivisionName:   pgtype.Text{String: "Rookie Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 15, Valid: true},
	})
	is.NoErr(err)

	// Register 15 rookies
	rm := NewRegistrationManager(store)
	rookieUserIDs := []string{}
	for i := 0; i < 15; i++ {
		userID := uuid.NewString()
		rookieUserIDs = append(rookieUserIDs, userID)
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1500-i*10))
		is.NoErr(err)
		err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      userID,
			SeasonID:    season1ID,
			DivisionID:  pgtype.UUID{Bytes: rookieDiv.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Create standings for rookies (simulate completed season)
	for i, userID := range rookieUserIDs {
		err = store.UpsertStanding(ctx, models.UpsertStandingParams{
			DivisionID:     rookieDiv.Uuid,
			UserID:         userID,
			Rank:           pgtype.Int4{Int32: int32(i + 1), Valid: true},
			Wins:           pgtype.Int4{Int32: int32(10 - i), Valid: true},
			Losses:         pgtype.Int4{Int32: int32(i), Valid: true},
			Draws:          pgtype.Int4{Int32: 0, Valid: true},
			Spread:         pgtype.Int4{Int32: int32(100 - i*20), Valid: true},
			GamesPlayed:    pgtype.Int4{Int32: 10, Valid: true},
			GamesRemaining: pgtype.Int4{Int32: 0, Valid: true},
			Result:         pgtype.Text{String: standingResultToString(pb.StandingResult_RESULT_PROMOTED), Valid: true},
		})
		is.NoErr(err)
	}

	// Season 2: Create 5 regular divisions
	season2ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season2ID,
		LeagueID:     leagueID,
		SeasonNumber: 2,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       "SCHEDULED",
	})
	is.NoErr(err)

	// Create 5 regular divisions
	divisions := []models.LeagueDivision{}
	for i := 1; i <= 5; i++ {
		div, err := store.CreateDivision(ctx, models.CreateDivisionParams{
			Uuid:           uuid.New(),
			SeasonID:       season2ID,
			DivisionNumber: int32(i),
			DivisionName:   pgtype.Text{String: fmt.Sprintf("Division %d", i), Valid: true},
			PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
		divisions = append(divisions, div)
	}

	// Register all rookies for Season 2
	for _, userID := range rookieUserIDs {
		err = rm.RegisterPlayer(ctx, userID, season2ID, 1500)
		is.NoErr(err)
	}

	// Graduate rookies
	gm := NewGraduationManager(store)
	result, err := gm.GraduateRookies(ctx, season1ID, season2ID)
	is.NoErr(err)

	// 15 rookies should be graduated
	is.Equal(len(result.GraduatedRookies), 15)

	// Check placement: ceil(15/6) = 3 per group, 5 groups
	// Starting div = max(2, 5 - 5 + 1) = 2
	// Groups: [3,3,3,3,3] -> Divs [2,3,4,5,5]

	// Count players per division
	divCounts := make(map[int32]int)
	for _, placed := range result.GraduatedRookies {
		// Find which division they're in
		for _, div := range divisions {
			if div.Uuid == placed.DivisionID {
				divCounts[div.DivisionNumber]++
				break
			}
		}
	}

	// Division 1 should have 0 (skipped)
	is.Equal(divCounts[1], 0)

	// Divisions 2-5 should have rookies
	is.True(divCounts[2] > 0)
	is.True(divCounts[3] > 0)
	is.True(divCounts[4] > 0)
	is.True(divCounts[5] > 0)

	// Total should be 15
	total := divCounts[1] + divCounts[2] + divCounts[3] + divCounts[4] + divCounts[5]
	is.Equal(total, 15)
}

func TestGraduateRookies_NoRookieDivisions(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	store, cleanup := setupTest(t)
	defer cleanup()

	// Create league
	leagueID := uuid.New()
	_, err := store.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid:        leagueID,
		Name:        "Test League",
		Description: pgtype.Text{String: "Test", Valid: true},
		Slug:        "test",
		Settings:    []byte(`{}`),
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
		CreatedBy:   pgtype.Int8{Int64: 1, Valid: true},
	})
	is.NoErr(err)

	// Season 1: No rookie divisions, just regular divisions
	season1ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season1ID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -30), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -16), Valid: true},
		Status:       "COMPLETED",
	})
	is.NoErr(err)

	// Season 2
	season2ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season2ID,
		LeagueID:     leagueID,
		SeasonNumber: 2,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       "SCHEDULED",
	})
	is.NoErr(err)

	// Graduate rookies (should be no-op)
	gm := NewGraduationManager(store)
	result, err := gm.GraduateRookies(ctx, season1ID, season2ID)
	is.NoErr(err)

	// Should have no graduated rookies
	is.Equal(len(result.GraduatedRookies), 0)
}
