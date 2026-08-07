package xwordgame

import (
	"fmt"

	"github.com/domino14/word-golib/tilemapping"
)

// MoveType is the kind of action a player took. The values mirror macondo's
// move.MoveType ordering so conversion at the event boundary stays a cast.
type MoveType uint8

const (
	MoveTypePlay MoveType = iota
	MoveTypeExchange
	MoveTypePass
	MoveTypeChallenge
	MoveTypePhonyTilesReturned
	MoveTypeChallengeBonus
	MoveTypeUnsuccessfulChallengePass
	MoveTypeEndgameTiles
	MoveTypeLostTileScore
	MoveTypeLostScoreOnTime
	MoveTypeUnset
)

func (m MoveType) String() string {
	switch m {
	case MoveTypePlay:
		return "play"
	case MoveTypeExchange:
		return "exchange"
	case MoveTypePass:
		return "pass"
	case MoveTypeChallenge:
		return "challenge"
	case MoveTypePhonyTilesReturned:
		return "phony-tiles-returned"
	case MoveTypeChallengeBonus:
		return "challenge-bonus"
	case MoveTypeUnsuccessfulChallengePass:
		return "unsuccessful-challenge-pass"
	case MoveTypeEndgameTiles:
		return "endgame-tiles"
	case MoveTypeLostTileScore:
		return "lost-tile-score"
	case MoveTypeLostScoreOnTime:
		return "lost-score-on-time"
	}
	return fmt.Sprintf("MoveType(%d)", uint8(m))
}

// Move is an action a player takes. It is a plain value with no back-reference
// to a game, board or alphabet -- unlike macondo's move.Move, which carries an
// alphabet pointer, a leave, an equity and a cached score for the move
// generator's benefit. A referee evaluates one move against one state; nothing
// here needs to be memoised.
type Move struct {
	Type MoveType

	// Row, Col and Vertical position a tile-placement play. Row and Col are the
	// coordinates of the first tile in Tiles, whether or not that square is
	// freshly covered.
	Row      int
	Col      int
	Vertical bool

	// Tiles is the play's span for MoveTypePlay: one entry per square covered,
	// where the zero value marks a played-through square whose letter is
	// already on the board. Blanks carry their designated letter with the blank
	// bit set.
	//
	// For MoveTypeExchange it is the tiles being returned to the bag.
	Tiles tilemapping.MachineWord
}

// direction returns the per-index row and column deltas for a placement.
func (m *Move) direction() (dRow, dCol int) {
	if m.Vertical {
		return 1, 0
	}
	return 0, 1
}

// TilesPlayed is the number of tiles the play takes off the rack -- that is,
// the span minus any played-through squares.
func (m *Move) TilesPlayed() int {
	n := 0
	for _, ml := range m.Tiles {
		if ml != 0 {
			n++
		}
	}
	return n
}

// PlayedTiles appends the tiles this play removes from the rack to dst.
func (m *Move) PlayedTiles(dst []tilemapping.MachineLetter) []tilemapping.MachineLetter {
	for _, ml := range m.Tiles {
		if ml != 0 {
			dst = append(dst, ml)
		}
	}
	return dst
}

// NewPlacementMove builds a tile-placement play.
func NewPlacementMove(row, col int, vertical bool, tiles tilemapping.MachineWord) *Move {
	return &Move{Type: MoveTypePlay, Row: row, Col: col, Vertical: vertical, Tiles: tiles}
}

// NewPassMove builds a pass.
func NewPassMove() *Move { return &Move{Type: MoveTypePass} }

// NewExchangeMove builds an exchange of the given tiles.
func NewExchangeMove(tiles tilemapping.MachineWord) *Move {
	return &Move{Type: MoveTypeExchange, Tiles: tiles}
}

// NewChallengeMove builds a challenge of the opponent's last play.
func NewChallengeMove() *Move { return &Move{Type: MoveTypeChallenge} }
