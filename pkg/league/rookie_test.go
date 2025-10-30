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

func TestPlaceRookies_FewRookies_PlaceInRegularDivisions(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	store, cleanup := setupTest(t)
	defer cleanup()

	// Create league and season
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

	seasonID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       "SCHEDULED",
	})
	is.NoErr(err)

	// Create 3 regular divisions
	div1, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       seasonID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	div2, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       seasonID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	div3, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       seasonID,
		DivisionNumber: 3,
		DivisionName:   pgtype.Text{String: "Division 3", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register 6 rookies with varying ratings
	rm := NewRegistrationManager(store)
	rookieRatings := []int32{1600, 1550, 1500, 1450, 1400, 1350}
	rookies := []CategorizedPlayer{}

	for i, rating := range rookieRatings {
		userID := uuid.NewString()
		err = rm.RegisterPlayer(ctx, userID, seasonID, rating)
		is.NoErr(err)

		reg, err := store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			UserID:   userID,
			SeasonID: seasonID,
		})
		is.NoErr(err)

		rookies = append(rookies, CategorizedPlayer{
			Registration: reg,
			Category:     PlayerCategoryNew,
			Rating:       rating,
		})
		_ = i
	}

	// Place rookies
	rookieManager := NewRookieManager(store)
	result, err := rookieManager.PlaceRookies(ctx, seasonID, rookies)
	is.NoErr(err)

	// Should not create rookie divisions
	is.Equal(len(result.CreatedDivisions), 0)
	is.Equal(len(result.PlacedInRookieDivisions), 0)

	// All 6 should be placed in regular divisions
	is.Equal(len(result.PlacedInRegularDivisions), 6)

	// Top 3 (1600, 1550, 1500) should be in division 2
	// Bottom 3 (1450, 1400, 1350) should be in division 3
	div2Count := 0
	div3Count := 0
	for _, placed := range result.PlacedInRegularDivisions {
		if placed.DivisionID == div2.Uuid {
			div2Count++
			// Should be higher rated players
			is.True(placed.Rating >= 1500)
		} else if placed.DivisionID == div3.Uuid {
			div3Count++
			// Should be lower rated players
			is.True(placed.Rating <= 1450)
		}
	}

	is.Equal(div2Count, 3)
	is.Equal(div3Count, 3)

	// Division 1 should be empty
	div1Regs, err := store.GetDivisionRegistrations(ctx, div1.Uuid)
	is.NoErr(err)
	is.Equal(len(div1Regs), 0)
}

func TestPlaceRookies_FewRookies_OneDivision(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	store, cleanup := setupTest(t)
	defer cleanup()

	// Create league and season
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

	seasonID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       "SCHEDULED",
	})
	is.NoErr(err)

	// Create only 1 regular division
	div1, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           uuid.New(),
		SeasonID:       seasonID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Register 5 rookies with varying ratings
	rm := NewRegistrationManager(store)
	rookieRatings := []int32{1600, 1550, 1500, 1450, 1400}
	rookies := []CategorizedPlayer{}

	for _, rating := range rookieRatings {
		userID := uuid.NewString()
		err = rm.RegisterPlayer(ctx, userID, seasonID, rating)
		is.NoErr(err)

		reg, err := store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			UserID:   userID,
			SeasonID: seasonID,
		})
		is.NoErr(err)

		rookies = append(rookies, CategorizedPlayer{
			Registration: reg,
			Category:     PlayerCategoryNew,
			Rating:       rating,
		})
	}

	// Place rookies
	rookieManager := NewRookieManager(store)
	result, err := rookieManager.PlaceRookies(ctx, seasonID, rookies)
	is.NoErr(err)

	// Should not create rookie divisions
	is.Equal(len(result.CreatedDivisions), 0)
	is.Equal(len(result.PlacedInRookieDivisions), 0)

	// All 5 should be placed in the single division
	is.Equal(len(result.PlacedInRegularDivisions), 5)

	// All should be in division 1
	for _, placed := range result.PlacedInRegularDivisions {
		is.Equal(placed.DivisionID, div1.Uuid)
	}

	// Verify all are in the division
	div1Regs, err := store.GetDivisionRegistrations(ctx, div1.Uuid)
	is.NoErr(err)
	is.Equal(len(div1Regs), 5)
}

func TestPlaceRookies_TenRookies_CreateOneDivision(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	store, cleanup := setupTest(t)
	defer cleanup()

	// Create league and season
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

	seasonID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       "SCHEDULED",
	})
	is.NoErr(err)

	// Register 10 rookies
	rm := NewRegistrationManager(store)
	rookies := []CategorizedPlayer{}

	for i := 0; i < 10; i++ {
		userID := uuid.NewString()
		rating := int32(1500 - i*10)
		err = rm.RegisterPlayer(ctx, userID, seasonID, rating)
		is.NoErr(err)

		reg, err := store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			UserID:   userID,
			SeasonID: seasonID,
		})
		is.NoErr(err)

		rookies = append(rookies, CategorizedPlayer{
			Registration: reg,
			Category:     PlayerCategoryNew,
			Rating:       rating,
		})
	}

	// Place rookies
	rookieManager := NewRookieManager(store)
	result, err := rookieManager.PlaceRookies(ctx, seasonID, rookies)
	is.NoErr(err)

	// Should create 1 rookie division
	is.Equal(len(result.CreatedDivisions), 1)
	is.Equal(len(result.PlacedInRookieDivisions), 10)
	is.Equal(len(result.PlacedInRegularDivisions), 0)

	// Verify division number is 100
	is.Equal(result.CreatedDivisions[0].DivisionNumber, int32(100))

	// Verify division name
	is.True(result.CreatedDivisions[0].DivisionName.Valid)
	is.Equal(result.CreatedDivisions[0].DivisionName.String, "Rookie Division 1")
}

func TestPlaceRookies_TwentyRookies_CreateOneDivision(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	store, cleanup := setupTest(t)
	defer cleanup()

	// Create league and season
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

	seasonID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       "SCHEDULED",
	})
	is.NoErr(err)

	// Register 20 rookies
	rm := NewRegistrationManager(store)
	rookies := []CategorizedPlayer{}

	for i := 0; i < 20; i++ {
		userID := uuid.NewString()
		rating := int32(1600 - i*10)
		err = rm.RegisterPlayer(ctx, userID, seasonID, rating)
		is.NoErr(err)

		reg, err := store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			UserID:   userID,
			SeasonID: seasonID,
		})
		is.NoErr(err)

		rookies = append(rookies, CategorizedPlayer{
			Registration: reg,
			Category:     PlayerCategoryNew,
			Rating:       rating,
		})
	}

	// Place rookies
	rookieManager := NewRookieManager(store)
	result, err := rookieManager.PlaceRookies(ctx, seasonID, rookies)
	is.NoErr(err)

	// Should create 1 rookie division of 20 (at max size)
	is.Equal(len(result.CreatedDivisions), 1)
	is.Equal(len(result.PlacedInRookieDivisions), 20)
	is.Equal(len(result.PlacedInRegularDivisions), 0)

	// Verify division number
	is.Equal(result.CreatedDivisions[0].DivisionNumber, int32(100))

	// Verify division has all 20 players
	div1Regs, err := store.GetDivisionRegistrations(ctx, result.CreatedDivisions[0].Uuid)
	is.NoErr(err)
	is.Equal(len(div1Regs), 20)
}

func TestPlaceRookies_ThirtyRookies_CreateBalancedDivisions(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	store, cleanup := setupTest(t)
	defer cleanup()

	// Create league and season
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

	seasonID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       "SCHEDULED",
	})
	is.NoErr(err)

	// Register 30 rookies
	rm := NewRegistrationManager(store)
	rookies := []CategorizedPlayer{}

	for i := 0; i < 30; i++ {
		userID := uuid.NewString()
		rating := int32(1700 - i*10)
		err = rm.RegisterPlayer(ctx, userID, seasonID, rating)
		is.NoErr(err)

		reg, err := store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			UserID:   userID,
			SeasonID: seasonID,
		})
		is.NoErr(err)

		rookies = append(rookies, CategorizedPlayer{
			Registration: reg,
			Category:     PlayerCategoryNew,
			Rating:       rating,
		})
	}

	// Place rookies
	rookieManager := NewRookieManager(store)
	result, err := rookieManager.PlaceRookies(ctx, seasonID, rookies)
	is.NoErr(err)

	// Should create 2 rookie divisions (30 / 15 = 2)
	is.Equal(len(result.CreatedDivisions), 2)
	is.Equal(len(result.PlacedInRookieDivisions), 30)

	// Each division should have 15 players
	for _, div := range result.CreatedDivisions {
		regs, err := store.GetDivisionRegistrations(ctx, div.Uuid)
		is.NoErr(err)
		is.Equal(len(regs), 15)
	}
}

func TestCalculateRookieDivisionSizes(t *testing.T) {
	is := is.New(t)

	tests := []struct {
		numRookies     int
		expectedSizes  []int
		description    string
	}{
		{9, []int{}, "fewer than 10 rookies"},
		{10, []int{10}, "exactly 10 rookies"},
		{15, []int{15}, "exactly 15 rookies (target)"},
		{16, []int{16}, "16 rookies - one division (within max)"},
		{20, []int{20}, "20 rookies - one division (at max)"},
		{21, []int{11, 10}, "21 rookies - two divisions"},
		{25, []int{13, 12}, "25 rookies - balanced divisions"},
		{30, []int{15, 15}, "30 rookies - two divisions at target"},
		{31, []int{11, 10, 10}, "31 rookies - three divisions"},
		{38, []int{13, 13, 12}, "38 rookies - three divisions"},
		{40, []int{14, 13, 13}, "40 rookies - three divisions"},
		{45, []int{15, 15, 15}, "45 rookies - three divisions at target"},
		{60, []int{15, 15, 15, 15}, "60 rookies - four divisions at target"},
		{80, []int{14, 14, 13, 13, 13, 13}, "80 rookies - six divisions"},
		{100, []int{15, 15, 14, 14, 14, 14, 14}, "100 rookies - seven divisions"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			sizes := calculateRookieDivisionSizes(tt.numRookies)
			is.Equal(len(sizes), len(tt.expectedSizes))

			for i, size := range sizes {
				is.Equal(size, tt.expectedSizes[i])
				// Verify size constraints
				is.True(size >= MinRookieDivisionSize)
				is.True(size <= MaxRookieDivisionSize)
			}

			// Verify total equals numRookies (only if divisions were created)
			if len(sizes) > 0 {
				total := 0
				for _, size := range sizes {
					total += size
				}
				is.Equal(total, tt.numRookies)
			}
		})
	}
}
