package auth

import (
	"errors"
	"testing"

	"github.com/matryer/is"
)

// Small settings keep the tests fast; the format is what matters here.
var testConfig = NewPasswordConfig(1, 1024, 1, 32)

func TestComparePassword(t *testing.T) {
	is := is.New(t)
	hash, err := GeneratePassword(testConfig, "correct horse")
	is.NoErr(err)

	ok, err := ComparePassword("correct horse", hash)
	is.NoErr(err)
	is.True(ok)

	ok, err = ComparePassword("wrong horse", hash)
	is.NoErr(err)
	is.True(!ok)
}

func TestNeedsRehash(t *testing.T) {
	is := is.New(t)
	hash, err := GeneratePassword(NewPasswordConfig(1, 1024, 4, 32), "correct horse")
	is.NoErr(err)

	is.True(NeedsRehash(testConfig, hash))
	is.True(!NeedsRehash(NewPasswordConfig(1, 1024, 4, 32), hash))
	is.True(!NeedsRehash(testConfig, "not a hash"))
}

// A check queued behind others for too long gives up instead of waiting.
func TestHasherBusy(t *testing.T) {
	is := is.New(t)
	hash, err := GeneratePassword(testConfig, "correct horse")
	is.NoErr(err)

	hasher <- struct{}{}
	_, err = ComparePassword("correct horse", hash)
	<-hasher
	is.True(errors.Is(err, ErrHasherBusy))

	ok, err := ComparePassword("correct horse", hash)
	is.NoErr(err)
	is.True(ok)
}
