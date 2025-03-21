package cwgame

import (
	"testing"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

func TestLeave(t *testing.T) {
	// Test blank exchange works
	is := is.New(t)
	l, err := tilemapping.Leave([]tilemapping.MachineLetter{0, 1, 2, 3, 4, 5},
		[]tilemapping.MachineLetter{0, 3, 5}, true)

	is.NoErr(err)
	is.Equal(l, tilemapping.MachineWord{1, 2, 4})
}
