package board

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/domino14/word-golib/cache"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type BonusSquare byte

type BoardLayout struct {
	Rows   int
	Cols   int
	Layout []byte
}

const (
	EmptySpace BonusSquare = 0
	Bonus2LS   BonusSquare = 2
	Bonus3LS   BonusSquare = 3
	Bonus4LS   BonusSquare = 4
	Bonus2WS   BonusSquare = 12
	Bonus3WS   BonusSquare = 13
	Bonus4WS   BonusSquare = 14
)

var CacheKeyPrefix = "boardlayout:"

// CacheLoadFunc is the function that loads an object into the global cache.
func CacheLoadFunc(cfg map[string]any, key string) (interface{}, error) {
	dist := strings.TrimPrefix(key, CacheKeyPrefix)
	return NamedBoardLayout(dist)
}

func bonusToEnum(c rune) BonusSquare {
	switch c {
	case '~':
		return Bonus4WS
	case '=':
		return Bonus3WS
	case '-':
		return Bonus2WS
	case '^':
		return Bonus4LS
	case '"':
		return Bonus3LS
	case '\'':
		return Bonus2LS
	case ' ':
		return EmptySpace
	default:
		panic("unrecognized char")
	}
}

func enumToBonus(b BonusSquare) rune {
	switch b {
	case Bonus4WS:
		return '~'
	case Bonus3WS:
		return '='
	case Bonus2WS:
		return '-'
	case Bonus4LS:
		return '^'
	case Bonus3LS:
		return '"'
	case Bonus2LS:
		return '\''
	case EmptySpace:
		return ' '
	default:
		panic("unrecognized bonus")
	}
}

func NewBoard(bl *BoardLayout) *ipc.GameBoard {
	dim := bl.Rows * bl.Cols

	return &ipc.GameBoard{
		NumRows: int32(bl.Rows),
		NumCols: int32(bl.Cols),
		Tiles:   make([]byte, dim),
		IsEmpty: true,
	}
}

func NamedBoardLayout(name string) (*BoardLayout, error) {
	var w bytes.Buffer
	var board []string
	switch name {
	case "", CrosswordGameLayout:
		board = CrosswordGameBoard

	case SuperCrosswordGameLayout:
		board = SuperCrosswordGameBoard
	default:
		return nil, errors.New("layout not supported")
	}

	for _, row := range board {
		for _, c := range row {
			err := w.WriteByte(byte(bonusToEnum(c)))
			if err != nil {
				return nil, err
			}
		}
	}
	return &BoardLayout{
		Rows:   len(board),
		Cols:   len(board[0]),
		Layout: w.Bytes(),
	}, nil
}

func GetBoardLayout(name string) (*BoardLayout, error) {
	key := CacheKeyPrefix + name
	obj, err := cache.Load(nil, key, CacheLoadFunc)
	if err != nil {
		return nil, err
	}
	ret, ok := obj.(*BoardLayout)
	if !ok {
		return nil, errors.New("could not read board from name")
	}
	return ret, nil
}

func GetLetter(board *ipc.GameBoard, row, col int) tilemapping.MachineLetter {
	idx := row*int(board.NumCols) + col
	return tilemapping.MachineLetter(board.Tiles[idx])
}

func PosExists(board *ipc.GameBoard, row, col int) bool {
	return row >= 0 && row < int(board.NumRows) && col >= 0 && col < int(board.NumCols)
}

func HasLetter(board *ipc.GameBoard, row, col int) bool {
	return GetLetter(board, row, col) != 0
}

func ErrorIfIllegalPlay(board *ipc.GameBoard, row, col int, vertical bool,
	word []tilemapping.MachineLetter) error {

	ri, ci := 0, 1
	if vertical {
		ri, ci = ci, ri
	}
	touchesCenterSquare := false
	bordersATile := false
	placedATile := false

	for idx, ml := range word {
		newrow, newcol := row+(ri*idx), col+(ci*idx)

		if board.IsEmpty && newrow == int(board.NumRows)>>1 && newcol == int(board.NumCols)>>1 {
			touchesCenterSquare = true
		}

		if newrow < 0 || newrow >= int(board.NumRows) || newcol < 0 || newcol >= int(board.NumCols) {
			return errors.New("play extends off of the board")
		}

		// 0 is played-through (or blank, but that shouldn't happen here)
		if ml == 0 {
			ml = GetLetter(board, newrow, newcol)
			if ml == 0 {
				return errors.New("a played-through marker was specified, but " +
					"there is no tile at the given location")
			}
			bordersATile = true
		} else {
			ml = GetLetter(board, newrow, newcol)
			if ml != 0 {
				return fmt.Errorf("tried to play through a letter already on "+
					"the board; please use the played-through marker (.) instead "+
					"(row %v col %v ml %v)", newrow, newcol, ml)
			}

			// We are placing a tile on this empty square. Check if we border
			// any other tiles.

			for d := -1; d <= 1; d += 2 {
				// only check perpendicular hooks
				checkrow, checkcol := newrow+ci*d, newcol+ri*d
				if PosExists(board, checkrow, checkcol) && GetLetter(board, checkrow, checkcol) != 0 {
					bordersATile = true
				}
			}

			placedATile = true
		}
	}

	if board.IsEmpty && !touchesCenterSquare {
		return errors.New("the first play must touch the center square")
	}
	if !board.IsEmpty && !bordersATile {
		return errors.New("your play must border a tile already on the board")
	}
	if !placedATile {
		return errors.New("your play must place a new tile")
	}
	if len(word) < 2 {
		return errors.New("your play must include at least two letters")
	}
	{
		checkrow, checkcol := row-ri, col-ci
		if PosExists(board, checkrow, checkcol) && GetLetter(board, checkrow, checkcol) != 0 {
			return errors.New("your play must include the whole word")
		}
	}
	{
		checkrow, checkcol := row+ri*len(word), col+ci*len(word)
		if PosExists(board, checkrow, checkcol) && GetLetter(board, checkrow, checkcol) != 0 {
			return errors.New("your play must include the whole word")
		}
	}
	return nil
}

// FormedWords returns an array of all machine words formed by this move.
// The move is assumed to be of type Play
func FormedWords(board *ipc.GameBoard, row, col int, vertical bool, mls []tilemapping.MachineLetter) ([]tilemapping.MachineWord, error) {
	// Reserve space for main word.
	words := []tilemapping.MachineWord{nil}
	mainWord := []tilemapping.MachineLetter{}

	ri, ci := 0, 1
	if vertical {
		ri, ci = ci, ri
	}

	if len(mls) == 0 {
		return nil, errors.New("function must be called with a tile placement play")
	}

	for idx, letter := range mls {
		// For the purpose of checking words, all letters should be unblanked.
		letter = letter.Unblank()
		newrow, newcol := row+(ri*idx), col+(ci*idx)

		// This is the main word.
		if letter == 0 {
			letter = GetLetter(board, newrow, newcol).Unblank()
			mainWord = append(mainWord, letter)
			continue
		}
		mainWord = append(mainWord, letter)
		crossWord := formedCrossWord(board, !vertical, letter, newrow, newcol)
		if crossWord != nil {
			words = append(words, crossWord)
		}
	}
	// Prepend the main word to the slice. We do this to establish a convention
	// that this slice always contains the main formed word first.
	// Space for this is already reserved upfront to avoid unnecessary copying.
	words[0] = mainWord

	return words, nil
}

func formedCrossWord(board *ipc.GameBoard, crossVertical bool, letter tilemapping.MachineLetter,
	row, col int) tilemapping.MachineWord {

	ri, ci := 0, 1
	if crossVertical {
		ri, ci = ci, ri
	}

	// Given the cross-word direction (crossVertical) and a letter located at row, col
	// find the cross-word that contains this letter (if any)
	// Look in the cross direction for newly played tiles.
	crossword := []tilemapping.MachineLetter{}

	newrow := row - ri
	newcol := col - ci
	// top/left and bottom/right row/column pairs.
	var tlr, tlc, brr, brc int

	// Find the top or left edge.
	for PosExists(board, newrow, newcol) && HasLetter(board, newrow, newcol) {
		newrow -= ri
		newcol -= ci
	}
	newrow += ri
	newcol += ci
	tlr = newrow
	tlc = newcol

	// Find bottom or right edge
	newrow, newcol = row, col
	newrow += ri
	newcol += ci
	for PosExists(board, newrow, newcol) && HasLetter(board, newrow, newcol) {
		newrow += ri
		newcol += ci
	}
	newrow -= ri
	newcol -= ci
	// what a ghetto function, sorry future me
	brr = newrow
	brc = newcol

	for rowiter, coliter := tlr, tlc; rowiter <= brr && coliter <= brc; rowiter, coliter = rowiter+ri, coliter+ci {
		if rowiter == row && coliter == col {
			crossword = append(crossword, letter.Unblank())
		} else {
			crossword = append(crossword, GetLetter(board, rowiter, coliter).Unblank())
		}
	}
	if len(crossword) < 2 {
		// there are no 1-letter words, Josh >:(
		return nil
	}
	return crossword
}

// PlayMove plays the move on the board and returns the score of the move.
func PlayMove(board *ipc.GameBoard, layoutName string, dist *tilemapping.LetterDistribution,
	mls []tilemapping.MachineLetter, row, col int, vertical bool) (int32, error) {

	layout, err := GetBoardLayout(layoutName)
	if err != nil {
		return 0, err
	}

	score := placeMoveTiles(board, layout, dist, mls, row, col, vertical)
	return score, nil

}

func placeMoveTiles(board *ipc.GameBoard, layout *BoardLayout, dist *tilemapping.LetterDistribution,
	mls []tilemapping.MachineLetter, row, col int, vertical bool) int32 {

	ri, ci := 0, 1
	// The cross direction is opposite the play direction.
	csDirection := ipc.GameEvent_VERTICAL
	if vertical {
		ri, ci = ci, ri
		csDirection = ipc.GameEvent_HORIZONTAL
	}

	tilesUsed := 0
	bingoBonus := 0
	mainWordScore := 0
	wordMultiplier := 1
	crossScores := 0

	for idx, tile := range mls {
		freshTile := false
		newrow, newcol := row+(ri*idx), col+(ci*idx)

		sqIdx := (newrow * int(board.NumCols)) + newcol
		letterMultiplier := 1
		thisWordMultiplier := 1
		if tile == 0 {
			// a play-through marker, hopefully.
			tile = tilemapping.MachineLetter(board.Tiles[sqIdx])
		} else {
			freshTile = true
			tilesUsed++
			board.Tiles[sqIdx] = byte(tile)

			bonusSq := BonusSquare(layout.Layout[sqIdx])
			switch bonusSq {
			case Bonus4WS:
				wordMultiplier *= 4
				thisWordMultiplier = 4
			case Bonus3WS:
				wordMultiplier *= 3
				thisWordMultiplier = 3
			case Bonus2WS:
				wordMultiplier *= 2
				thisWordMultiplier = 2
			case Bonus2LS:
				letterMultiplier = 2
			case Bonus3LS:
				letterMultiplier = 3
			case Bonus4LS:
				letterMultiplier = 4
			}
		}
		// else all the multipliers are 1.
		if freshTile && board.IsEmpty {
			board.IsEmpty = false
		}
		cs := getCrossScore(board, dist, newrow, newcol, csDirection)
		ls := 0
		if tile > 0 {
			// Count the score of a tile, even if it's a played-through one
			// see `if tile == 0` case above.
			ls = dist.Score(tile)
		}
		mainWordScore += ls * letterMultiplier
		// We only add cross scores if we are making an "across" word).
		actualCrossWord := false
		if !vertical {
			actualCrossWord = (newrow > 0 && HasLetter(board, newrow-1, newcol)) || (newrow < layout.Rows-1 && HasLetter(board, newrow+1, newcol))
		} else {
			// fmt.Println("col", col, "row", row, "idx", idx, "layout", layout.Rows, layout.Cols)
			actualCrossWord = (newcol > 0 && HasLetter(board, newrow, newcol-1)) || (newcol < layout.Cols-1 && HasLetter(board, newrow, newcol+1))
		}

		if freshTile && actualCrossWord {
			crossScores += ls*letterMultiplier*thisWordMultiplier + int(cs)*thisWordMultiplier
		}
	}
	if tilesUsed == 7 {
		bingoBonus = 50
	}
	return int32(mainWordScore*wordMultiplier + crossScores + bingoBonus)

}

// UnplaceMoveTiles unplaces the tiles -- that is, removes them from the board
func UnplaceMoveTiles(board *ipc.GameBoard, mls []tilemapping.MachineLetter, row, col int, vertical bool) error {
	ri, ci := 0, 1
	if vertical {
		ri, ci = ci, ri
	}
	for idx, tile := range mls {
		newrow, newcol := row+(ri*idx), col+(ci*idx)

		sqIdx := (newrow * int(board.NumCols)) + newcol

		if tile == 0 {
			// nothing to unplace, tile was already there.
			if board.Tiles[sqIdx] == byte(0) {
				return errors.New("mismatch with played-through marker")
			}
			continue
		} else {
			board.Tiles[sqIdx] = byte(0)
			if newrow == int(board.NumRows)>>1 && newcol == int(board.NumCols)>>1 {
				// If we are unplaying a tile that was in the center square,
				// that is a good proxy for whether the board is now empty.
				board.IsEmpty = true
			}
		}
	}
	return nil
}

func getCrossScore(board *ipc.GameBoard, dist *tilemapping.LetterDistribution,
	row, col int, direction ipc.GameEvent_Direction) int {
	// look both ways along direction from row, col
	ri, ci := 0, 1
	if direction == ipc.GameEvent_VERTICAL {
		ri, ci = ci, ri
	}

	cs := 0
	for rit, cit := row+ri, col+ci; rit < int(board.NumRows) && cit < int(board.NumCols); rit, cit = rit+ri, cit+ci {
		l := GetLetter(board, rit, cit)
		if l == 0 {
			break
		}
		cs += dist.Score(l)
	}
	// look the other way
	for rit, cit := row-ri, col-ci; rit >= 0 && cit >= 0; rit, cit = rit-ri, cit-ci {
		l := GetLetter(board, rit, cit)
		if l == 0 {
			break
		}
		cs += dist.Score(l)
	}
	return cs
}

func ToFEN(board *ipc.GameBoard, dist *tilemapping.LetterDistribution) string {
	var bd strings.Builder
	alph := dist.TileMapping()

	for i := 0; i < int(board.NumRows); i++ {
		var r strings.Builder
		zeroCt := 0
		for j := 0; j < int(board.NumCols); j++ {
			l := GetLetter(board, i, j)
			if l == 0 {
				zeroCt++
				continue
			}
			// Otherwise, it's a letter.
			if zeroCt > 0 {
				r.WriteString(strconv.Itoa(zeroCt))
				zeroCt = 0
			}
			v := l.UserVisible(alph, false)
			rc := utf8.RuneCountInString(v)
			if rc > 1 {
				r.WriteString("[")
			}
			r.WriteString(l.UserVisible(alph, false))
			if rc > 1 {
				r.WriteString("]")
			}
		}
		if zeroCt > 0 {
			r.WriteString(strconv.Itoa(zeroCt))
		}
		bd.WriteString(r.String())
		if i != int(board.NumRows)-1 {
			bd.WriteString("/")
		}
	}
	return bd.String()
}
