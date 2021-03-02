package gameplay_test

import (
	"context"
	"testing"
	"time"

	"github.com/lithammer/shortuuid"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"

	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/stores/stats"
	ts "github.com/domino14/liwords/pkg/stores/tournament"
	"github.com/domino14/liwords/pkg/stores/user"
)

func TestHandleAbort(t *testing.T) {
	is := is.New(t)
	recreateDB()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	ustore := userStore(cstr)
	lstore := listStatStore(cstr)
	cfg, gstore := gameStore(cstr, ustore)
	tstore := tournamentStore(cfg, gstore)

	g, _, cancel, donechan, consumer := makeGame(cfg, ustore, gstore)

	evtID := shortuuid.New()

	// Jesse requests an abort.
	metaEvt := &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    "3xpEkpRAy3AizbVmDg3kdi", // "jesse"
		GameId:      g.GameID(),
		OrigEventId: evtID,
	}

	err := gameplay.HandleMetaEvent(context.Background(), metaEvt,
		consumer.ch, gstore, ustore, lstore, tstore)

	is.NoErr(err)

	// Cesar accepts the abort
	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_ABORT_ACCEPTED,
		PlayerId:    "xjCWug7EZtDxDHX5fRZTLo", // "cesar4"
		GameId:      g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		consumer.ch, gstore, ustore, lstore, tstore)
	is.NoErr(err)

	cancel()
	<-donechan

	// expected events: game history, request abort, game deletion (from lobby),
	// game ended event, abort accepted
	log.Debug().Interface("evts", consumer.evts).Msg("evts")
	is.Equal(len(consumer.evts), 5)

	ustore.(*user.DBStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
	gstore.(*game.Cache).Disconnect()
	tstore.(*ts.Cache).Disconnect()
}
