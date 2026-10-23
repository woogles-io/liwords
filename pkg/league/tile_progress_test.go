package league

import (
	"os"
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/macondo/board"
	macondoconfig "github.com/domino14/macondo/config"
	macondogame "github.com/domino14/macondo/game"
	"github.com/domino14/macondo/gcgio"
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

// TestTileProgressRealGame replays the first ten turns of a real game,
// phony included, and checks the counts against the board and the bag.
func TestTileProgressRealGame(t *testing.T) {
	is := is.New(t)
	mcfg := macondoconfig.DefaultConfig()
	if p := os.Getenv("MACONDO_DATA_PATH"); p != "" {
		mcfg.Set(macondoconfig.ConfigDataPath, p)
	}
	mcfg.SetDefault(macondoconfig.ConfigDefaultLexicon, "CSW21")
	mcfg.SetDefault(macondoconfig.ConfigDefaultLetterDistribution, "english")
	f, err := os.Open("../omgwords/testdata/vs_josh.gcg")
	is.NoErr(err)
	defer f.Close()
	hist, err := gcgio.ParseGCGFromReader(mcfg, f)
	is.NoErr(err)
	rules, err := macondogame.NewBasicGameRules(mcfg, "CSW21", board.CrosswordGameLayout,
		"english", macondogame.CrossScoreOnly, macondogame.VarClassic)
	is.NoErr(err)

	// A live game's history ends at its current turn. Through cesar's
	// DEMIR.P: cesar has DJINN, sCRIEVE, .AZY and DEMIR.P down (MAZEY came
	// back off), josh WITHY, MA.TERLy, ALGAE and OO.E.
	partial := proto.Clone(hist).(*macondopb.GameHistory)
	partial.Events = partial.Events[:10]
	g, err := macondogame.NewFromHistory(partial, rules, 10)
	is.NoErr(err)
	dist, err := tilemapping.GetDistribution(mcfg.WGLConfig(), "english")
	is.NoErr(err)
	played, unplayed := tileProgress(g.History(), dist)
	is.Equal(played, [2]int32{21, 20})
	is.Equal(unplayed, int32(59))
	// the bag and both racks hold exactly the unplayed tiles.
	is.Equal(unplayed, int32(g.Bag().TilesRemaining())+int32(g.RackFor(0).NumTiles())+int32(g.RackFor(1).NumTiles()))

	// The whole game: cesar went out, josh was left holding BLONST.
	g, err = macondogame.NewFromHistory(hist, rules, len(hist.Events))
	is.NoErr(err)
	played, unplayed = tileProgress(g.History(), dist)
	is.Equal(played, [2]int32{50, 44})
	is.Equal(unplayed, int32(6))
}
