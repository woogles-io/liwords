package entity

import (
	"fmt"

	"github.com/domino14/word-golib/tilemapping"
)

var AllowedNewGameLexica []string

func init() {
	AllowedNewGameLexica = append(AllowedNewGameLexica,
		"CSW24X",
		"CSW24",
		"ECWL",
		"FILE2017",
		"FRA24",
		"NSF25",
		"NSWL23",
		"NWL23",
		"RD29",
		"DISC2",
		"OSPS52",
		"SLV26",
	)
}

// LetterDistributionForLexicon returns the letter distribution that must be
// used with the given lexicon. A lexicon's word graph and its letter
// distribution have to agree on the alphabet: if they don't, consumers walk
// the word graph and index into the (shorter) tile mapping with machine
// letters that don't exist in it. Macondo's move generator does exactly that,
// so a mismatch takes the bot down with an out-of-range panic rather than
// returning an error.
//
// Because the distribution is fully determined by the lexicon, it should be
// derived here rather than taken from a client.
func LetterDistributionForLexicon(lexicon string) (string, error) {
	ld, err := tilemapping.ProbableLetterDistributionName(lexicon)
	if err != nil {
		return "", fmt.Errorf("no letter distribution is known for lexicon %s", lexicon)
	}
	return ld, nil
}

// ValidateLetterDistribution returns an error if the given letter distribution
// cannot be used with the given lexicon. An empty distribution is allowed;
// callers are expected to fill it in with LetterDistributionForLexicon.
func ValidateLetterDistribution(lexicon, letterDistribution string) error {
	if letterDistribution == "" {
		return nil
	}
	expected, err := LetterDistributionForLexicon(lexicon)
	if err != nil {
		return err
	}
	if letterDistribution == expected {
		return nil
	}
	// The super variant uses a bigger bag, but the same alphabet.
	if expected == "english" && letterDistribution == "english_super" {
		return nil
	}
	return fmt.Errorf("letter distribution %s cannot be used with lexicon %s; expected %s",
		letterDistribution, lexicon, expected)
}
