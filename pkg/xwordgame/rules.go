// A game's immutable configuration.
//
// Rules is deliberately not part of State. A snapshot holds the position and
// nothing else; the configuration lives beside it as plain columns on
// ongoing_games and is resolved once at load time into shared, cached objects.
// A BoardLayout and a LetterDistribution are read-only and process-wide, so a
// Rules value is safe to share across goroutines and games.

package xwordgame

import (
	"fmt"

	"github.com/domino14/word-golib/tilemapping"
)

// DefaultExchangeLimit is how many tiles must remain in the bag before an
// exchange is allowed. Seven is standard; macondo calls it the same thing.
const DefaultExchangeLimit = 7

// Rules is everything about a game that does not change while it is played.
type Rules struct {
	// Layout is the board's premium-square arrangement. Required.
	Layout *BoardLayout
	// LetterDistribution supplies tile values and counts. Required.
	LetterDistribution *tilemapping.LetterDistribution
	// Lexicon validates words. Required to adjudicate a challenge, and to
	// apply a tile play under the void rule; optional otherwise.
	Lexicon Lexicon
	// Variant selects exact-spelling or anagram matching.
	Variant Variant
	// ChallengeRule is how the game settles challenges.
	ChallengeRule ChallengeRule
	// ExchangeLimit is the minimum number of tiles that must remain in the bag
	// for an exchange to be legal. Zero means DefaultExchangeLimit.
	ExchangeLimit int
	// FivePointMode selects flat or per-word scoring under the five-point
	// challenge rule, and is ignored by every other rule.
	FivePointMode FivePointMode
}

// exchangeLimit resolves the zero value to the default.
func (r *Rules) exchangeLimit() int {
	if r.ExchangeLimit == 0 {
		return DefaultExchangeLimit
	}
	return r.ExchangeLimit
}

// validate checks that the rules carry what any move needs.
func (r *Rules) validate() error {
	if r == nil {
		return fmt.Errorf("xwordgame: rules are required")
	}
	if r.Layout == nil {
		return fmt.Errorf("xwordgame: rules need a board layout")
	}
	if r.LetterDistribution == nil {
		return fmt.Errorf("xwordgame: rules need a letter distribution")
	}
	if r.ExchangeLimit < 0 {
		return fmt.Errorf("xwordgame: exchange limit %d is negative", r.ExchangeLimit)
	}
	return nil
}
