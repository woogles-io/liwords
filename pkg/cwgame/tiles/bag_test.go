package tiles

import (
	"os"
	"reflect"
	"testing"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/cwgame/runemapping"
	"github.com/matryer/is"
)

var DataDir = os.Getenv("DATA_PATH")
var DefaultConfig = &config.Config{DataPath: DataDir}

func TestBag(t *testing.T) {
	is := is.New(t)

	ld, err := EnglishLetterDistribution(DefaultConfig)
	is.NoErr(err)
	bag := TileBag(ld)
	if len(bag.Tiles) != ld.numLetters {
		t.Error("Tile bag and letter distribution do not match.")
	}
	tileMap := make(map[rune]uint8)
	numTiles := 0
	ml := make([]runemapping.MachineLetter, 1)

	for range bag.Tiles {
		err := Draw(bag, 1, ml)
		numTiles++
		uv := ml[0].UserVisible(ld.runemapping)
		t.Logf("Drew a %c! , %v", uv, numTiles)
		if err != nil {
			t.Error("Error drawing from tile bag.", err)
		}
		tileMap[uv]++
	}
	if !reflect.DeepEqual(tileMap, ld.Distribution) {
		t.Error("Distribution and tilemap were not identical.")
	}
	err = Draw(bag, 1, ml)
	if err == nil {
		t.Error("Should not have been able to draw from an empty bag.")
	}
}

func TestDraw(t *testing.T) {
	is := is.New(t)

	ld, err := EnglishLetterDistribution(DefaultConfig)
	is.NoErr(err)
	bag := TileBag(ld)
	ml := make([]runemapping.MachineLetter, 7)
	err = Draw(bag, 7, ml)
	is.NoErr(err)

	if len(bag.Tiles) != 93 {
		t.Errorf("Length was %v, expected 93", len(bag.Tiles))
	}
}

func TestDrawAtMost(t *testing.T) {
	is := is.New(t)

	ld, err := EnglishLetterDistribution(DefaultConfig)
	is.NoErr(err)
	bag := TileBag(ld)
	ml := make([]runemapping.MachineLetter, 7)
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

	ld, err := EnglishLetterDistribution(DefaultConfig)
	is.NoErr(err)
	bag := TileBag(ld)
	ml := make([]runemapping.MachineLetter, 7)
	err = Draw(bag, 7, ml)
	is.NoErr(err)
	newML := make([]runemapping.MachineLetter, 7)
	err = Exchange(bag, ml[:5], newML[:5])
	is.NoErr(err)
	is.Equal(len(bag.Tiles), 93)
}

// func TestRemoveTiles(t *testing.T) {
// 	is := is.New(t)

// 	ld, err := EnglishLetterDistribution(DefaultConfig)
// 	is.NoErr(err)
// 	bag := ld.TileBag()
// 	is.Equal(len(bag), 100)
// 	toRemove := []MachineLetter{
// 		9, 14, 24, 4, 3, 20, 4, 11, 21, 6, 22, 14, 8, 0, 8, 15, 6, 5, 4,
// 		19, 0, 24, 8, 17, 17, 18, 2, 11, 8, 14, 1, 8, 0, 20, 7, 0, 8, 10,
// 		0, 11, 13, 25, 11, 14, 5, 8, 19, 4, 12, 8, 18, 4, 3, 19, 14, 19,
// 		1, 0, 13, 4, 19, 14, 4, 17, 20, 6, 21, 104, 3, 7, 0, 3, 14, 22,
// 		4, 8, 13, 16, 20, 4, 18, 19, 4, 23, 4, 2, 17, 12, 14, 0, 13,
// 	}
// 	is.Equal(len(toRemove), 91)
// 	err = bag.RemoveTiles(toRemove)
// 	is.NoErr(err)
// 	is.Equal(len(bag), 9)
// }
