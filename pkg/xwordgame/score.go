package xwordgame

import (
	"fmt"

	"github.com/domino14/word-golib/tilemapping"
)

// BingoBonus is awarded for using every tile on a full rack.
const BingoBonus = 50

// ScoreMove returns the score of a tile placement.
//
// This is the one part of the port that is a reimplementation rather than a
// transcription, so it is the part the parity harness matters most for.
//
// macondo's board.ScoreWord reads a per-square cache of the score of the
// perpendicular word already on the board (GetCrossScore), maintained by the
// cross_set package. That cache exists so a move generator can score tens of
// thousands of candidate plays per turn without re-walking the board. A referee
// scores one move per turn, and a cache of derived state has to survive a
// reload -- meaning either recomputing it on load (the same work) or persisting
// ~882 values per 21x21 board in the snapshot, tripling its size to carry data
// that is a pure function of the board, and reintroducing state that can go
// stale. So we walk the words instead.
//
// The play is scored against the board as it stands BEFORE the tiles are
// placed; do not call this after PlaceMoveTiles.
func (s *State) ScoreMove(layout *BoardLayout, ld *tilemapping.LetterDistribution, m *Move) (int, error) {
	if m.Type != MoveTypePlay {
		return 0, fmt.Errorf("xwordgame: expected a tile placement, got %s", m.Type)
	}
	if layout.Dim() != int(s.dim) {
		return 0, fmt.Errorf("xwordgame: layout %q is %dx%d but the board is %dx%d",
			layout.Name(), layout.Dim(), layout.Dim(), s.dim, s.dim)
	}
	dRow, dCol := m.direction()

	mainScore := 0
	mainWordMultiplier := 1
	crossScores := 0
	tilesPlayed := 0

	for idx, ml := range m.Tiles {
		row, col := m.Row+dRow*idx, m.Col+dCol*idx
		if !s.PosExists(row, col) {
			return 0, fmt.Errorf("xwordgame: play extends off the board at row %d col %d", row, col)
		}

		if ml == 0 {
			// Played through: the tile is already on the board, and its square's
			// premium was consumed when it was first covered.
			mainScore += ld.Score(s.TileAt(row, col))
			continue
		}

		tilesPlayed++
		bonus := layout.Bonus(row, col)
		letterScore := ld.Score(ml) * bonus.LetterMultiplier()
		wordMultiplier := bonus.WordMultiplier()

		mainScore += letterScore
		mainWordMultiplier *= wordMultiplier

		// The perpendicular word, if this tile creates one. Scored with the same
		// premium square: a fresh tile on a double-word counts double in both
		// directions.
		if cross := s.crossWordScore(ld, m.Vertical, row, col); cross >= 0 {
			crossScores += (cross + letterScore) * wordMultiplier
		}
	}

	score := mainScore*mainWordMultiplier + crossScores
	if tilesPlayed == RackTileLimit {
		score += BingoBonus
	}
	return score, nil
}

// crossWordScore returns the summed value of the tiles already on the board in
// the word perpendicular to the play through (row, col), or -1 if placing a
// tile there forms no cross word.
//
// The freshly placed tile's own contribution is added by the caller, which
// knows its letter multiplier.
func (s *State) crossWordScore(ld *tilemapping.LetterDistribution, playVertical bool, row, col int) int {
	// The cross word runs perpendicular to the play.
	dRow, dCol := 1, 0
	if playVertical {
		dRow, dCol = 0, 1
	}

	score := 0
	found := false

	for _, sign := range [2]int{-1, 1} {
		r, c := row+dRow*sign, col+dCol*sign
		for s.PosExists(r, c) && s.HasLetter(r, c) {
			score += ld.Score(s.TileAt(r, c))
			found = true
			r += dRow * sign
			c += dCol * sign
		}
	}
	if !found {
		return -1
	}
	return score
}

// ScoreWords is a convenience for scoring a play and returning the words it
// forms in one pass, since callers almost always need both: the score to apply
// and the words to keep for a possible challenge.
func (s *State) ScoreWords(layout *BoardLayout, ld *tilemapping.LetterDistribution, m *Move) (int, []tilemapping.MachineWord, error) {
	score, err := s.ScoreMove(layout, ld, m)
	if err != nil {
		return 0, nil, err
	}
	words, err := s.FormedWords(m)
	if err != nil {
		return 0, nil, err
	}
	return score, words, nil
}

// RackScore is the total face value of a set of tiles, used for the end-of-game
// rack adjustments.
func RackScore(ld *tilemapping.LetterDistribution, tiles []tilemapping.MachineLetter) int {
	score := 0
	for _, ml := range tiles {
		score += ld.Score(ml)
	}
	return score
}
