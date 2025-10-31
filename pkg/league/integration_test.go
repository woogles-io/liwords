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

// TestFullPlacementFlow_20Players tests what happens with 20 total players
func TestFullPlacementFlow_20Players(t *testing.T) {
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

	// Season 1: 20 players all new
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

	// Register 20 players in Season 1
	rm := NewRegistrationManager(store)
	season1Players := []string{}
	for i := 0; i < 20; i++ {
		userID := uuid.NewString()
		season1Players = append(season1Players, userID)
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1500-i*10))
		is.NoErr(err)
	}

	// Categorize - all should be new
	regs1, err := rm.GetSeasonRegistrations(ctx, season1ID)
	is.NoErr(err)

	categorized1, err := rm.CategorizeRegistrations(ctx, leagueID, season1ID, regs1)
	is.NoErr(err)
	is.Equal(len(categorized1), 20)

	// Place rookies - should create 1 rookie division of 20 (at max)
	rookieManager := NewRookieManager(store)
	rookieResult1, err := rookieManager.PlaceRookies(ctx, season1ID, categorized1)
	is.NoErr(err)

	is.Equal(len(rookieResult1.CreatedDivisions), 1)
	is.Equal(len(rookieResult1.PlacedInRookieDivisions), 20)
	is.Equal(rookieResult1.CreatedDivisions[0].DivisionNumber, int32(100))

	// Season 2: All 20 return
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

	// Create 1 regular division (Division 1) for the graduated rookies
	div1S2, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season2ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// All 20 register for Season 2
	for _, userID := range season1Players {
		err = rm.RegisterPlayer(ctx, userID, season2ID, int32(1500))
		is.NoErr(err)
	}

	// Categorize - all should be returning
	regs2, err := rm.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)

	categorized2, err := rm.CategorizeRegistrations(ctx, leagueID, season2ID, regs2)
	is.NoErr(err)
	is.Equal(len(categorized2), 20)

	// All should be categorized as returning
	for _, cp := range categorized2 {
		is.Equal(cp.Category, PlayerCategoryReturning)
	}

	// Place returning players - they should go back to their rookie division
	// But rookie division 100 doesn't exist in Season 2, so they should go to lowest division (Division 1)
	pm := NewPlacementManager(store)
	placementResult, err := pm.PlaceReturningPlayers(ctx, leagueID, season2ID, 2, categorized2)
	is.NoErr(err)

	// All should be placed in the lowest division since their rookie division doesn't exist
	is.Equal(len(placementResult.PlacedReturning), 0)
	is.Equal(len(placementResult.PlacedInLowestDivision), 20)
	is.Equal(placementResult.PlacedInLowestDivision[0].DivisionID, div1S2.Uuid)

	// Verify all 20 are in Division 1
	div1Regs, err := store.GetDivisionRegistrations(ctx, div1S2.Uuid)
	is.NoErr(err)
	is.Equal(len(div1Regs), 20)
}

// TestFullPlacementFlow_21Players tests what happens with 21 total players
func TestFullPlacementFlow_21Players(t *testing.T) {
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

	rm := NewRegistrationManager(store)

	// Season 0: Establish 15 players with history
	season0ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season0ID,
		LeagueID:     leagueID,
		SeasonNumber: 0,
		StartDate:    pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -30), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -16), Valid: true},
		Status:       "COMPLETED",
	})
	is.NoErr(err)

	// Create 2 divisions in Season 0
	div1S0, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season0ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	div2S0, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season0ID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register 15 players in Season 0 (8 in Div 1, 7 in Div 2)
	div1Players := []string{}
	div2Players := []string{}

	for i := 0; i < 8; i++ {
		userID := uuid.NewString()
		div1Players = append(div1Players, userID)
		err = rm.RegisterPlayer(ctx, userID, season0ID, int32(1600-i*10))
		is.NoErr(err)
		err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      userID,
			SeasonID:    season0ID,
			DivisionID:  pgtype.UUID{Bytes: div1S0.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	for i := 0; i < 7; i++ {
		userID := uuid.NewString()
		div2Players = append(div2Players, userID)
		err = rm.RegisterPlayer(ctx, userID, season0ID, int32(1500-i*10))
		is.NoErr(err)
		err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      userID,
			SeasonID:    season0ID,
			DivisionID:  pgtype.UUID{Bytes: div2S0.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Season 1: 15 returning players + 6 new players = 21 total
	season1ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season1ID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       "SCHEDULED",
	})
	is.NoErr(err)

	// Create 2 regular divisions in Season 1
	div1, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season1ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	div2, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season1ID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register the 15 returning players for Season 1
	for _, userID := range div1Players {
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1600))
		is.NoErr(err)
	}
	for _, userID := range div2Players {
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1500))
		is.NoErr(err)
	}

	// Register 6 new players
	newPlayers := []string{}
	for i := 0; i < 6; i++ {
		userID := uuid.NewString()
		newPlayers = append(newPlayers, userID)
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1400-i*10))
		is.NoErr(err)
	}

	// Total: 21 players (15 returning + 6 new)
	regs, err := rm.GetSeasonRegistrations(ctx, season1ID)
	is.NoErr(err)
	is.Equal(len(regs), 21)

	// Categorize
	categorized, err := rm.CategorizeRegistrations(ctx, leagueID, season1ID, regs)
	is.NoErr(err)

	// Filter by category
	rookies := []CategorizedPlayer{}
	returning := []CategorizedPlayer{}
	for _, cp := range categorized {
		if cp.Category == PlayerCategoryNew {
			rookies = append(rookies, cp)
		} else {
			returning = append(returning, cp)
		}
	}

	is.Equal(len(rookies), 6)
	is.Equal(len(returning), 15)

	// Place returning players first
	pm := NewPlacementManager(store)
	placementResult, err := pm.PlaceReturningPlayers(ctx, leagueID, season1ID, 1, returning)
	is.NoErr(err)
	is.Equal(len(placementResult.PlacedReturning), 15)

	// Place rookies (< 10, so they go into regular divisions)
	rookieManager := NewRookieManager(store)
	rookieResult, err := rookieManager.PlaceRookies(ctx, season1ID, rookies)
	is.NoErr(err)

	// Should not create rookie divisions
	is.Equal(len(rookieResult.CreatedDivisions), 0)
	is.Equal(len(rookieResult.PlacedInRegularDivisions), 6)

	// Rookies should be split between Division 1 and Division 2
	// Top 3 go to Division 1, bottom 3 go to Division 2
	div1RookieCount := 0
	div2RookieCount := 0
	for _, placed := range rookieResult.PlacedInRegularDivisions {
		if placed.DivisionID == div1.Uuid {
			div1RookieCount++
		} else if placed.DivisionID == div2.Uuid {
			div2RookieCount++
		}
	}

	is.Equal(div1RookieCount, 3)
	is.Equal(div2RookieCount, 3)

	// Final division sizes:
	// Division 1: 8 returning + 3 rookies = 11
	// Division 2: 7 returning + 3 rookies = 10
	div1Final, err := store.GetDivisionRegistrations(ctx, div1.Uuid)
	is.NoErr(err)
	is.Equal(len(div1Final), 11)

	div2Final, err := store.GetDivisionRegistrations(ctx, div2.Uuid)
	is.NoErr(err)
	is.Equal(len(div2Final), 10)
}

// TestFullPlacementFlow_22Players tests 22 players: 12 returning, 10 new
func TestFullPlacementFlow_22Players(t *testing.T) {
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

	rm := NewRegistrationManager(store)

	// Season 0: Establish 12 players with history
	season0ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season0ID,
		LeagueID:     leagueID,
		SeasonNumber: 0,
		StartDate:    pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -30), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -16), Valid: true},
		Status:       "COMPLETED",
	})
	is.NoErr(err)

	// Create 1 division in Season 0
	div1S0, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season0ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register 12 players in Season 0
	returningPlayers := []string{}
	for i := 0; i < 12; i++ {
		userID := uuid.NewString()
		returningPlayers = append(returningPlayers, userID)
		err = rm.RegisterPlayer(ctx, userID, season0ID, int32(1600-i*10))
		is.NoErr(err)
		err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      userID,
			SeasonID:    season0ID,
			DivisionID:  pgtype.UUID{Bytes: div1S0.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Season 1: 12 returning + 10 new = 22 total
	season1ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season1ID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       "SCHEDULED",
	})
	is.NoErr(err)

	// Create 1 regular division in Season 1
	div1, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season1ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register the 12 returning players for Season 1
	for _, userID := range returningPlayers {
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1600))
		is.NoErr(err)
	}

	// Register 10 new players
	for i := 0; i < 10; i++ {
		userID := uuid.NewString()
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1400-i*10))
		is.NoErr(err)
	}

	// Total: 22 players (12 returning + 10 new)
	regs, err := rm.GetSeasonRegistrations(ctx, season1ID)
	is.NoErr(err)
	is.Equal(len(regs), 22)

	// Categorize
	categorized, err := rm.CategorizeRegistrations(ctx, leagueID, season1ID, regs)
	is.NoErr(err)

	rookies := []CategorizedPlayer{}
	returning := []CategorizedPlayer{}
	for _, cp := range categorized {
		if cp.Category == PlayerCategoryNew {
			rookies = append(rookies, cp)
		} else {
			returning = append(returning, cp)
		}
	}

	is.Equal(len(rookies), 10)
	is.Equal(len(returning), 12)

	// Place returning players first
	pm := NewPlacementManager(store)
	placementResult, err := pm.PlaceReturningPlayers(ctx, leagueID, season1ID, 1, returning)
	is.NoErr(err)
	is.Equal(len(placementResult.PlacedReturning), 12)

	// Place rookies (exactly 10, so creates 1 rookie division)
	rookieManager := NewRookieManager(store)
	rookieResult, err := rookieManager.PlaceRookies(ctx, season1ID, rookies)
	is.NoErr(err)

	// Should create 1 rookie division of 10
	is.Equal(len(rookieResult.CreatedDivisions), 1)
	is.Equal(len(rookieResult.PlacedInRookieDivisions), 10)
	is.Equal(rookieResult.CreatedDivisions[0].DivisionNumber, int32(100))

	// Final state:
	// Division 1 (regular): 12 players
	// Division 100 (rookie): 10 players
	div1Final, err := store.GetDivisionRegistrations(ctx, div1.Uuid)
	is.NoErr(err)
	is.Equal(len(div1Final), 12)

	rookieDiv, err := store.GetDivisionRegistrations(ctx, rookieResult.CreatedDivisions[0].Uuid)
	is.NoErr(err)
	is.Equal(len(rookieDiv), 10)
}

// TestFullPlacementFlow_23Players tests 23 players: 12 returning, 11 new
func TestFullPlacementFlow_23Players(t *testing.T) {
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

	rm := NewRegistrationManager(store)

	// Season 0: Establish 12 players with history
	season0ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season0ID,
		LeagueID:     leagueID,
		SeasonNumber: 0,
		StartDate:    pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -30), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -16), Valid: true},
		Status:       "COMPLETED",
	})
	is.NoErr(err)

	// Create 1 division in Season 0
	div1S0, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season0ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register 12 players in Season 0
	returningPlayers := []string{}
	for i := 0; i < 12; i++ {
		userID := uuid.NewString()
		returningPlayers = append(returningPlayers, userID)
		err = rm.RegisterPlayer(ctx, userID, season0ID, int32(1600-i*10))
		is.NoErr(err)
		err = store.UpdateRegistrationDivision(ctx, models.UpdateRegistrationDivisionParams{
			UserID:      userID,
			SeasonID:    season0ID,
			DivisionID:  pgtype.UUID{Bytes: div1S0.Uuid, Valid: true},
			FirstsCount: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Season 1: 12 returning + 11 new = 23 total
	season1ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season1ID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       "SCHEDULED",
	})
	is.NoErr(err)

	// Create 1 regular division in Season 1
	div1, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       season1ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register the 12 returning players for Season 1
	for _, userID := range returningPlayers {
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1600))
		is.NoErr(err)
	}

	// Register 11 new players
	for i := 0; i < 11; i++ {
		userID := uuid.NewString()
		err = rm.RegisterPlayer(ctx, userID, season1ID, int32(1400-i*10))
		is.NoErr(err)
	}

	// Total: 23 players (12 returning + 11 new)
	regs, err := rm.GetSeasonRegistrations(ctx, season1ID)
	is.NoErr(err)
	is.Equal(len(regs), 23)

	// Categorize
	categorized, err := rm.CategorizeRegistrations(ctx, leagueID, season1ID, regs)
	is.NoErr(err)

	rookies := []CategorizedPlayer{}
	returning := []CategorizedPlayer{}
	for _, cp := range categorized {
		if cp.Category == PlayerCategoryNew {
			rookies = append(rookies, cp)
		} else {
			returning = append(returning, cp)
		}
	}

	is.Equal(len(rookies), 11)
	is.Equal(len(returning), 12)

	// Place returning players first
	pm := NewPlacementManager(store)
	placementResult, err := pm.PlaceReturningPlayers(ctx, leagueID, season1ID, 1, returning)
	is.NoErr(err)
	is.Equal(len(placementResult.PlacedReturning), 12)

	// Place rookies (11, so creates 1 rookie division of 11)
	rookieManager := NewRookieManager(store)
	rookieResult, err := rookieManager.PlaceRookies(ctx, season1ID, rookies)
	is.NoErr(err)

	// Should create 1 rookie division of 11
	is.Equal(len(rookieResult.CreatedDivisions), 1)
	is.Equal(len(rookieResult.PlacedInRookieDivisions), 11)
	is.Equal(rookieResult.CreatedDivisions[0].DivisionNumber, int32(100))

	// Final state:
	// Division 1 (regular): 12 players
	// Division 100 (rookie): 11 players
	div1Final, err := store.GetDivisionRegistrations(ctx, div1.Uuid)
	is.NoErr(err)
	is.Equal(len(div1Final), 12)

	rookieDiv, err := store.GetDivisionRegistrations(ctx, rookieResult.CreatedDivisions[0].Uuid)
	is.NoErr(err)
	is.Equal(len(rookieDiv), 11)
}
