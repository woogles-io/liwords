package board

import (
	"testing"

	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/word-golib/tilemapping"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"github.com/woogles-io/liwords/pkg/omgwords/game/gamestate"
)

var DefaultMacondoConfig = macondoconfig.DefaultConfig()

func buildBoardFlatBuffer() *gamestate.Board {
	builder := flatbuffers.NewBuilder(512)
	boardOffset, err := BuildBoard(builder, "CrosswordGame")
	if err != nil {
		panic(err)
	}

	builder.Finish(boardOffset)
	bd := gamestate.GetRootAsBoard(builder.FinishedBytes(), 0)
	return bd
}

func TestFormedWords(t *testing.T) {
	is := is.New(t)

	bd := buildBoardFlatBuffer()

	ld, err := tilemapping.NamedLetterDistribution(DefaultMacondoConfig.WGLConfig(), "english")
	is.NoErr(err)
	tm := ld.TileMapping()
	setFromPlaintext(bd, VsOxy, tm)

	mls, err := tilemapping.ToMachineLetters("OX.P...B..AZ..E", tm)
	is.NoErr(err)

	words, err := FormedWords(bd, 0, 0, true, mls)
	is.NoErr(err)

	is.Equal(len(words), 8)
	// convert all words to user-visible
	uvWords := make([]string, 8)
	for idx, w := range words {
		uvWords[idx] = w.UserVisible(tm)
	}
	is.Equal(uvWords, []string{"OXYPHENBUTAZONE", "OPACIFYING", "XIS", "PREQUALIFIED", "BRAINWASHING",
		"AWAKENERS", "ZONETIME", "EJACULATING"})

}

func TestPlayMoveGiant(t *testing.T) {
	is := is.New(t)

	dist, err := tilemapping.GetDistribution(DefaultMacondoConfig.WGLConfig(), "english")
	is.NoErr(err)

	bd := buildBoardFlatBuffer()

	ld, err := tilemapping.NamedLetterDistribution(DefaultMacondoConfig.WGLConfig(), "english")
	is.NoErr(err)
	tm := ld.TileMapping()
	setFromPlaintext(bd, VsOxy, tm)

	mls, err := tilemapping.ToMachineLetters("OX.P...B..AZ..E", tm)
	is.NoErr(err)

	score, err := PlayMove(bd, "CrosswordGame", dist, mls, 0, 0, true)
	is.NoErr(err)
	is.Equal(score, int32(1780))
}

func TestMoveInBetween(t *testing.T) {
	is := is.New(t)

	dist, err := tilemapping.GetDistribution(DefaultMacondoConfig.WGLConfig(), "english")
	is.NoErr(err)

	bd := buildBoardFlatBuffer()
	tm := dist.TileMapping()

	setFromPlaintext(bd, VsMatt, tm)

	mls, err := tilemapping.ToMachineLetters("TAEL", tm)
	is.NoErr(err)

	score, err := PlayMove(bd, "CrosswordGame", dist, mls, 8, 10, true)
	is.NoErr(err)
	is.Equal(score, int32(38))
}

func TestToFENEmpty(t *testing.T) {
	is := is.New(t)

	dist, err := tilemapping.GetDistribution(DefaultMacondoConfig.WGLConfig(), "english")
	is.NoErr(err)

	bd := buildBoardFlatBuffer()

	is.Equal(ToFEN(bd, dist), "15/15/15/15/15/15/15/15/15/15/15/15/15/15/15")
}

func TestToFEN(t *testing.T) {
	is := is.New(t)

	dist, err := tilemapping.GetDistribution(DefaultMacondoConfig.WGLConfig(), "english")
	is.NoErr(err)

	bd := buildBoardFlatBuffer()

	tm := dist.TileMapping()

	setFromPlaintext(bd, VsMatt, tm)

	is.Equal(ToFEN(bd, dist),
		"7ZEP1F3/1FLUKY3R1R3/5EX2A1U3/2SCARIEST1I3/9TOT3/6GO1LO4/6OR1ETA3/6JABS1b3/5QI4A3/5I1N3N3/3ReSPOND1D3/1HOE3V3O3/1ENCOMIA3N3/7T7/3VENGED6")
}

func TestToFENCatalan(t *testing.T) {
	is := is.New(t)

	dist, err := tilemapping.GetDistribution(DefaultMacondoConfig.WGLConfig(), "catalan")
	is.NoErr(err)

	bd := buildBoardFlatBuffer()
	bd.MutateTiles(7*15+6, 1)
	bd.MutateTiles(7*15+7, 13)
	bd.MutateTiles(7*15+8, 1)

	is.Equal(ToFEN(bd, dist), "15/15/15/15/15/15/15/6A[L·L]A6/15/15/15/15/15/15/15")
}

func BenchmarkPlayMove(b *testing.B) {
	dist, _ := tilemapping.GetDistribution(DefaultMacondoConfig.WGLConfig(), "english")
	bd := buildBoardFlatBuffer()

	tm := dist.TileMapping()
	lv := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)
	defer zerolog.SetGlobalLevel(lv)
	mls, _ := tilemapping.ToMachineLetters("TAEL", tm)

	// ~29us per operation on themonolith
	for i := 0; i < b.N; i++ {
		setFromPlaintext(bd, VsMatt, tm)
		PlayMove(bd, "CrosswordGame", dist, mls, 8, 10, true)
	}
}
