package cwgame

import "github.com/domino14/liwords/rpc/api/proto/ipc"

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
	challengeRule    ipc.ChallengeRule
	secondsPerPlayer []int
	maxOvertimeMins  int
	incrementSeconds int
	untimed          bool
}

func NewBasicGameRules(lexicon, boardLayout, letterDist string, challengeRule ipc.ChallengeRule,
	variant Variant, seconds []int, overtimeMins int, increment int, untimed bool) *GameRules {
	return &GameRules{
		boardLayout:      boardLayout,
		lexicon:          lexicon,
		distname:         letterDist,
		challengeRule:    challengeRule,
		variant:          variant,
		secondsPerPlayer: seconds,
		maxOvertimeMins:  overtimeMins,
		incrementSeconds: increment,
		untimed:          untimed,
	}
}
