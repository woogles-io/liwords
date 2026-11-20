package league

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lithammer/shortuuid/v4"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	leaguestore "github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// fakeGameCreator writes a bare games row per game, with its players and
// league fields, and marks it started. It stops the way a start cut off by a
// deploy does: after creating stopAfter games (0: never), or, with
// stopBeforeStart, between inserting a game and starting it.
type fakeGameCreator struct {
	pool            *pgxpool.Pool
	games           map[string]*entity.Game
	ids             map[*entity.Game]string
	created         int
	started         int
	stopAfter       int
	stopBeforeStart bool
}

func newFakeGameCreator(pool *pgxpool.Pool) *fakeGameCreator {
	return &fakeGameCreator{pool: pool, games: map[string]*entity.Game{}, ids: map[*entity.Game]string{}}
}

func (f *fakeGameCreator) InstantiateNewGame(ctx context.Context, users [2]*entity.User, req *pb.GameRequest, tdata *entity.TournamentData, opts ...gameplay.GameOption) (*entity.Game, error) {
	if f.stopAfter > 0 && f.created >= f.stopAfter {
		return nil, errors.New("cut off")
	}
	g := &entity.Game{}
	for _, o := range opts {
		o(g)
	}
	id := shortuuid.New()
	var division any
	if g.LeagueDivisionID != nil {
		division = *g.LeagueDivisionID
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO games (uuid, player0_id, player1_id, league_division_id, started, created_at, updated_at)
		VALUES ($1, $2, $3, $4, false, NOW(), NOW())`, id, users[0].ID, users[1].ID, division); err != nil {
		return nil, err
	}
	f.games[id], f.ids[g] = g, id
	f.created++
	return g, nil
}

func (f *fakeGameCreator) StartGame(ctx context.Context, g *entity.Game, _ user.Store) error {
	if f.stopBeforeStart {
		f.stopBeforeStart = false
		return errors.New("cut off")
	}
	if _, err := f.pool.Exec(ctx, `UPDATE games SET started = true WHERE uuid = $1`, f.ids[g]); err != nil {
		return err
	}
	f.started++
	return nil
}

func (f *fakeGameCreator) GetGame(ctx context.Context, id string) (*entity.Game, error) {
	g, ok := f.games[id]
	if !ok {
		return nil, errors.New("no such game")
	}
	return g, nil
}

const startPlayers = 6

// seasonToStart is a SCHEDULED season past its start, with one division of
// startPlayers players registered at the same moment.
func seasonToStart(t *testing.T, ctx context.Context, store *leaguestore.DBStore) (leagueID, seasonID, divID uuid.UUID) {
	is := is.New(t)
	leagueID, seasonID, divID = uuid.New(), uuid.New(), uuid.New()
	_, err := store.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid: leagueID, Name: "Resume League", Slug: uuid.NewString(), Settings: []byte(`{}`),
		IsActive: pgtype.Bool{Bool: true, Valid: true}, CreatedBy: pgtype.Int8{Int64: 1, Valid: true},
	})
	is.NoErr(err)
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid: seasonID, LeagueID: leagueID, SeasonNumber: 1,
		StartDate: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		EndDate:   pgtype.Timestamptz{Time: time.Now().AddDate(0, 1, 0), Valid: true},
		Status:    int32(pb.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid: divID, SeasonID: seasonID, DivisionNumber: 1,
		DivisionName: pgtype.Text{String: "Division 1", Valid: true},
	})
	is.NoErr(err)
	for i := int32(1); i <= startPlayers; i++ {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID: i, SeasonID: seasonID,
			DivisionID: pgtype.UUID{Bytes: divID, Valid: true},
			// The same moment for all: user_id orders them.
			RegistrationDate: pgtype.Timestamptz{Time: time.Unix(1700000000, 0), Valid: true},
			Status:           pgtype.Text{String: "ACTIVE", Valid: true},
		})
		is.NoErr(err)
	}
	return leagueID, seasonID, divID
}

// checkStartedOnce checks the division has one started game per pair, and
// the season is marked with all its games created.
func checkStartedOnce(t *testing.T, ctx context.Context, store *leaguestore.DBStore, seasonID, divID uuid.UUID) {
	is := is.New(t)
	rows, err := store.GetDivisionGamePlayers(ctx, divID)
	is.NoErr(err)
	is.Equal(len(rows), startPlayers*(startPlayers-1)/2)
	seen := map[string]bool{}
	for _, r := range rows {
		a, b := r.Player0ID.Int32, r.Player1ID.Int32
		if a > b {
			a, b = b, a
		}
		k := fmt.Sprint(a, b)
		is.True(!seen[k]) // a pair with two games
		seen[k] = true
		is.True(r.Started.Bool)
	}
	season, err := store.GetSeason(ctx, seasonID)
	is.NoErr(err)
	is.True(season.GamesCreatedAt.Valid)
	is.True(season.StartedAt.Valid)
}

// A season start cut off partway leaves the season ACTIVE without
// games_created_at; the next run creates only the games still missing.
func TestSeasonStartCarriesOnAfterCutOff(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	allStores, store, cleanup := setupTest(t)
	defer cleanup()
	leagueID, seasonID, divID := seasonToStart(t, ctx, store)

	creator := newFakeGameCreator(allStores.DBPool)
	creator.stopAfter = 4
	run := func() *LifecycleRunnerResult {
		res, err := RunLeagueLifecycleTasks(ctx, config.DefaultConfig(), allStores, creator, leagueID, RealClock{})
		is.NoErr(err)
		return res
	}

	run() // cut off after 4 games
	season, err := store.GetSeason(ctx, seasonID)
	is.NoErr(err)
	is.Equal(season.Status, int32(pb.SeasonStatus_SEASON_ACTIVE))
	is.True(!season.GamesCreatedAt.Valid)
	is.Equal(creator.created, 4)

	creator.stopAfter = 0
	res := run()
	is.True(res.SeasonStarted)
	is.Equal(res.GamesCreated, startPlayers*(startPlayers-1)/2-4)
	checkStartedOnce(t, ctx, store, seasonID, divID)

	// A third run does nothing.
	res = run()
	is.Equal(res.GamesCreated, 0)
	is.Equal(creator.created, startPlayers*(startPlayers-1)/2)
}

// A start cut off between creating a game and starting it leaves that game
// in its division, unstarted; the next run starts it rather than creating
// the pair another.
func TestSeasonStartStartsGameLeftUnstarted(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	allStores, store, cleanup := setupTest(t)
	defer cleanup()
	leagueID, seasonID, divID := seasonToStart(t, ctx, store)

	creator := newFakeGameCreator(allStores.DBPool)
	creator.stopBeforeStart = true
	_, err := RunLeagueLifecycleTasks(ctx, config.DefaultConfig(), allStores, creator, leagueID, RealClock{})
	is.NoErr(err)
	is.Equal(creator.created, 1)
	is.Equal(creator.started, 0)

	_, err = RunLeagueLifecycleTasks(ctx, config.DefaultConfig(), allStores, creator, leagueID, RealClock{})
	is.NoErr(err)
	is.Equal(creator.created, startPlayers*(startPlayers-1)/2) // no second game for the first pair
	is.Equal(creator.started, startPlayers*(startPlayers-1)/2)
	checkStartedOnce(t, ctx, store, seasonID, divID)
}
