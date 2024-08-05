package tiles

import (
	"reflect"
	"sort"
	"testing"

	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

var DefaultConfig = macondoconfig.DefaultConfig()

func TestBag(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	bag := TileBag(ld)
	if len(bag.Tiles) != 100 {
		t.Error("Tile bag and letter distribution do not match.")
	}
	tileMap := make([]uint8, len(ld.Distribution()))
	numTiles := 0
	ml := make([]tilemapping.MachineLetter, 1)

	for range bag.Tiles {
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

func TestNorwegianBag(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.NamedLetterDistribution(DefaultConfig.WGLConfig(), "norwegian")
	is.NoErr(err)
	bag := TileBag(ld)
	if len(bag.Tiles) != 100 {
		t.Error("Tile bag and letter distribution do not match.")
	}
	tileMap := make([]uint8, len(ld.Distribution()))
	numTiles := 0
	ml := make([]tilemapping.MachineLetter, 1)

	for range bag.Tiles {
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

func TestDraw(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	bag := TileBag(ld)
	ml := make([]tilemapping.MachineLetter, 7)
	err = Draw(bag, 7, ml)
	is.NoErr(err)

	if len(bag.Tiles) != 93 {
		t.Errorf("Length was %v, expected 93", len(bag.Tiles))
	}
}

func TestDrawAtMost(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	bag := TileBag(ld)
	ml := make([]tilemapping.MachineLetter, 7)
	for i := 0; i < 14; i++ {
		err := Draw(bag, 7, ml)
		is.NoErr(err)
	}
	is.Equal(len(bag.Tiles), 2)
	drawn, err := DrawAtMost(bag, 7, ml)
	is.NoErr(err)
	is.Equal(drawn, 2)
	is.Equal(len(bag.Tiles), 0)
	// Try to draw one more time.
	drawn, err = DrawAtMost(bag, 7, ml)
	is.NoErr(err)
	is.Equal(drawn, 0)
	is.Equal(len(bag.Tiles), 0)
}

func TestExchange(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	bag := TileBag(ld)
	ml := make([]tilemapping.MachineLetter, 7)
	err = Draw(bag, 7, ml)
	is.NoErr(err)
	newML := make([]tilemapping.MachineLetter, 7)
	err = Exchange(bag, ml[:5], newML[:5])
	is.NoErr(err)
	is.Equal(len(bag.Tiles), 93)
}

func TestRemoveTiles(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	bag := TileBag(ld)
	is.Equal(len(bag.Tiles), 100)
	toRemove := []tilemapping.MachineLetter{
		10, 15, 25, 5, 4, 21, 5, 12, 22, 7, 23, 15, 9, 1, 9, 16, 7, 6, 5,
		20, 1, 25, 9, 18, 18, 19, 3, 12, 9, 15, 2, 9, 1, 21, 8, 1, 9, 11,
		1, 12, 14, 26, 12, 15, 6, 9, 20, 5, 13, 9, 19, 5, 4, 20, 15, 20,
		2, 1, 14, 5, 20, 15, 5, 18, 21, 7, 22, 0, 4, 8, 1, 4, 15, 23,
		5, 9, 14, 17, 21, 5, 19, 20, 5, 24, 5, 3, 18, 13, 15, 1, 14}
	is.Equal(len(toRemove), 91)
	err = RemoveTiles(bag, toRemove)
	is.NoErr(err)
	is.Equal(len(bag.Tiles), 9)
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
	bag := TileBag(ld)
	is.Equal(len(bag.Tiles), 100)
	toRemove := []tilemapping.MachineLetter{30}
	err = RemoveTiles(bag, toRemove)
	is.NoErr(err)
	is.Equal(len(bag.Tiles), 99)
}

func TestPutBack(t *testing.T) {
	is := is.New(t)

	ld, err := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	is.NoErr(err)
	bag := TileBag(ld)
	is.Equal(len(bag.Tiles), 100)
	toRemove := []tilemapping.MachineLetter{
		10, 15, 25, 5, 4, 21, 5, 12, 22, 7, 23, 15, 9, 1, 9, 16, 7, 6, 5,
		20, 1, 25, 9, 18, 18, 19, 3, 12, 9, 15, 2, 9, 1, 21, 8, 1, 9, 11,
		1, 12, 14, 26, 12, 15, 6, 9, 20, 5, 13, 9, 19, 5, 4, 20, 15, 20,
		2, 1, 14, 5, 20, 15, 5, 18, 21, 7, 22, 0, 4, 8, 1, 4, 15, 23,
		5, 9, 14, 17, 21, 5, 19, 20, 5, 24, 5, 3, 18, 13, 15, 1, 14}
	is.Equal(len(toRemove), 91)
	err = RemoveTiles(bag, toRemove)
	is.NoErr(err)
	is.Equal(len(bag.Tiles), 9)
	PutBack(bag, toRemove)
	is.Equal(len(bag.Tiles), 100)
	// Make sure the bag is the same as a brand new bag.

	sort.Slice(bag.Tiles, func(i, j int) bool { return bag.Tiles[i] < bag.Tiles[j] })
	newBag := TileBag(ld)
	sort.Slice(newBag.Tiles, func(i, j int) bool { return newBag.Tiles[i] < newBag.Tiles[j] })
	is.Equal(bag.Tiles, newBag.Tiles)
}

func BenchmarkRemoveTiles(b *testing.B) {
	ld, _ := tilemapping.EnglishLetterDistribution(DefaultConfig.WGLConfig())
	// remove 14 tiles
	toRemove := []tilemapping.MachineLetter{
		10, 15, 25, 5, 4, 21, 5, 12, 22, 7, 23, 15, 9, 1}
	b.ResetTimer()
	// 4473 ns/op on themonolith
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		bag := TileBag(ld)
		b.StartTimer()
		RemoveTiles(bag, toRemove)
	}
}
