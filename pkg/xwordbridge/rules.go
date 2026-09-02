package xwordbridge

// Building xwordgame.Rules from the names liwords stores.
//
// A game's configuration reaches us as five strings and an enum -- whatever
// went into the GameRequest, or whatever a GameHistory recorded -- and turning
// those into a Rules value means resolving a board layout, a letter
// distribution and a lexicon, each from a different place. Doing that in one
// helper matters more than it looks: the corpus test and the production load
// path have to agree on every default, and a lexicon that resolves in the test
// but not in production would make the corpus evidence worthless.

import (
	"fmt"

	macondoboard "github.com/domino14/macondo/board"
	macondoconfig "github.com/domino14/macondo/config"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/kwg"
	"github.com/domino14/word-golib/tilemapping"

	"github.com/woogles-io/liwords/pkg/xwordgame"
)

// RulesSpec is a game's configuration as liwords stores it: names, not objects.
//
// Every field may be empty; Defaults fills in what macondo would have used, so
// a history that predates a field still resolves.
type RulesSpec struct {
	Lexicon            string
	BoardLayout        string
	LetterDistribution string
	Variant            string
	ChallengeRule      macondopb.ChallengeRule
}

// Defaults returns the spec with empty names replaced by the ones macondo
// applies when a history leaves them unset.
func (s RulesSpec) Defaults() RulesSpec {
	if s.BoardLayout == "" {
		s.BoardLayout = macondoboard.CrosswordGameLayout
	}
	if s.LetterDistribution == "" {
		s.LetterDistribution = "english"
	}
	if s.Variant == "" {
		s.Variant = "classic"
	}
	return s
}

// SpecFromHistory reads a spec off a GameHistory.
func SpecFromHistory(hist *macondopb.GameHistory) RulesSpec {
	return RulesSpec{
		Lexicon:            hist.Lexicon,
		BoardLayout:        hist.BoardLayout,
		LetterDistribution: hist.LetterDistribution,
		Variant:            hist.Variant,
		ChallengeRule:      hist.ChallengeRule,
	}.Defaults()
}

// RulesFor resolves a spec into a Rules value.
//
// TrustRecordedPlays is deliberately not settable here. ReplayHistory turns it
// on for itself, because it is a property of replaying a log rather than of the
// game's configuration, and a second way to set it would eventually be set the
// wrong way for live play.
//
// A lexicon that fails to load is not an error. Most of what a replay does --
// placing tiles, scoring, tracking the bag -- needs no word list at all, and a
// game played under a lexicon this server no longer ships should still load.
// The rules that do need one degrade honestly: an adjudication without a
// lexicon reports that it could not decide, rather than guessing.
func RulesFor(cfg *macondoconfig.Config, spec RulesSpec) (*xwordgame.Rules, error) {
	spec = spec.Defaults()

	layout, err := xwordgame.NamedLayout(spec.BoardLayout)
	if err != nil {
		return nil, fmt.Errorf("xwordbridge: board layout %q: %w", spec.BoardLayout, err)
	}
	ld, err := tilemapping.GetDistribution(cfg.WGLConfig(), spec.LetterDistribution)
	if err != nil {
		return nil, fmt.Errorf("xwordbridge: letter distribution %q: %w", spec.LetterDistribution, err)
	}
	cr, err := ChallengeRuleFromMacondo(spec.ChallengeRule)
	if err != nil {
		return nil, err
	}

	r := &xwordgame.Rules{
		Layout:             layout,
		LetterDistribution: ld,
		Variant:            xwordgame.Variant(spec.Variant),
		ChallengeRule:      cr,
		ExchangeLimit:      xwordgame.ExchangeLimitForLexicon(spec.Lexicon),
	}
	if spec.Lexicon != "" {
		k, err := kwg.GetKWG(cfg.WGLConfig(), spec.Lexicon,
			kwg.WithDistribution(spec.LetterDistribution))
		if err == nil {
			r.Lexicon = kwg.Lexicon{KWG: *k}
		}
	}
	return r, nil
}
