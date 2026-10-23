package league

import (
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/config"
)

func TestTilesPlayedByPlayer(t *testing.T) {
	is := is.New(t)
	cfg := config.DefaultConfig()
	dist, err := tilemapping.GetDistribution(cfg.WGLConfig(), "english")
	is.NoErr(err)

	play := func(p uint32, tiles string) *macondopb.GameEvent {
		return &macondopb.GameEvent{Type: macondopb.GameEvent_TILE_PLACEMENT_MOVE, PlayerIndex: p, PlayedTiles: tiles}
	}
	events := []*macondopb.GameEvent{
		play(0, "BANJO"),
		// six new tiles through the board's A; a lowercase blank counts.
		play(1, "BU.sUTI"),
		// a phony, then taken back.
		play(0, "ZZXQ"),
		{Type: macondopb.GameEvent_PHONY_TILES_RETURNED, PlayerIndex: 0},
		{Type: macondopb.GameEvent_PASS, PlayerIndex: 1},
		play(0, "RE"),
		// unreadable tiles are skipped, not miscounted.
		play(1, "!!"),
	}
	is.Equal(tilesPlayedByPlayer(events, dist.TileMapping()), [2]int32{7, 6})
}
