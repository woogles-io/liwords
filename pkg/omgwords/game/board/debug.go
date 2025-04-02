package board

import (
	"fmt"
	"os"
	"strings"

	"github.com/domino14/word-golib/tilemapping"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/woogles-io/liwords/pkg/omgwords/game/gamestate"
)

var (
	ColorSupport = os.Getenv("LIWORDS_DISABLE_COLOR") != "on"
)

func sqDisplayStr(st *gamestate.GameState, layout *BoardLayout, i, j int, rm *tilemapping.TileMapping) string {
	idx := i*int(st.NumBoardCols()) + j
	if st.Board(idx) != 0 {
		return string(rm.Letter(tilemapping.MachineLetter(st.Board(idx))))
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

func ToUserVisibleString(st *gamestate.GameState, layoutName string, rm *tilemapping.TileMapping) (string, error) {
	layout, err := GetBoardLayout(layoutName)
	if err != nil {
		return "", err
	}

	var str strings.Builder
	paddingStr := "   "
	row := paddingStr
	for i := 0; i < int(st.NumBoardCols()); i++ {
		row = row + fmt.Sprintf("%c", 'A'+i) + " "
	}
	str.WriteString(row + "\n")
	str.WriteString(paddingStr + strings.Repeat("-", int(st.NumBoardCols())*2) + "\n")
	for i := 0; i < int(st.NumBoardRows()); i++ {
		row := fmt.Sprintf("%2d|", i+1)
		for j := 0; j < int(st.NumBoardCols()); j++ {
			row = row + sqDisplayStr(st, layout, i, j, rm) + " "
		}
		row += "|"
		str.WriteString(row)
		str.WriteString("\n")
	}
	str.WriteString(paddingStr + strings.Repeat("-", int(st.NumBoardCols())*2) + "\n")
	str.WriteString("\n")
	return str.String(), nil
}

func cwGameStateWithBoard() *gamestate.GameState {
	builder := flatbuffers.NewBuilder(512)
	boardVector := builder.CreateByteVector(make([]byte, 15*15))

	gamestate.GameStateStart(builder)
	gamestate.GameStateAddBoard(builder, boardVector)
	tt := gamestate.GameStateEnd(builder)
	builder.Finish(tt)

	st := gamestate.GetRootAsGameState(builder.FinishedBytes(), 0)
	return st
}
