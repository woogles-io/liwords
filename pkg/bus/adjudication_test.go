package bus_test

import (
	"context"
	"os"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var pkg = "bus_test"

var gameReq = &pb.GameRequest{
	Lexicon: "CSW21",
	Rules: &pb.GameRules{
		BoardLayoutName:        entity.CrosswordGame,
		LetterDistributionName: "English",
		VariantName:            "classic",
	},
	InitialTimeSeconds: 25 * 60,
	IncrementSeconds:   0,
	ChallengeRule:      macondopb.ChallengeRule_FIVE_POINT,
	GameMode:           pb.GameMode_REAL_TIME,
	RatingMode:         pb.RatingMode_RATED,
	RequestId:          "yeet",
	OriginalRequestId:  "originalyeet",
	MaxOvertimeMinutes: 10,
}

var DefaultConfig = config.DefaultConfig()

func ctxForTests() context.Context {
	ctx := context.Background()
	ctx = log.Logger.WithContext(ctx)
	ctx = DefaultConfig.WithContext(ctx)
	return ctx
}

func recreateDB() *stores.Stores {
	err := common.RecreateTestDB(pkg)
	if err != nil {
		panic(err)
	}
	pool, err := common.OpenTestingDB(pkg)
	if err != nil {
		panic(err)
	}

	cfg := DefaultConfig
	cfg.DBConnDSN = common.TestingPostgresConnDSN(pkg)
	stores, err := stores.NewInitializedStores(pool, nil, cfg)
	if err != nil {
		panic(err)
	}

	// Insert test users
	for _, u := range []*entity.User{
		{Username: "cesar4", Email: "cesar4@woogles.io", UUID: "xjCWug7EZtDxDHX5fRZTLo"},
		{Username: "jesse", Email: "jesse@woogles.io", UUID: "3xpEkpRAy3AizbVmDg3kdi"},
	} {
		err = stores.UserStore.New(context.Background(), u)
		if err != nil {
			log.Fatal().Err(err).Msg("error")
		}
	}
	return stores
}

func teardownStores(stores *stores.Stores) {
	stores.UserStore.Disconnect()
	stores.GameStore.Disconnect()
	stores.TournamentStore.Disconnect()
	stores.ListStatStore.Disconnect()
	stores.NotorietyStore.Disconnect()
}

type TestGameOption func(*pb.GameRequest)

func WithCorrespondenceMode(incrementSeconds int32, timeBankMinutes int32) TestGameOption {
	return func(gr *pb.GameRequest) {
		gr.GameMode = pb.GameMode_CORRESPONDENCE
		gr.IncrementSeconds = incrementSeconds
		gr.TimeBankMinutes = timeBankMinutes
		gr.InitialTimeSeconds = incrementSeconds
		gr.MaxOvertimeMinutes = 0 // Correspondence games don't use max overtime
	}
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

func makeCorrespondenceGame(stores *stores.Stores, cfg *config.Config, opts ...TestGameOption) (*entity.Game, *entity.FakeNower, context.CancelFunc, chan bool, *evtConsumer) {
	ctx := ctxForTests()
	cesar, _ := stores.UserStore.Get(ctx, "cesar4")
	jesse, _ := stores.UserStore.Get(ctx, "jesse")

	gr := proto.Clone(gameReq).(*pb.GameRequest)

	// Apply options
	for _, opt := range opts {
		opt(gr)
	}

	g, _ := gameplay.InstantiateNewGame(ctx, stores.GameStore, cfg, [2]*entity.User{jesse, cesar}, gr, nil)

	// Set up event channel and consumer
	ch := make(chan *entity.EventWrapper)
	donechan := make(chan bool)
	consumer := &evtConsumer{}
	stores.GameStore.SetGameEventChan(ch)

	cctx, cancel := context.WithCancel(ctx)
	go consumer.consumeEventChan(cctx, ch, donechan)

	// Use FakeNower for time control
	nower := entity.NewFakeNower(1000)
	g.SetTimerModule(nower)

	// Start the game
	gameplay.StartGame(ctx, stores, ch, g)

	return g, nower, cancel, donechan, consumer
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

// TestCorrespondenceTimeoutWithoutTimeBank tests that a correspondence game
// times out when the player exceeds the allowed turn time and has no time bank.
func TestCorrespondenceTimeoutWithoutTimeBank(t *testing.T) {
	is := is.New(t)
	stores := recreateDB()
	defer teardownStores(stores)

	cfg := DefaultConfig

	// Create correspondence game with 60 second increment and no time bank
	g, nower, cancel, donechan, _ := makeCorrespondenceGame(stores, cfg, WithCorrespondenceMode(60, 0))

	// Advance time by 120 seconds (2x the allowed time)
	// Player should timeout immediately as there's no time bank
	nower.Sleep(120 * 1000) // 120 seconds in milliseconds

	// Check that TimeRanOut returns true
	playerOnTurn := g.Game.PlayerOnTurn()
	is.True(g.TimeRanOut(playerOnTurn))

	// Call TimedOut to end the game
	ctx := ctxForTests()
	playerID := g.History().Players[playerOnTurn].UserId
	err := gameplay.TimedOut(ctx, stores, playerID, g.GameID())
	is.NoErr(err)

	// Reload game and check it ended due to timeout
	reloadedGame, err := stores.GameStore.Get(ctx, g.GameID())
	is.NoErr(err)
	is.Equal(reloadedGame.GameEndReason, pb.GameEndReason_TIME)

	// Clean up: cancel context and wait for goroutine to finish
	cancel()
	<-donechan
}

// TestCorrespondenceTimeoutWithExhaustedTimeBank tests that a correspondence game
// times out when the player exceeds the allowed turn time and exhausts their time bank.
func TestCorrespondenceTimeoutWithExhaustedTimeBank(t *testing.T) {
	is := is.New(t)
	stores := recreateDB()
	defer teardownStores(stores)

	cfg := DefaultConfig

	// Create correspondence game with 60 second increment and 2 minutes time bank
	g, nower, cancel, donechan, _ := makeCorrespondenceGame(stores, cfg, WithCorrespondenceMode(60, 2))

	// Time bank should be 2 * 60 * 1000 = 120000 ms
	// If player takes 240 seconds (4 minutes), deficit = 240 - 60 = 180 seconds = 180000 ms
	// Time bank can only cover 120000 ms, so player should timeout
	nower.Sleep(240 * 1000) // 240 seconds = 4 minutes

	// Check that TimeRanOut returns true
	playerOnTurn := g.Game.PlayerOnTurn()
	is.True(g.TimeRanOut(playerOnTurn))

	// Call TimedOut to end the game
	ctx := ctxForTests()
	playerID := g.History().Players[playerOnTurn].UserId
	err := gameplay.TimedOut(ctx, stores, playerID, g.GameID())
	is.NoErr(err)

	// Reload game and check it ended due to timeout
	reloadedGame, err := stores.GameStore.Get(ctx, g.GameID())
	is.NoErr(err)
	is.Equal(reloadedGame.GameEndReason, pb.GameEndReason_TIME)

	// Clean up: cancel context and wait for goroutine to finish
	cancel()
	<-donechan
}

// TestCorrespondenceNoTimeoutWithTimeBank tests that a correspondence game
// does NOT timeout when the player exceeds the allowed turn time but the time bank covers it.
func TestCorrespondenceNoTimeoutWithTimeBank(t *testing.T) {
	is := is.New(t)
	stores := recreateDB()
	defer teardownStores(stores)

	cfg := DefaultConfig

	// Create correspondence game with 60 second increment and 5 minutes time bank
	g, nower, cancel, donechan, _ := makeCorrespondenceGame(stores, cfg, WithCorrespondenceMode(60, 5))

	// Time bank should be 5 * 60 * 1000 = 300000 ms
	// If player takes 120 seconds (2 minutes), deficit = 120 - 60 = 60 seconds = 60000 ms
	// Time bank can cover this, so player should NOT timeout
	nower.Sleep(120 * 1000) // 120 seconds = 2 minutes

	// Check that TimeRanOut returns FALSE (time bank covers it)
	playerOnTurn := g.Game.PlayerOnTurn()
	is.True(!g.TimeRanOut(playerOnTurn)) // Should NOT timeout

	// Verify game is still ongoing
	is.Equal(g.GameEndReason, pb.GameEndReason_NONE)

	// Clean up: cancel context and wait for goroutine to finish
	cancel()
	<-donechan
}

// TestCorrespondenceWithinAllowedTime tests that a correspondence game
// does NOT timeout when the player is within the allowed turn time.
func TestCorrespondenceWithinAllowedTime(t *testing.T) {
	is := is.New(t)
	stores := recreateDB()
	defer teardownStores(stores)

	cfg := DefaultConfig

	// Create correspondence game with 60 second increment and no time bank
	g, nower, cancel, donechan, _ := makeCorrespondenceGame(stores, cfg, WithCorrespondenceMode(60, 0))

	// Advance time by 30 seconds (within allowed 60 seconds)
	// Player should NOT timeout
	nower.Sleep(30 * 1000) // 30 seconds

	// Check that TimeRanOut returns FALSE
	playerOnTurn := g.Game.PlayerOnTurn()
	is.True(!g.TimeRanOut(playerOnTurn)) // Should NOT timeout

	// Verify game is still ongoing
	is.Equal(g.GameEndReason, pb.GameEndReason_NONE)

	// Clean up: cancel context and wait for goroutine to finish
	cancel()
	<-donechan
}
