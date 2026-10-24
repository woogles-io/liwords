package bus

import (
	"context"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func readyTestSetup(t *testing.T) (context.Context, *Bus, *stores.Stores, [2]*entity.User) {
	is := is.New(t)
	st := recreateInternalTestDB()
	t.Cleanup(st.Disconnect)

	cfg := config.DefaultConfig()
	ctx, cancel := context.WithCancel(cfg.WithContext(log.Logger.WithContext(context.Background())))
	t.Cleanup(cancel)

	// Starting a game sends events; nothing listens in this test.
	ch := make(chan *entity.EventWrapper, 100)
	st.GameStore.SetGameEventChan(ch)
	go func() {
		for {
			select {
			case <-ch:
			case <-ctx.Done():
				return
			}
		}
	}()

	var users [2]*entity.User
	for i, u := range []*entity.User{
		{Username: "readyone", Email: "readyone@example.com", UUID: "rdy1aaaaaaaaaaaaaaaaaa"},
		{Username: "readytwo", Email: "readytwo@example.com", UUID: "rdy2aaaaaaaaaaaaaaaaaa"},
	} {
		is.NoErr(st.UserStore.New(ctx, u))
		got, err := st.UserStore.Get(ctx, u.Username)
		is.NoErr(err)
		users[i] = got
	}
	return ctx, &Bus{stores: st, gameEventChan: ch}, st, users
}

func readyTestGame(t *testing.T, ctx context.Context, st *stores.Stores,
	users [2]*entity.User, mode pb.GameMode) *entity.Game {

	is := is.New(t)
	req := &pb.GameRequest{
		Lexicon: "CSW21",
		Rules: &pb.GameRules{BoardLayoutName: entity.CrosswordGame,
			LetterDistributionName: "English", VariantName: "classic"},
		InitialTimeSeconds: 25 * 60,
		ChallengeRule:      macondopb.ChallengeRule_FIVE_POINT,
		GameMode:           mode,
		RatingMode:         pb.RatingMode_RATED,
		RequestId:          "ready-" + mode.String(),
	}
	g, err := gameplay.InstantiateNewGame(ctx, st.GameStore, config.DefaultConfig(), users, req, nil)
	is.NoErr(err)
	return g
}

func readyFlag(t *testing.T, ctx context.Context, gameID string) int64 {
	pool, err := common.OpenTestingDB(internalTestPkg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var rf int64
	if err := pool.QueryRow(ctx, "SELECT ready_flag FROM games WHERE uuid = $1", gameID).Scan(&rf); err != nil {
		t.Fatal(err)
	}
	return rf
}

// The client sends ReadyForGame whenever a player opens the game room. Once a
// game has started that must do nothing -- not mark the player ready, and not
// try to start the game again. A correspondence game starts without the ready
// step, so this is the first anyone hears from either player.
func TestReadyForStartedGameDoesNothing(t *testing.T) {
	is := is.New(t)
	ctx, b, st, users := readyTestSetup(t)

	g := readyTestGame(t, ctx, st, users, pb.GameMode_CORRESPONDENCE)
	is.NoErr(gameplay.StartGame(ctx, st, b.gameEventChan, g))

	for _, u := range users {
		is.NoErr(b.readyForGame(ctx, &pb.ReadyForGame{GameId: g.GameID()}, u.UUID))
		is.NoErr(b.readyForGame(ctx, &pb.ReadyForGame{GameId: g.GameID()}, u.UUID))
	}
	is.Equal(readyFlag(t, ctx, g.GameID()), int64(0))
}

// Before the start, both players being ready still starts the game.
func TestReadyFromBothPlayersStartsTheGame(t *testing.T) {
	is := is.New(t)
	ctx, b, st, users := readyTestSetup(t)

	g := readyTestGame(t, ctx, st, users, pb.GameMode_REAL_TIME)
	is.NoErr(st.GameStore.Set(ctx, g))

	is.NoErr(b.readyForGame(ctx, &pb.ReadyForGame{GameId: g.GameID()}, users[0].UUID))
	loaded, err := st.GameStore.Get(ctx, g.GameID())
	is.NoErr(err)
	is.True(!loaded.Started)

	is.NoErr(b.readyForGame(ctx, &pb.ReadyForGame{GameId: g.GameID()}, users[1].UUID))
	loaded, err = st.GameStore.Get(ctx, g.GameID())
	is.NoErr(err)
	is.True(loaded.Started)
	is.Equal(readyFlag(t, ctx, g.GameID()), int64(3))
}
