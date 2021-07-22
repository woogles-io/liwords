package gameplay_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	pkgmod "github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/stores/mod"
	"github.com/domino14/liwords/pkg/stores/stats"
	ts "github.com/domino14/liwords/pkg/stores/tournament"
	"github.com/domino14/liwords/pkg/stores/user"
	"github.com/domino14/liwords/pkg/tournament"
	pkguser "github.com/domino14/liwords/pkg/user"
	gs "github.com/domino14/liwords/rpc/api/proto/game_service"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/alphabet"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

func gameStore(dbURL string, userStore pkguser.Store) (*config.Config, gameplay.GameStore) {
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

func tournamentStore(cfg *config.Config, gs gameplay.GameStore) tournament.TournamentStore {
	tmp, err := ts.NewDBStore(cfg, gs)
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	tournamentStore := ts.NewCache(tmp)
	return tournamentStore
}

func notorietyStore(dbURL string) pkgmod.NotorietyStore {
	n, err := mod.NewNotorietyStore(TestingDBConnStr + " dbname=liwords_test")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return n
}

type evtConsumer struct {
	evts []*entity.EventWrapper
	ch   chan *entity.EventWrapper
}

func (ec *evtConsumer) consumeEventChan(ctx context.Context,
	ch chan *entity.EventWrapper,
	done chan bool) {

	ec.ch = ch

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

func makeGame(cfg *config.Config, ustore pkguser.Store, gstore gameplay.GameStore) (
	*entity.Game, *entity.FakeNower, context.CancelFunc, chan bool, *evtConsumer) {

	ctx := context.Background()
	cesar, _ := ustore.Get(ctx, "cesar4")
	jesse, _ := ustore.Get(ctx, "jesse")
	// see the gameReq in game_stats_test.go in this package
	gr := proto.Clone(gameReq).(*pb.GameRequest)

	gr.IncrementSeconds = 5
	gr.MaxOvertimeMinutes = 0
	g, _ := gameplay.InstantiateNewGame(ctx, gstore, cfg, [2]*entity.User{cesar, jesse},
		1, gr, nil)

	ch := make(chan *entity.EventWrapper)
	donechan := make(chan bool)
	consumer := &evtConsumer{}
	gstore.SetGameEventChan(ch)

	cctx, cancel := context.WithCancel(ctx)
	go consumer.consumeEventChan(cctx, ch, donechan)

	nower := entity.NewFakeNower(1234)
	g.SetTimerModule(nower)

	gameplay.StartGame(ctx, gstore, ustore, ch, g.GameID())

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
	nstore := notorietyStore(cstr)
	cfg, gstore := gameStore(cstr, ustore)
	tstore := tournamentStore(cfg, gstore)

	g, _, cancel, donechan, consumer := makeGame(cfg, ustore, gstore)

	is.Equal(g.PlayerOnTurn(), 1)

	cge := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         g.GameID(),
		PositionCoords: "8D",
		Tiles:          "BANJO",
	}

	// User ID below is "cesar4" who's not on turn.
	_, err := gameplay.HandleEvent(context.Background(), gstore, ustore, nstore, lstore, tstore,
		"xjCWug7EZtDxDHX5fRZTLo", cge)

	is.Equal(err.Error(), "player not on turn")

	cancel()
	<-donechan
	// It should just be a single GameHistory event.
	is.Equal(len(consumer.evts), 1)
	ustore.(*user.DBStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
	nstore.(*mod.NotorietyStore).Disconnect()
	gstore.(*game.Cache).Disconnect()
	tstore.(*ts.Cache).Disconnect()
}

func Test5ptBadWord(t *testing.T) {
	is := is.New(t)
	recreateDB()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	ustore := userStore(cstr)
	lstore := listStatStore(cstr)
	nstore := notorietyStore(cstr)
	cfg, gstore := gameStore(cstr, ustore)
	tstore := tournamentStore(cfg, gstore)

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
	nower.Sleep(3750) // 3.75 secs
	_, err := gameplay.HandleEvent(context.Background(), gstore, ustore, nstore, lstore, tstore,
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
	nstore.(*mod.NotorietyStore).Disconnect()
	gstore.(*game.Cache).Disconnect()
	tstore.(*ts.Cache).Disconnect()
}

func TestDoubleChallengeBadWord(t *testing.T) {
	is := is.New(t)
	recreateDB()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	ustore := userStore(cstr)
	lstore := listStatStore(cstr)
	nstore := notorietyStore(cstr)
	cfg, gstore := gameStore(cstr, ustore)
	tstore := tournamentStore(cfg, gstore)

	g, nower, cancel, donechan, consumer := makeGame(cfg, ustore, gstore)

	cge := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         g.GameID(),
		PositionCoords: "8D",
		Tiles:          "BANJOER",
	}
	g.SetChallengeRule(macondopb.ChallengeRule_DOUBLE)
	g.SetRacksForBoth([]*alphabet.Rack{
		alphabet.RackFromString("AGLSYYZ", g.Alphabet()),
		alphabet.RackFromString("ABEJNOR", g.Alphabet()),
	})
	// "jesse" plays a word after some time
	nower.Sleep(3750) // 3.75 secs
	_, err := gameplay.HandleEvent(context.Background(), gstore, ustore, nstore, lstore, tstore,
		"3xpEkpRAy3AizbVmDg3kdi", cge)

	is.NoErr(err)
	// "cesar4" waits a while before challenging this very plausible word.
	nower.Sleep(7620)
	_, err = gameplay.HandleEvent(context.Background(), gstore, ustore, nstore, lstore, tstore,
		"xjCWug7EZtDxDHX5fRZTLo", &pb.ClientGameplayEvent{
			Type:   pb.ClientGameplayEvent_CHALLENGE_PLAY,
			GameId: g.GameID(),
		})
	is.NoErr(err)

	// Kill the go-routine and let's see the events.
	cancel()
	<-donechan
	log.Info().Interface("evts", consumer.evts).Msg("evts")
	// evts: history, banjoer*, challenge, phony_tiles_returned
	is.Equal(len(consumer.evts), 4)
	// get some fields to make sure the move was played properly.
	evt := consumer.evts[1].Event.(*pb.ServerGameplayEvent)
	is.Equal(evt.Event.Score, int32(88))
	is.Equal(evt.UserId, "3xpEkpRAy3AizbVmDg3kdi")
	is.Equal(evt.TimeRemaining, int32((25*60000)+1250))
	sge := consumer.evts[2].Event.(*pb.ServerChallengeResultEvent)
	is.Equal(sge.Valid, false)
	evt = consumer.evts[3].Event.(*pb.ServerGameplayEvent)
	is.Equal(evt.Event.LostScore, int32(88))
	is.Equal(evt.Event.Type, macondopb.GameEvent_PHONY_TILES_RETURNED)
	// Time remaining here is for the person who made the challenge.
	// We don't give them their time back. They get time back after they
	// make some valid move, after challenging the play off.
	is.Equal(evt.TimeRemaining, int32((25*60000)-7620))

	ustore.(*user.DBStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
	nstore.(*mod.NotorietyStore).Disconnect()
	gstore.(*game.Cache).Disconnect()
	tstore.(*ts.Cache).Disconnect()
}

func TestDoubleChallengeGoodWord(t *testing.T) {
	is := is.New(t)
	recreateDB()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	ustore := userStore(cstr)
	lstore := listStatStore(cstr)
	nstore := notorietyStore(cstr)
	cfg, gstore := gameStore(cstr, ustore)
	tstore := tournamentStore(cfg, gstore)

	g, nower, cancel, donechan, consumer := makeGame(cfg, ustore, gstore)

	cge := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         g.GameID(),
		PositionCoords: "8D",
		Tiles:          "BANJO",
	}
	g.SetChallengeRule(macondopb.ChallengeRule_DOUBLE)
	g.SetRacksForBoth([]*alphabet.Rack{
		alphabet.RackFromString("AGLSYYZ", g.Alphabet()),
		alphabet.RackFromString("ABEJNOR", g.Alphabet()),
	})
	// "jesse" plays a word after some time
	nower.Sleep(3750) // 3.75 secs
	_, err := gameplay.HandleEvent(context.Background(), gstore, ustore, nstore, lstore, tstore,
		"3xpEkpRAy3AizbVmDg3kdi", cge)

	is.NoErr(err)
	// "cesar4" waits a while before challenging BANJO for some reason.
	nower.Sleep(7620)
	_, err = gameplay.HandleEvent(context.Background(), gstore, ustore, nstore, lstore, tstore,
		"xjCWug7EZtDxDHX5fRZTLo", &pb.ClientGameplayEvent{
			Type:   pb.ClientGameplayEvent_CHALLENGE_PLAY,
			GameId: g.GameID(),
		})
	is.NoErr(err)

	// Kill the go-routine and let's see the events.
	cancel()
	<-donechan
	log.Info().Interface("evts", consumer.evts).Msg("evts")
	// evts: history, banjo, challenge, unsuccessful_chall_turn_loss
	is.Equal(len(consumer.evts), 4)
	// get some fields to make sure the move was played properly.
	evt := consumer.evts[1].Event.(*pb.ServerGameplayEvent)
	is.Equal(evt.Event.Score, int32(34))
	is.Equal(evt.UserId, "3xpEkpRAy3AizbVmDg3kdi")
	is.Equal(evt.TimeRemaining, int32((25*60000)+1250))
	sge := consumer.evts[2].Event.(*pb.ServerChallengeResultEvent)
	is.Equal(sge.Valid, true)
	evt = consumer.evts[3].Event.(*pb.ServerGameplayEvent)
	is.Equal(evt.Event.Type, macondopb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS)
	// Time remaining here is for the person who made the challenge.
	// They lose their turn but still get 5 seconds back.
	is.Equal(evt.TimeRemaining, int32((25*60000)-2620))

	ustore.(*user.DBStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
	nstore.(*mod.NotorietyStore).Disconnect()
	gstore.(*game.Cache).Disconnect()
	tstore.(*ts.Cache).Disconnect()
}

func TestQuickdata(t *testing.T) {
	is := is.New(t)
	recreateDB()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	ustore := userStore(cstr)
	lstore := listStatStore(cstr)
	nstore := notorietyStore(cstr)
	cfg, gstore := gameStore(cstr, ustore)
	tstore := tournamentStore(cfg, gstore)

	g, nower, cancel, donechan, _ := makeGame(cfg, ustore, gstore)
	ctx := context.WithValue(context.Background(), gameplay.ConfigCtxKey("config"), &DefaultConfig)

	cge1 := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         g.GameID(),
		PositionCoords: "8D",
		Tiles:          "BANJO",
	}
	cge2 := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         g.GameID(),
		PositionCoords: "I8",
		Tiles:          "SYZYGAL",
	}
	g.SetChallengeRule(macondopb.ChallengeRule_TRIPLE)
	g.SetRacksForBoth([]*alphabet.Rack{
		alphabet.RackFromString("AGLSYYZ", g.Alphabet()),
		alphabet.RackFromString("ABEJNOR", g.Alphabet()),
	})
	// "jesse" plays a word after some time
	nower.Sleep(3750) // 3.75 secs
	_, err := gameplay.HandleEvent(ctx, gstore, ustore, nstore, lstore, tstore,
		"3xpEkpRAy3AizbVmDg3kdi", cge1)

	is.NoErr(err)

	// "cesar4" plays a word after some time
	nower.Sleep(4750) // 4.75 secs
	_, err = gameplay.HandleEvent(ctx, gstore, ustore, nstore, lstore, tstore,
		"xjCWug7EZtDxDHX5fRZTLo", cge2)

	is.NoErr(err)

	// "jesse" waits a while before challenging SYZYGAL for some reason.
	nower.Sleep(7620)
	entGame, err := gameplay.HandleEvent(ctx, gstore, ustore, nstore, lstore, tstore,
		"3xpEkpRAy3AizbVmDg3kdi", &pb.ClientGameplayEvent{
			Type:   pb.ClientGameplayEvent_CHALLENGE_PLAY,
			GameId: g.GameID(),
		})
	is.NoErr(err)

	// Check the quickdata
	is.Equal(entGame.Quickdata.PlayerInfo, []*gs.PlayerInfo{
		{UserId: "xjCWug7EZtDxDHX5fRZTLo", Nickname: "cesar4", First: false, Rating: "1500?"},
		{UserId: "3xpEkpRAy3AizbVmDg3kdi", Nickname: "jesse", First: true, Rating: "1500?"},
	})
	is.Equal(entGame.Quickdata.FinalScores[0], int32(93))
	is.Equal(entGame.Quickdata.FinalScores[1], int32(34))
	is.Equal(entGame.Quickdata.OriginalRequestId, gameReq.OriginalRequestId)

	// Kill the go-routine
	cancel()
	<-donechan
	ustore.(*user.DBStore).Disconnect()
	nstore.(*mod.NotorietyStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
	gstore.(*game.Cache).Disconnect()
	tstore.(*ts.Cache).Disconnect()
}
