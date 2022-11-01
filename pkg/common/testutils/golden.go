package testutils

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/matryer/is"
)

func slurp(filename string) string {
	contents, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return string(contents)
}

func updateGolden(filename string, bts []byte) {
	// write the bts to filename
	os.WriteFile(filename, bts, 0600)
}

func CompareGolden(t *testing.T, goldenFile string, actualRepr []byte, goldenFileUpdate bool) {
	is := is.New(t)
	if goldenFileUpdate {
		updateGolden(goldenFile, actualRepr)
	} else {
		expected := slurp(goldenFile)
		is.Equal(expected, string(actualRepr))
	}
}

func CompareGoldenJson(t *testing.T, goldenFile string, actualRepr []byte, goldenFileUpdate bool) {
	is := is.New(t)
	if goldenFileUpdate {
		updateGolden(goldenFile, actualRepr)
	} else {
		expectedContents, err := os.ReadFile(goldenFile)
		is.NoErr(err)
		var v1, v2 any
		err1 := json.Unmarshal(expectedContents, &v1)
		is.NoErr(err1)
		err2 := json.Unmarshal(actualRepr, &v2)
		is.NoErr(err2)
		is.True(reflect.DeepEqual(v1, v2))
	}
}
