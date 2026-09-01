package game

// Tests for the shadow/snapshot hook that hangs off DBStore.Set.
//
// This code runs on the live move path, so what matters is less that it works
// than that it cannot hurt: with the flags off it must do nothing at all, and
// with them on it must not fail a move, whatever it is handed. These tests are
// about that guarantee. Whether the snapshot is *correct* is settled elsewhere,
// against 113,010 real games (pkg/xwordbridge/corpus_test.go).

import (
	"context"
	"sync/atomic"
	"testing"

	macondoboard "github.com/domino14/macondo/board"
	macondoconfig "github.com/domino14/macondo/config"
	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"
	"github.com/rs/zerolog"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// shadowGame builds a started macondo game wrapped in an entity.Game, which is
// what Set receives.
func shadowGame(t *testing.T) *entity.Game {
	t.Helper()
	cfg := macondoconfig.DefaultConfig()
	rules, err := macondogame.NewBasicGameRules(cfg, "CSW21",
		macondoboard.CrosswordGameLayout, "english", macondogame.CrossScoreOnly, "")
	if err != nil {
		t.Skipf("macondo data not available: %v", err)
	}
	mcg, err := macondogame.NewGame(rules, []*macondopb.PlayerInfo{
		{Nickname: "p1", RealName: "Player One"},
		{Nickname: "p2", RealName: "Player Two"},
	})
	if err != nil {
		t.Fatal(err)
	}
	mcg.StartGame()
	g := entity.NewGame(mcg, &pb.GameRequest{
		Lexicon: "CSW21",
		Rules: &pb.GameRules{
			BoardLayoutName:        macondoboard.CrosswordGameLayout,
			LetterDistributionName: "english",
			VariantName:            "classic",
		},
		InitialTimeSeconds: 900,
	})
	return g
}

// loadedConfig is a Config with its macondo settings populated, which is the
// only way to get one: the macondo config is unexported and set by Load.
func loadedConfig(t *testing.T) *config.Config {
	t.Helper()
	c := &config.Config{}
	if err := c.Load(nil); err != nil {
		t.Skipf("cannot load config: %v", err)
	}
	return c
}

// ctxWith returns a context carrying a config with the given flags.
func ctxWith(shadow, write bool) context.Context {
	c := &config.Config{ShadowXwordState: shadow, WriteXwordState: write}
	return c.WithContext(context.Background())
}

func counterTotal() uint64 {
	var n uint64
	for _, c := range []*atomic.Uint64{
		&panicCount, &snapshotSkipCount, &paramsSkipCount, &upsertErrCount,
		&deleteErrCount, &invalidStateCount, &notConservedCount,
		&decodeFailCount, &roundTripCount,
	} {
		n += c.Load()
	}
	return n
}

// With both flags off this must not touch the database, read the game, or log.
// It is the only thing standing between a bug in here and every move on the
// server, so it is worth asserting rather than assuming.
func TestShadowDoesNothingWhenDisabled(t *testing.T) {
	is := is.New(t)
	g := shadowGame(t)

	// A DBStore with no queries and no config proves the point: reading either
	// would panic, and the recover would record it.
	before := counterTotal()
	s := &DBStore{}
	s.maybeWriteOngoingGame(ctxWith(false, false), g)
	s.maybeWriteOngoingGame(context.Background(), g) // no config at all
	is.Equal(counterTotal(), before)
}

// Shadow mode reads the game and validates the snapshot. It writes nothing, so
// it needs no ongoing_games table and can be deployed on its own.
func TestShadowModeNeedsNoDatabase(t *testing.T) {
	is := is.New(t)
	g := shadowGame(t)

	// Config but no queries and no pool: shadow mode needs the config to
	// resolve a letter distribution, and must not need anything else. A query
	// against the nil Queries would panic, and the recover would count it.
	before := counterTotal()
	s := &DBStore{cfg: loadedConfig(t)}
	s.maybeWriteOngoingGame(ctxWith(true, false), g)

	// A freshly started game converts and validates cleanly, so nothing is
	// counted -- in particular the panic counter, which is what would fire if
	// shadow mode reached for the database.
	is.Equal(counterTotal(), before)
}

// The guarantee that matters most: whatever is wrong with the game, the move
// still succeeds.
func TestShadowNeverPanics(t *testing.T) {
	is := is.New(t)

	for _, tc := range []struct {
		name string
		game func(*testing.T) *entity.Game
	}{
		{"a game mid-transposition", func(t *testing.T) *entity.Game {
			g := shadowGame(t)
			// macondo flips the board in place while scoring vertical plays.
			// Reading one here is refused, not mistranslated.
			g.Game.Board().Transpose()
			return g
		}},
		{"a game with no history", func(t *testing.T) *entity.Game {
			g := shadowGame(t)
			g.Game.SetHistory(nil)
			return g
		}},
		{"a finished game", func(t *testing.T) *entity.Game {
			g := shadowGame(t)
			g.Game.SetPlaying(macondopb.PlayState_GAME_OVER)
			return g
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := tc.game(t)
			s := &DBStore{}
			// Must return normally. A panic escaping here would take down a
			// real move.
			s.maybeWriteOngoingGame(ctxWith(true, false), g)
			is.True(true)
		})
	}
}

// A systemic failure must not emit one log line per move for every game on the
// server. The count rides on each line so the scale stays visible.
func TestLogBoundedThinsOutAfterABurst(t *testing.T) {
	is := is.New(t)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	var counter atomic.Uint64
	emitted := 0
	logger := zerolog.New(zerolog.NewTestWriter(t)).Hook(
		zerolog.HookFunc(func(e *zerolog.Event, l zerolog.Level, msg string) {
			emitted++
		}))

	for range logBurst + 3*logEvery {
		logBounded(logger.Error(), &counter, "test-event")
	}

	// Everything in the burst, then one per interval.
	is.Equal(counter.Load(), uint64(logBurst+3*logEvery))
	is.Equal(emitted, logBurst+3)
}
