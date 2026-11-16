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
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// fakeGameCreator writes a bare games row per game, with its players and
// division, and fails once it has created stopAfter games (0: never), the
// way a start cut off by a deploy stops.
type fakeGameCreator struct {
	pool      *pgxpool.Pool
	players   map[*entity.Game][2]*entity.User
	created   int
	stopAfter int
}

func (f *fakeGameCreator) InstantiateNewGame(ctx context.Context, users [2]*entity.User, req *pb.GameRequest, tdata *entity.TournamentData) (*entity.Game, error) {
	if f.stopAfter > 0 && f.created >= f.stopAfter {
		return nil, errors.New("cut off")
	}
	g := &entity.Game{}
	f.players[g] = users
	return g, nil
}

func (f *fakeGameCreator) StartGame(ctx context.Context, g *entity.Game) error {
	users := f.players[g]
	_, err := f.pool.Exec(ctx, `INSERT INTO games (uuid, player0_id, player1_id, league_division_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())`,
		shortuuid.New(), users[0].ID, users[1].ID, *g.LeagueDivisionID)
	if err != nil {
		return err
	}
	f.created++
	return nil
}

// A season start cut off partway leaves the season ACTIVE without
// games_created_at; the next run creates only the games still missing,
// so every pair plays exactly once.
func TestSeasonStartCarriesOnAfterCutOff(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	allStores, store, cleanup := setupTest(t)
	defer cleanup()

	leagueID := uuid.New()
	_, err := store.CreateLeague(ctx, models.CreateLeagueParams{
		Uuid: leagueID, Name: "Resume League", Slug: uuid.NewString(), Settings: []byte(`{}`),
		IsActive: pgtype.Bool{Bool: true, Valid: true}, CreatedBy: pgtype.Int8{Int64: 1, Valid: true},
	})
	is.NoErr(err)
	seasonID := uuid.New()
	_, err = store.CreateSeason(ctx, models.CreateSeasonParams{
		Uuid: seasonID, LeagueID: leagueID, SeasonNumber: 1,
		StartDate: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		EndDate:   pgtype.Timestamptz{Time: time.Now().AddDate(0, 1, 0), Valid: true},
		Status:    int32(pb.SeasonStatus_SEASON_SCHEDULED),
	})
	is.NoErr(err)
	divID := uuid.New()
	_, err = store.CreateDivision(ctx, models.CreateDivisionParams{
		Uuid: divID, SeasonID: seasonID, DivisionNumber: 1,
		DivisionName: pgtype.Text{String: "Division 1", Valid: true},
	})
	is.NoErr(err)
	const players = 6
	for i := int32(1); i <= players; i++ {
		_, err := store.RegisterPlayer(ctx, models.RegisterPlayerParams{
			UserID: i, SeasonID: seasonID,
			DivisionID: pgtype.UUID{Bytes: divID, Valid: true},
			// The same date for all: user_id orders them.
			RegistrationDate: pgtype.Timestamptz{Time: time.Unix(1700000000, 0), Valid: true},
			Status:           pgtype.Text{String: "ACTIVE", Valid: true},
		})
		is.NoErr(err)
	}

	creator := &fakeGameCreator{pool: allStores.DBPool, players: map[*entity.Game][2]*entity.User{}, stopAfter: 4}
	cfg := config.DefaultConfig()
	run := func() *LifecycleRunnerResult {
		res, err := RunLeagueLifecycleTasks(ctx, cfg, allStores, creator, leagueID, RealClock{})
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
	is.Equal(res.GamesCreated, players*(players-1)/2-4)

	season, err = store.GetSeason(ctx, seasonID)
	is.NoErr(err)
	is.True(season.GamesCreatedAt.Valid)
	is.True(season.StartedAt.Valid)

	// Every pair once.
	rows, err := store.GetDivisionGamePlayers(ctx, divID)
	is.NoErr(err)
	is.Equal(len(rows), players*(players-1)/2)
	seen := map[string]bool{}
	for _, r := range rows {
		a, b := r.Player0ID.Int32, r.Player1ID.Int32
		if a > b {
			a, b = b, a
		}
		k := fmt.Sprint(a, b)
		is.True(!seen[k])
		seen[k] = true
	}

	// A third run creates nothing.
	res = run()
	is.Equal(res.GamesCreated, 0)
	is.Equal(creator.created, players*(players-1)/2)
}
