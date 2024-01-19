package stores

import (
	"flag"
	"os"
	"testing"

	"github.com/domino14/liwords/pkg/common/testutils"
	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/matryer/is"
	"google.golang.org/protobuf/encoding/protojson"
)

var DataDir = os.Getenv("DATA_PATH")
var DefaultConfig = config.DefaultConfig()

var goldenFileUpdate bool

func init() {
	flag.BoolVar(&goldenFileUpdate, "update", false, "update golden files")
}

func TestMigrateEnglish(t *testing.T) {
	is := is.New(t)
	bts, err := os.ReadFile("./testdata/english_game.json")
	is.NoErr(err)

	gdoc := &ipc.GameDocument{}
	err = protojson.Unmarshal(bts, gdoc)
	is.NoErr(err)
	err = MigrateGameDocument(&DefaultConfig, gdoc)
	is.NoErr(err)

	dump, err := protojson.Marshal(gdoc)
	is.NoErr(err)

	testutils.CompareGoldenJson(t, "./testdata/english_game.golden", dump, goldenFileUpdate)

}
