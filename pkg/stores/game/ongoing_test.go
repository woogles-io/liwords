package game

// ongoing_games has to track a live game exactly, because a listing that
// disagrees with the game it names is worse than no listing at all.

import (
	"context"
	"testing"

	"github.com/matryer/is"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const ongoingGID = "wJxURccCgSAPivUvj4QdYL"

func ongoingCount(t *testing.T, gstore *DBStore) int {
	t.Helper()
	n, err := gstore.queries.CountOngoingGames(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return int(n)
}

// writeOngoingCtx enables the flag for one context only.
//
// config.DefaultConfig() returns a package-level singleton *Config, so
// `cfg := DefaultConfig; cfg.Flag = true` sets that flag for the whole process
// and every test that runs after. Copy the struct.
func writeOngoingCtx() context.Context {
	cfg := *DefaultConfig
	cfg.WriteOngoingGames = true
	return cfg.WithContext(context.Background())
}

func TestOngoingRowFollowsTheGame(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := writeOngoingCtx()

	is.Equal(ongoingCount(t, gstore), 0)

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, entGame))

	is.Equal(ongoingCount(t, gstore), 1)
	row, err := gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(int(row.PlayState), int(entGame.Playing()))
	is.Equal(int(row.OnTurn.Int16), entGame.PlayerOnTurn())
	is.Equal(row.Lexicon, entGame.History().Lexicon)

	// A move moves the row with it.
	before := len(entGame.History().Events)
	_, err = entGame.PlayScoringMove("8E", "AGUE", true)
	is.NoErr(err)
	is.NoErr(gstore.StageTurns(entGame, before, entGame.History().Events[before:]))
	is.NoErr(gstore.Set(ctx, entGame))

	row, err = gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(int(row.OnTurn.Int16), entGame.PlayerOnTurn())
}

// A finished game is not ongoing, and stops being listed in the same
// transaction that records that it ended.
func TestOngoingRowGoesWhenTheGameEnds(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := writeOngoingCtx()

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, entGame))
	is.Equal(ongoingCount(t, gstore), 1)

	entGame.SetGameEndReason(pb.GameEndReason_STANDARD)
	entGame.SetPlaying(macondopb.PlayState_GAME_OVER)
	is.NoErr(gstore.Set(ctx, entGame))
	is.Equal(ongoingCount(t, gstore), 0)
}

// Readiness is ORed in one bit per player. A save landing between the two
// players readying must not wipe the first -- which is the bug games.ready_flag
// has today, and the reason the upsert leaves the column alone.
func TestOngoingSaveDoesNotClobberReadiness(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := writeOngoingCtx()

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, entGame))

	flag, err := gstore.queries.SetOngoingGameReady(ctx, models.SetOngoingGameReadyParams{
		GameUuid: ongoingGID, PlayerIdx: 0,
	})
	is.NoErr(err)
	is.Equal(flag, int64(1))

	// Another save, before the second player is ready.
	is.NoErr(gstore.Set(ctx, entGame))

	row, err := gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(row.ReadyFlag, int64(1))
}

// Off by default: nothing is written unless the flag says so.
func TestOngoingWriteIsOffByDefault(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, entGame))
	is.Equal(ongoingCount(t, gstore), 0)
}

// Readiness is accumulated by the database, one bit per player, so a save
// between the two players readying must leave the first player's bit alone.
//
// UpdateGame used to write a literal 0 to games.ready_flag on every save --
// entity.Game has no such field, so there was nothing else to write. A meta
// event arriving while the first player waited was enough: the second player's
// ready then returned 2 rather than 3, StartGame never fired, and the first
// player had to ready up again.
func TestSaveDoesNotClobberReadiness(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := context.Background()

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)

	// The fixture arrives with bits already set; start from nobody ready.
	_, err = gstore.dbPool.Exec(ctx, "UPDATE games SET ready_flag = 0 WHERE uuid = $1", ongoingGID)
	is.NoErr(err)

	// Player 0 is ready.
	flag, err := gstore.SetReady(ctx, ongoingGID, 0)
	is.NoErr(err)
	is.Equal(flag, 1)

	// Anything at all saves the game while player 1 is still deciding.
	is.NoErr(gstore.Set(ctx, entGame))

	// Player 1 is ready. Both bits must be set, or the game never starts.
	flag, err = gstore.SetReady(ctx, ongoingGID, 1)
	is.NoErr(err)
	is.Equal(flag, 3)
}

func contains(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// A game that was live before the flag went on, and is not saved again, gets
// its row from the backfill -- the same row a save would have written.
func TestOngoingBackfill(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := writeOngoingCtx()

	// One player readied before the row existed.
	_, err := gstore.dbPool.Exec(ctx, "UPDATE games SET ready_flag = 1 WHERE uuid = $1", ongoingGID)
	is.NoErr(err)

	ids, err := gstore.ListLiveGamesWithoutOngoingRow(ctx)
	is.NoErr(err)
	is.True(contains(ids, ongoingGID))

	wrote, err := gstore.BackfillOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.True(wrote)

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	row, err := gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(int(row.PlayState), int(entGame.Playing()))
	is.Equal(int(row.OnTurn.Int16), entGame.PlayerOnTurn())
	is.Equal(row.Lexicon, entGame.History().Lexicon)
	is.Equal(row.Player0ID, int32(entGame.PlayerDBIDs[0]))
	is.Equal(row.ReadyFlag, int64(1)) // copied, not reset

	// Idempotent: the second run finds the row and leaves it.
	wrote, err = gstore.BackfillOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.True(!wrote)
	ids, err = gstore.ListLiveGamesWithoutOngoingRow(ctx)
	is.NoErr(err)
	is.True(!contains(ids, ongoingGID))
}

// A game that ended is not ongoing, so the backfill must not give it a row --
// whether the listing was read before it ended or after.
func TestOngoingBackfillSkipsEndedGames(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := writeOngoingCtx()

	ids, err := gstore.ListLiveGamesWithoutOngoingRow(ctx)
	is.NoErr(err)
	is.True(contains(ids, ongoingGID))

	// It ends after the listing was read. Saved with the flag off, so there is
	// no row for the save to delete.
	off := DefaultConfig.WithContext(context.Background())
	entGame, err := gstore.Get(off, ongoingGID)
	is.NoErr(err)
	entGame.SetGameEndReason(pb.GameEndReason_STANDARD)
	entGame.SetPlaying(macondopb.PlayState_GAME_OVER)
	is.NoErr(gstore.Set(off, entGame))

	wrote, err := gstore.BackfillOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.True(!wrote)
	is.Equal(ongoingCount(t, gstore), 0)

	ids, err = gstore.ListLiveGamesWithoutOngoingRow(ctx)
	is.NoErr(err)
	is.True(!contains(ids, ongoingGID))
}

// The first save after the flag goes on copies readiness from the games row,
// and from then on SetReady keeps both in step.
func TestOngoingReadinessFollowsTheGame(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := writeOngoingCtx()

	_, err := gstore.dbPool.Exec(ctx, "UPDATE games SET ready_flag = 0 WHERE uuid = $1", ongoingGID)
	is.NoErr(err)

	// Player 0 readies before the game has a row.
	flag, err := gstore.SetReady(ctx, ongoingGID, 0)
	is.NoErr(err)
	is.Equal(flag, 1)
	is.Equal(ongoingCount(t, gstore), 0)

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, entGame))
	row, err := gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(row.ReadyFlag, int64(1))

	// Player 1 readies once the row exists: both rows get the bit.
	flag, err = gstore.SetReady(ctx, ongoingGID, 1)
	is.NoErr(err)
	is.Equal(flag, 3)
	row, err = gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(row.ReadyFlag, int64(3))

	// Readying again changes nothing and is not an error.
	flag, err = gstore.SetReady(ctx, ongoingGID, 1)
	is.NoErr(err)
	is.Equal(flag, 0)
}
