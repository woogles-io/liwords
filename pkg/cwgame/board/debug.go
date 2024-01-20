package board

import (
	"fmt"
	"os"
	"strings"

	"github.com/domino14/word-golib/tilemapping"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

var (
	ColorSupport = os.Getenv("LIWORDS_DISABLE_COLOR") != "on"
)

func sqDisplayStr(board *ipc.GameBoard, layout *BoardLayout, i, j int, rm *tilemapping.TileMapping) string {
	idx := i*int(board.NumCols) + j
	if board.Tiles[idx] != 0 {
		return string(rm.Letter(tilemapping.MachineLetter(board.Tiles[idx])))
	}
	repr := string(enumToBonus(BonusSquare(layout.Layout[idx])))
	if !ColorSupport {
		return repr
	}
	switch BonusSquare(layout.Layout[idx]) {
	case Bonus4WS:
		return fmt.Sprintf("\033[33m%s\033[0m", repr)
	case Bonus3WS:
		return fmt.Sprintf("\033[31m%s\033[0m", repr)
	case Bonus2WS:
		return fmt.Sprintf("\033[35m%s\033[0m", repr)
	case Bonus4LS:
		return fmt.Sprintf("\033[95m%s\033[0m", repr)
	case Bonus3LS:
		return fmt.Sprintf("\033[34m%s\033[0m", repr)
	case Bonus2LS:
		return fmt.Sprintf("\033[36m%s\033[0m", repr)
	case EmptySpace:
		return " "
	default:
		return "?"
	}
}

func ToUserVisibleString(board *ipc.GameBoard, layoutName string, rm *tilemapping.TileMapping) (string, error) {
	layout, err := GetBoardLayout(layoutName)
	if err != nil {
		return "", err
	}

	var str strings.Builder
	paddingStr := "   "
	row := paddingStr
	for i := 0; i < int(board.NumCols); i++ {
		row = row + fmt.Sprintf("%c", 'A'+i) + " "
	}
	str.WriteString(row + "\n")
	str.WriteString(paddingStr + strings.Repeat("-", int(board.NumCols)*2) + "\n")
	for i := 0; i < int(board.NumRows); i++ {
		row := fmt.Sprintf("%2d|", i+1)
		for j := 0; j < int(board.NumCols); j++ {
			row = row + sqDisplayStr(board, layout, i, j, rm) + " "
		}
		row += "|"
		str.WriteString(row)
		str.WriteString("\n")
	}
	str.WriteString(paddingStr + strings.Repeat("-", int(board.NumCols)*2) + "\n")
	str.WriteString("\n")
	return str.String(), nil
}
