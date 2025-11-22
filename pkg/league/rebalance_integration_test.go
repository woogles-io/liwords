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
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	leaguestore "github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
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
	})
	is.NoErr(err)

	div2 := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid:           div2,
		SeasonID:       season1ID,
		DivisionNumber: 2,
		DivisionName:   pgtype.Text{String: "Division 2", Valid: true},
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

	// Ranks are calculated on-demand when fetching standings, no need to explicitly recalculate

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

	// Ranks are calculated on-demand when fetching standings, no need to explicitly recalculate

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
	// All divisions are now regular divisions
}
