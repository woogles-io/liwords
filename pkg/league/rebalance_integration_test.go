package league

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	leaguestore "github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
)

const integrationPkg = "league_rebalance_integration_test"

func setupIntegrationTest(t *testing.T) (*stores.Stores, func()) {
	err := common.RecreateTestDB(integrationPkg)
	if err != nil {
		t.Fatal(err)
	}

	pool, err := common.OpenTestingDB(integrationPkg)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.DBConnDSN = common.TestingPostgresConnDSN(integrationPkg)

	leagueStore, err := leaguestore.NewDBStore(cfg, pool)
	if err != nil {
		t.Fatal(err)
	}

	userStore, err := user.NewDBStore(pool)
	if err != nil {
		t.Fatal(err)
	}

	// Create test users
	err = createIntegrationTestUsers(pool)
	if err != nil {
		t.Fatal(err)
	}

	// Create minimal stores object for testing
	allStores := &stores.Stores{
		LeagueStore: leagueStore,
		UserStore:   userStore,
	}

	cleanup := func() {
		pool.Close()
	}

	return allStores, cleanup
}

// createIntegrationTestUsers creates test users in the database for integration tests
func createIntegrationTestUsers(pool *pgxpool.Pool) error {
	ustore, err := user.NewDBStore(pool)
	if err != nil {
		return err
	}
	// Don't disconnect - the pool is shared with the league store

	ctx := context.Background()

	// Create 100 test users (enough for all integration tests)
	for i := 1; i <= 100; i++ {
		u := &entity.User{
			Username: fmt.Sprintf("testuser%d", i),
			Email:    fmt.Sprintf("testuser%d@test.com", i),
			UUID:     fmt.Sprintf("test-uuid-%d", i),
		}
		err = ustore.New(ctx, u)
		if err != nil {
			return fmt.Errorf("failed to create test user %d: %w", i, err)
		}
	}

	return nil
}

// Helper to create a league and season for testing
func createLeagueAndSeason(t *testing.T, ctx context.Context, allStores *stores.Stores) (uuid.UUID, uuid.UUID) {
	store := allStores.LeagueStore
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
	if err != nil {
		t.Fatal(err)
	}

	seasonID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         seasonID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	if err != nil {
		t.Fatal(err)
	}

	return leagueID, seasonID
}

func TestCreateRookieDivisionsAndAssign_20Rookies(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	_, seasonID := createLeagueAndSeason(t, ctx, allStores)

	// Create 20 rookie players with varying ratings
	for i := 0; i < 20; i++ {
		userDBID := int32(i + 1)

		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:   userDBID,
			SeasonID: seasonID,
			Status:   pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	// Fetch the registrations
	regs, err := store.GetSeasonRegistrations(ctx, seasonID)
	is.NoErr(err)
	is.Equal(len(regs), 20)

	// Create CategorizedPlayer list
	rookies := make([]CategorizedPlayer, 20)
	for i := 0; i < 20; i++ {
		rookies[i] = CategorizedPlayer{
			Registration: regs[i],
			Category:     PlayerCategoryNew,
			Rating:       0,
		}
	}

	// Sort by rating (highest first) as the orchestrator does
	sort.Slice(rookies, func(i, j int) bool {
		return rookies[i].Rating > rookies[j].Rating
	})

	// Create rookie divisions and assign
	rm := NewRebalanceManager(allStores)
	result, err := rm.CreateRookieDivisionsAndAssign(ctx, seasonID, rookies, 15)
	is.NoErr(err)

	// Verify results
	is.Equal(len(result.CreatedDivisions), 1)                // Should create 1 division
	is.Equal(len(result.PlacedInRookieDivisions), 20)        // All 20 assigned
	is.Equal(len(result.PlacedInRegularDivisions), 0)        // None in regular

	// Verify division properties
	div := result.CreatedDivisions[0]
	is.Equal(div.DivisionNumber, int32(RookieDivisionNumberBase)) // Division 100
	is.Equal(div.DivisionName.String, "Rookie Division 1")
	is.Equal(div.PlayerCount.Int32, int32(20))

	// Verify all players are assigned to the division
	for _, placed := range result.PlacedInRookieDivisions {
		is.Equal(placed.DivisionID, div.Uuid)
		is.Equal(placed.DivisionName, "Rookie Division 1")
	}

	// Verify players are sorted by rating (highest first)
	for i := 1; i < len(result.PlacedInRookieDivisions); i++ {
		prev := result.PlacedInRookieDivisions[i-1].Rating
		curr := result.PlacedInRookieDivisions[i].Rating
		is.True(prev >= curr) // Should be descending order
	}

	// Verify in database
	divs, err := store.GetDivisionsBySeason(ctx, seasonID)
	is.NoErr(err)
	is.Equal(len(divs), 1)
	is.Equal(divs[0].DivisionNumber, int32(RookieDivisionNumberBase))

	divRegs, err := store.GetDivisionRegistrations(ctx, div.Uuid)
	is.NoErr(err)
	is.Equal(len(divRegs), 20)
}

func TestCreateRookieDivisionsAndAssign_45Rookies(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	_, seasonID := createLeagueAndSeason(t, ctx, allStores)

	// Create 45 rookie players
	for i := 0; i < 45; i++ {
		userDBID := int32(i + 1)

		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:   userDBID,
			SeasonID: seasonID,
			Status:   pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	// Fetch the registrations
	regs, err := store.GetSeasonRegistrations(ctx, seasonID)
	is.NoErr(err)
	is.Equal(len(regs), 45)

	// Create CategorizedPlayer list
	rookies := make([]CategorizedPlayer, 45)
	for i := 0; i < 45; i++ {
		rookies[i] = CategorizedPlayer{
			Registration: regs[i],
			Category:     PlayerCategoryNew,
			Rating:       0,
		}
	}

	// Sort by rating (highest first)
	sort.Slice(rookies, func(i, j int) bool {
		return rookies[i].Rating > rookies[j].Rating
	})

	// Create rookie divisions and assign
	rm := NewRebalanceManager(allStores)
	result, err := rm.CreateRookieDivisionsAndAssign(ctx, seasonID, rookies, 15)
	is.NoErr(err)

	// Verify results - should create 3 divisions of 15 each
	is.Equal(len(result.CreatedDivisions), 3)
	is.Equal(len(result.PlacedInRookieDivisions), 45)

	// Verify division properties
	for i, div := range result.CreatedDivisions {
		expectedDivNum := int32(RookieDivisionNumberBase + i)
		expectedDivName := "Rookie Division " + string(rune('1'+i))

		is.Equal(div.DivisionNumber, expectedDivNum)
		is.True(div.DivisionName.String == expectedDivName)
		is.Equal(div.PlayerCount.Int32, int32(15)) // Each should have 15
	}

	// Verify player distribution
	divisionCounts := make(map[uuid.UUID]int)
	for _, placed := range result.PlacedInRookieDivisions {
		divisionCounts[placed.DivisionID]++
	}
	is.Equal(len(divisionCounts), 3) // 3 different divisions
	for _, count := range divisionCounts {
		is.Equal(count, 15) // Each division has 15
	}

	// Verify top 15 players went to Division 100
	div1Players := 0
	for i, placed := range result.PlacedInRookieDivisions {
		if i < 15 {
			is.Equal(placed.DivisionID, result.CreatedDivisions[0].Uuid) // First 15 to Div 100
			div1Players++
		}
	}
	is.Equal(div1Players, 15)
}

func TestCreateRookieDivisionsAndAssign_TooFewRookies(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	_, seasonID := createLeagueAndSeason(t, ctx, allStores)

	// Create only 8 rookies (below minimum)
	for i := 0; i < 8; i++ {
		userDBID := int32(i + 1)

		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:   userDBID,
			SeasonID: seasonID,
			Status:   pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	// Fetch the registrations
	regs, err := store.GetSeasonRegistrations(ctx, seasonID)
	is.NoErr(err)
	is.Equal(len(regs), 8)

	// Create CategorizedPlayer list
	rookies := make([]CategorizedPlayer, 8)
	for i := 0; i < 8; i++ {
		rookies[i] = CategorizedPlayer{
			Registration: regs[i],
			Category:     PlayerCategoryNew,
			Rating:       0,
		}
	}

	// Attempt to create rookie divisions - should fail
	rm := NewRebalanceManager(allStores)
	_, err = rm.CreateRookieDivisionsAndAssign(ctx, seasonID, rookies, 15)
	is.True(err != nil) // Should return error
}

func TestCreateRookieDivisionsAndAssign_100Rookies(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	_, seasonID := createLeagueAndSeason(t, ctx, allStores)

	// Create 100 rookie players
	for i := 0; i < 100; i++ {
		userDBID := int32(i + 1)

		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:   userDBID,
			SeasonID: seasonID,
			Status:   pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	// Fetch the registrations
	regs, err := store.GetSeasonRegistrations(ctx, seasonID)
	is.NoErr(err)
	is.Equal(len(regs), 100)

	// Create CategorizedPlayer list
	rookies := make([]CategorizedPlayer, 100)
	for i := 0; i < 100; i++ {
		rookies[i] = CategorizedPlayer{
			Registration: regs[i],
			Category:     PlayerCategoryNew,
			Rating:       0,
		}
	}

	// Sort by rating
	sort.Slice(rookies, func(i, j int) bool {
		return rookies[i].Rating > rookies[j].Rating
	})

	// Create rookie divisions
	rm := NewRebalanceManager(allStores)
	result, err := rm.CreateRookieDivisionsAndAssign(ctx, seasonID, rookies, 15)
	is.NoErr(err)

	// Verify all 100 assigned
	is.Equal(len(result.PlacedInRookieDivisions), 100)

	// Verify all divisions are within size constraints
	divisionCounts := make(map[uuid.UUID]int)
	for _, placed := range result.PlacedInRookieDivisions {
		divisionCounts[placed.DivisionID]++
	}

	for _, count := range divisionCounts {
		is.True(count >= MinRookieDivisionSize) // At least 10
		is.True(count <= MaxRookieDivisionSize) // At most 20
	}

	// Verify divisions were created in database
	divs, err := store.GetDivisionsBySeason(ctx, seasonID)
	is.NoErr(err)
	is.Equal(len(divs), len(result.CreatedDivisions))

	// Verify all division numbers start at RookieDivisionNumberBase
	for i, div := range divs {
		is.Equal(div.DivisionNumber, int32(RookieDivisionNumberBase+i))
	}
}

func TestRebalanceDivisions_30ReturningPlayers(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	leagueID, season1ID := createLeagueAndSeason(t, ctx, allStores)

	// Create 2 divisions in Season 1
	div1 := uuid.New()
	_, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div1,
		SeasonID:       season1ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 15, Valid: true},
	})
	is.NoErr(err)

	div2 := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div2,
		SeasonID:       season1ID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 15, Valid: true},
	})
	is.NoErr(err)

	// Register 30 players in Season 1 (15 in each division)
	playerIDs := make([]int32, 30)
	for i := 0; i < 30; i++ {
		userDBID := int32(i + 1)
		playerIDs[i] = userDBID

		divID := div1
		if i >= 15 {
			divID = div2
		}

		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:     userDBID,
			SeasonID:   season1ID,
			Status:     pgtype.Text{String: "REGISTERED", Valid: true},
			DivisionID: pgtype.UUID{Bytes: divID, Valid: true},
		})
		is.NoErr(err)

		// Create standings for each player
		err = store.UpsertStanding(ctx, models.UpsertStandingParams{
			UserID:     userDBID,
			DivisionID: divID,
			Wins:       pgtype.Int4{Int32: int32(10 - i%15), Valid: true},
			Losses:     pgtype.Int4{Int32: int32(i % 15), Valid: true},
			Draws:      pgtype.Int4{Int32: 0, Valid: true},
			Spread:     pgtype.Int4{Int32: int32(100 - i*3), Valid: true},
		})
		is.NoErr(err)
	}

	// Recalculate ranks
	err = store.RecalculateRanks(ctx, div1)
	is.NoErr(err)
	err = store.RecalculateRanks(ctx, div2)
	is.NoErr(err)

	// Create Season 2
	season2ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season2ID,
		LeagueID:     leagueID,
		SeasonNumber: 2,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)

	// Register all 30 players for Season 2
	for _, userID := range playerIDs {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:   userID,
			SeasonID: season2ID,
			Status:   pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	// Fetch registrations
	regs, err := store.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)
	is.Equal(len(regs), 30)

	// Create CategorizedPlayer list
	categorized := make([]CategorizedPlayer, 30)
	for i := 0; i < 30; i++ {
		categorized[i] = CategorizedPlayer{
			Registration: regs[i],
			Category:     PlayerCategoryReturning,
			Rating:       0,
		}
	}

	// Run rebalancing
	rm := NewRebalanceManager(allStores)
	result, err := rm.RebalanceDivisions(ctx, leagueID, season1ID, season2ID, 2, categorized, 15)
	is.NoErr(err)

	// Verify divisions created - should be round(30/15) = 2 divisions
	is.Equal(result.DivisionsCreated, 2)

	// Verify all players were assigned
	is.Equal(result.PlayersAssigned, 30)

	// Verify divisions exist in database
	divs, err := store.GetDivisionsBySeason(ctx, season2ID)
	is.NoErr(err)
	is.Equal(len(divs), 2)

	// Verify division numbers are 1 and 2
	divNums := []int32{divs[0].DivisionNumber, divs[1].DivisionNumber}
	sort.Slice(divNums, func(i, j int) bool { return divNums[i] < divNums[j] })
	is.Equal(divNums[0], int32(1))
	is.Equal(divNums[1], int32(2))

	// Verify all registrations have divisions assigned
	finalRegs, err := store.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)
	is.Equal(len(finalRegs), 30)
	for _, reg := range finalRegs {
		is.True(reg.DivisionID.Valid) // All should have divisions
	}
}

func TestRebalanceDivisions_45Players(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	leagueID, season1ID := createLeagueAndSeason(t, ctx, allStores)

	// Create 3 divisions in Season 1
	divIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		divID := uuid.New()
		divIDs[i] = divID
		_, err := store.CreateDivision(ctx, models.CreateDivisionParams{
			Uuid:           divID,
			SeasonID:       season1ID,
			DivisionNumber: int32(i + 1),
			DivisionName:   pgtype.Text{String: "Division " + string(rune('1'+i)), Valid: true},
			PlayerCount:    pgtype.Int4{Int32: 15, Valid: true},
		})
		is.NoErr(err)
	}

	// Register 45 players
	playerIDs := make([]int32, 45)
	for i := 0; i < 45; i++ {
		userDBID := int32(i + 1)
		playerIDs[i] = userDBID

		divID := divIDs[i/15] // 15 per division

		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:     userDBID,
			SeasonID:   season1ID,
			Status:     pgtype.Text{String: "REGISTERED", Valid: true},
			DivisionID: pgtype.UUID{Bytes: divID, Valid: true},
		})
		is.NoErr(err)

		// Create standings
		err = store.UpsertStanding(ctx, models.UpsertStandingParams{
			UserID:     userDBID,
			DivisionID: divID,
			Wins:       pgtype.Int4{Int32: int32(10), Valid: true},
			Losses:     pgtype.Int4{Int32: int32(5), Valid: true},
			Draws:      pgtype.Int4{Int32: 0, Valid: true},
			Spread:     pgtype.Int4{Int32: int32(50), Valid: true},
		})
		is.NoErr(err)
	}

	// Recalculate ranks for all divisions
	for _, divID := range divIDs {
		err := store.RecalculateRanks(ctx, divID)
		is.NoErr(err)
	}

	// Create Season 2
	season2ID := uuid.New()
	_, err := store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season2ID,
		LeagueID:     leagueID,
		SeasonNumber: 2,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)

	// Register all 45 players for Season 2
	for _, userID := range playerIDs {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:   userID,
			SeasonID: season2ID,
			Status:   pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	// Fetch registrations
	regs, err := store.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)
	is.Equal(len(regs), 45)

	// Create CategorizedPlayer list
	categorized := make([]CategorizedPlayer, 45)
	for i := 0; i < 45; i++ {
		categorized[i] = CategorizedPlayer{
			Registration: regs[i],
			Category:     PlayerCategoryReturning,
			Rating:       0,
		}
	}

	// Run rebalancing
	rm := NewRebalanceManager(allStores)
	result, err := rm.RebalanceDivisions(ctx, leagueID, season1ID, season2ID, 2, categorized, 15)
	is.NoErr(err)

	// Verify divisions created - should be round(45/15) = 3 divisions
	is.Equal(result.DivisionsCreated, 3)
	is.Equal(result.PlayersAssigned, 45)

	// Verify divisions exist
	divs, err := store.GetDivisionsBySeason(ctx, season2ID)
	is.NoErr(err)
	is.Equal(len(divs), 3)

	// Verify each division has players (should be ~15 each)
	for _, div := range divs {
		regs, err := store.GetDivisionRegistrations(ctx, div.Uuid)
		is.NoErr(err)
		is.True(len(regs) >= 12) // Should be close to 15, at least above minimum
		is.True(len(regs) <= 20) // Should not exceed reasonable bounds
	}
}

func TestRebalanceDivisions_8NewRookies(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	_, seasonID := createLeagueAndSeason(t, ctx, allStores)

	// Create 8 new rookies (below the MinPlayersForRookieDivision threshold)
	// These should be handled by regular division rebalancing
	for i := 0; i < 8; i++ {
		userDBID := int32(i + 1)

		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:   userDBID,
			SeasonID: seasonID,
			Status:   pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	// Fetch registrations
	regs, err := store.GetSeasonRegistrations(ctx, seasonID)
	is.NoErr(err)
	is.Equal(len(regs), 8)

	// Create CategorizedPlayer list
	categorized := make([]CategorizedPlayer, 8)
	for i := 0; i < 8; i++ {
		categorized[i] = CategorizedPlayer{
			Registration: regs[i],
			Category:     PlayerCategoryNew,
			Rating:       0,
		}
	}

	// Run rebalancing - should create 1 division for these 8 rookies
	rm := NewRebalanceManager(allStores)
	result, err := rm.RebalanceDivisions(ctx, uuid.New(), uuid.New(), seasonID, 1, categorized, 15)
	is.NoErr(err)

	// Verify 1 division created
	is.Equal(result.DivisionsCreated, 1)
	is.Equal(result.PlayersAssigned, 8)

	// Verify division is a regular division (not rookie)
	divs, err := store.GetDivisionsBySeason(ctx, seasonID)
	is.NoErr(err)
	is.Equal(len(divs), 1)
	is.True(divs[0].DivisionNumber < RookieDivisionNumberBase) // Should be regular division number
}

func TestSeasonOrchestrator_FullWorkflow_30Returning_20Rookies(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	leagueID, season1ID := createLeagueAndSeason(t, ctx, allStores)

	// Create 2 divisions in Season 1
	div1 := uuid.New()
	_, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div1,
		SeasonID:       season1ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 15, Valid: true},
	})
	is.NoErr(err)

	div2 := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div2,
		SeasonID:       season1ID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 15, Valid: true},
	})
	is.NoErr(err)

	// Register 30 returning players in Season 1
	returningPlayerIDs := make([]int32, 30)
	for i := 0; i < 30; i++ {
		userDBID := int32(i + 1)
		returningPlayerIDs[i] = userDBID

		divID := div1
		if i >= 15 {
			divID = div2
		}

		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:     userDBID,
			SeasonID:   season1ID,
			Status:     pgtype.Text{String: "REGISTERED", Valid: true},
			DivisionID: pgtype.UUID{Bytes: divID, Valid: true},
		})
		is.NoErr(err)

		// Create standings
		err = store.UpsertStanding(ctx, models.UpsertStandingParams{
			UserID:     userDBID,
			DivisionID: divID,
			Wins:       pgtype.Int4{Int32: int32(10), Valid: true},
			Losses:     pgtype.Int4{Int32: int32(5), Valid: true},
			Draws:      pgtype.Int4{Int32: 0, Valid: true},
			Spread:     pgtype.Int4{Int32: int32(50), Valid: true},
		})
		is.NoErr(err)
	}

	// Recalculate ranks
	err = store.RecalculateRanks(ctx, div1)
	is.NoErr(err)
	err = store.RecalculateRanks(ctx, div2)
	is.NoErr(err)

	// Create Season 2
	season2ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season2ID,
		LeagueID:     leagueID,
		SeasonNumber: 2,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)

	// Register 30 returning players for Season 2
	for _, userID := range returningPlayerIDs {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:   userID,
			SeasonID: season2ID,
			Status:   pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	// Register 20 new rookies for Season 2
	for i := 0; i < 20; i++ {
		userDBID := int32(31 + i) // IDs 31-50

		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:   userDBID,
			SeasonID: season2ID,
			Status:   pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	// Run full season orchestration
	orchestrator := NewSeasonOrchestrator(allStores)
	const idealDivisionSize = 15
	result, err := orchestrator.PrepareNextSeasonDivisions(ctx, leagueID, season1ID, season2ID, 2, idealDivisionSize)
	is.NoErr(err)

	// Verify result counts
	is.Equal(result.TotalRegistrations, 50)       // 30 + 20
	is.Equal(result.ReturningPlayers, 30)
	is.Equal(result.NewPlayers, 20)
	is.Equal(result.PlacedReturning, 30)
	is.Equal(result.PlacedInRookieDivs, 20)
	is.Equal(result.RegularDivisionsUsed, 2)      // round(30/15) = 2
	is.Equal(result.RookieDivisionsCreated, 1)    // 20 rookies → 1 rookie division

	// Verify divisions in database
	divs, err := store.GetDivisionsBySeason(ctx, season2ID)
	is.NoErr(err)
	is.Equal(len(divs), 3) // 2 regular + 1 rookie

	// Count regular vs rookie divisions
	regularDivs := 0
	rookieDivs := 0
	for _, div := range divs {
		if div.DivisionNumber >= RookieDivisionNumberBase {
			rookieDivs++
		} else {
			regularDivs++
		}
	}
	is.Equal(regularDivs, 2)
	is.Equal(rookieDivs, 1)

	// Verify all 50 players have divisions assigned
	regs, err := store.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)
	is.Equal(len(regs), 50)
	for _, reg := range regs {
		is.True(reg.DivisionID.Valid) // All should have divisions
	}
}

func TestSeasonOrchestrator_8RookiesIntoRegularDivisions(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	leagueID, season1ID := createLeagueAndSeason(t, ctx, allStores)

	// Create 1 division in Season 1 with 20 returning players
	div1 := uuid.New()
	_, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div1,
		SeasonID:       season1ID,
		DivisionNumber: 1,
		DivisionName:   pgtype.Text{String: "Division 1", Valid: true},
		PlayerCount:    pgtype.Int4{Int32: 20, Valid: true},
	})
	is.NoErr(err)

	// Register 20 returning players
	returningPlayerIDs := make([]int32, 20)
	for i := 0; i < 20; i++ {
		userDBID := int32(i + 1)
		returningPlayerIDs[i] = userDBID

		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:     userDBID,
			SeasonID:   season1ID,
			Status:     pgtype.Text{String: "REGISTERED", Valid: true},
			DivisionID: pgtype.UUID{Bytes: div1, Valid: true},
		})
		is.NoErr(err)

		err = store.UpsertStanding(ctx, models.UpsertStandingParams{
			UserID:     userDBID,
			DivisionID: div1,
			Wins:       pgtype.Int4{Int32: 10, Valid: true},
			Losses:     pgtype.Int4{Int32: 5, Valid: true},
			Draws:      pgtype.Int4{Int32: 0, Valid: true},
			Spread:     pgtype.Int4{Int32: 50, Valid: true},
		})
		is.NoErr(err)
	}

	err = store.RecalculateRanks(ctx, div1)
	is.NoErr(err)

	// Create Season 2
	season2ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season2ID,
		LeagueID:     leagueID,
		SeasonNumber: 2,
		StartDate:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:       int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)

	// Register 20 returning players
	for _, userID := range returningPlayerIDs {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:   userID,
			SeasonID: season2ID,
			Status:   pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	// Register only 8 new rookies (below MinPlayersForRookieDivision)
	for i := 0; i < 8; i++ {
		userDBID := int32(21 + i) // IDs 21-28

		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID:   userDBID,
			SeasonID: season2ID,
			Status:   pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	// Run orchestration
	orchestrator := NewSeasonOrchestrator(allStores)
	const idealDivisionSize = 15
	result, err := orchestrator.PrepareNextSeasonDivisions(ctx, leagueID, season1ID, season2ID, 2, idealDivisionSize)
	is.NoErr(err)

	// Verify 8 rookies were included in regular rebalancing
	is.Equal(result.TotalRegistrations, 28)       // 20 + 8
	is.Equal(result.ReturningPlayers, 20)
	is.Equal(result.NewPlayers, 8)
	is.Equal(result.PlacedInRegularDivs, 8)       // Rookies went to regular divisions
	is.Equal(result.RookieDivisionsCreated, 0)    // No rookie divisions created

	// Verify only regular divisions exist (no rookie divisions)
	divs, err := store.GetDivisionsBySeason(ctx, season2ID)
	is.NoErr(err)
	for _, div := range divs {
		is.True(div.DivisionNumber < RookieDivisionNumberBase) // All should be regular
	}

	// Verify all 28 players assigned
	regs, err := store.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)
	is.Equal(len(regs), 28)
	for _, reg := range regs {
		is.True(reg.DivisionID.Valid)
	}
}
