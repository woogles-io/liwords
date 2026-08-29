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
	"strings"

	"github.com/domino14/word-golib/tilemapping"
)

// DefaultExchangeLimit is how many tiles must remain in the bag before an
// exchange is allowed. Seven is standard; macondo calls it the same thing.
const DefaultExchangeLimit = 7

// SpanishExchangeLimit is the minimum under FISE rules, which allow an exchange
// while any tile at all remains in the bag.
const SpanishExchangeLimit = 1

// ExchangeLimitForLexicon returns the minimum bag size an exchange requires
// under the rules that go with a lexicon.
//
// Spanish is the exception: FISE permits an exchange with a single tile left,
// where every other ruleset we support requires a full rack. The corpus has a
// 2025 FILE2017 game that exchanged with four tiles in the bag, which is
// perfectly legal and which a flat limit of seven rejects.
//
// Sniffing the lexicon name is how macondo decides this too (lexicon.IsSpanish,
// commented there as "a little bit ghetto"). It is kept as an explicit helper
// rather than buried in Rules so that a caller can see the decision being made
// and override it.
func ExchangeLimitForLexicon(lexiconName string) int {
	l := strings.ToLower(lexiconName)
	if strings.HasPrefix(l, "fise") || strings.HasPrefix(l, "file") {
		return SpanishExchangeLimit
	}
	return DefaultExchangeLimit
}

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
	// for an exchange to be legal. Zero means DefaultExchangeLimit; Spanish
	// games need SpanishExchangeLimit. See ExchangeLimitForLexicon.
	ExchangeLimit int
	// FivePointMode selects flat or per-word scoring under the five-point
	// challenge rule, and is ignored by every other rule.
	FivePointMode FivePointMode

	// TrustRecordedPlays says the caller is replaying moves a referee already
	// accepted, so the word check under the void rule should be skipped.
	//
	// Woogles games are played under the lexicon of their day, and lexicons
	// change: the corpus has a 2021 ECWL game whose SEZ is not in today's ECWL.
	// Re-deciding it now would reject a play that really happened.
	//
	// It covers the lexicon and nothing else. Everything that depends on the
	// position rather than on a word list -- geometry, rack contents, the
	// exchange minimum, every state transition -- is still checked, because
	// those are what a replay exists to verify. In particular the exchange
	// minimum is a rule to be modelled, not a check to skip; see
	// ExchangeLimitForLexicon.
	//
	// Never set this for live play.
	TrustRecordedPlays bool
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
