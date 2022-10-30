package common

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"
	"google.golang.org/protobuf/encoding/protojson"
)

var goldenFileUpdate bool

func init() {
	flag.BoolVar(&goldenFileUpdate, "update", false, "update golden files")
}

func slurp(filename string) string {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return string(contents)
}

func updateGolden(filename string, bts []byte) {
	// write the bts to filename
	os.WriteFile(filename, bts, 0600)
}

func compareGolden(t *testing.T, goldenFile string, actualRepr []byte) {
	is := is.New(t)
	if goldenFileUpdate {
		updateGolden(goldenFile, actualRepr)
	} else {
		expected := slurp(goldenFile)
		is.Equal(expected, string(actualRepr))
	}
}

func compareGoldenJson(t *testing.T, goldenFile string, actualRepr []byte) {
	is := is.New(t)
	if goldenFileUpdate {
		updateGolden(goldenFile, actualRepr)
	} else {
		expectedContents, err := ioutil.ReadFile(goldenFile)
		is.NoErr(err)
		var v1, v2 any
		err1 := json.Unmarshal(expectedContents, &v1)
		is.NoErr(err1)
		err2 := json.Unmarshal(actualRepr, &v2)
		is.NoErr(err2)
		is.True(reflect.DeepEqual(v1, v2))
	}
}

func TestMigrateHistory(t *testing.T) {
	is := is.New(t)

	testcases := []struct {
		name       string
		file       string
		goldenfile string
	}{
		{"without secondwentfirst", "game_history1.json", "game_history1.golden"},
		{"with secondwentfirst", "game_history2.json", "game_history2.golden"},
	}
	for _, tc := range testcases {
		cts, err := ioutil.ReadFile("./testdata/" + tc.file)
		is.NoErr(err)

		hist := &macondopb.GameHistory{}
		err = protojson.Unmarshal(cts, hist)
		is.NoErr(err)
		hist2, migrated := MigrateGameHistory(hist)
		is.True(migrated)
		marshaller := &protojson.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}
		b2, err := marshaller.Marshal(hist2)
		is.NoErr(err)
		compareGoldenJson(t, "./testdata/"+tc.goldenfile, b2)
	}

}
