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
	gstore.shadowCompareTurns(logged, gid, rules, want, racks, len(reloaded.History().Events))

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
