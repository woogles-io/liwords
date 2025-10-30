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

func TestPlaceReturningPlayers_AllReturning(t *testing.T) {
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

	// Create Season 1 with 2 divisions
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

	div1S1, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season1ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	div2S1, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season1ID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register 10 players in Season 1 (5 in each division)
	rm := NewRegistrationManager(store)
	div1Players := []string{}
	div2Players := []string{}

	for i := 0; i < 5; i++ {
		userID := uuid.NewString()
		div1Players = append(div1Players, userID)
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1500+i*10))
		is.NoErr(err)
		err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      userID,
			SeasonID:    season1ID,
			DivisionID:  pgtype.UUID{Bytes: div1S1.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	for i := 0; i < 5; i++ {
		userID := uuid.NewString()
		div2Players = append(div2Players, userID)
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1400+i*10))
		is.NoErr(err)
		err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      userID,
			SeasonID:    season1ID,
			DivisionID:  pgtype.UUID{Bytes: div2S1.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Create Season 2 with same 2 divisions
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

	div1S2, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season2ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	div2S2, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season2ID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register all 10 returning players in Season 2
	for _, userID := range append(div1Players, div2Players...) {
		err = rm.RegisterPlayer(ctx, userID, season2ID, int32(1500))
		is.NoErr(err)
	}

	// Categorize registrations
	regs, err := rm.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)

	categorized, err := rm.CategorizeRegistrations(ctx, leagueID, season2ID, regs)
	is.NoErr(err)
	is.Equal(len(categorized), 10)

	// Place returning players
	pm := NewPlacementManager(store)
	result, err := pm.PlaceReturningPlayers(ctx, leagueID, season2ID, categorized)
	is.NoErr(err)

	// All 10 should be placed as returning
	is.Equal(len(result.PlacedReturning), 10)
	is.Equal(len(result.NeedingRookiePlacement), 0)
	is.Equal(len(result.PlacedInLowestDivision), 0)

	// Verify players are in correct divisions
	div1Count := 0
	div2Count := 0
	for _, placed := range result.PlacedReturning {
		if placed.DivisionID == div1S2.Uuid {
			div1Count++
			// Verify this player was in div1 originally
			found := false
			for _, id := range div1Players {
				if id == placed.Registration.UserID {
					found = true
					break
				}
			}
			is.True(found)
		} else if placed.DivisionID == div2S2.Uuid {
			div2Count++
			// Verify this player was in div2 originally
			found := false
			for _, id := range div2Players {
				if id == placed.Registration.UserID {
					found = true
					break
				}
			}
			is.True(found)
		}
	}

	is.Equal(div1Count, 5)
	is.Equal(div2Count, 5)
}

func TestPlaceReturningPlayers_Mixed(t *testing.T) {
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

	// Create Season 1 with 1 division
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

	div1S1, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season1ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register 3 returning players in Season 1
	rm := NewRegistrationManager(store)
	returningPlayers := []string{}
	for i := 0; i < 3; i++ {
		userID := uuid.NewString()
		returningPlayers = append(returningPlayers, userID)
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1500+i*10))
		is.NoErr(err)
		err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      userID,
			SeasonID:    season1ID,
			DivisionID:  pgtype.UUID{Bytes: div1S1.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Create Season 2 with same division
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

	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season2ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register 3 returning + 4 new players in Season 2
	for _, userID := range returningPlayers {
		err = rm.RegisterPlayer(ctx, userID, season2ID, int32(1500))
		is.NoErr(err)
	}

	newPlayers := []string{}
	for i := 0; i < 4; i++ {
		userID := uuid.NewString()
		newPlayers = append(newPlayers, userID)
		err = rm.RegisterPlayer(ctx, userID, season2ID, int32(1400+i*10))
		is.NoErr(err)
	}

	// Categorize and place
	regs, err := rm.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)

	categorized, err := rm.CategorizeRegistrations(ctx, leagueID, season2ID, regs)
	is.NoErr(err)
	is.Equal(len(categorized), 7)

	pm := NewPlacementManager(store)
	result, err := pm.PlaceReturningPlayers(ctx, leagueID, season2ID, categorized)
	is.NoErr(err)

	// 3 returning, 4 rookies
	is.Equal(len(result.PlacedReturning), 3)
	is.Equal(len(result.NeedingRookiePlacement), 4)
	is.Equal(len(result.PlacedInLowestDivision), 0)

	// Verify rookies are the new players
	for _, rookie := range result.NeedingRookiePlacement {
		found := false
		for _, id := range newPlayers {
			if id == rookie.Registration.UserID {
				found = true
				break
			}
		}
		is.True(found)
	}
}

func TestPlaceReturningPlayers_MissingDivision(t *testing.T) {
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

	// Create Season 1 with 2 divisions
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

	div1S1, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season1ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	div2S1, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season1ID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register players in both divisions
	rm := NewRegistrationManager(store)
	div1Player := uuid.NewString()
	div2Player := uuid.NewString()

	err = rm.RegisterPlayer(ctx, div1Player, season1ID, 1500)
	is.NoErr(err)
	err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
		UserID:      div1Player,
		SeasonID:    season1ID,
		DivisionID:  pgtype.UUID{Bytes: div1S1.Uuid, Valid: true},
		FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	err = rm.RegisterPlayer(ctx, div2Player, season1ID, 1400)
	is.NoErr(err)
	err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
		UserID:      div2Player,
		SeasonID:    season1ID,
		DivisionID:  pgtype.UUID{Bytes: div2S1.Uuid, Valid: true},
		FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Create Season 2 with ONLY division 1 (division 2 is gone)
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

	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season2ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)
	// Note: No division 2 in season 2

	// Both players register for Season 2
	err = rm.RegisterPlayer(ctx, div1Player, season2ID, 1500)
	is.NoErr(err)
	err = rm.RegisterPlayer(ctx, div2Player, season2ID, 1400)
	is.NoErr(err)

	// Categorize and place
	regs, err := rm.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)

	categorized, err := rm.CategorizeRegistrations(ctx, leagueID, season2ID, regs)
	is.NoErr(err)

	pm := NewPlacementManager(store)
	result, err := pm.PlaceReturningPlayers(ctx, leagueID, season2ID, categorized)
	is.NoErr(err)

	// div1Player should be placed in their original division
	// div2Player should be placed in the lowest division (div 1, since div 2 doesn't exist)
	is.Equal(len(result.PlacedReturning), 1)
	is.Equal(len(result.NeedingRookiePlacement), 0)
	is.Equal(len(result.PlacedInLowestDivision), 1)

	is.Equal(result.PlacedReturning[0].Registration.UserID, div1Player)
	is.Equal(result.PlacedInLowestDivision[0].Registration.UserID, div2Player)

	// Both players should now be in division 1
	div1AfterPlacement, err := store.GetDivisionRegistrations(ctx, result.PlacedReturning[0].DivisionID)
	is.NoErr(err)
	is.True(len(div1AfterPlacement) >= 2) // Should have both players
}
