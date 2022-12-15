package runemapping_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/cwgame/dawg"
	"github.com/domino14/liwords/pkg/cwgame/runemapping"

	"github.com/matryer/is"
)

var DataDir = os.Getenv("DATA_PATH")

func TestLoadAlphabetFromDawg(t *testing.T) {
	is := is.New(t)
	dg, err := dawg.GetDawg(&config.Config{DataPath: DataDir}, "NWL20")

	is.NoErr(err)
	alph := dg.GetRuneMapping()

	is.Equal(alph.Letter(0), '?')
	is.Equal(alph.Letter(1), 'A')
	is.Equal(alph.Letter(25), 'Y')
	is.Equal(alph.Letter(255), 'a')
	is.Equal(alph.Letter(230), 'z')

	v, err := alph.Val('F')
	is.NoErr(err)
	is.Equal(v, runemapping.MachineLetter(6))

	v, err = alph.Val('?')
	is.NoErr(err)
	is.Equal(v, runemapping.MachineLetter(0))

	v, err = alph.Val('x')
	is.NoErr(err)
	is.Equal(v, runemapping.MachineLetter(232))

	_, err = alph.Val('é')
	is.Equal(err, fmt.Errorf("letter `é` not found in alphabet"))
}
