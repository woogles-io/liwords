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
// auto-passes (instead of forfeiting) when the player exceeds the allowed turn
// time and has no time bank.
func TestCorrespondenceTimeoutWithoutTimeBank(t *testing.T) {
	is := is.New(t)
	stores := recreateDB()
	defer stores.Disconnect()

	cfg := DefaultConfig

	// Create correspondence game with 60 second increment and no time bank
	g, nower, cancel, donechan, _ := makeCorrespondenceGame(stores, cfg, WithCorrespondenceMode(60, 0))

	// Advance time by 120 seconds (2x the allowed time)
	// Player should timeout immediately as there's no time bank
	nower.Sleep(120 * 1000) // 120 seconds in milliseconds

	// Check that TimeRanOut returns true
	playerOnTurn := g.Game.PlayerOnTurn()
	is.True(g.TimeRanOut(playerOnTurn))
	numEvtsBefore := len(g.History().Events)

	// Call TimedOut. For correspondence games this auto-passes rather than
	// forfeiting.
	ctx := ctxForTests()
	playerID := g.History().Players[playerOnTurn].UserId
	err := gameplay.TimedOut(ctx, stores, playerID, g.GameID())
	is.NoErr(err)

	// Reload game and check it did NOT end. The on-turn player was
	// auto-passed and the turn advanced to the opponent.
	reloadedGame, err := stores.GameStore.Get(ctx, g.GameID())
	is.NoErr(err)
	is.Equal(reloadedGame.GameEndReason, pb.GameEndReason_NONE)
	is.Equal(reloadedGame.Game.Playing(), macondopb.PlayState_PLAYING)
	is.Equal(reloadedGame.Game.PlayerOnTurn(), 1-playerOnTurn)
	// A pass event was recorded for the timed-out player.
	newEvts := reloadedGame.History().Events
	is.Equal(len(newEvts), numEvtsBefore+1)
	is.Equal(newEvts[len(newEvts)-1].Type, macondopb.GameEvent_PASS)

	// Clean up: cancel context and wait for goroutine to finish
	cancel()
	<-donechan
}

// TestCorrespondenceTimeoutWithExhaustedTimeBank tests that a correspondence
// game auto-passes (instead of forfeiting) when the player exceeds the allowed
// turn time and exhausts their time bank.
func TestCorrespondenceTimeoutWithExhaustedTimeBank(t *testing.T) {
	is := is.New(t)
	stores := recreateDB()
	defer stores.Disconnect()

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

	// Call TimedOut. For correspondence games this auto-passes rather than
	// forfeiting.
	ctx := ctxForTests()
	playerID := g.History().Players[playerOnTurn].UserId
	err := gameplay.TimedOut(ctx, stores, playerID, g.GameID())
	is.NoErr(err)

	// Reload game and check it did NOT end. The bank was drained to zero and
	// the on-turn player was auto-passed.
	reloadedGame, err := stores.GameStore.Get(ctx, g.GameID())
	is.NoErr(err)
	is.Equal(reloadedGame.GameEndReason, pb.GameEndReason_NONE)
	is.Equal(reloadedGame.Game.Playing(), macondopb.PlayState_PLAYING)
	is.Equal(reloadedGame.Game.PlayerOnTurn(), 1-playerOnTurn)
	// The timed-out player's bank is fully drained.
	is.Equal(reloadedGame.Timers.TimeBank[playerOnTurn], int64(0))

	// Clean up: cancel context and wait for goroutine to finish
	cancel()
	<-donechan
}

// TestCorrespondenceNoTimeoutWithTimeBank tests that a correspondence game
// does NOT timeout when the player exceeds the allowed turn time but the time bank covers it.
func TestCorrespondenceNoTimeoutWithTimeBank(t *testing.T) {
	is := is.New(t)
	stores := recreateDB()
	defer stores.Disconnect()

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
	defer stores.Disconnect()

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

// TestCorrespondenceRepeatedTimeoutsEndViaSixPasses tests that repeated
// correspondence timeouts produce auto-passes that accumulate into the
// six-consecutive-pass rule, ending the game naturally (winner by score)
// rather than by timeout/forfeit. Both players abandon the game.
func TestCorrespondenceRepeatedTimeoutsEndViaSixPasses(t *testing.T) {
	is := is.New(t)
	stores := recreateDB()
	defer stores.Disconnect()

	cfg := DefaultConfig

	// 60 second increment, no time bank.
	g, nower, cancel, donechan, _ := makeCorrespondenceGame(stores, cfg, WithCorrespondenceMode(60, 0))
	// Make reloaded games (including those loaded inside TimedOut) share the
	// same fake nower so the timeout checks are deterministic.
	stores.GameStore.SetTimerModuleCreator(func() entity.Nower {
		return nower
	})

	ctx := ctxForTests()

	// Drive timeouts until the game ends. Each timeout auto-passes the
	// on-turn player; six consecutive passes end the game. Guard with a
	// generous iteration cap to avoid an infinite loop on regression.
	var reloadedGame *entity.Game
	var err error
	ended := false
	for i := 0; i < 12; i++ {
		// Advance well past the per-turn allowance so the on-turn player
		// times out.
		nower.Sleep(120 * 1000)

		reloadedGame, err = stores.GameStore.Get(ctx, g.GameID())
		is.NoErr(err)
		if reloadedGame.Game.Playing() == macondopb.PlayState_GAME_OVER {
			ended = true
			break
		}
		playerOnTurn := reloadedGame.Game.PlayerOnTurn()
		playerID := reloadedGame.History().Players[playerOnTurn].UserId
		err = gameplay.TimedOut(ctx, stores, playerID, g.GameID())
		is.NoErr(err)
	}

	is.True(ended) // game must have ended via passes

	// Reload final state.
	reloadedGame, err = stores.GameStore.Get(ctx, g.GameID())
	is.NoErr(err)
	is.Equal(reloadedGame.Game.Playing(), macondopb.PlayState_GAME_OVER)
	// Ended naturally via the six-consecutive-pass rule, NOT a timeout
	// forfeit. Six passes produce CONSECUTIVE_ZEROES (winner by score).
	is.Equal(reloadedGame.GameEndReason, pb.GameEndReason_CONSECUTIVE_ZEROES)
	is.True(reloadedGame.GameEndReason != pb.GameEndReason_TIME)
	// A winner was decided by score (winner index is 0 or 1, not -1).
	is.True(reloadedGame.WinnerIdx == 0 || reloadedGame.WinnerIdx == 1)

	// Exactly six consecutive scoreless passes were recorded.
	evts := reloadedGame.History().Events
	passCount := 0
	for _, e := range evts {
		if e.Type == macondopb.GameEvent_PASS {
			passCount++
		}
	}
	is.Equal(passCount, 6)

	cancel()
	<-donechan
}

// TestRealtimeTimeoutStillForfeits tests that a real-time (non-correspondence)
// game still forfeits on timeout. The auto-pass change applies only to
// correspondence games.
func TestRealtimeTimeoutStillForfeits(t *testing.T) {
	is := is.New(t)
	stores := recreateDB()
	defer stores.Disconnect()

	cfg := DefaultConfig

	// No correspondence option: the default gameReq is REAL_TIME with a
	// 25-minute clock and 10 minutes of max overtime.
	g, nower, cancel, donechan, _ := makeCorrespondenceGame(stores, cfg)
	is.True(!g.IsCorrespondence())
	stores.GameStore.SetTimerModuleCreator(func() entity.Nower {
		return nower
	})

	// Advance past the 25-minute clock plus 10 minutes of overtime.
	nower.Sleep((25 + 10 + 1) * 60 * 1000)

	playerOnTurn := g.Game.PlayerOnTurn()
	is.True(g.TimeRanOut(playerOnTurn))

	ctx := ctxForTests()
	playerID := g.History().Players[playerOnTurn].UserId
	err := gameplay.TimedOut(ctx, stores, playerID, g.GameID())
	is.NoErr(err)

	// Real-time games still forfeit on timeout.
	reloadedGame, err := stores.GameStore.Get(ctx, g.GameID())
	is.NoErr(err)
	is.Equal(reloadedGame.GameEndReason, pb.GameEndReason_TIME)
	is.Equal(reloadedGame.Game.Playing(), macondopb.PlayState_GAME_OVER)
	// The player who timed out loses.
	is.Equal(reloadedGame.WinnerIdx, 1-playerOnTurn)

	cancel()
	<-donechan
}

// TestCorrespondenceAutoPassOpponentClockCorrect tests that when a
// correspondence auto-pass is processed late (e.g. up to 60s after expiry by
// the adjudicator ticker), the opponent's clock starts correctly: the opponent
// keeps their full per-turn increment and full time bank, unaffected by how
// late the timed-out player's pass was recorded.
func TestCorrespondenceAutoPassOpponentClockCorrect(t *testing.T) {
	is := is.New(t)
	stores := recreateDB()
	defer stores.Disconnect()

	cfg := DefaultConfig

	// 60 second increment, 5 minute time bank.
	incrementSeconds := int32(60)
	bankMinutes := int32(5)
	g, nower, cancel, donechan, _ := makeCorrespondenceGame(stores, cfg,
		WithCorrespondenceMode(incrementSeconds, bankMinutes))
	stores.GameStore.SetTimerModuleCreator(func() entity.Nower {
		return nower
	})

	playerOnTurn := g.Game.PlayerOnTurn()
	opponent := 1 - playerOnTurn

	// Simulate the adjudicator firing very late: the player blew their
	// per-turn allowance, their whole bank, AND another ~10 minutes of
	// ticker latency on top.
	overByMs := int64(incrementSeconds)*1000 +
		int64(bankMinutes)*60*1000 +
		int64(10)*60*1000
	nower.Sleep(overByMs)

	is.True(g.TimeRanOut(playerOnTurn))

	ctx := ctxForTests()
	playerID := g.History().Players[playerOnTurn].UserId
	err := gameplay.TimedOut(ctx, stores, playerID, g.GameID())
	is.NoErr(err)

	reloadedGame, err := stores.GameStore.Get(ctx, g.GameID())
	is.NoErr(err)
	// Game continues; opponent is on turn.
	is.Equal(reloadedGame.GameEndReason, pb.GameEndReason_NONE)
	is.Equal(reloadedGame.Game.PlayerOnTurn(), opponent)

	// The timed-out player's bank is fully drained, capped at zero (never
	// negative regardless of how late the pass was processed).
	is.Equal(reloadedGame.Timers.TimeBank[playerOnTurn], int64(0))

	// The opponent's clock is untouched by the latency: full bank remaining
	// and a full per-turn increment available from their turn start.
	is.Equal(reloadedGame.Timers.TimeBank[opponent], int64(bankMinutes)*60*1000)
	// Opponent is now on turn; their remaining (cached, since no time has
	// elapsed on their turn yet in this reloaded snapshot) is the full
	// increment.
	is.Equal(reloadedGame.CachedTimeRemaining(opponent), int(incrementSeconds)*1000)

	cancel()
	<-donechan
}
