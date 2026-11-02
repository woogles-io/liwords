package memento

import (
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

// The super German set has its own counts, the German set's scores, and is
// drawn on the 21x21 board.
func TestGermanSuperDrawer(t *testing.T) {
	is := is.New(t)
	bd, ok := BoardDrawers["german_super"]
	is.True(ok)
	is.Equal(len(bd.BoardConfig), 21)
	ld := bd.LetterDistribution
	is.Equal(ld.NumTotalLetters(), uint(200))
	german := BoardDrawers["german"].LetterDistribution
	is.Equal(german.NumTotalLetters(), uint(102))
	is.Equal(ld.TileMapping().NumLetters(), german.TileMapping().NumLetters())
	for ml := tilemapping.MachineLetter(0); ml < tilemapping.MachineLetter(german.TileMapping().NumLetters()); ml++ {
		is.Equal(ld.Score(ml), german.Score(ml))
	}
}
