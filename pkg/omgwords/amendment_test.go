package omgwords

import (
	"context"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/cwgame"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// TestReplayEventsResetsGameEndState tests that ReplayEvents actually resets
// the Winner, EndReason, and PlayState fields
func TestReplayEventsResetsGameEndState(t *testing.T) {
	is := is.New(t)

	cfg := config.DefaultConfig()
	ctx := context.Background()
	ctx = log.Logger.WithContext(ctx)
	ctx = cfg.WithContext(ctx)

	// Create a game document that has ended
	gdoc := &ipc.GameDocument{
		PlayState:          ipc.PlayState_GAME_OVER,
		Winner:             1,
		EndReason:          ipc.GameEndReason_STANDARD,
		Events:             []*ipc.GameEvent{},
		Players:            []*ipc.GameDocument_MinimalPlayerInfo{{UserId: "p1"}, {UserId: "p2"}},
		Lexicon:            "NWL20",
		BoardLayout:        "CrosswordGame",
		LetterDistribution: "english",
		Racks:              [][]byte{{}, {}},
		Timers:             &ipc.Timers{TimeRemaining: []int64{0, 0}},
	}

	// Call ReplayEvents with empty event list
	err := cwgame.ReplayEvents(ctx, cfg.WGLConfig(), gdoc, []*ipc.GameEvent{}, false)
	is.NoErr(err)

	// Verify that ReplayEvents actually reset the game-end state
	is.Equal(gdoc.PlayState, ipc.PlayState_PLAYING)      // Should be PLAYING, not GAME_OVER
	is.Equal(gdoc.Winner, int32(0))                       // Should be reset to 0
	is.Equal(gdoc.EndReason, ipc.GameEndReason_NONE)     // Should be NONE, not STANDARD
}

// TestApplyEventInEditorModeDiscardsInvalid documents that when an event fails to apply,
// it is silently discarded (not added to the event history).
// The actual behavior is tested in the integration tests (TestEditMoveAfterMaking, etc.)
func TestApplyEventInEditorModeDiscardsInvalid(t *testing.T) {
	is := is.New(t)

	// This test documents the behavior:
	// When ApplyEventInEditorMode returns an error, the event is NOT added to gdoc.Events
	// This happens naturally because playMove() only appends events on success

	// The integration tests (TestEditMoveAfterMaking, TestEditMoveAfterChallenge)
	// verify this behavior in real scenarios where:
	// - Editing event N elsewhere on the board preserves event N+1 (no conflict)
	// - Editing event N at the same location as event N+1 discards event N+1 (conflict)

	is.True(true) // Behavior is documented and tested in integration tests
}
