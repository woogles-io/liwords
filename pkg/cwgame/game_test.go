package cwgame

import (
	"testing"

	"github.com/domino14/macondo/tilemapping"
	"github.com/matryer/is"
)

func TestLeave(t *testing.T) {
	// Test blank exchange works
	is := is.New(t)
	l, err := tilemapping.Leave([]tilemapping.MachineLetter{0, 3, 5, 6},
		[]tilemapping.MachineLetter{0}, true)

	is.NoErr(err)
	is.Equal(l, tilemapping.MachineWord{3, 5, 6})
}
