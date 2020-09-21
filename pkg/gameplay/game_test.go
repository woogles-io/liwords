package gameplay

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/stores/stats"
	"github.com/domino14/liwords/pkg/stores/user"
	pkguser "github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/alphabet"
)

type FakeNower struct {
	fakeMeow int64
}

func (f FakeNower) Now() int64 {
	return f.fakeMeow
}

func gameStore(dbURL string, userStore pkguser.Store) (*config.Config, GameStore) {
	cfg := &config.Config{}
	cfg.MacondoConfig = DefaultConfig
	cfg.DBConnString = dbURL

	tmp, err := game.NewDBStore(cfg, userStore)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	gameStore := game.NewCache(tmp)
	return cfg, gameStore
}

type evtConsumer struct {
	evts []*entity.EventWrapper
}

func (ec *evtConsumer) consumeEventChan(ctx context.Context,
	ch chan *entity.EventWrapper,
	done chan bool) {
	defer func() { done <- true }()
	for {
		select {
		case msg := <-ch:
			ec.evts = append(ec.evts, msg)
		case <-ctx.Done():
			return
		}
	}
}

func makeGame(cfg *config.Config, ustore pkguser.Store, gstore GameStore) (
	*entity.Game, *FakeNower, context.CancelFunc, chan bool, *evtConsumer) {

	ctx := context.Background()
	cesar, _ := ustore.Get(ctx, "cesar4")
	jesse, _ := ustore.Get(ctx, "jesse")
	// see the gameReq in game_test.go in this package
	gr := proto.Clone(gameReq).(*pb.GameRequest)

	gr.IncrementSeconds = 5
	gr.MaxOvertimeMinutes = 0
	g, _ := InstantiateNewGame(ctx, gstore, cfg, [2]*entity.User{cesar, jesse},
		1, gr)

	ch := make(chan *entity.EventWrapper)
	donechan := make(chan bool)
	consumer := &evtConsumer{}

	cctx, cancel := context.WithCancel(ctx)
	go consumer.consumeEventChan(cctx, ch, donechan)

	nower := &FakeNower{}
	g.SetTimerModule(nower)

	StartGame(ctx, gstore, ch, g.GameID())

	return g, nower, cancel, donechan, consumer
}

func TestInitializeGame(t *testing.T) {
	is := is.New(t)
	recreateDB()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	ustore := userStore(cstr)
	lstore := listStatStore(cstr)
	cfg, gstore := gameStore(cstr, ustore)

	g, _, cancel, donechan, consumer := makeGame(cfg, ustore, gstore)

	is.Equal(g.PlayerOnTurn(), 1)
	cancel()
	<-donechan
	// It should just be a single GameHistory event.
	is.Equal(len(consumer.evts), 1)
	ustore.(*user.DBStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
	gstore.(*game.Cache).Disconnect()
}

func TestWrongTurn(t *testing.T) {
	is := is.New(t)
	recreateDB()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	ustore := userStore(cstr)
	lstore := listStatStore(cstr)
	cfg, gstore := gameStore(cstr, ustore)

	g, _, cancel, donechan, consumer := makeGame(cfg, ustore, gstore)

	is.Equal(g.PlayerOnTurn(), 1)

	cge := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         g.GameID(),
		PositionCoords: "8D",
		Tiles:          "BANJO",
	}

	// User ID below is "cesar4" who's not on turn.
	_, err := HandleEvent(context.Background(), gstore, ustore, lstore,
		"xjCWug7EZtDxDHX5fRZTLo", cge)

	is.Equal(err.Error(), "player not on turn")

	cancel()
	<-donechan
	// It should just be a single GameHistory event.
	is.Equal(len(consumer.evts), 1)
	ustore.(*user.DBStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
	gstore.(*game.Cache).Disconnect()
}

func Test5ptBadWord(t *testing.T) {
	is := is.New(t)
	recreateDB()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	ustore := userStore(cstr)
	lstore := listStatStore(cstr)
	cfg, gstore := gameStore(cstr, ustore)

	g, nower, cancel, donechan, consumer := makeGame(cfg, ustore, gstore)

	cge := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         g.GameID(),
		PositionCoords: "8D",
		Tiles:          "BANJO",
	}
	g.SetRacksForBoth([]*alphabet.Rack{
		alphabet.RackFromString("AGLSYYZ", g.Alphabet()),
		alphabet.RackFromString("ABEJNOR", g.Alphabet()),
	})
	// "jesse" plays a word after some time
	nower.fakeMeow += 3750 // 3.75 secs
	_, err := HandleEvent(context.Background(), gstore, ustore, lstore,
		"3xpEkpRAy3AizbVmDg3kdi", cge)

	is.NoErr(err)

	// Kill the go-routine and let's see the events.
	cancel()
	<-donechan

	is.Equal(len(consumer.evts), 2)
	// get some fields to make sure the move was played properly.
	evt := consumer.evts[1].Event.(*pb.ServerGameplayEvent)
	is.Equal(evt.Event.Score, int32(34))
	is.Equal(evt.UserId, "3xpEkpRAy3AizbVmDg3kdi")
	// starting time is 25*60 secs, plus a 5-second increment, and they spent 3750 ms on the move.
	// TimeRemaining is in ms.
	is.Equal(evt.TimeRemaining, int32((25*60000)+1250))

	ustore.(*user.DBStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
	gstore.(*game.Cache).Disconnect()
}
