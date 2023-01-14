package board

var (
	// CrosswordGameBoard is a board for a fun Crossword Game, featuring lots
	// of wingos and blonks.
	CrosswordGameBoard []string
	// SuperCrosswordGameBoard is a board for a bigger Crossword game, featuring
	// even more wingos and blonks.
	SuperCrosswordGameBoard []string
)

const (
	CrosswordGameLayout      = "CrosswordGame"
	SuperCrosswordGameLayout = "SuperCrosswordGame"
)

func init() {
	CrosswordGameBoard = []string{
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
	SuperCrosswordGameBoard = []string{
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
}
