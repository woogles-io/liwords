package league

import (
	"context"
	"fmt"
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

const pkg = "league_test"

func setupTest(t *testing.T) (*stores.Stores, *leaguestore.DBStore, func()) {
	err := common.RecreateTestDB(pkg)
	if err != nil {
		t.Fatal(err)
	}

	pool, err := common.OpenTestingDB(pkg)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.DBConnDSN = common.TestingPostgresConnDSN(pkg)

	leagueStore, err := leaguestore.NewDBStore(cfg, pool)
	if err != nil {
		t.Fatal(err)
	}

	userStore, err := user.NewDBStore(pool)
	if err != nil {
		t.Fatal(err)
	}

	// Create test users
	err = createTestUsers(pool)
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

	return allStores, leagueStore, cleanup
}

// createTestUsers creates test users in the database for league tests
func createTestUsers(pool *pgxpool.Pool) error {
	ustore, err := user.NewDBStore(pool)
	if err != nil {
		return err
	}
	// Don't disconnect - the pool is shared with the league store

	ctx := context.Background()

	// Create 100 test users (enough for all tests)
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

func TestRegisterPlayer(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, store, cleanup := setupTest(t)
	defer cleanup()

	// Create a test league and season
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
		Status:       int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)

	// Test registration
	rm := NewRegistrationManager(store, RealClock{}, allStores)

	userID := int32(1)

	err = rm.RegisterPlayer(ctx, userID, seasonID)
	is.NoErr(err)

	// Verify registration was stored
	regs, err := rm.GetSeasonRegistrations(ctx, seasonID)
	is.NoErr(err)
	is.Equal(len(regs), 1)
	is.Equal(regs[0].UserID, userID)
	is.Equal(regs[0].Status.String, "REGISTERED")
	is.True(!regs[0].DivisionID.Valid) // No division assigned yet
}

func TestRegisterMultiplePlayers(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, store, cleanup := setupTest(t)
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
		Status:       int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)

	// Register 50 players
	rm := NewRegistrationManager(store, RealClock{}, allStores)

	for i := 0; i < 50; i++ {
		userID := int32(i + 1)
		err = rm.RegisterPlayer(ctx, userID, seasonID)
		is.NoErr(err)
	}

	// Verify all 50 were stored
	regs, err := rm.GetSeasonRegistrations(ctx, seasonID)
	is.NoErr(err)
	is.Equal(len(regs), 50)
}

func TestCategorizeRegistrations_AllNew(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, store, cleanup := setupTest(t)
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
		Status:       int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)

	// Register 10 new players (first season, so all should be "NEW")
	rm := NewRegistrationManager(store, RealClock{}, allStores)

	for i := 0; i < 10; i++ {
		userID := int32(i + 1)
		err = rm.RegisterPlayer(ctx, userID, seasonID)
		is.NoErr(err)
	}

	// Get registrations and categorize them
	regs, err := rm.GetSeasonRegistrations(ctx, seasonID)
	is.NoErr(err)

	categorized, err := rm.CategorizeRegistrations(ctx, leagueID, seasonID, regs)
	is.NoErr(err)
	is.Equal(len(categorized), 10)

	// All should be categorized as NEW
	for _, cp := range categorized {
		is.Equal(cp.Category, PlayerCategoryNew)
	}
}

func TestCategorizeRegistrations_Mixed(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, store, cleanup := setupTest(t)
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

	// Create Season 1
	season1ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid:         season1ID,
		LeagueID:     leagueID,
		SeasonNumber: 1,
		StartDate:    pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -30), Valid: true},
		EndDate:      pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -16), Valid: true},
		Status:       int32(ipc.SeasonStatus_SEASON_COMPLETED),
	})
	is.NoErr(err)

	// Register 5 players in Season 1
	rm := NewRegistrationManager(store, RealClock{}, allStores)
	returningPlayerIDs := []int32{}
	for i := 0; i < 5; i++ {
		userID := int32(i + 1)
		returningPlayerIDs = append(returningPlayerIDs, userID)
		err = rm.RegisterPlayer(ctx, userID, season1ID)
		is.NoErr(err)
	}

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

	// Register the 5 returning players in Season 2
	for _, userID := range returningPlayerIDs {
		err = rm.RegisterPlayer(ctx, userID, season2ID)
		is.NoErr(err)
	}

	// Register 5 new players in Season 2
	newPlayerIDs := []int32{}
	for i := 0; i < 5; i++ {
		userID := int32(6 + i) // IDs 6-10
		newPlayerIDs = append(newPlayerIDs, userID)
		err = rm.RegisterPlayer(ctx, userID, season2ID)
		is.NoErr(err)
	}

	// Get registrations and categorize them
	regs, err := rm.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)

	categorized, err := rm.CategorizeRegistrations(ctx, leagueID, season2ID, regs)
	is.NoErr(err)
	is.Equal(len(categorized), 10)

	// Count categories
	newCount := 0
	returningCount := 0
	for _, cp := range categorized {
		if cp.Category == PlayerCategoryNew {
			newCount++
			// Verify this player is in our newPlayerIDs list
			found := false
			for _, id := range newPlayerIDs {
				if id == cp.Registration.UserID {
					found = true
					break
				}
			}
			is.True(found) // New player should be in newPlayerIDs list
		} else {
			returningCount++
			// Verify this player is in our returningPlayerIDs list
			found := false
			for _, id := range returningPlayerIDs {
				if id == cp.Registration.UserID {
					found = true
					break
				}
			}
			is.True(found) // Returning player should be in returningPlayerIDs list
		}
	}

	is.Equal(newCount, 5)
	is.Equal(returningCount, 5)
}
