package league

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/models"
)

// TestMergeDivisions_BasicMerge tests merging Division 3 into Division 2
func TestMergeDivisions_BasicMerge(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	store, cleanup := setupTest(t)
	defer cleanup()

	mdm := NewManualDivisionManager(store)

	// Create league and season
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

	// Create 3 divisions
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

	div3ID := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div3ID,
		SeasonID:       seasonID,
		DivisionNumber: 3,
		DivisionName:   pgtype.Text{String: "Division 3", Valid: true},
	})
	is.NoErr(err)

	// Register 3 players in Division 3
	player1 := uuid.New().String()
	player2 := uuid.New().String()
	player3 := uuid.New().String()

	for _, playerID := range []string{player1, player2, player3} {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:      playerID,
			SeasonID:    seasonID,
			DivisionID:  pgtype.UUID{Bytes: div3ID, Valid: true},
			SeasonsAway: pgtype.Int4{Int32: 0, Valid: true},
		})
		is.NoErr(err)
	}

	// Merge Division 3 into Division 2
	result, err := mdm.MergeDivisions(ctx, seasonID, div2ID, div3ID)
	is.NoErr(err)
	is.Equal(result.PlayersAffected, 3)
	is.Equal(result.DeletedDivisionID, div3ID)
	is.Equal(result.ReceivingDivisionID, div2ID)

	// Verify Division 3 no longer exists
	_, err = store.GetDivision(ctx, div3ID)
	is.True(err != nil) // Division should be deleted

	// Verify all 3 players are now in Division 2
	div2Players, err := store.GetDivisionRegistrations(ctx, div2ID)
	is.NoErr(err)
	is.Equal(len(div2Players), 3)

	// Verify divisions were renumbered (Division 3 deleted, so we should have 1, 2)
	divisions, err := store.GetDivisionsBySeason(ctx, seasonID)
	is.NoErr(err)

	regularDivs := 0
	for _, div := range divisions {
		if div.DivisionNumber < RookieDivisionNumberBase {
			regularDivs++
		}
	}
	is.Equal(regularDivs, 2) // Only 2 divisions left
}

// TestMergeDivisions_WithRenumbering tests that divisions get renumbered after merge
func TestMergeDivisions_WithRenumbering(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	store, cleanup := setupTest(t)
	defer cleanup()

	mdm := NewManualDivisionManager(store)

	// Create league and season
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

	// Create 5 divisions: [1, 2, 3, 4, 5]
	divIDs := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		divIDs[i] = uuid.New()
		_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
			Uuid:           divIDs[i],
			SeasonID:       seasonID,
			DivisionNumber: int32(i + 1),
			DivisionName:   pgtype.Text{String: fmt.Sprintf("Division %d", i+1), Valid: true},
		})
		is.NoErr(err)
	}

	// Merge Division 3 into Division 2
	// Expected result: [1, 2, 3, 4] (old 4→3, old 5→4)
	result, err := mdm.MergeDivisions(ctx, seasonID, divIDs[1], divIDs[2])
	is.NoErr(err)
	is.True(result.DivisionsRenumbered > 0) // Should have renumbered divisions

	// Verify final division numbers are sequential: 1, 2, 3, 4
	divisions, err := store.GetDivisionsBySeason(ctx, seasonID)
	is.NoErr(err)

	divNumbers := []int32{}
	for _, div := range divisions {
		if div.DivisionNumber < RookieDivisionNumberBase {
			divNumbers = append(divNumbers, div.DivisionNumber)
		}
	}

	// Sort for comparison
	sort.Slice(divNumbers, func(i, j int) bool {
		return divNumbers[i] < divNumbers[j]
	})

	is.Equal(len(divNumbers), 4)
	is.Equal(divNumbers[0], int32(1))
	is.Equal(divNumbers[1], int32(2))
	is.Equal(divNumbers[2], int32(3))
	is.Equal(divNumbers[3], int32(4))
}

// TestMovePlayer_Success tests moving a player between divisions
func TestMovePlayer_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	store, cleanup := setupTest(t)
	defer cleanup()

	mdm := NewManualDivisionManager(store)

	// Create league and season
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

	// Create 2 divisions
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

	// Register player in Division 1
	playerID := uuid.New().String()
	_, err = store.RegisterPlayer(ctx, models.RegisterPlayerParams{
		UserID:      playerID,
		SeasonID:    seasonID,
		DivisionID:  pgtype.UUID{Bytes: div1ID, Valid: true},
		SeasonsAway: pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Move player from Division 1 to Division 2
	result, err := mdm.MovePlayer(ctx, playerID, seasonID, div1ID, div2ID)
	is.NoErr(err)
	is.Equal(result.Success, true)
	is.Equal(result.UserID, playerID)
	is.Equal(result.PreviousDivisionID, div1ID)
	is.Equal(result.NewDivisionID, div2ID)

	// Verify player is now in Division 2
	reg, err := store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
		SeasonID: seasonID,
		UserID:   playerID,
	})
	is.NoErr(err)
	is.True(reg.DivisionID.Valid)
	is.Equal(uuid.UUID(reg.DivisionID.Bytes), div2ID)
}

// TestMovePlayer_InvalidDivision tests error handling for invalid division
func TestMovePlayer_InvalidDivision(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	store, cleanup := setupTest(t)
	defer cleanup()

	mdm := NewManualDivisionManager(store)

	// Create league and season
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

	// Create 1 division
	div1ID := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div1ID,
		SeasonID:       seasonID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
	})
	is.NoErr(err)

	// Register player in Division 1
	playerID := uuid.New().String()
	_, err = store.RegisterPlayer(ctx, models.RegisterPlayerParams{
		UserID:      playerID,
		SeasonID:    seasonID,
		DivisionID:  pgtype.UUID{Bytes: div1ID, Valid: true},
		SeasonsAway: pgtype.Int4{Int32: 0, Valid: true},
	})
	is.NoErr(err)

	// Try to move player from wrong division - should fail
	wrongDivID := uuid.New()
	_, err = mdm.MovePlayer(ctx, playerID, seasonID, wrongDivID, div1ID)
	is.True(err != nil) // Should return error
}

// TestCreateDivision_AtEnd tests creating a new division at the bottom
func TestCreateDivision_AtEnd(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	store, cleanup := setupTest(t)
	defer cleanup()

	mdm := NewManualDivisionManager(store)

	// Create league and season
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

	// Create 2 divisions
	for i := 1; i <= 2; i++ {
		_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
			Uuid:           uuid.New(),
			SeasonID:       seasonID,
			DivisionNumber: int32(i),
			DivisionName:   pgtype.Text{String: fmt.Sprintf("Division %d", i), Valid: true},
		})
		is.NoErr(err)
	}

	// Create new Division 3 at the end
	newDiv, err := mdm.CreateDivision(ctx, seasonID, 3, "Division 3")
	is.NoErr(err)
	is.Equal(newDiv.DivisionNumber, int32(3))

	// Verify we now have 3 divisions numbered 1, 2, 3
	divisions, err := store.GetDivisionsBySeason(ctx, seasonID)
	is.NoErr(err)

	regularDivs := 0
	for _, div := range divisions {
		if div.DivisionNumber < RookieDivisionNumberBase {
			regularDivs++
		}
	}
	is.Equal(regularDivs, 3)
}

// TestCreateDivision_InsertMiddle tests creating a division in the middle and renumbering
func TestCreateDivision_InsertMiddle(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	store, cleanup := setupTest(t)
	defer cleanup()

	mdm := NewManualDivisionManager(store)

	// Create league and season
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

	// Create divisions 1, 2, 3
	divIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		divIDs[i] = uuid.New()
		_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
			Uuid:           divIDs[i],
			SeasonID:       seasonID,
			DivisionNumber: int32(i + 1),
			DivisionName:   pgtype.Text{String: fmt.Sprintf("Division %d", i+1), Valid: true},
		})
		is.NoErr(err)
	}

	// Insert new Division 2 in the middle
	// Expected: old Division 2 becomes Division 3, old Division 3 becomes Division 4
	newDiv, err := mdm.CreateDivision(ctx, seasonID, 2, "New Division 2")
	is.NoErr(err)
	is.Equal(newDiv.DivisionNumber, int32(2))

	// Verify we now have 4 divisions numbered 1, 2, 3, 4
	divisions, err := store.GetDivisionsBySeason(ctx, seasonID)
	is.NoErr(err)

	divNumbers := []int32{}
	for _, div := range divisions {
		if div.DivisionNumber < RookieDivisionNumberBase {
			divNumbers = append(divNumbers, div.DivisionNumber)
		}
	}

	sort.Slice(divNumbers, func(i, j int) bool {
		return divNumbers[i] < divNumbers[j]
	})

	is.Equal(len(divNumbers), 4)
	is.Equal(divNumbers[0], int32(1))
	is.Equal(divNumbers[1], int32(2))
	is.Equal(divNumbers[2], int32(3))
	is.Equal(divNumbers[3], int32(4))

	// Verify the old Division 2 got shifted to Division 3
	oldDiv2, err := store.GetDivision(ctx, divIDs[1])
	is.NoErr(err)
	is.Equal(oldDiv2.DivisionNumber, int32(3)) // Was 2, now 3
}
