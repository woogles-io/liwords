package cwgame

import (
	"testing"

	"github.com/domino14/word-golib/kwg"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/matryer/is"
)

// The CSW15 word graph must not be a relabeled copy of a later edition.
// CSW19 dropped BERGALI and TURPSES and added OK and ZE.
func TestCSW15IsNotALaterEdition(t *testing.T) {
	is := is.New(t)
	gd, err := kwg.GetKWG(DefaultConfig.WGLConfig(), "CSW15")
	is.NoErr(err)
	hasWord := func(w string) bool {
		mw, err := tilemapping.ToMachineWord(w, gd.GetAlphabet())
		is.NoErr(err)
		return kwg.FindMachineWord(gd, mw)
	}
	is.True(hasWord("BERGALI"))
	is.True(hasWord("TURPSES"))
	is.True(!hasWord("OK"))
	is.True(!hasWord("ZE"))
}
