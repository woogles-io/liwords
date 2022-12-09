package cwgame

type Variant string

const (
	VarClassic  Variant = "classic"
	VarWordSmog         = "wordsmog"
	// Redundant information, but we are deciding to treat different board
	// layouts as different variants.
	VarClassicSuper  = "classic_super"
	VarWordSmogSuper = "wordsmog_super"
)

type GameRules struct {
	boardLayout      string
	lexicon          string
	distname         string
	variant          Variant
	secondsPerPlayer []int
	maxOvertimeMins  int
	incrementSeconds int
}

func NewBasicGameRules(lexicon, boardLayout, letterDist string, variant Variant,
	seconds []int, overtimeMins int, increment int) *GameRules {
	return &GameRules{
		boardLayout:      boardLayout,
		lexicon:          lexicon,
		distname:         letterDist,
		variant:          variant,
		secondsPerPlayer: seconds,
		maxOvertimeMins:  overtimeMins,
		incrementSeconds: increment,
	}
}
