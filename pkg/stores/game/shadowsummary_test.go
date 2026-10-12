package game

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/rs/zerolog"

	"github.com/woogles-io/liwords/pkg/stores/user"
)

// The summary reports each count and then starts again from zero, so every
// line covers its own period and nothing is counted twice.
func TestShadowSummaryReportsAndResets(t *testing.T) {
	is := is.New(t)
	var c shadowCounts
	c.ok.Add(3)
	c.mismatch.Add(1)
	c.noTurns.Add(2)

	read := func() map[string]any {
		var buf syncBuffer
		c.report(zerolog.New(&buf), "load", time.Hour)
		var rec map[string]any
		is.NoErr(json.Unmarshal([]byte(buf.String()), &rec))
		return rec
	}

	first := read()
	is.Equal(first["message"], "shadow-summary")
	is.Equal(first["shadow"], "load")
	is.Equal(first["ok"], 3.0)
	is.Equal(first["mismatch"], 1.0)
	is.Equal(first["no_turns"], 2.0)
	is.Equal(first["torn"], 0.0)

	second := read()
	is.Equal(second["ok"], 0.0)
	is.Equal(second["mismatch"], 0.0)
	is.Equal(second["no_turns"], 0.0)
}

// A game with no turn rows is still a load the shadow saw, and is counted as
// such: it is the difference between "nothing to compare" and "never ran".
func TestShadowLoadCountsGamesWithoutTurns(t *testing.T) {
	is := is.New(t)
	ustore, gstore := recreateDB()
	defer gstore.Disconnect()
	defer ustore.(*user.DBStore).Disconnect()

	const gid = "wJxURccCgSAPivUvj4QdYL"
	ctx := context.Background()
	// Saved once, so the games row carries its racks as a live game's does;
	// no moves, so it has no turn rows.
	entGame, err := gstore.Get(ctx, gid)
	is.NoErr(err)
	is.NoErr(gstore.Set(ctx, entGame))

	before := loadShadowCounts.noTurns.Load()
	shadowLoad(t, gstore, gid)
	is.Equal(loadShadowCounts.noTurns.Load()-before, int64(1))
}
