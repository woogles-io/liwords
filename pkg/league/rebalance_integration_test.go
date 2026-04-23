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

// TestEnforceGuarantees_PromotedPlayerGuaranteed verifies that a player who earned
// promotion is always placed in a better division, even when high-scoring STAYED
// players would otherwise crowd them out via the priority-bucket algorithm.
//
// Scenario: 30 players across 2 divisions of 15.
//   - All 15 div-1 players: RESULT_STAYED   (placement STAYED, virtual div 1, score ~240k)
//   - Players 16-18 in div2: RESULT_PROMOTED (placement PROMOTED, virtual div 1, score ~230k)
//   - Players 19-30 in div2: RESULT_STAYED   (virtual div 2, score ~140k)
//
// Without enforcement the 15 STAYED-div1 players (240k) beat the 3 PROMOTED-div2
// players (230k), pushing the promoted players into div 2 — a guarantee violation.
// With enforcement they must end up in div 1.
func TestEnforceGuarantees_PromotedPlayerGuaranteed(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	leagueID, season1ID := createLeagueAndSeason(t, ctx, allStores)

	// Season 1: two divisions, 15 players each.
	div1S1 := uuid.New()
	_, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid: div1S1, SeasonID: season1ID, DivisionNumber: 1,
		DivisionName: pgtype.Text{String: "Division 1", Valid: true},
	})
	is.NoErr(err)

	div2S1 := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid: div2S1, SeasonID: season1ID, DivisionNumber: 2,
		DivisionName: pgtype.Text{String: "Division 2", Valid: true},
	})
	is.NoErr(err)

	// Register and create standings for 30 players.
	for i := 1; i <= 30; i++ {
		divID := div1S1
		if i > 15 {
			divID = div2S1
		}
		_, err = store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID: int32(i), SeasonID: season1ID,
			Status:     pgtype.Text{String: "REGISTERED", Valid: true},
			DivisionID: pgtype.UUID{Bytes: divID, Valid: true},
		})
		is.NoErr(err)
		err = store.UpsertStanding(ctx, models.UpsertStandingParams{
			UserID: int32(i), DivisionID: divID,
			Wins: pgtype.Int4{Int32: 5, Valid: true}, Losses: pgtype.Int4{Int32: 5, Valid: true},
		})
		is.NoErr(err)
	}

	// Set outcomes: players 1-15 STAYED in div1; players 16-18 PROMOTED in div2.
	for i := 1; i <= 15; i++ {
		err = store.UpdateStandingResult(ctx, models.UpdateStandingResultParams{
			DivisionID: div1S1, UserID: int32(i),
			Result: pgtype.Int4{Int32: int32(ipc.StandingResult_RESULT_STAYED), Valid: true},
		})
		is.NoErr(err)
	}
	for i := 16; i <= 18; i++ {
		err = store.UpdateStandingResult(ctx, models.UpdateStandingResultParams{
			DivisionID: div2S1, UserID: int32(i),
			Result: pgtype.Int4{Int32: int32(ipc.StandingResult_RESULT_PROMOTED), Valid: true},
		})
		is.NoErr(err)
	}
	for i := 19; i <= 30; i++ {
		err = store.UpdateStandingResult(ctx, models.UpdateStandingResultParams{
			DivisionID: div2S1, UserID: int32(i),
			Result: pgtype.Int4{Int32: int32(ipc.StandingResult_RESULT_STAYED), Valid: true},
		})
		is.NoErr(err)
	}

	// Season 2: register all 30 players.
	season2ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid: season2ID, LeagueID: leagueID, SeasonNumber: 2,
		StartDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:   pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:    int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)

	for i := 1; i <= 30; i++ {
		_, err = store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID: int32(i), SeasonID: season2ID,
			Status: pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	regs, err := store.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)
	is.Equal(len(regs), 30)

	categorized := make([]CategorizedPlayer, 30)
	for i, r := range regs {
		categorized[i] = CategorizedPlayer{Registration: r, Category: PlayerCategoryReturning}
	}

	rm := NewRebalanceManager(allStores)
	result, err := rm.RebalanceDivisions(ctx, leagueID, season1ID, season2ID, 2, categorized, 15)
	is.NoErr(err)
	is.Equal(result.DivisionsCreated, 2)

	// Find the season-2 division with number 1.
	s2Divs, err := store.GetDivisionsBySeason(ctx, season2ID)
	is.NoErr(err)
	div1S2 := uuid.UUID{}
	for _, d := range s2Divs {
		if d.DivisionNumber == 1 {
			div1S2 = d.Uuid
		}
	}
	is.True(div1S2 != uuid.UUID{})

	// Players 16-18 (PROMOTED from div2) must all be in div1.
	for i := 16; i <= 18; i++ {
		reg, err := store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			SeasonID: season2ID, UserID: int32(i),
		})
		is.NoErr(err)
		is.True(reg.DivisionID.Valid)
		is.Equal(uuid.UUID(reg.DivisionID.Bytes), div1S2) // must be div 1

		is.True(reg.PlacementStatus.Valid)
		is.Equal(ipc.PlacementStatus(reg.PlacementStatus.Int32), ipc.PlacementStatus_PLACEMENT_PROMOTED)
	}
}

// TestEnforceGuarantees_StayedPlayerNotRelegated verifies that a STAYED player
// is never placed in a worse division than their previous one, even when
// relegated players crowd them out.
//
// Scenario: 30 players across 2 divisions of 15.
//   - Div-1: 2 relegated (virtual div 2, score ~150k) + 13 stayed (virtual div 1, ~240k).
//   - Div-2: 3 relegated (virtual div 3 — beyond last div, overflow to div 2)
//             + 12 stayed (virtual div 2, ~140k).
//
// The 2 relegated from div1 and 3 relegated from div2 both have high relegated bonus (50k).
// We construct a case where some STAYED-from-div1 players (ceiling=div1) are crowded out.
//
// Specifically: make div1 have 14 stayed + 1 relegated.  In virtual div2 there are the
// relegated player (from div1, virtual div2, ~150k) plus 3 relegated from div2 (virtual
// div3 = overflow to last → div2 in a 2-division league, ~150k), pushing all 4 relegated
// into the bucket that was supposed to be div2 for the 12-stayed-from-div2 players.
//
// A simpler direct scenario: 16 players target div1 but only 15 bucket slots exist.
// The 16th is a STAYED player whose ceiling is div1 — enforcement must move them up.
func TestEnforceGuarantees_StayedPlayerNotRelegated(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	allStores, cleanup := setupIntegrationTest(t)
	defer cleanup()

	store := allStores.LeagueStore
	leagueID, season1ID := createLeagueAndSeason(t, ctx, allStores)

	// Season 1: one division with 16 players, all STAYED.
	div1S1 := uuid.New()
	_, err := store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid: div1S1, SeasonID: season1ID, DivisionNumber: 1,
		DivisionName: pgtype.Text{String: "Division 1", Valid: true},
	})
	is.NoErr(err)

	// Extra div to satisfy the "has previous season div" constraint for the 16th player.
	div2S1 := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid: div2S1, SeasonID: season1ID, DivisionNumber: 2,
		DivisionName: pgtype.Text{String: "Division 2", Valid: true},
	})
	is.NoErr(err)

	// Players 1-15 stayed in div1; player 16 also stayed in div1.
	// Season 2 idealDivisionSize=15 → bucket size 15. All 16 targeting virtual div1.
	// Without enforcement: player 16 (lowest rank → lowest score) slips into div2.
	// With enforcement: must be moved back to div1.
	for i := 1; i <= 16; i++ {
		_, err = store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID: int32(i), SeasonID: season1ID,
			Status:     pgtype.Text{String: "REGISTERED", Valid: true},
			DivisionID: pgtype.UUID{Bytes: div1S1, Valid: true},
		})
		is.NoErr(err)
		err = store.UpsertStanding(ctx, models.UpsertStandingParams{
			UserID: int32(i), DivisionID: div1S1,
			Wins: pgtype.Int4{Int32: int32(16 - i), Valid: true}, // Different wins to get distinct ranks.
		})
		is.NoErr(err)
		err = store.UpdateStandingResult(ctx, models.UpdateStandingResultParams{
			DivisionID: div1S1, UserID: int32(i),
			Result: pgtype.Int4{Int32: int32(ipc.StandingResult_RESULT_STAYED), Valid: true},
		})
		is.NoErr(err)
	}
	// Also register player 17 in div2 (STAYED) so season 2 has enough players for 2 divs.
	for i := 17; i <= 29; i++ {
		_, err = store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID: int32(i), SeasonID: season1ID,
			Status:     pgtype.Text{String: "REGISTERED", Valid: true},
			DivisionID: pgtype.UUID{Bytes: div2S1, Valid: true},
		})
		is.NoErr(err)
		err = store.UpsertStanding(ctx, models.UpsertStandingParams{
			UserID: int32(i), DivisionID: div2S1,
			Wins: pgtype.Int4{Int32: 5, Valid: true},
		})
		is.NoErr(err)
		err = store.UpdateStandingResult(ctx, models.UpdateStandingResultParams{
			DivisionID: div2S1, UserID: int32(i),
			Result: pgtype.Int4{Int32: int32(ipc.StandingResult_RESULT_STAYED), Valid: true},
		})
		is.NoErr(err)
	}

	// Season 2: register all 29 players.
	season2ID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid: season2ID, LeagueID: leagueID, SeasonNumber: 2,
		StartDate: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		EndDate:   pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 14), Valid: true},
		Status:    int32(ipc.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)

	for i := 1; i <= 29; i++ {
		_, err = store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID: int32(i), SeasonID: season2ID,
			Status: pgtype.Text{String: "REGISTERED", Valid: true},
		})
		is.NoErr(err)
	}

	regs, err := store.GetSeasonRegistrations(ctx, season2ID)
	is.NoErr(err)
	is.Equal(len(regs), 29)

	categorized := make([]CategorizedPlayer, 29)
	for i, r := range regs {
		categorized[i] = CategorizedPlayer{Registration: r, Category: PlayerCategoryReturning}
	}

	rm := NewRebalanceManager(allStores)
	result, err := rm.RebalanceDivisions(ctx, leagueID, season1ID, season2ID, 2, categorized, 15)
	is.NoErr(err)
	is.Equal(result.PlayersAssigned, 29)

	// Find div1 in season 2.
	s2Divs, err := store.GetDivisionsBySeason(ctx, season2ID)
	is.NoErr(err)
	var div1S2UUID uuid.UUID
	for _, d := range s2Divs {
		if d.DivisionNumber == 1 {
			div1S2UUID = d.Uuid
		}
	}
	is.True(div1S2UUID != uuid.UUID{})

	// All 16 STAYED-from-div1 players must be in div1.
	for i := 1; i <= 16; i++ {
		reg, err := store.GetPlayerRegistration(ctx, models.GetPlayerRegistrationParams{
			SeasonID: season2ID, UserID: int32(i),
		})
		is.NoErr(err)
		is.True(reg.DivisionID.Valid)
		is.Equal(uuid.UUID(reg.DivisionID.Bytes), div1S2UUID)

		is.True(reg.PlacementStatus.Valid)
		// Player 16 had the lowest score (rank 16 in a 16-player div, rank component = 16-16 = 0).
		// After enforcement they are placed in div1 — CorrectPlacementStatus labels them STAYED.
		status := ipc.PlacementStatus(reg.PlacementStatus.Int32)
		is.True(status == ipc.PlacementStatus_PLACEMENT_STAYED || status == ipc.PlacementStatus_PLACEMENT_PROMOTED)
	}
}
