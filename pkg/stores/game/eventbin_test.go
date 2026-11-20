package game

// game_turns rows are written in two encodings for one release, and read in
// whichever the row has.

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	"github.com/woogles-io/liwords/pkg/xwordbridge"
)

func TestTurnsAreWrittenInBothEncodings(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := context.Background()

	entGame := playOneMove(t, gstore, atomGID, "8E", "AGUE")
	is.NoErr(gstore.Set(ctx, entGame))
	want := entGame.History().Events[0]

	var js, bin []byte
	is.NoErr(gstore.dbPool.QueryRow(ctx,
		`SELECT event::text, event_bin FROM game_turns WHERE game_uuid = $1 AND turn_idx = 0`,
		atomGID).Scan(&js, &bin))

	// `event` is what the previous release reads, so it must still be whole.
	fromJSON := &macondopb.GameEvent{}
	is.NoErr(protojson.Unmarshal(js, fromJSON))
	is.True(proto.Equal(fromJSON, want))

	fromBin := &macondopb.GameEvent{}
	is.NoErr(proto.Unmarshal(bin, fromBin))
	is.True(proto.Equal(fromBin, want))

	// And the readers are handed the binary copy.
	turns, err := gstore.GetTurns(ctx, atomGID)
	is.NoErr(err)
	is.Equal(len(turns), 1)
	is.Equal(turns[0].Event, bin)
}

// A row written before event_bin existed has only the protojson copy, and must
// still be readable: live correspondence games carry such rows for weeks.
func TestTurnsWithoutBinaryCopyStillRead(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := context.Background()

	old := &macondopb.GameEvent{
		Type: macondopb.GameEvent_TILE_PLACEMENT_MOVE, PlayerIndex: 1,
		Position: "8E", PlayedTiles: "AGUE", Score: 10, Cumulative: 10,
	}
	js, err := protojson.Marshal(old)
	is.NoErr(err)
	is.NoErr(gstore.queries.AppendGameTurn(ctx, models.AppendGameTurnParams{
		GameUuid: atomGID, TurnIdx: 0, Event: js,
	}))

	turns, err := gstore.GetTurns(ctx, atomGID)
	is.NoErr(err)
	is.Equal(len(turns), 1)
	got, err := xwordbridge.DecodeTurn(turns[0].Event)
	is.NoErr(err)
	is.True(proto.Equal(got, old))

	last, err := gstore.GetLastTurn(ctx, atomGID)
	is.NoErr(err)
	got, err = xwordbridge.DecodeTurn(last.Event)
	is.NoErr(err)
	is.True(proto.Equal(got, old))

	row, err := gstore.queries.GetGameWithTurns(ctx, models.GetGameWithTurnsParams{
		Uuid: pgtype.Text{String: atomGID, Valid: true}, WithTurns: true,
	})
	is.NoErr(err)
	evts, err := xwordbridge.DecodeTurns(row.TurnEvents)
	is.NoErr(err)
	is.Equal(len(evts), 1)
	is.True(proto.Equal(evts[0], old))
}
