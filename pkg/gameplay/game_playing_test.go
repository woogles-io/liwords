package gameplay_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var pkg = "gameplay_test"

func ctxForTests() context.Context {
	ctx := context.Background()
	ctx = log.Logger.WithContext(ctx)
	ctx = DefaultConfig.WithContext(ctx)
	return ctx
}

// func gameStore(userStore pkguser.Store) (*config.Config, *game.Cache) {
// 	cfg := DefaultConfig
// 	cfg.DBConnDSN = common.TestingPostgresConnDSN(pkg)

// 	tmp, err := game.NewDBStore(&cfg, userStore)
// 	if err != nil {
// 		log.Fatal().Err(err).Msg("error")
// 	}
// 	gameStore := game.NewCache(tmp)
// 	return &cfg, gameStore
// }

// func tournamentStore(cfg *config.Config, gs *game.Cache) tournament.TournamentStore {
// 	tmp, err := ts.NewDBStore(cfg, gs)
// 	if err != nil {
// 		log.Fatal().Err(err).Msg("error")
// 	}
// 	tournamentStore := ts.NewCache(tmp)
// 	return tournamentStore
// }

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

type TestGameOption func(*pb.GameRequest)

func WithIncrementSeconds(seconds int32) TestGameOption {
	return func(gr *pb.GameRequest) {
		gr.IncrementSeconds = seconds
	}
}

func WithMaxOvertimeMinutes(minutes int32) TestGameOption {
	return func(gr *pb.GameRequest) {
		gr.MaxOvertimeMinutes = minutes
	}
}

func WithInitialTimeSeconds(seconds int32) TestGameOption {
	return func(gr *pb.GameRequest) {
		gr.InitialTimeSeconds = seconds
	}
}

func makeGame(cfg *config.Config, stores *stores.Stores, opts ...TestGameOption) (
	*entity.Game, *entity.FakeNower, context.CancelFunc, chan bool, *evtConsumer) {

	ctx := ctxForTests()
	cesar, _ := stores.UserStore.Get(ctx, "cesar4")
	jesse, _ := stores.UserStore.Get(ctx, "jesse")
	// see the gameReq in game_stats_test.go in this package
	gr := proto.Clone(gameReq).(*pb.GameRequest)

	gr.IncrementSeconds = 5
	gr.MaxOvertimeMinutes = 0

	// Apply any custom options
	for _, opt := range opts {
		opt(gr)
	}

	g, _ := gameplay.InstantiateNewGame(ctx, stores.GameStore, cfg, [2]*entity.User{jesse, cesar},
		gr, nil)

	ch := make(chan *entity.EventWrapper)
	donechan := make(chan bool)
	consumer := &evtConsumer{}
	stores.GameStore.SetGameEventChan(ch)

	cctx, cancel := context.WithCancel(ctx)
	go consumer.consumeEventChan(cctx, ch, donechan)

	nower := entity.NewFakeNower(1234)
	g.SetTimerModule(nower)

	gameplay.StartGame(ctx, stores, ch, g)

	return g, nower, cancel, donechan, consumer
}

func TestInitializeGame(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame()

	is.Equal(gs.g.PlayerOnTurn(), 0)
	gs.cancel()
	<-gs.donechan
	// It should just be a single GameHistory event.
	is.Equal(len(gs.consumer.evts), 1)
	teardownGame(gs)
}

func TestWrongTurn(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame()

	is.Equal(gs.g.PlayerOnTurn(), 0)

	cge := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         gs.g.GameID(),
		PositionCoords: "8D",
		MachineLetters: []byte{2, 1, 14, 10, 15},
	}
	ctx := ctxForTests()
	// User ID below is "cesar4" who's not on turn.
	_, err := gameplay.HandleEvent(ctx, gs.stores, "xjCWug7EZtDxDHX5fRZTLo", cge)

	is.Equal(err.Error(), "player not on turn")

	gs.cancel()
	<-gs.donechan
	// It should just be a single GameHistory event.
	is.Equal(len(gs.consumer.evts), 1)
	teardownGame(gs)
}

func Test5ptBadWord(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame()

	ctx := ctxForTests()

	cge := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         gs.g.GameID(),
		PositionCoords: "8D",
		MachineLetters: []byte{2, 1, 14, 10, 15},
	}
	gs.g.SetRacksForBoth([]*tilemapping.Rack{
		tilemapping.RackFromString("ABEJNOR", gs.g.Alphabet()),
		tilemapping.RackFromString("AGLSYYZ", gs.g.Alphabet()),
	})
	// "jesse" plays a word after some time
	gs.nower.Sleep(3750) // 3.75 secs
	_, err := gameplay.HandleEvent(ctx, gs.stores, "3xpEkpRAy3AizbVmDg3kdi", cge)

	is.NoErr(err)

	// Kill the go-routine and let's see the events.
	gs.cancel()
	<-gs.donechan

	is.Equal(len(gs.consumer.evts), 2)
	// get some fields to make sure the move was played properly.
	evt := gs.consumer.evts[1].Event.(*pb.ServerGameplayEvent)
	is.Equal(evt.Event.Score, int32(34))
	is.Equal(evt.UserId, "3xpEkpRAy3AizbVmDg3kdi")
	// starting time is 25*60 secs, plus a 5-second increment, and they spent 3750 ms on the move.
	// TimeRemaining is in ms.
	is.Equal(evt.TimeRemaining, int32((25*60000)+1250))

	teardownGame(gs)
}

func TestDoubleChallengeBadWord(t *testing.T) {
	is := is.New(t)
	gs := setupNewGame()
	ctx := ctxForTests()

	cge := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         gs.g.GameID(),
		PositionCoords: "8D",
		MachineLetters: []byte{2, 1, 14, 10, 15, 5, 18}, // BANJOER
	}
	gs.g.SetChallengeRule(macondopb.ChallengeRule_DOUBLE)
	gs.g.SetRacksForBoth([]*tilemapping.Rack{
		tilemapping.RackFromString("ABEJNOR", gs.g.Alphabet()),
		tilemapping.RackFromString("AGLSYYZ", gs.g.Alphabet()),
	})
	// "jesse" plays a word after some time
	gs.nower.Sleep(3750) // 3.75 secs
	_, err := gameplay.HandleEvent(ctx, gs.stores, "3xpEkpRAy3AizbVmDg3kdi", cge)

	is.NoErr(err)
	// "cesar4" waits a while before challenging this very plausible word.
	gs.nower.Sleep(7620)
	_, err = gameplay.HandleEvent(ctx, gs.stores,
		"xjCWug7EZtDxDHX5fRZTLo", &pb.ClientGameplayEvent{
			Type:   pb.ClientGameplayEvent_CHALLENGE_PLAY,
			GameId: gs.g.GameID(),
		})
	is.NoErr(err)

	// Kill the go-routine and let's see the events.
	gs.cancel()
	<-gs.donechan
	log.Info().Interface("evts", gs.consumer.evts).Msg("evts")
	// evts: history, banjoer*, challenge, phony_tiles_returned
	is.Equal(len(gs.consumer.evts), 4)
	// get some fields to make sure the move was played properly.
	evt := gs.consumer.evts[1].Event.(*pb.ServerGameplayEvent)
	is.Equal(evt.Event.Score, int32(88))
	is.Equal(evt.UserId, "3xpEkpRAy3AizbVmDg3kdi")
	is.Equal(evt.TimeRemaining, int32((25*60000)+1250))
	sge := gs.consumer.evts[2].Event.(*pb.ServerChallengeResultEvent)
	is.Equal(sge.Valid, false)
	evt = gs.consumer.evts[3].Event.(*pb.ServerGameplayEvent)
	is.Equal(evt.Event.LostScore, int32(88))
	is.Equal(evt.Event.Type, macondopb.GameEvent_PHONY_TILES_RETURNED)
	// Time remaining here is for the person who made the challenge.
	// We don't give them their time back. They get time back after they
	// make some valid move, after challenging the play off.
	is.Equal(evt.TimeRemaining, int32((25*60000)-7620))

	teardownGame(gs)
}

func loadGameForTest(testid string, t *testing.T) *gamesetup {
	pool, stores, _ := recreateDB()
	ctx := context.Background()

	testdataDir := filepath.Join(".", "testdata", testid)
	historyPath := filepath.Join(testdataDir, "history.json")
	historyBytes, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("failed to read history.json: %v", err)
	}
	hist := &macondopb.GameHistory{}
	if err := protojson.Unmarshal(historyBytes, hist); err != nil {
		t.Fatalf("failed to unmarshal history.json: %v", err)
	}
	bpb, err := proto.Marshal(hist)
	if err != nil {
		t.Fatalf("failed to marshal history.json: %v", err)
	}

	_, err = pool.Exec(ctx, `INSERT INTO games(uuid, timers, player0_id, player1_id, started, game_end_reason, type, game_request, history, quickdata)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		"98uhDKtp", `{
        "lu": 1707439121587,
        "mo": 1,
        "tr": [
            200000,
            3599676
        ],
        "ts": 1707435505427
    }`, 1, 2, true, 0, 0, `{
        "rules": {
            "boardLayoutName": "CrosswordGame",
            "letterDistributionName": "english"
        },
        "lexicon": "CSW21",
        "requestId": "KKaVaNtAc6byE3tJXzLrP8",
        "playerVsBot": true,
        "challengeRule": "FIVE_POINT",
        "originalRequestId": "KKaVaNtAc6byE3tJXzLrP8",
        "initialTimeSeconds": 3600,
        "maxOvertimeMinutes": 1
    }`,
		bpb, `{}`)
	if err != nil {
		t.Fatalf("failed to insert game: %v", err)
	}

	entGame, err := stores.GameStore.Get(ctx, "98uhDKtp")
	if err != nil {
		t.Fatalf("failed to get game: %v", err)
	}

	ch := make(chan *entity.EventWrapper)
	donechan := make(chan bool)
	consumer := &evtConsumer{}
	stores.GameStore.SetGameEventChan(ch)

	cctx, cancel := context.WithCancel(ctx)
	go consumer.consumeEventChan(cctx, ch, donechan)

	nower := entity.NewFakeNower(1707439121587)
	entGame.SetTimerModule(nower)
	entGame.RegisterChangeHook(ch)

	// set the timer module factory for the game store
	stores.GameStore.SetTimerModuleCreator(func() entity.Nower {
		return nower
	})

	return &gamesetup{
		entGame, nower, cancel, donechan, consumer, stores,
	}
}

func TestEndOfGameChallengeBadWord(t *testing.T) {
	is := is.New(t)
	gs := loadGameForTest("game3", t)
	ctx := ctxForTests()

	fmt.Println(gs.g.Game.ToDisplayText())

	// "jesse" plays a word after some time
	gs.nower.Sleep(3750) // 3.75 secs

	cge := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         gs.g.GameID(),
		PositionCoords: "1A",
		MachineLetters: []byte{18, 9, 16, 21, 18, 9 | 0x80, 1, 0}, // RIPURiA(N)*
	}

	_, err := gameplay.HandleEvent(ctx, gs.stores, "xjCWug7EZtDxDHX5fRZTLo", cge)
	is.NoErr(err)
	is.Equal(gs.g.Game.Playing(), macondopb.PlayState_WAITING_FOR_FINAL_PASS)
	is.Equal(gs.g.Game.History().PlayState, macondopb.PlayState_WAITING_FOR_FINAL_PASS)

	fmt.Println(gs.g.Game.ToDisplayText())

	//  "mina" HastyBot waits a while before challenging this very plausible word.
	gs.nower.Sleep(7620)
	// unload the game to simulate a restart
	gs.stores.GameStore.Unload(ctx, gs.g.GameID())

	entGame, err := gameplay.HandleEvent(ctx, gs.stores,
		"qUQkST8CendYA3baHNoPjk", &pb.ClientGameplayEvent{
			Type:   pb.ClientGameplayEvent_CHALLENGE_PLAY,
			GameId: gs.g.GameID(),
		})
	is.NoErr(err)
	// Update gs.g with the freshly loaded game
	gs.g = entGame
	is.Equal(gs.g.Game.Playing(), macondopb.PlayState_PLAYING)
	is.Equal(gs.g.Game.History().PlayState, macondopb.PlayState_PLAYING)

	gs.nower.Sleep(1000)
	// unload the game to simulate a restart
	gs.stores.GameStore.Unload(ctx, gs.g.GameID())

	cge2 := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         gs.g.GameID(),
		PositionCoords: "N6",
		MachineLetters: []byte{4, 5, 2, 1, 20, 5}, // DEBATE
	}

	entGame, err = gameplay.HandleEvent(ctx, gs.stores, "qUQkST8CendYA3baHNoPjk", cge2)
	is.NoErr(err)
	gs.g = entGame
	fmt.Println(gs.g.Game.ToDisplayText())
	is.Equal(gs.g.Game.Playing(), macondopb.PlayState_PLAYING)
	is.Equal(gs.g.Game.History().PlayState, macondopb.PlayState_PLAYING)
	// unload the game to simulate a restart
	gs.stores.GameStore.Unload(ctx, gs.g.GameID())

	cge3 := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         gs.g.GameID(),
		PositionCoords: "9M",
		MachineLetters: []byte{16, 0}, // P(A)
	}

	entGame, err = gameplay.HandleEvent(ctx, gs.stores, "xjCWug7EZtDxDHX5fRZTLo", cge3)
	is.NoErr(err)
	gs.g = entGame
	fmt.Println(gs.g.Game.ToDisplayText())
	is.Equal(gs.g.Game.Playing(), macondopb.PlayState_PLAYING)
	is.Equal(gs.g.Game.History().PlayState, macondopb.PlayState_PLAYING)

	// unload the game to simulate a restart
	gs.stores.GameStore.Unload(ctx, gs.g.GameID())

	cge4 := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         gs.g.GameID(),
		PositionCoords: "8N",
		MachineLetters: []byte{0, 9}, // (B)I
	}

	entGame, err = gameplay.HandleEvent(ctx, gs.stores, "qUQkST8CendYA3baHNoPjk", cge4)
	is.NoErr(err)
	gs.g = entGame
	fmt.Println(gs.g.Game.ToDisplayText())
	is.Equal(gs.g.Game.Playing(), macondopb.PlayState_WAITING_FOR_FINAL_PASS)
	is.Equal(gs.g.Game.History().PlayState, macondopb.PlayState_WAITING_FOR_FINAL_PASS)

	entGame, err = gameplay.HandleEvent(ctx, gs.stores, "xjCWug7EZtDxDHX5fRZTLo",
		&pb.ClientGameplayEvent{
			Type:   pb.ClientGameplayEvent_PASS,
			GameId: gs.g.GameID(),
		})
	is.NoErr(err)
	gs.g = entGame
	fmt.Println(gs.g.Game.ToDisplayText())
	is.Equal(gs.g.Game.Playing(), macondopb.PlayState_GAME_OVER)
	is.Equal(gs.g.GameEndReason, pb.GameEndReason_STANDARD)

	// Kill the go-routine and let's see the events.
	gs.cancel()
	<-gs.donechan
	log.Info().Interface("evts", gs.consumer.evts).Msg("evts")

	for idx, evt := range gs.consumer.evts {
		log.Info().Int("idx", idx).Interface("evt", evt).Msg("evt")
	}

	teardownGame(gs)
}

func TestDoubleChallengeGoodWord(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()

	gs := setupNewGame()

	cge := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         gs.g.GameID(),
		PositionCoords: "8D",
		MachineLetters: []byte{2, 1, 14, 10, 15},
	}
	gs.g.SetChallengeRule(macondopb.ChallengeRule_DOUBLE)
	gs.g.SetRacksForBoth([]*tilemapping.Rack{
		tilemapping.RackFromString("ABEJNOR", gs.g.Alphabet()),
		tilemapping.RackFromString("AGLSYYZ", gs.g.Alphabet()),
	})
	// "jesse" plays a word after some time
	gs.nower.Sleep(3750) // 3.75 secs
	_, err := gameplay.HandleEvent(ctx, gs.stores, "3xpEkpRAy3AizbVmDg3kdi", cge)

	is.NoErr(err)
	// "cesar4" waits a while before challenging BANJO for some reason.
	gs.nower.Sleep(7620)
	_, err = gameplay.HandleEvent(ctx, gs.stores,
		"xjCWug7EZtDxDHX5fRZTLo", &pb.ClientGameplayEvent{
			Type:   pb.ClientGameplayEvent_CHALLENGE_PLAY,
			GameId: gs.g.GameID(),
		})
	is.NoErr(err)

	// Kill the go-routine and let's see the events.
	gs.cancel()
	<-gs.donechan
	log.Info().Interface("evts", gs.consumer.evts).Msg("evts")
	// evts: history, banjo, challenge, unsuccessful_chall_turn_loss
	is.Equal(len(gs.consumer.evts), 4)
	// get some fields to make sure the move was played properly.
	evt := gs.consumer.evts[1].Event.(*pb.ServerGameplayEvent)
	is.Equal(evt.Event.Score, int32(34))
	is.Equal(evt.UserId, "3xpEkpRAy3AizbVmDg3kdi")
	is.Equal(evt.TimeRemaining, int32((25*60000)+1250))
	sge := gs.consumer.evts[2].Event.(*pb.ServerChallengeResultEvent)
	is.Equal(sge.Valid, true)
	evt = gs.consumer.evts[3].Event.(*pb.ServerGameplayEvent)
	is.Equal(evt.Event.Type, macondopb.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS)
	// Time remaining here is for the person who made the challenge.
	// They lose their turn but still get 5 seconds back.
	is.Equal(evt.TimeRemaining, int32((25*60000)-2620))

	teardownGame(gs)
}

func TestQuickdata(t *testing.T) {
	is := is.New(t)
	ctx := ctxForTests()

	gs := setupNewGame()

	cge1 := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         gs.g.GameID(),
		PositionCoords: "8D",
		MachineLetters: []byte{2, 1, 14, 10, 15}, // BANJO
	}
	cge2 := &pb.ClientGameplayEvent{
		Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
		GameId:         gs.g.GameID(),
		PositionCoords: "I8",
		MachineLetters: []byte{19, 25, 26, 25, 7, 1, 12}, // SYZYGAL
	}
	gs.g.SetChallengeRule(macondopb.ChallengeRule_TRIPLE)
	gs.g.SetRacksForBoth([]*tilemapping.Rack{
		tilemapping.RackFromString("ABEJNOR", gs.g.Alphabet()),
		tilemapping.RackFromString("AGLSYYZ", gs.g.Alphabet()),
	})
	// "jesse" plays a word after some time
	gs.nower.Sleep(3750) // 3.75 secs
	_, err := gameplay.HandleEvent(ctx, gs.stores, "3xpEkpRAy3AizbVmDg3kdi", cge1)

	is.NoErr(err)

	// "cesar4" plays a word after some time
	gs.nower.Sleep(4750) // 4.75 secs
	_, err = gameplay.HandleEvent(ctx, gs.stores, "xjCWug7EZtDxDHX5fRZTLo", cge2)

	is.NoErr(err)

	// "jesse" waits a while before challenging SYZYGAL for some reason.
	gs.nower.Sleep(7620)
	entGame, err := gameplay.HandleEvent(ctx, gs.stores,
		"3xpEkpRAy3AizbVmDg3kdi", &pb.ClientGameplayEvent{
			Type:   pb.ClientGameplayEvent_CHALLENGE_PLAY,
			GameId: gs.g.GameID(),
		})
	is.NoErr(err)

	// Check the quickdata
	is.Equal(entGame.Quickdata.PlayerInfo, []*pb.PlayerInfo{
		{UserId: "3xpEkpRAy3AizbVmDg3kdi", Nickname: "jesse", First: true, Rating: "1500?"},
		{UserId: "xjCWug7EZtDxDHX5fRZTLo", Nickname: "cesar4", First: false, Rating: "1500?"},
	})
	is.Equal(entGame.Quickdata.FinalScores[0], int32(34))
	is.Equal(entGame.Quickdata.FinalScores[1], int32(93))
	is.Equal(entGame.Quickdata.OriginalRequestId, gameReq.OriginalRequestId)

	// Kill the go-routine
	gs.cancel()
	<-gs.donechan

	teardownGame(gs)
}
