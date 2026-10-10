package xwordgame

import (
	"errors"
	"fmt"

	"github.com/domino14/word-golib/tilemapping"
)

// Board geometry, placement legality and word formation.
//
// Ported from macondo/board/board.go, with two deliberate omissions:
//
//   - No cross-sets, anchors or extension sets. Those exist to make move
//     generation fast and have no role in refereeing a single submitted move.
//   - No Transpose. macondo flips rowMul/colMul so ScoreWord can always iterate
//     horizontally; the functions we actually need (ErrorIfIllegalPlay,
//     FormedWords) already work off direction vectors, and scoring is rewritten
//     to do the same. Dropping it removes a mutable "is the board currently
//     transposed" mode that every caller would otherwise have to reason about.
//
// The squares themselves live on State, so these are State methods.

// PosExists reports whether the coordinates are on the board.
func (s *State) PosExists(row, col int) bool {
	d := int(s.dim)
	return row >= 0 && row < d && col >= 0 && col < d
}

// HasLetter reports whether a tile occupies the square.
func (s *State) HasLetter(row, col int) bool {
	return s.TileAt(row, col) != 0
}

// IsBoardEmpty reports whether no tile has been played yet.
func (s *State) IsBoardEmpty() bool {
	for _, ml := range s.Board() {
		if ml != 0 {
			return false
		}
	}
	return true
}

// TilesOnBoard counts the tiles played so far.
//
// macondo keeps this as an incrementing counter on the board. We scan instead:
// it is 441 bytes at worst, once per turn, and a value derived on demand cannot
// drift from the thing it describes. Same reasoning as not caching cross-scores.
func (s *State) TilesOnBoard() int {
	n := 0
	for _, ml := range s.Board() {
		if ml != 0 {
			n++
		}
	}
	return n
}

// ErrorIfIllegalPlay reports whether a tile placement is a legal Crossword Game
// move. It does NOT check that the words formed are in the lexicon -- that is
// the challenge rule's business.
//
// Ported from macondo board.go:801.
func (s *State) ErrorIfIllegalPlay(layout *BoardLayout, m *Move) error {
	return s.errorIfIllegalPlay(layout, m, false)
}

// errorIfIllegalPlay is ErrorIfIllegalPlay with the option to waive the rules
// that constrain what is allowed rather than where tiles land. See
// Rules.TrustRecordedPlays.
func (s *State) errorIfIllegalPlay(layout *BoardLayout, m *Move, trusted bool) error {
	if m.Type != MoveTypePlay {
		return fmt.Errorf("xwordgame: expected a tile placement, got %s", m.Type)
	}
	if layout.Dim() != int(s.dim) {
		return fmt.Errorf("xwordgame: layout %q is %dx%d but the board is %dx%d",
			layout.Name(), layout.Dim(), layout.Dim(), s.dim, s.dim)
	}
	dRow, dCol := m.direction()

	boardEmpty := s.IsBoardEmpty()
	touchesCenter := false
	bordersATile := false
	placedATile := false

	for idx, ml := range m.Tiles {
		row, col := m.Row+dRow*idx, m.Col+dCol*idx

		if !s.PosExists(row, col) {
			return errors.New("play extends off of the board")
		}
		if boardEmpty && row == layout.StartRow() && col == layout.StartCol() {
			touchesCenter = true
		}

		if ml == 0 {
			// A played-through square: there must actually be a tile here.
			if !s.HasLetter(row, col) {
				return errors.New("a played-through marker was specified, but " +
					"there is no tile at the given location")
			}
			bordersATile = true
			continue
		}

		if existing := s.TileAt(row, col); existing != 0 {
			return fmt.Errorf("tried to play through a letter already on the "+
				"board; please use the played-through marker instead "+
				"(row %d col %d tile %d)", row, col, existing)
		}
		// A fresh tile on an empty square. Does it hook onto anything
		// perpendicular?
		for d := -1; d <= 1; d += 2 {
			// Perpendicular to the play direction.
			checkRow, checkCol := row+dCol*d, col+dRow*d
			if s.PosExists(checkRow, checkCol) && s.HasLetter(checkRow, checkCol) {
				bordersATile = true
			}
		}
		placedATile = true
	}

	if boardEmpty && !touchesCenter {
		return errors.New("the first play must touch the center square")
	}
	if !boardEmpty && !bordersATile {
		return errors.New("your play must border a tile already on the board")
	}
	if !placedATile {
		return errors.New("your play must place a new tile")
	}
	if len(m.Tiles) < 2 && !trusted {
		return errors.New("your play must include at least two letters")
	}
	// The span must cover the whole main word: no tile may abut either end.
	if r, c := m.Row-dRow, m.Col-dCol; s.PosExists(r, c) && s.HasLetter(r, c) {
		return errors.New("your play must include the whole word")
	}
	n := len(m.Tiles)
	if r, c := m.Row+dRow*n, m.Col+dCol*n; s.PosExists(r, c) && s.HasLetter(r, c) {
		return errors.New("your play must include the whole word")
	}
	return nil
}

// FormedWords returns every word the play creates: the main word first, then
// each cross word, in the order the tiles forming them appear.
//
// All letters are returned unblanked, since blankness affects scoring but not
// spelling. Ported from macondo board.go:882.
func (s *State) FormedWords(m *Move) ([]tilemapping.MachineWord, error) {
	if m.Type != MoveTypePlay {
		return nil, fmt.Errorf("xwordgame: expected a tile placement, got %s", m.Type)
	}
	dRow, dCol := m.direction()

	// Slot zero is reserved for the main word so it is always first.
	words := make([]tilemapping.MachineWord, 1, 1+len(m.Tiles))
	mainWord := make(tilemapping.MachineWord, 0, len(m.Tiles))

	for idx, ml := range m.Tiles {
		row, col := m.Row+dRow*idx, m.Col+dCol*idx
		if ml == 0 {
			mainWord = append(mainWord, s.TileAt(row, col).Unblank())
			continue
		}
		mainWord = append(mainWord, ml.Unblank())
		if cross := s.formedCrossWord(!m.Vertical, ml, row, col); cross != nil {
			words = append(words, cross)
		}
	}
	words[0] = mainWord
	return words, nil
}

// formedCrossWord returns the word perpendicular to the play that runs through
// (row, col) once `letter` is placed there, or nil if that would be a lone
// tile. Ported from macondo board.go:922.
func (s *State) formedCrossWord(crossVertical bool, letter tilemapping.MachineLetter, row, col int) tilemapping.MachineWord {
	dRow, dCol := 0, 1
	if crossVertical {
		dRow, dCol = 1, 0
	}

	// Walk back to the top/left edge of the contiguous run.
	startRow, startCol := row-dRow, col-dCol
	for s.PosExists(startRow, startCol) && s.HasLetter(startRow, startCol) {
		startRow -= dRow
		startCol -= dCol
	}
	startRow += dRow
	startCol += dCol

	// Walk forward to the bottom/right edge.
	endRow, endCol := row+dRow, col+dCol
	for s.PosExists(endRow, endCol) && s.HasLetter(endRow, endCol) {
		endRow += dRow
		endCol += dCol
	}
	endRow -= dRow
	endCol -= dCol

	word := make(tilemapping.MachineWord, 0, MaxBoardDim)
	for r, c := startRow, startCol; r <= endRow && c <= endCol; r, c = r+dRow, c+dCol {
		if r == row && c == col {
			word = append(word, letter.Unblank())
		} else {
			word = append(word, s.TileAt(r, c).Unblank())
		}
	}
	if len(word) < 2 {
		// There are no one-letter words.
		return nil
	}
	return word
}

// PlaceMoveTiles writes a play's fresh tiles onto the board. It does not
// validate; call ErrorIfIllegalPlay first.
func (s *State) PlaceMoveTiles(m *Move) {
	dRow, dCol := m.direction()
	for idx, ml := range m.Tiles {
		if ml == 0 {
			continue
		}
		s.SetTileAt(m.Row+dRow*idx, m.Col+dCol*idx, ml)
	}
}

// UnplaceMoveTiles removes a play's fresh tiles from the board, for returning
// phony tiles after a successful challenge.
func (s *State) UnplaceMoveTiles(m *Move) {
	dRow, dCol := m.direction()
	for idx, ml := range m.Tiles {
		if ml == 0 {
			continue
		}
		s.SetTileAt(m.Row+dRow*idx, m.Col+dCol*idx, 0)
	}
}
