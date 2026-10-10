package game

// The load-path shadow, against a real database.
//
// What is being tested is not the reconstruction -- that is covered against
// 113,010 production games in pkg/xwordbridge -- but the plumbing around it:
// that the rows a real save writes are the rows the shadow reads back, that the
// racks and rules it feeds in come from the right columns, and that the torn
// read between AppendTurns and Set is recognised as a torn read rather than
// reported as a divergence.

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// syncBuffer is a bytes.Buffer safe to write from one goroutine while another
// reads it.
//
// A plain bytes.Buffer is not, and the shadow is code whose whole purpose is to
// run in the background: anything holding a context whose logger writes here
// may log from a goroutine this test never sees. Costing a mutex to be immune
// to that is cheaper than rediscovering it in CI.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// shadowLoad runs the load-path shadow for a game and returns the messages it
// logged, in order.
func shadowLoad(t *testing.T, gstore *DBStore, gameID string) []string {
	t.Helper()
	var buf syncBuffer
	logged := zerolog.New(&buf).Level(zerolog.DebugLevel).WithContext(context.Background())

	// Load with the shadow flag OFF, so Get does not spawn a shadow of its own.
	// This helper runs the work synchronously and reports what *it* logged; a
	// background goroutine logging into the same buffer would be noise at best,
	// and it is what the buffer used to be Reset() to discard.
	//
	// A copy of DefaultConfig each time: it is a process-wide singleton
	// pointer, so setting a flag on it leaks into every test that follows.
	plain := *DefaultConfig
	entGame, err := gstore.Get(plain.WithContext(logged), gameID)
	if err != nil {
		t.Fatal(err)
	}

	cfg := *DefaultConfig
	cfg.ShadowTurnsLoad = true
	ctx := cfg.WithContext(logged)

	row, err := gstore.queries.GetGameWithTurns(ctx, models.GetGameWithTurnsParams{
		WithTurns: true, Uuid: common.ToPGTypeText(gameID),
	})
	if err != nil {
		t.Fatal(err)
	}
	work := gstore.shadowLoadWork(ctx, gameFromRow(row), row.TurnEvents, entGame)
	if work == nil {
		t.Fatal("shadow declined to run on a live game")
	}
	// On this goroutine, so everything it logs is in the buffer by the time we
	// read it.
	work(ctx)

	var msgs []string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var rec struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		msgs = append(msgs, rec.Message)
		t.Log(line)
	}
	return msgs
}

func hasPrefix(msgs []string, want string) bool {
	for _, m := range msgs {
		if strings.HasPrefix(m, want) {
			return true
		}
	}
	return false
}

// A game saved through the real write path reconstructs from its own rows.
func TestShadowLoadMatchesAfterRealMoves(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()

	const gid = "wJxURccCgSAPivUvj4QdYL"
	ctx := context.Background()

	// Two moves, written the way gameplay writes them: append the new events to
	// game_turns, then save the game.
	for _, mv := range []struct{ coords, word string }{{"8E", "AGUE"}, {"E8", "AVE"}} {
		entGame, err := gstore.Get(ctx, gid)
		is.NoErr(err)
		before := len(entGame.History().Events)
		_, err = entGame.PlayScoringMove(mv.coords, mv.word, true)
		is.NoErr(err)
		evts := entGame.History().Events[before:]
		is.NoErr(gstore.StageTurns(entGame, before, evts))
		is.NoErr(gstore.Set(ctx, entGame))
	}

	msgs := shadowLoad(t, gstore, gid)
	is.True(hasPrefix(msgs, "shadow-load-ok"))
	is.True(!hasPrefix(msgs, "shadow-load-mismatch"))
}

// The race the read path was disabled for: terminal events are committed to
// game_turns before game_end_reason is committed to games, so a load in between
// sees a finished log beside a live row. It must be named as a torn read, not
// as a reconstruction failure -- the whole point of measuring it is to find out
// how often it happens before serializing the writers.
func TestShadowLoadReportsTornRead(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()

	const gid = "wJxURccCgSAPivUvj4QdYL"
	ctx := context.Background()

	entGame, err := gstore.Get(ctx, gid)
	is.NoErr(err)
	before := len(entGame.History().Events)
	_, err = entGame.PlayScoringMove("8E", "AGUE", true)
	is.NoErr(err)
	is.NoErr(gstore.StageTurns(entGame, before, entGame.History().Events[before:]))
	is.NoErr(gstore.Set(ctx, entGame))

	// AppendTurns has committed the ending; Set has not yet been called with it.
	n := len(entGame.History().Events)
	is.NoErr(gstore.AppendTurns(ctx, gid, n, []*macondopb.GameEvent{{
		Type:        macondopb.GameEvent_END_RACK_PTS,
		PlayerIndex: 0,
		Rack:        "AEIJVVW",
		Score:       20,
		Cumulative:  30,
	}}))

	msgs := shadowLoad(t, gstore, gid)
	is.True(hasPrefix(msgs, "shadow-load-torn"))
	is.True(!hasPrefix(msgs, "shadow-load-mismatch"))
}

// A finished game is served from S3 and its turns are archived away, so there
// is nothing to compare and the shadow must not run at all.
func TestShadowLoadSkipsFinishedGames(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()

	const gid = "wJxURccCgSAPivUvj4QdYL"
	// Load and save with the flag off: a Get with it on spawns a shadow
	// goroutine that would still be running when the deferred Disconnect closes
	// the pool underneath it. Only shadowLoadWork needs the flag, and it just
	// decides whether to return work -- it starts nothing.
	plain := *DefaultConfig
	plainCtx := plain.WithContext(context.Background())

	cfg := *DefaultConfig
	cfg.ShadowTurnsLoad = true
	ctx := cfg.WithContext(context.Background())

	entGame, err := gstore.Get(plainCtx, gid)
	is.NoErr(err)
	entGame.SetGameEndReason(pb.GameEndReason_STANDARD)
	entGame.SetPlaying(macondopb.PlayState_GAME_OVER)
	is.NoErr(gstore.Set(plainCtx, entGame))

	row, err := gstore.queries.GetGameWithTurns(ctx, models.GetGameWithTurnsParams{
		WithTurns: true, Uuid: common.ToPGTypeText(gid),
	})
	is.NoErr(err)
	is.Equal(gstore.shadowLoadWork(ctx, gameFromRow(row), row.TurnEvents, entGame), nil)
}
