package game

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func metaEvt(typ pb.GameMetaEvent_EventType, player, orig string, at time.Time) *pb.GameMetaEvent {
	return &pb.GameMetaEvent{
		Type: typ, PlayerId: player, OrigEventId: orig, GameId: atomGID,
		Timestamp: timestamppb.New(at), Expiry: 60000,
	}
}

func storedMetaEvents(t *testing.T, gstore *DBStore, gid string) ([]*pb.GameMetaEvent, []time.Time) {
	t.Helper()
	rows, err := gstore.queries.GetGameMetaEvents(context.Background(), gid)
	if err != nil {
		t.Fatal(err)
	}
	evts := make([]*pb.GameMetaEvent, len(rows))
	ats := make([]time.Time, len(rows))
	for i, r := range rows {
		if int(r.Idx) != i {
			t.Fatalf("row %d has idx %d", i, r.Idx)
		}
		evts[i] = &pb.GameMetaEvent{}
		if err := json.Unmarshal(r.Event, evts[i]); err != nil {
			t.Fatal(err)
		}
		ats[i] = r.CreatedAt.Time
	}
	return evts, ats
}

func sameMetaEvents(is *is.I, got, want []*pb.GameMetaEvent) {
	is.Equal(len(got), len(want))
	for i := range want {
		is.True(proto.Equal(got[i], want[i]))
	}
}

// Each save writes the events it hasn't written yet, and leaves the rest.
func TestSaveAppendsMetaEvents(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := context.Background()

	at := time.Date(2026, 11, 12, 3, 4, 5, 678_000_000, time.UTC)
	g, err := gstore.Get(ctx, atomGID)
	is.NoErr(err)
	g.MetaEvents.Events = append(g.MetaEvents.Events,
		metaEvt(pb.GameMetaEvent_REQUEST_ABORT, "p1", "e1", at))
	is.NoErr(gstore.Set(ctx, g))

	got, ats := storedMetaEvents(t, gstore, atomGID)
	sameMetaEvents(is, got, g.MetaEvents.Events)
	is.True(ats[0].Equal(at)) // when it happened, not when it was saved

	// The answer is saved later, from a fresh load.
	g, err = gstore.Get(ctx, atomGID)
	is.NoErr(err)
	g.MetaEvents.Events = append(g.MetaEvents.Events,
		metaEvt(pb.GameMetaEvent_ABORT_DENIED, "p2", "e1", at.Add(time.Minute)))
	is.NoErr(gstore.Set(ctx, g))

	got, _ = storedMetaEvents(t, gstore, atomGID)
	sameMetaEvents(is, got, g.MetaEvents.Events)

	// And the column still says the same thing: it's what is read.
	g, err = gstore.Get(ctx, atomGID)
	is.NoErr(err)
	sameMetaEvents(is, g.MetaEvents.Events, got)
}

// Games saved before the table existed only have the column; the backfill
// copies them, and copying twice changes nothing.
func TestBackfillCopiesMetaEvents(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()
	ctx := context.Background()

	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	want := []*pb.GameMetaEvent{
		metaEvt(pb.GameMetaEvent_REQUEST_ADJUDICATION, "p1", "n1", at),
		metaEvt(pb.GameMetaEvent_ADJUDICATION_DENIED, "p2", "n1", at.Add(time.Second)),
	}
	// Written the way the previous release wrote them: the column only.
	is.NoErr(gstore.UpdateMetaEvents(ctx, atomGID, entity.MetaEventData{Events: want}))
	got, _ := storedMetaEvents(t, gstore, atomGID)
	is.Equal(len(got), 0)

	maxID, err := gstore.MaxGameID(ctx)
	is.NoErr(err)
	// Every other game in the fixtures has {"events": null} or no events,
	// which must be skipped rather than fail the batch.
	n, err := gstore.BackfillMetaEvents(ctx, 0, int32(maxID)+1)
	is.NoErr(err)
	is.Equal(n, int64(2))

	got, ats := storedMetaEvents(t, gstore, atomGID)
	sameMetaEvents(is, got, want)
	is.True(ats[1].Equal(at.Add(time.Second)))

	n, err = gstore.BackfillMetaEvents(ctx, 0, int32(maxID)+1)
	is.NoErr(err)
	is.Equal(n, int64(0))
}
