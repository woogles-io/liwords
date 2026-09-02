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
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// shadowLoad runs the load-path shadow for a game and returns the messages it
// logged, in order.
func shadowLoad(t *testing.T, gstore *DBStore, gameID string) []string {
	t.Helper()
	var buf bytes.Buffer
	ctx := zerolog.New(&buf).Level(zerolog.DebugLevel).WithContext(context.Background())
	cfg := DefaultConfig
	cfg.ShadowTurnsLoad = true
	ctx = cfg.WithContext(ctx)

	entGame, err := gstore.Get(ctx, gameID)
	if err != nil {
		t.Fatal(err)
	}
	row, err := gstore.queries.GetGame(ctx, common.ToPGTypeText(gameID))
	if err != nil {
		t.Fatal(err)
	}
	work := gstore.shadowLoadWork(ctx, row, entGame)
	if work == nil {
		t.Fatal("shadow declined to run on a live game")
	}
	buf.Reset()
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
		is.NoErr(gstore.AppendTurns(ctx, gid, before, evts))
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
	is.NoErr(gstore.AppendTurns(ctx, gid, before, entGame.History().Events[before:]))
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
	cfg := DefaultConfig
	cfg.ShadowTurnsLoad = true
	ctx := cfg.WithContext(context.Background())

	entGame, err := gstore.Get(ctx, gid)
	is.NoErr(err)
	entGame.SetGameEndReason(pb.GameEndReason_STANDARD)
	entGame.SetPlaying(macondopb.PlayState_GAME_OVER)
	is.NoErr(gstore.Set(ctx, entGame))

	row, err := gstore.queries.GetGame(ctx, common.ToPGTypeText(gid))
	is.NoErr(err)
	is.Equal(gstore.shadowLoadWork(ctx, row, entGame), nil)
}
