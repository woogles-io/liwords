package xwordgame

import (
	"fmt"
	"sync"
)

// BonusSquare is a premium square, encoded as the ASCII character used in the
// textual board layouts. The byte values match macondo's so that layout strings
// and CGP/GCG interchange stay compatible.
type BonusSquare byte

const (
	// NoBonus is a plain square.
	NoBonus BonusSquare = ' '
	// Bonus2LS doubles the letter placed on it.
	Bonus2LS BonusSquare = '\''
	// Bonus3LS triples the letter placed on it.
	Bonus3LS BonusSquare = '"'
	// Bonus4LS quadruples the letter placed on it (super boards only).
	Bonus4LS BonusSquare = '^'
	// Bonus2WS doubles the word played through it.
	Bonus2WS BonusSquare = '-'
	// Bonus3WS triples the word played through it.
	Bonus3WS BonusSquare = '='
	// Bonus4WS quadruples the word played through it (super boards only).
	Bonus4WS BonusSquare = '~'
)

// LetterMultiplier is the factor applied to a tile freshly placed on this
// square.
func (b BonusSquare) LetterMultiplier() int {
	switch b {
	case Bonus2LS:
		return 2
	case Bonus3LS:
		return 3
	case Bonus4LS:
		return 4
	}
	return 1
}

// WordMultiplier is the factor applied to a word played through this square,
// when a tile is freshly placed on it.
func (b BonusSquare) WordMultiplier() int {
	switch b {
	case Bonus2WS:
		return 2
	case Bonus3WS:
		return 3
	case Bonus4WS:
		return 4
	}
	return 1
}

// Layout names, matching macondo and the values stored in
// game_request.rules.board_layout_name.
const (
	CrosswordGameLayout      = "CrosswordGame"
	SuperCrosswordGameLayout = "SuperCrosswordGame"
)

// BoardLayout is the immutable premium-square arrangement for a board. It is
// pure configuration: it holds no game state, so a single instance is shared
// process-wide by every game using that layout.
type BoardLayout struct {
	name    string
	dim     int
	bonuses []BonusSquare
}

// Name returns the layout's canonical name.
func (l *BoardLayout) Name() string { return l.name }

// Dim returns the board's side length.
func (l *BoardLayout) Dim() int { return l.dim }

// Bonus returns the premium square at the given coordinates.
func (l *BoardLayout) Bonus(row, col int) BonusSquare {
	return l.bonuses[row*l.dim+col]
}

// StartRow and StartCol give the centre square, which the first play must
// cover. Both dimensions we support are odd, so this is exact.
func (l *BoardLayout) StartRow() int { return l.dim / 2 }
func (l *BoardLayout) StartCol() int { return l.dim / 2 }

var (
	layoutsOnce sync.Once
	layouts     map[string]*BoardLayout
	layoutErr   error
)

// NamedLayout returns the shared layout with the given name. The returned value
// is immutable and safe for concurrent use.
func NamedLayout(name string) (*BoardLayout, error) {
	layoutsOnce.Do(initLayouts)
	if layoutErr != nil {
		return nil, layoutErr
	}
	l, ok := layouts[name]
	if !ok {
		return nil, fmt.Errorf("xwordgame: unknown board layout %q", name)
	}
	return l, nil
}

func initLayouts() {
	layouts = map[string]*BoardLayout{}
	for name, desc := range map[string][]string{
		CrosswordGameLayout:      crosswordGameBoard,
		SuperCrosswordGameLayout: superCrosswordGameBoard,
	} {
		l, err := parseLayout(name, desc)
		if err != nil {
			layoutErr = err
			return
		}
		layouts[name] = l
	}
}

// parseLayout turns the textual description into a bonus grid, validating that
// it is square, within bounds, and made only of known premium squares. A typo
// in a layout string would otherwise mis-score every game on that board.
func parseLayout(name string, desc []string) (*BoardLayout, error) {
	dim := len(desc)
	if dim < 1 || dim > MaxBoardDim {
		return nil, fmt.Errorf("xwordgame: layout %q has %d rows, want 1..%d", name, dim, MaxBoardDim)
	}
	bonuses := make([]BonusSquare, 0, dim*dim)
	for r, row := range desc {
		if len(row) != dim {
			return nil, fmt.Errorf("xwordgame: layout %q row %d has %d columns, want %d", name, r, len(row), dim)
		}
		for c := range dim {
			b := BonusSquare(row[c])
			switch b {
			case NoBonus, Bonus2LS, Bonus3LS, Bonus4LS, Bonus2WS, Bonus3WS, Bonus4WS:
			default:
				return nil, fmt.Errorf("xwordgame: layout %q has unknown bonus %q at row %d col %d", name, string(row[c]), r, c)
			}
			bonuses = append(bonuses, b)
		}
	}
	return &BoardLayout{name: name, dim: dim, bonuses: bonuses}, nil
}

// Ported verbatim from macondo/board/layouts.go. These must not be
// "cleaned up" -- every character is a premium square.
var crosswordGameBoard = []string{
	`=  '   =   '  =`,
	` -   "   "   - `,
	`  -   ' '   -  `,
	`'  -   '   -  '`,
	`    -     -    `,
	` "   "   "   " `,
	`  '   ' '   '  `,
	`=  '   -   '  =`,
	`  '   ' '   '  `,
	` "   "   "   " `,
	`    -     -    `,
	`'  -   '   -  '`,
	`  -   ' '   -  `,
	` -   "   "   - `,
	`=  '   =   '  =`,
}

var superCrosswordGameBoard = []string{
	`~  '   =  '  =   '  ~`,
	` -  "   -   -   "  - `,
	`  -  ^   - -   ^  -  `,
	`'  =  '   =   '  =  '`,
	` "  -   "   "   -  " `,
	`  ^  -   ' '   -  ^  `,
	`   '  -   '   -  '   `,
	`=      -     -      =`,
	` -  "   "   "   "  - `,
	`  -  '   ' '   '  -  `,
	`'  =  '   -   '  =  '`,
	`  -  '   ' '   '  -  `,
	` -  "   "   "   "  - `,
	`=      -     -      =`,
	`   '  -   '   -  '   `,
	`  ^  -   ' '   -  ^  `,
	` "  -   "   "   -  " `,
	`'  =  '   =   '  =  '`,
	`  -  ^   - -   ^  -  `,
	` -  "   -   -   "  - `,
	`~  '   =  '  =   '  ~`,
}
