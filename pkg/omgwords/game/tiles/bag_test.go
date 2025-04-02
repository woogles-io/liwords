package tiles

import (
	"crypto/rand"
	"reflect"
	"sort"
	"testing"

	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/word-golib/tilemapping"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/omgwords/game/gamestate"
)

var DefaultConfig = macondoconfig.DefaultConfig()

type GameStateOption func(*gameStateOptions)

type gameStateOptions struct {
	seed [32]byte
}

func WithSeed(seed [32]byte) GameStateOption {
	return func(opts *gameStateOptions) {
		opts.seed = seed
	}
}

func cwGameStateWithBag(ld *tilemapping.LetterDistribution, opts ...GameStateOption) *gamestate.GameState {
	builder := flatbuffers.NewBuilder(512)
	options := gameStateOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	var seed [32]byte
	if options.seed == [32]byte{} {
		// rand.Read is supposed to never return an error.
		rand.Read(seed[:])
	} else {
		seed = options.seed
	}

	tileBagOffset := BuildTileBag(builder, ld, seed)
	gamestate.GameStateStart(builder)
	gamestate.GameStateAddBag(builder, tileBagOffset)
	tt := gamestate.GameStateEnd(builder)
	builder.Finish(tt)

	st := gamestate.GetRootAsGameState(builder.FinishedBytes(), 0)
	return st
}

func TestBag(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)

	st := cwGameStateWithBag(ld)
	bag := st.Bag(nil)
	if len(bag.BagBytes()) != 100 {
		t.Error("Tile bag and letter distribution do not match.")
	}
	tileMap := make([]uint8, len(ld.Distribution()))
	numTiles := 0
	ml := make([]tilemapping.MachineLetter, 1)

	for range bag.BagBytes() {
		err := Draw(bag, 1, ml)
		numTiles++
		uv := ml[0].UserVisible(ld.TileMapping(), false)
		t.Logf("Drew a %v!, %v", uv, numTiles)
		if err != nil {
			t.Error("Error drawing from tile bag.", err)
		}
		tileMap[ml[0]]++
	}
	if !reflect.DeepEqual(tileMap, ld.Distribution()) {
		t.Error("Distribution and tilemap were not identical.")
	}
	err = Draw(bag, 1, ml)
	if err == nil {
		t.Error("Should not have been able to draw from an empty bag.")
	}
}

func TestRepeatableOrderWithSeed(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)

	st := cwGameStateWithBag(ld, WithSeed([32]byte{
		1, 2, 3, 4, 5, 6, 7, 8,
		9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24,
		25, 26, 27, 28, 29, 30, 31}))
	bag := st.Bag(nil)

	tilesDrawn := make([]tilemapping.MachineLetter, 0)

	ml := make([]tilemapping.MachineLetter, 1)

	for range bag.BagBytes() {
		err := Draw(bag, 1, ml)
		if err != nil {
			t.Error("Error drawing from tile bag.", err)
		}
		tilesDrawn = append(tilesDrawn, ml[0])
	}

	st2 := cwGameStateWithBag(ld, WithSeed([32]byte{
		1, 2, 3, 4, 5, 6, 7, 8,
		9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24,
		25, 26, 27, 28, 29, 30, 31}))

	bag2 := st2.Bag(nil)
	ml2 := make([]tilemapping.MachineLetter, 1)
	tilesDrawn2 := make([]tilemapping.MachineLetter, 0)

	for range bag2.BagBytes() {
		err := Draw(bag2, 1, ml2)
		if err != nil {
			t.Error("Error drawing from tile bag.", err)
		}
		tilesDrawn2 = append(tilesDrawn2, ml2[0])
	}
	is.Equal(tilesDrawn, tilesDrawn2)
}

func TestNorwegianBag(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.NamedLetterDistribution(DefaultConfig.WGLConfig(), "norwegian")
	is.NoErr(err)
	st := cwGameStateWithBag(ld)
	bag := st.Bag(nil)
	if len(bag.BagBytes()) != 100 {
		t.Error("Tile bag and letter distribution do not match.")
	}
	tileMap := make([]uint8, len(ld.Distribution()))
	numTiles := 0
	ml := make([]tilemapping.MachineLetter, 1)

	for range bag.BagBytes() {
		err := Draw(bag, 1, ml)
		numTiles++
		uv := ml[0].UserVisible(ld.TileMapping(), false)
		t.Logf("Drew a %v!, %v", uv, numTiles)
		is.NoErr(err)
		tileMap[ml[0]]++
	}

	if !reflect.DeepEqual(tileMap, ld.Distribution()) {
		t.Error("Distribution and tilemap were not identical.")
	}
	err = Draw(bag, 1, ml)
	if err == nil {
		t.Error("Should not have been able to draw from an empty bag.")
	}
}

func TestDraw(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	st := cwGameStateWithBag(ld)
	bag := st.Bag(nil)
	ml := make([]tilemapping.MachineLetter, 7)
	err = Draw(bag, 7, ml)
	is.NoErr(err)

	is.Equal(int(bag.NumTilesInBag()), 93)
	is.Equal(len(bag.BagBytes()), 100)
}

func TestDrawAtMost(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	st := cwGameStateWithBag(ld)
	bag := st.Bag(nil)
	ml := make([]tilemapping.MachineLetter, 7)
	for i := 0; i < 14; i++ {
		err := Draw(bag, 7, ml)
		is.NoErr(err)
	}
	is.Equal(int(bag.NumTilesInBag()), 2)
	drawn, err := DrawAtMost(bag, 7, ml)
	is.NoErr(err)
	is.Equal(drawn, 2)
	is.Equal(int(bag.NumTilesInBag()), 0)
	// Try to draw one more time.
	drawn, err = DrawAtMost(bag, 7, ml)
	is.NoErr(err)
	is.Equal(drawn, 0)
	is.Equal(int(bag.NumTilesInBag()), 0)
}

func TestExchange(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	st := cwGameStateWithBag(ld)
	bag := st.Bag(nil)
	ml := make([]tilemapping.MachineLetter, 7)
	err = Draw(bag, 7, ml)
	is.NoErr(err)
	newML := make([]tilemapping.MachineLetter, 7)
	err = Exchange(bag, ml[:5], newML[:5])
	is.NoErr(err)
	is.Equal(int(bag.NumTilesInBag()), 93)
}

func TestRemoveTiles(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	st := cwGameStateWithBag(ld)
	bag := st.Bag(nil)
	is.Equal(int(bag.NumTilesInBag()), 100)
	toRemove := []tilemapping.MachineLetter{
		10, 15, 25, 5, 4, 21, 5, 12, 22, 7, 23, 15, 9, 1, 9, 16, 7, 6, 5,
		20, 1, 25, 9, 18, 18, 19, 3, 12, 9, 15, 2, 9, 1, 21, 8, 1, 9, 11,
		1, 12, 14, 26, 12, 15, 6, 9, 20, 5, 13, 9, 19, 5, 4, 20, 15, 20,
		2, 1, 14, 5, 20, 15, 5, 18, 21, 7, 22, 0, 4, 8, 1, 4, 15, 23,
		5, 9, 14, 17, 21, 5, 19, 20, 5, 24, 5, 3, 18, 13, 15, 1, 14}
	is.Equal(len(toRemove), 91)
	err = RemoveTiles(bag, toRemove)
	is.NoErr(err)
	is.Equal(int(bag.NumTilesInBag()), 9)
	// Draw these last tiles and compare to what they should be
	todraw := make([]tilemapping.MachineLetter, 9)
	err = Draw(bag, 9, todraw)
	is.NoErr(err)
	sort.Slice(todraw, func(i, j int) bool { return todraw[i] < todraw[j] })
	is.Equal(todraw, []tilemapping.MachineLetter{0, 1, 5, 14, 14, 16, 18, 18, 19})
}

func TestRemoveNorwegianTile(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.NamedLetterDistribution(DefaultConfig.WGLConfig(), "norwegian")
	is.NoErr(err)
	st := cwGameStateWithBag(ld)
	bag := st.Bag(nil)
	is.Equal(int(bag.NumTilesInBag()), 100)
	toRemove := []tilemapping.MachineLetter{30}
	err = RemoveTiles(bag, toRemove)
	is.NoErr(err)
	is.Equal(int(bag.NumTilesInBag()), 99)
}

func TestPutBack(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	st := cwGameStateWithBag(ld)
	bag := st.Bag(nil)
	is.Equal(int(bag.NumTilesInBag()), 100)
	toRemove := []tilemapping.MachineLetter{
		10, 15, 25, 5, 4, 21, 5, 12, 22, 7, 23, 15, 9, 1, 9, 16, 7, 6, 5,
		20, 1, 25, 9, 18, 18, 19, 3, 12, 9, 15, 2, 9, 1, 21, 8, 1, 9, 11,
		1, 12, 14, 26, 12, 15, 6, 9, 20, 5, 13, 9, 19, 5, 4, 20, 15, 20,
		2, 1, 14, 5, 20, 15, 5, 18, 21, 7, 22, 0, 4, 8, 1, 4, 15, 23,
		5, 9, 14, 17, 21, 5, 19, 20, 5, 24, 5, 3, 18, 13, 15, 1, 14}
	is.Equal(len(toRemove), 91)
	err = RemoveTiles(bag, toRemove)
	is.NoErr(err)
	is.Equal(int(bag.NumTilesInBag()), 9)
	PutBack(bag, toRemove)
	is.Equal(int(bag.NumTilesInBag()), 100)
	// Make sure the bag is the same as a brand new bag.

	Sort(bag)

	newSt := cwGameStateWithBag(ld)
	newBag := newSt.Bag(nil)
	Sort(newBag)
	is.Equal(bag.BagBytes(), newBag.BagBytes())
}

// func BenchmarkRemoveTiles(b *testing.B) {
// 	ld, _ := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
// 	// remove 14 tiles
// 	toRemove := []tilemapping.MachineLetter{
// 		10, 15, 25, 5, 4, 21, 5, 12, 22, 7, 23, 15, 9, 1}
// 	b.ResetTimer()
// 	// 4473 ns/op on themonolith
// 	for i := 0; i < b.N; i++ {
// 		b.StopTimer()
// 		st := cwGameStateWithBag(ld)
// 		bag := st.Bag(nil)
// 		b.StartTimer()
// 		RemoveTiles(bag, toRemove)
// 	}
// }

func BenchmarkDraw(b *testing.B) {
	ld, _ := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		st := cwGameStateWithBag(ld)
		bag := st.Bag(nil)
		ml := make([]tilemapping.MachineLetter, 7)
		b.StartTimer()
		Draw(bag, 7, ml)
	}
}
