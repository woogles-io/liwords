package tiles

import (
	"testing"

	"github.com/domino14/liwords/pkg/cwgame/runemapping"
	"github.com/matryer/is"
)

func TestLetterDistributionScores(t *testing.T) {
	is := is.New(t)
	ld, err := EnglishLetterDistribution(DefaultConfig)
	is.NoErr(err)

	is.Equal(ld.Score(0), 0)
	is.Equal(ld.Score(254), 0)
	is.Equal(ld.Score(25), 4)
	is.Equal(ld.Score(26), 10)
	is.Equal(ld.Score(8), 4)
	is.Equal(ld.Score(1), 1)
}

func TestLetterDistributionWordScore(t *testing.T) {
	is := is.New(t)
	ld, err := EnglishLetterDistribution(DefaultConfig)
	is.NoErr(err)

	word := "CoOKIE"
	mls, err := runemapping.ToMachineLetters(word, ld.RuneMapping())
	is.NoErr(err)
	is.Equal(ld.WordScore(mls), 11)
}
