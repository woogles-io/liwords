package gameplay_test

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/lithammer/shortuuid"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/domino14/liwords/pkg/gameplay"
)

func TestHandleAbort(t *testing.T) {
	is := is.New(t)
	gsetup := setupNewGame()
	evtID := shortuuid.New()

	// Jesse requests an abort.
	metaEvt := &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    "3xpEkpRAy3AizbVmDg3kdi", // "jesse"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err := gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)

	is.NoErr(err)

	// Cesar accepts the abort
	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_ABORT_ACCEPTED,
		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)
	is.NoErr(err)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history, request abort, game deletion (from lobby),
	// game ended event, abort accepted
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 5)
	is.Equal(gsetup.g.Playing(), macondopb.PlayState_GAME_OVER)

	teardownGame(gsetup)
}

func TestHandleAbortTooManyTurns(t *testing.T) {
	is := is.New(t)
	gsetup := setupNewGame()
	evtID := shortuuid.New()

	histjson, err := ioutil.ReadFile("./testdata/game2/history.json")
	is.NoErr(err)
	hist := &macondopb.GameHistory{}
	err = json.Unmarshal(histjson, hist)
	is.NoErr(err)

	// Overwrite this test history a bit.
	for _, e := range hist.Events {
		if e.Nickname == "Mina" {
			e.Nickname = "jesse"
		}
	}
	hist.Players[0].Nickname = "jesse"
	hist.Players[0].UserId = "3xpEkpRAy3AizbVmDg3kdi"
	hist.Uid = gsetup.g.GameID()

	gsetup.g.SetHistory(hist)
	// This test is a little broken in that the game is actually already over,
	// but this wasn't changed in the gsetup.g -- it's ok, we're just measuring
	// the length of the events for it, for now.

	// Jesse requests an abort.
	metaEvt := &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    "3xpEkpRAy3AizbVmDg3kdi", // "jesse"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)
	is.Equal(err, gameplay.ErrTooManyTurns)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 1)

	teardownGame(gsetup)
}

func TestHandleAbortAcceptWrongId(t *testing.T) {
	is := is.New(t)
	gsetup := setupNewGame()
	evtID := shortuuid.New()

	// Jesse requests an abort.
	metaEvt := &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    "3xpEkpRAy3AizbVmDg3kdi", // "jesse"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err := gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)

	is.NoErr(err)

	// Cesar accepts the abort but passes in a wrong event id.
	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_ABORT_ACCEPTED,
		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
		GameId:      gsetup.g.GameID(),
		OrigEventId: "FOOBAR",
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)
	is.Equal(err, gameplay.ErrNoMatchingEvent)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history, request abort
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 2)

	teardownGame(gsetup)
}

func TestHandleAbortDeny(t *testing.T) {
	is := is.New(t)
	gsetup := setupNewGame()
	evtID := shortuuid.New()

	// Jesse requests an abort.
	metaEvt := &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    "3xpEkpRAy3AizbVmDg3kdi", // "jesse"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err := gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)

	is.NoErr(err)

	// Cesar denies the abort.
	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_ABORT_DENIED,
		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)
	is.NoErr(err)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history, request abort, deny abort
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 3)

	teardownGame(gsetup)
}

func TestHandleTooManyAborts(t *testing.T) {
	is := is.New(t)
	gsetup := setupNewGame()
	evtID := shortuuid.New()

	// Jesse requests an abort.
	metaEvt := &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    "3xpEkpRAy3AizbVmDg3kdi", // "jesse"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err := gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)

	is.NoErr(err)

	// Cesar denies the abort.
	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_ABORT_DENIED,
		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)
	is.NoErr(err)

	// Jesse requests an abort again.
	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    "3xpEkpRAy3AizbVmDg3kdi", // "jesse"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)
	is.Equal(err, gameplay.ErrTooManyAborts)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history, request abort, deny abort. The second
	// request never gets sent.
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 3)

	teardownGame(gsetup)
}

func TestHandleAbortTooLate(t *testing.T) {
	is := is.New(t)
	gsetup := setupNewGame()
	evtID := shortuuid.New()

	// cesar4 requests an abort after waiting too long.
	// jesse's timer has been running this whole time.
	// Sleep almost 25 minutes (the full time)
	gsetup.nower.Sleep((25*60 - 10) * 1000)

	metaEvt := &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err := gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)

	is.Equal(err, gameplay.ErrPleaseWaitToEnd)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 1)

	teardownGame(gsetup)
}

func TestHandleAbortAutoaccept(t *testing.T) {
	is := is.New(t)
	gsetup := setupNewGame()
	evtID := shortuuid.New()

	metaEvt := &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err := gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.lstore,
		gsetup.tstore)

	is.NoErr(err)

	// Sleep 61 seconds
	gsetup.nower.Sleep(61 * 1000)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 1)

	teardownGame(gsetup)
}
