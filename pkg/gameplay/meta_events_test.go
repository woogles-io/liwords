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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
		gsetup.tstore)
	is.NoErr(err)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history, request abort, game deletion (from lobby),
	// game ended event, active game entry, abort accepted
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 6)
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
		gsetup.tstore)

	is.NoErr(err)

	// Sleep 61 seconds
	gsetup.nower.Sleep(61 * 1000)

	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_TIMER_EXPIRED,
		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
		gsetup.tstore)
	is.NoErr(err)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history, request abort, game deletion (from lobby),
	// game ended event, active game entry
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 5)

	teardownGame(gsetup)
}

func TestHandleAbortAutoacceptTooEarly(t *testing.T) {
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
		gsetup.tstore)

	is.NoErr(err)

	// Sleep 59 seconds
	gsetup.nower.Sleep(59 * 1000)

	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_TIMER_EXPIRED,
		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
		gsetup.tstore)
	is.Equal(err, gameplay.ErrMetaEventExpirationIncorrect)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history, request abort
	// this game wasn't aborted (this really shouldn't happen in the front end tho)
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 2)

	teardownGame(gsetup)
}

func TestHandleAbortDenyThenAccept(t *testing.T) {
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
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
		gsetup.tstore)

	is.NoErr(err)

	// Cesar denies the abort
	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_ABORT_DENIED,
		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
		gsetup.tstore)
	is.NoErr(err)

	gsetup.nower.Sleep(120 * 1000)
	// Now cesar is losing, can he accept an abort he denied? What a cheater!
	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_ABORT_ACCEPTED,
		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
		GameId:      gsetup.g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
		gsetup.tstore)
	is.Equal(err, gameplay.ErrNoMatchingEvent)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history, request abort, deny abort
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 3)

	teardownGame(gsetup)
}

// Both aborting can just be handled by the front-end accepting an outstanding
// abort request.
// func TestBothAbort(t *testing.T) {
// 	is := is.New(t)
// 	gsetup := setupNewGame()

// 	// Jesse requests an abort.
// 	metaEvt := &pb.GameMetaEvent{
// 		Timestamp:   timestamppb.New(time.Now()),
// 		Type:        pb.GameMetaEvent_REQUEST_ABORT,
// 		PlayerId:    "3xpEkpRAy3AizbVmDg3kdi", // "jesse"
// 		GameId:      gsetup.g.GameID(),
// 		OrigEventId: shortuuid.New(),
// 	}

// 	err := gameplay.HandleMetaEvent(context.Background(), metaEvt,
// 		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
// 		gsetup.tstore)

// 	is.NoErr(err)

// 	// At nearly the same time cesar4 requests an abort. The abort should
// 	// just go through.
// 	metaEvt = &pb.GameMetaEvent{
// 		Timestamp:   timestamppb.New(time.Now()),
// 		Type:        pb.GameMetaEvent_REQUEST_ABORT,
// 		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "jesse"
// 		GameId:      gsetup.g.GameID(),
// 		OrigEventId: shortuuid.New(),
// 	}
// 	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
// 		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
// 		gsetup.tstore)

// 	is.NoErr(err)

// 	gsetup.cancel()
// 	<-gsetup.donechan

// 	// expected events: game history, request abort, request abort, game deletion (from lobby),
// 	// game ended event, abort accepted
// 	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
// 	is.Equal(len(gsetup.consumer.evts), 6)
// 	is.Equal(gsetup.g.Playing(), macondopb.PlayState_GAME_OVER)

// 	teardownGame(gsetup)
// }

// Test that making a play auto-answers/cancels a nudge.
// XXX: Should this also be done only on the front end?
// It's a pain to do this on the backend.
// func TestPlayCancelsNudge(t *testing.T) {
// 	is := is.New(t)
// 	gsetup := setupNewGame()
// 	evtID := shortuuid.New()

// 	metaEvt := &pb.GameMetaEvent{
// 		Timestamp:   timestamppb.New(time.Now()),
// 		Type:        pb.GameMetaEvent_REQUEST_ADJUDICATION,
// 		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
// 		GameId:      gsetup.g.GameID(),
// 		OrigEventId: evtID,
// 	}

// 	err := gameplay.HandleMetaEvent(context.Background(), metaEvt,
// 		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
// 		gsetup.tstore)

// 	is.NoErr(err)

// 	// Jesse makes a play instead of answering the adjudication.
// 	cge := &pb.ClientGameplayEvent{
// 		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
// 		GameId:         gsetup.g.GameID(),
// 		PositionCoords: "8D",
// 		Tiles:          "BANJO",
// 	}
// 	gsetup.g.SetRacksForBoth([]*alphabet.Rack{
// 		alphabet.RackFromString("AGLSYYZ", gsetup.g.Alphabet()),
// 		alphabet.RackFromString("ABEJNOR", gsetup.g.Alphabet()),
// 	})
// 	// "jesse" plays a word after some time
// 	gsetup.nower.Sleep(3750) // 3.75 secs
// 	_, err = gameplay.HandleEvent(context.Background(), gsetup.gstore, gsetup.ustore,
// 		gsetup.lstore, gsetup.tstore, "3xpEkpRAy3AizbVmDg3kdi", cge)
// 	is.NoErr(err)
// 	// Sleep longer than the nudge interval of seconds
// 	gsetup.nower.Sleep(180 * 1000)

// 	metaEvt = &pb.GameMetaEvent{
// 		Timestamp:   timestamppb.New(time.Now()),
// 		Type:        pb.GameMetaEvent_TIMER_EXPIRED,
// 		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
// 		GameId:      gsetup.g.GameID(),
// 		OrigEventId: evtID,
// 	}

// 	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
// 		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
// 		gsetup.tstore)
// 	is.NoErr(err)

// 	gsetup.cancel()
// 	<-gsetup.donechan

// 	// expected events: game history, request abort, game deletion (from lobby),
// 	// game ended event
// 	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
// 	is.Equal(len(gsetup.consumer.evts), 4)

// 	teardownGame(gsetup)
// }

// test disallow abort AND adjudication to be sent

func TestDisallowAbortAndAdjudication(t *testing.T) {
	is := is.New(t)
	gsetup := setupNewGame()

	// Jesse requests an abort.
	metaEvt := &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    "3xpEkpRAy3AizbVmDg3kdi", // "jesse"
		GameId:      gsetup.g.GameID(),
		OrigEventId: shortuuid.New(),
	}

	err := gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
		gsetup.tstore)

	is.NoErr(err)

	// Jesse requests an adjudication
	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ADJUDICATION,
		PlayerId:    "3xpEkpRAy3AizbVmDg3kdi",
		GameId:      gsetup.g.GameID(),
		OrigEventId: shortuuid.New(),
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		gsetup.consumer.ch, gsetup.gstore, gsetup.ustore, gsetup.nstore, gsetup.lstore,
		gsetup.tstore)
	is.Equal(err, gameplay.ErrAlreadyOutstandingRequest)

	gsetup.cancel()
	<-gsetup.donechan

	// expected events: game history, request abort
	log.Debug().Interface("evts", gsetup.consumer.evts).Msg("evts")
	is.Equal(len(gsetup.consumer.evts), 2)
	is.Equal(gsetup.g.Playing(), macondopb.PlayState_PLAYING)

	teardownGame(gsetup)
}
