package board

import (
	"fmt"
	"os"
	"strings"

	"github.com/domino14/word-golib/tilemapping"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/woogles-io/liwords/pkg/cwgame/board"
	"github.com/woogles-io/liwords/pkg/omgwords/game/gamestate"
)

var (
	ColorSupport = os.Getenv("LIWORDS_DISABLE_COLOR") != "on"
)

func sqDisplayStr(bd *gamestate.Board, layout *BoardLayout, i, j int, rm *tilemapping.TileMapping) string {
	idx := i*int(bd.NumCols()) + j
	if bd.Tiles(idx) != 0 {
		return string(rm.Letter(tilemapping.MachineLetter(bd.Tiles(idx))))
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

func ToUserVisibleString(bd *gamestate.Board, layoutName string, rm *tilemapping.TileMapping) (string, error) {
	layout, err := GetBoardLayout(layoutName)
	if err != nil {
		return "", err
	}

	var str strings.Builder
	paddingStr := "   "
	row := paddingStr
	for i := 0; i < int(bd.NumCols()); i++ {
		row = row + fmt.Sprintf("%c", 'A'+i) + " "
	}
	str.WriteString(row + "\n")
	str.WriteString(paddingStr + strings.Repeat("-", int(bd.NumCols())*2) + "\n")
	for i := 0; i < int(bd.NumRows()); i++ {
		row := fmt.Sprintf("%2d|", i+1)
		for j := 0; j < int(bd.NumCols()); j++ {
			row = row + sqDisplayStr(bd, layout, i, j, rm) + " "
		}
		row += "|"
		str.WriteString(row)
		str.WriteString("\n")
	}
	str.WriteString(paddingStr + strings.Repeat("-", int(bd.NumCols())*2) + "\n")
	str.WriteString("\n")
	return str.String(), nil
}

func BuildBoard(builder *flatbuffers.Builder, layoutName string) (flatbuffers.UOffsetT, error) {

	var enVal gamestate.BoardType
	rows, cols := 15, 15
	switch layoutName {
	case board.CrosswordGameLayout, "":
		enVal = gamestate.BoardTypeCrosswordGame
	case board.SuperCrosswordGameLayout:
		enVal = gamestate.BoardTypeSuperCrosswordGame
		rows, cols = 21, 21
	default:
		return 0, fmt.Errorf("unsupported board layout: %s", layoutName)
	}

	boardVector := builder.CreateByteVector(make([]byte, rows*cols))

	gamestate.BoardStart(builder)
	gamestate.BoardAddTiles(builder, boardVector)
	gamestate.BoardAddBoardType(builder, enVal)
	gamestate.BoardAddNumRows(builder, uint8(rows))
	gamestate.BoardAddNumCols(builder, uint8(cols))
	return gamestate.BoardEnd(builder), nil
}
