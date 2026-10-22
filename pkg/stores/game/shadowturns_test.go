package game

// The save-path shadow against a game macondo rebuilt from history rather than
// one it played live. Added while splitting PR #1975; see
// docs/mikado/pr1975_split.md.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog"

	"github.com/woogles-io/liwords/pkg/stores/user"
	"github.com/woogles-io/liwords/pkg/xwordbridge"
	"github.com/woogles-io/liwords/pkg/xwordgame"
)

// Per-player turn counts are lost by macondo's replay, so a game loaded from
// history disagrees with game_turns on them. That must not be reported as a
// mismatch: it is macondo's bug, nothing reads the count, and every
// correspondence game would trip it.
func TestShadowTurnsIgnoresTurnCountsLostByReplay(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()

	const gid = "wJxURccCgSAPivUvj4QdYL"
	ctx := context.Background()

	// Played live, on one object, the way a cached game is.
	live, err := gstore.Get(ctx, gid)
	is.NoErr(err)
	for _, mv := range []struct{ coords, word string }{{"8E", "AGUE"}, {"E8", "AVE"}} {
		before := len(live.History().Events)
		_, err = live.PlayScoringMove(mv.coords, mv.word, true)
		is.NoErr(err)
		is.NoErr(gstore.AppendTurns(ctx, gid, before, live.History().Events[before:]))
		is.NoErr(gstore.Set(ctx, live))
	}

	// Loaded again: macondo rebuilds it with NewFromHistory.
	reloaded, err := gstore.Get(ctx, gid)
	is.NoErr(err)
	livePos, err := xwordbridge.StateFromGame(&live.Game)
	is.NoErr(err)
	want, err := xwordbridge.StateFromGame(&reloaded.Game)
	is.NoErr(err)

	// The precondition. If this fails, macondo's replay keeps turn counts now,
	// and the exclusion in shadowCompareTurns can go.
	is.True(livePos.PlayerTurns != want.PlayerTurns)

	spec := xwordbridge.SpecFromHistory(reloaded.History())
	if spec.Lexicon == "" && reloaded.GameReq != nil {
		spec.Lexicon = reloaded.GameReq.Lexicon
	}
	rules, err := xwordbridge.RulesFor(gstore.cfg.MacondoConfig(), spec)
	is.NoErr(err)
	racks := make([]string, xwordgame.MaxPlayers)
	for p := range racks {
		racks[p] = reloaded.Game.RackLettersFor(p)
	}

	var buf syncBuffer
	logged := zerolog.New(&buf).Level(zerolog.DebugLevel).WithContext(ctx)
	okBefore := turnsShadowCounts.ok.Load()
	gstore.shadowCompareTurns(logged, gid, rules, want, racks, len(reloaded.History().Events))
	is.Equal(turnsShadowCounts.ok.Load()-okBefore, int64(1))

	var msgs []string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var rec struct {
			Message string `json:"message"`
		}
		is.NoErr(json.Unmarshal([]byte(line), &rec))
		msgs = append(msgs, rec.Message)
		t.Log(line)
	}
	is.True(hasPrefix(msgs, "shadow-turns-ok"))
	is.True(!hasPrefix(msgs, "shadow-turns-mismatch"))
}

// The shadow runs in a goroutine, so the next move can commit its rows before
// the shadow reads them; a bot's reply often does. That is the game moving on,
// not a row the history lacks, and the shadow must compare the rows its
// snapshot was taken against rather than report a mismatch.
func TestShadowTurnsComparesSnapshotWhenGameMovedOn(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()

	const gid = "wJxURccCgSAPivUvj4QdYL"
	ctx := context.Background()

	live, err := gstore.Get(ctx, gid)
	is.NoErr(err)
	play := func(coords, word string) {
		before := len(live.History().Events)
		_, err := live.PlayScoringMove(coords, word, true)
		is.NoErr(err)
		is.NoErr(gstore.AppendTurns(ctx, gid, before, live.History().Events[before:]))
		is.NoErr(gstore.Set(ctx, live))
	}

	// What SpawnShadowCompare captures after the first move...
	play("8E", "AGUE")
	want, err := xwordbridge.StateFromGame(&live.Game)
	is.NoErr(err)
	wantEvents := len(live.History().Events)
	racks := make([]string, xwordgame.MaxPlayers)
	for p := range racks {
		racks[p] = live.Game.RackLettersFor(p)
	}
	spec := xwordbridge.SpecFromHistory(live.History())
	if spec.Lexicon == "" && live.GameReq != nil {
		spec.Lexicon = live.GameReq.Lexicon
	}
	rules, err := xwordbridge.RulesFor(gstore.cfg.MacondoConfig(), spec)
	is.NoErr(err)

	// ...and the reply, committed before the shadow gets to read the table.
	play("E8", "AVE")

	var buf syncBuffer
	logged := zerolog.New(&buf).Level(zerolog.DebugLevel).WithContext(ctx)
	okBefore := turnsShadowCounts.ok.Load()
	gstore.shadowCompareTurns(logged, gid, rules, want, racks, wantEvents)

	var msgs []string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var rec struct {
			Message string `json:"message"`
		}
		is.NoErr(json.Unmarshal([]byte(line), &rec))
		msgs = append(msgs, rec.Message)
		t.Log(line)
	}
	is.True(hasPrefix(msgs, "shadow-turns-ok"))
	is.True(!hasPrefix(msgs, "shadow-turns-mismatch"))
	is.Equal(turnsShadowCounts.ok.Load()-okBefore, int64(1))
}

// The shadow runs for finished games too, since the save that ends a game is
// the one that commits its last rows. In a finished position the replay drops
// the last play's words -- nothing can be challenged once the game is over --
// while macondo's live game keeps them. That is not a mismatch.
func TestShadowTurnsIgnoresWordsFormedInFinishedGame(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()

	const gid = "wJxURccCgSAPivUvj4QdYL"
	ctx := context.Background()

	live, err := gstore.Get(ctx, gid)
	is.NoErr(err)
	before := len(live.History().Events)
	_, err = live.PlayScoringMove("8E", "AGUE", true)
	is.NoErr(err)
	is.NoErr(gstore.AppendTurns(ctx, gid, before, live.History().Events[before:]))
	is.NoErr(gstore.Set(ctx, live))

	want, err := xwordbridge.StateFromGame(&live.Game)
	is.NoErr(err)
	// The game ending on that play, as macondo reports it: over, with the
	// play's words still attached.
	want.PlayState = xwordgame.GameOver
	is.True(len(want.LastWordsFormed) > 0)

	racks := make([]string, xwordgame.MaxPlayers)
	for p := range racks {
		racks[p] = live.Game.RackLettersFor(p)
	}
	spec := xwordbridge.SpecFromHistory(live.History())
	if spec.Lexicon == "" && live.GameReq != nil {
		spec.Lexicon = live.GameReq.Lexicon
	}
	rules, err := xwordbridge.RulesFor(gstore.cfg.MacondoConfig(), spec)
	is.NoErr(err)

	var buf syncBuffer
	logged := zerolog.New(&buf).Level(zerolog.DebugLevel).WithContext(ctx)
	okBefore := turnsShadowCounts.ok.Load()
	gstore.shadowCompareTurns(logged, gid, rules, want, racks, len(live.History().Events))

	var msgs []string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var rec struct {
			Message string `json:"message"`
		}
		is.NoErr(json.Unmarshal([]byte(line), &rec))
		msgs = append(msgs, rec.Message)
		t.Log(line)
	}
	is.True(hasPrefix(msgs, "shadow-turns-ok"))
	is.True(!hasPrefix(msgs, "shadow-turns-mismatch"))
	is.Equal(turnsShadowCounts.ok.Load()-okBefore, int64(1))
}
