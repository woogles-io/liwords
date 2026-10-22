package league

import (
	"context"
	"strings"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/google/uuid"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	gamestore "github.com/woogles-io/liwords/pkg/stores/game"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const forceFinishPkg = "league_forcefinish_test"

// setupForceFinishTest builds the full set of stores, because force-finish
// plays the game's ending -- ratings, standings, saves -- rather than just
// writing league rows.
func setupForceFinishTest(t *testing.T) (context.Context, *stores.Stores, chan *entity.EventWrapper, func()) {
	if err := common.RecreateTestDB(forceFinishPkg); err != nil {
		t.Fatal(err)
	}
	pool, err := common.OpenTestingDB(forceFinishPkg)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.DBConnDSN = common.TestingPostgresConnDSN(forceFinishPkg)
	allStores, err := stores.NewInitializedStores(pool, nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := createIntegrationTestUsers(pool); err != nil {
		t.Fatal(err)
	}

	// Ending a game sends events; nothing listens in this test.
	ch := make(chan *entity.EventWrapper, 100)
	allStores.GameStore.SetGameEventChan(ch)
	ctx, cancel := context.WithCancel(cfg.WithContext(log.Logger.WithContext(context.Background())))
	go func() {
		for {
			select {
			case <-ch:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ctx, allStores, ch, func() {
		cancel()
		allStores.Disconnect()
	}
}

// createSeasonGame starts a correspondence game in the season, the way
// season start does: instantiate, tag with the league, then start, which saves.
func createSeasonGame(t *testing.T, ctx context.Context, allStores *stores.Stores,
	ch chan *entity.EventWrapper, leagueID, seasonID uuid.UUID, p0, p1 string) string {

	is := is.New(t)
	u0, err := allStores.UserStore.Get(ctx, p0)
	is.NoErr(err)
	u1, err := allStores.UserStore.Get(ctx, p1)
	is.NoErr(err)
	req := &ipc.GameRequest{
		Lexicon: "CSW21",
		Rules: &ipc.GameRules{BoardLayoutName: entity.CrosswordGame,
			LetterDistributionName: "English", VariantName: "classic"},
		InitialTimeSeconds: 3 * 24 * 3600,
		ChallengeRule:      macondopb.ChallengeRule_FIVE_POINT,
		GameMode:           ipc.GameMode_CORRESPONDENCE,
		RatingMode:         ipc.RatingMode_RATED,
		RequestId:          uuid.NewString(),
	}
	g, err := gameplay.InstantiateNewGame(ctx, allStores.GameStore, config.DefaultConfig(),
		[2]*entity.User{u0, u1}, req, nil)
	is.NoErr(err)
	divisionID := uuid.New()
	g.LeagueID = &leagueID
	g.SeasonID = &seasonID
	g.LeagueDivisionID = &divisionID
	is.NoErr(gameplay.StartGame(ctx, allStores, ch, g))
	return g.GameID()
}

// Season close force-finishes every unfinished game in the season. Each game
// is ended under its lock, so a game that stays locked past the wait is left
// for the next run -- the hourly league runner calls close again until it
// succeeds -- and that run ends only what is still unfinished.
func TestForceFinishUnfinishedGames(t *testing.T) {
	is := is.New(t)
	ctx, allStores, ch, cleanup := setupForceFinishTest(t)
	defer cleanup()

	leagueID, seasonID := createLeagueAndSeason(t, ctx, allStores)
	a := createSeasonGame(t, ctx, allStores, ch, leagueID, seasonID, "testuser1", "testuser2")
	b := createSeasonGame(t, ctx, allStores, ch, leagueID, seasonID, "testuser3", "testuser4")
	c := createSeasonGame(t, ctx, allStores, ch, leagueID, seasonID, "testuser5", "testuser6")
	done := createSeasonGame(t, ctx, allStores, ch, leagueID, seasonID, "testuser7", "testuser8")
	is.NoErr(gameplay.ForfeitGame(ctx, done, 0, allStores))

	// b is busy for the whole first run, as if a move on it never finished.
	held, err := allStores.GameStore.LockGame(ctx, b)
	is.NoErr(err)

	ffm := NewForceFinishManager(allStores)
	first, err := ffm.ForceFinishUnfinishedGames(ctx, seasonID)
	is.NoErr(err)
	held.Unlock(ctx)

	is.Equal(first.TotalGames, 3) // the forfeited game was not picked up
	is.Equal(first.ForceForfeitGames, 2)
	is.Equal(len(first.Errors), 1)
	is.True(strings.Contains(first.Errors[0], b))
	is.True(strings.Contains(first.Errors[0], gamestore.ErrGameLockBusy.Error()))

	endReason := func(id string) ipc.GameEndReason {
		g, err := allStores.GameStore.Get(ctx, id)
		is.NoErr(err)
		return g.GameEndReason
	}
	eventCount := func(id string) int {
		g, err := allStores.GameStore.Get(ctx, id)
		is.NoErr(err)
		return len(g.History().Events)
	}
	is.Equal(endReason(a), ipc.GameEndReason_ADJUDICATED)
	is.Equal(endReason(b), ipc.GameEndReason_NONE)
	is.Equal(endReason(c), ipc.GameEndReason_ADJUDICATED)
	is.Equal(endReason(done), ipc.GameEndReason_FORCE_FORFEIT)
	aEvents, cEvents := eventCount(a), eventCount(c)

	// The next hourly run.
	second, err := ffm.ForceFinishUnfinishedGames(ctx, seasonID)
	is.NoErr(err)
	is.Equal(second.TotalGames, 1)
	is.Equal(second.ForceForfeitGames, 1)
	is.Equal(len(second.Errors), 0)

	is.Equal(endReason(b), ipc.GameEndReason_ADJUDICATED)
	// The games ended by the first run were not ended again.
	is.Equal(eventCount(a), aEvents)
	is.Equal(eventCount(c), cEvents)
	is.Equal(endReason(done), ipc.GameEndReason_FORCE_FORFEIT)

	// Nothing left.
	third, err := ffm.ForceFinishUnfinishedGames(ctx, seasonID)
	is.NoErr(err)
	is.Equal(third.TotalGames, 0)
}
