package game

import (
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// gamesMetaEvents reads the games copy of a game's meta events, as text.
func gamesMetaEvents(t *testing.T, s *DBStore, gid string) string {
	t.Helper()
	var me *string
	if err := s.dbPool.QueryRow(testCtx(), `SELECT meta_events::text FROM games WHERE uuid = $1`, gid).Scan(&me); err != nil {
		t.Fatal(err)
	}
	if me == nil {
		return "<null>"
	}
	return *me
}

// A live game's meta events are saved to its ongoing row only, and a load
// reads them from there.
func TestMetaEventsLiveInTheOngoingRow(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	before := gamesMetaEvents(t, gstore, ongoingGID)

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(len(entGame.MetaEvents.Events), 0)
	entGame.MetaEvents.Events = append(entGame.MetaEvents.Events, &pb.GameMetaEvent{
		OrigEventId: "nudge1",
		Type:        pb.GameMetaEvent_REQUEST_ADJUDICATION,
		PlayerId:    "somebody",
		GameId:      ongoingGID,
		Expiry:      120000,
	})
	is.NoErr(gstore.Set(ctx, entGame))

	is.Equal(gamesMetaEvents(t, gstore, ongoingGID), before) // games copy untouched

	reloaded, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(len(reloaded.MetaEvents.Events), 1)
	is.Equal(reloaded.MetaEvents.Events[0].OrigEventId, "nudge1")
	is.Equal(reloaded.MetaEvents.Events[0].Type, pb.GameMetaEvent_REQUEST_ADJUDICATION)

	// The denial is appended and saved the same way.
	reloaded.MetaEvents.Events = append(reloaded.MetaEvents.Events, &pb.GameMetaEvent{
		OrigEventId: "nudge1",
		Type:        pb.GameMetaEvent_ADJUDICATION_DENIED,
		PlayerId:    "someone else",
		GameId:      ongoingGID,
	})
	is.NoErr(gstore.Set(ctx, reloaded))
	again, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(len(again.MetaEvents.Events), 2)
	is.Equal(again.MetaEvents.Events[1].Type, pb.GameMetaEvent_ADJUDICATION_DENIED)
}

// A game with no ongoing row -- a finished one, or one that never got a row
// -- keeps whatever the games column holds.
func TestMetaEventsWithoutOngoingRowComeFromGames(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := testCtx()

	_, err := gstore.dbPool.Exec(ctx, `UPDATE games SET meta_events =
		'{"events": [{"type": 1, "orig_event_id": "old", "game_id": "`+ongoingGID+`"}]}'
		WHERE uuid = $1`, ongoingGID)
	is.NoErr(err)
	is.Equal(ongoingCount(t, gstore), 0)

	entGame, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(len(entGame.MetaEvents.Events), 1)
	is.Equal(entGame.MetaEvents.Events[0].OrigEventId, "old")

	// The first save gives it a row carrying those events, and from then on
	// the row is what is read.
	is.NoErr(gstore.Set(ctx, entGame))
	_, err = gstore.dbPool.Exec(ctx, `UPDATE games SET meta_events = '{"events": null}' WHERE uuid = $1`, ongoingGID)
	is.NoErr(err)
	reloaded, err := gstore.Get(ctx, ongoingGID)
	is.NoErr(err)
	is.Equal(len(reloaded.MetaEvents.Events), 1)
}
