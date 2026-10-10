package cwgame

import (
	"testing"

	"github.com/domino14/word-golib/kwg"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

func lexiconHasWord(is *is.I, lexicon string) func(string) bool {
	gd, err := kwg.GetKWG(DefaultConfig.WGLConfig(), lexicon)
	is.NoErr(err)
	return func(w string) bool {
		mw, err := tilemapping.ToMachineWord(w, gd.GetAlphabet())
		is.NoErr(err)
		return kwg.FindMachineWord(gd, mw)
	}
}

// The CSW15 word graph must not be a relabeled copy of a later edition.
// CSW19 dropped BERGALI and TURPSES and added OK and ZE.
func TestCSW15IsNotALaterEdition(t *testing.T) {
	is := is.New(t)
	hasWord := lexiconHasWord(is, "CSW15")
	is.True(hasWord("BERGALI"))
	is.True(hasWord("TURPSES"))
	is.True(!hasWord("OK"))
	is.True(!hasWord("ZE"))
}

// The NSF26 word graph must not be a relabeled copy of NSF25, and it is
// built without the 1-letter and longer-than-15 words the source list has.
func TestNSF26IsNotAnEarlierEdition(t *testing.T) {
	is := is.New(t)
	hasWord := lexiconHasWord(is, "NSF26")
	is.True(hasWord("BIOFORSKER"))
	is.True(hasWord("LAKSESALATEN"))
	is.True(hasWord("KJEMPEVONDE"))
	is.True(!hasWord("SEPTRET"))
	is.True(!hasWord("FESTDOPA"))
	is.True(!hasWord("POSTFIKSA"))
	is.True(!hasWord("Å"))
	is.True(!hasWord("ABONNEMENTSAFTEN"))

	old := lexiconHasWord(is, "NSF25")
	is.True(!old("BIOFORSKER"))
	is.True(old("SEPTRET"))
}
