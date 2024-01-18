package board

import (
	"regexp"

	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/word-golib/tilemapping"
)

var boardPlaintextRegex = regexp.MustCompile(`\|(.+)\|`)
var userRackRegex = regexp.MustCompile(`(?U).+\s+([A-Z\?]*)\s+-?[0-9]+`)

// SetFromPlaintext sets the board from the given plaintext board.
// It returns a list of all played machine letters (tiles) so that the
// caller can reconcile the tile bag appropriately.
func setFromPlaintext(board *ipc.GameBoard, qText string,
	rm *tilemapping.TileMapping) {

	// Take a Quackle Plaintext Board and turn it into an internal structure.
	result := boardPlaintextRegex.FindAllStringSubmatch(qText, -1)
	if len(result) != 15 {
		panic("Wrongly implemented")
	}

	var err error
	var letter tilemapping.MachineLetter
	for i := range result {
		// result[i][1] has the string
		j := -1
		for _, ch := range result[i][1] {
			j++
			if j%2 != 0 {
				continue
			}
			letter, err = rm.Val(string(ch))
			pos := i*15 + (j / 2)
			if err != nil {
				// Ignore the error; we are passing in a space or another
				// board marker.
				board.Tiles[pos] = 0
			} else {
				board.Tiles[pos] = byte(letter)
				// playedTiles = append(playedTiles, letter)
			}
		}
	}

}
