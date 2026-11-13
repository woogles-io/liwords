package game

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// gamePlayersStartingRatings reads a finished game's starting ratings from
// game_players, in player order.
func gamePlayersStartingRatings(t *testing.T, gstore *DBStore, gid string) [2]*entity.StartingRating {
	t.Helper()
	rows, err := gstore.dbPool.Query(context.Background(),
		`SELECT starting_rating, starting_rating_provisional FROM game_players
		 WHERE game_uuid = $1 ORDER BY player_index`, gid)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out [2]*entity.StartingRating
	i := 0
	for rows.Next() {
		var r pgtype.Float4
		var p pgtype.Bool
		if err := rows.Scan(&r, &p); err != nil {
			t.Fatal(err)
		}
		out[i] = startingRatingFromColumns(r, p)
		i++
	}
	if i != 2 {
		t.Fatalf("game %s has %d game_players rows, want 2", gid, i)
	}
	return out
}

// A game's starting ratings are written to its ongoing row when it is
// created, read back on every load, never changed by a save, and copied to
// game_players when it ends.
func TestStartingRatingsFollowTheGame(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := writeOngoingCtx()

	const gid = "startingRating00000000"
	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	entGame.History().Uid = gid
	entGame.Started = false
	// Player 1 has no rating, like an anonymous player.
	entGame.StartingRatings = [2]*entity.StartingRating{{Rating: 1612.25, Provisional: false}, nil}
	entGame.Quickdata.PlayerInfo[1].Rating = "UnratedAnon"
	is.NoErr(gstore.Create(ctx, entGame))

	row, err := gstore.queries.GetOngoingGame(ctx, gid)
	is.NoErr(err)
	is.Equal(row.Player0StartingRating, pgtype.Float4{Float32: 1612.25, Valid: true})
	is.Equal(row.Player0StartingRatingProvisional, pgtype.Bool{Bool: false, Valid: true})
	is.True(!row.Player1StartingRating.Valid)
	is.True(!row.Player1StartingRatingProvisional.Valid)

	// A load reads them back.
	entGame, err = gstore.Get(ctx, gid)
	is.NoErr(err)
	is.Equal(*entGame.StartingRatings[0], entity.StartingRating{Rating: 1612.25, Provisional: false})
	is.Equal(entGame.StartingRatings[1], (*entity.StartingRating)(nil))

	// A save never changes them, even one that carries others.
	entGame.StartingRatings[0] = &entity.StartingRating{Rating: 2000, Provisional: true}
	is.NoErr(gstore.Set(ctx, entGame))
	row, err = gstore.queries.GetOngoingGame(ctx, gid)
	is.NoErr(err)
	is.Equal(row.Player0StartingRating, pgtype.Float4{Float32: 1612.25, Valid: true})
	is.Equal(row.Player0StartingRatingProvisional, pgtype.Bool{Bool: false, Valid: true})

	// The game ends: its ongoing row goes, and game_players gets the ratings
	// it was loaded with.
	entGame, err = gstore.Get(ctx, gid)
	is.NoErr(err)
	entGame.SetGameEndReason(pb.GameEndReason_RESIGNED)
	entGame.SetPlaying(macondopb.PlayState_GAME_OVER)
	is.NoErr(gstore.Set(ctx, entGame))
	is.NoErr(gstore.InsertGamePlayers(ctx, entGame))
	got := gamePlayersStartingRatings(t, gstore, gid)
	is.Equal(*got[0], entity.StartingRating{Rating: 1612.25, Provisional: false})
	is.Equal(got[1], (*entity.StartingRating)(nil))
}

// A game created before the columns existed has an ongoing row without
// starting ratings, or no row at all. A load takes them from its quickdata,
// and the next save fills in the row.
func TestStartingRatingsFromQuickdata(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := writeOngoingCtx()

	// The fixture's quickdata has "1600?" and "500?", and it has no row.
	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(*entGame.StartingRatings[0], entity.StartingRating{Rating: 1600, Provisional: true})
	is.Equal(*entGame.StartingRatings[1], entity.StartingRating{Rating: 500, Provisional: true})

	// A row written without them, as by the code before them.
	entGame.StartingRatings = [2]*entity.StartingRating{}
	is.NoErr(gstore.Set(ctx, entGame))
	row, err := gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.True(!row.Player0StartingRating.Valid)

	// Loading and saving fills it in.
	entGame, err = gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, entGame))
	row, err = gstore.queries.GetOngoingGame(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(row.Player0StartingRating, pgtype.Float4{Float32: 1600, Valid: true})
	is.Equal(row.Player0StartingRatingProvisional, pgtype.Bool{Bool: true, Valid: true})
	is.Equal(row.Player1StartingRating, pgtype.Float4{Float32: 500, Valid: true})
}
