package board

import (
	"regexp"

	"github.com/domino14/liwords/pkg/cwgame/runemapping"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
)

var boardPlaintextRegex = regexp.MustCompile(`\|(.+)\|`)
var userRackRegex = regexp.MustCompile(`(?U).+\s+([A-Z\?]*)\s+-?[0-9]+`)

// SetFromPlaintext sets the board from the given plaintext board.
// It returns a list of all played machine letters (tiles) so that the
// caller can reconcile the tile bag appropriately.
func setFromPlaintext(board *ipc.GameBoard, qText string,
	rm *runemapping.RuneMapping) {

	// Take a Quackle Plaintext Board and turn it into an internal structure.
	// playedTiles := []runemapping.MachineLetter(nil)
	result := boardPlaintextRegex.FindAllStringSubmatch(qText, -1)
	if len(result) != 15 {
		panic("Wrongly implemented")
	}

	var err error
	var letter runemapping.MachineLetter
	for i := range result {
		// result[i][1] has the string
		j := -1
		for _, ch := range result[i][1] {
			j++
			if j%2 != 0 {
				continue
			}
			letter, err = rm.Val(ch)
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
	// userRacks := userRackRegex.FindAllStringSubmatch(qText, -1)
	// for i := range userRacks {
	// 	if i > 1 { // only the first two lines that match
	// 		break
	// 	}
	// 	rack := userRacks[i][1]
	// 	// rackTiles := []runemapping.MachineLetter{}
	// 	for _, ch := range rack {
	// 		letter, err = rm.Val(ch)
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		// rackTiles = append(rackTiles, letter)
	// 	}
	// }
}
