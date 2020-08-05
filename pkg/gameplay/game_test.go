package gameplay

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/jinzhu/gorm"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/user"
	pkguser "github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")
var TestingDBConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"

func userStore(dbURL string) pkguser.Store {
	ustore, err := user.NewDBStore(TestingDBConnStr + " dbname=liwords_test")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return ustore
}

func recreateDB() {
	// Create a database.
	db, err := gorm.Open("postgres", TestingDBConnStr+" dbname=postgres")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	defer db.Close()
	db = db.Exec("DROP DATABASE IF EXISTS liwords_test")
	if db.Error != nil {
		log.Fatal().Err(db.Error).Msg("error")
	}
	db = db.Exec("CREATE DATABASE liwords_test")
	if db.Error != nil {
		log.Fatal().Err(db.Error).Msg("error")
	}
	// Create a user table. Initialize the user store.
	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")

	// Insert a couple of users into the table.

	for _, u := range []*entity.User{
		{Username: "cesar4", Email: "cesar4@woogles.io", UUID: "xjCWug7EZtDxDHX5fRZTLo"},
		{Username: "Mina", Email: "mina@gmail.com", UUID: "qUQkST8CendYA3baHNoPjk"},
		{Username: "jesse", Email: "jesse@woogles.io", UUID: "3xpEkpRAy3AizbVmDg3kdi"},
	} {
		err = ustore.New(context.Background(), u)
		if err != nil {
			log.Fatal().Err(err).Msg("error")
		}
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func variantKey(req *pb.GameRequest) entity.VariantKey {
	timefmt, variant, err := entity.VariantFromGameReq(req)
	if err != nil {
		panic(err)
	}
	return entity.ToVariantKey(req.Lexicon, variant, timefmt)
}

func TestComputeGameStats(t *testing.T) {
	is := is.New(t)
	recreateDB()
	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")

	histjson, err := ioutil.ReadFile("./testdata/game1/history.json")
	is.NoErr(err)
	hist := &macondopb.GameHistory{}
	err = json.Unmarshal(histjson, hist)
	is.NoErr(err)

	reqjson, err := ioutil.ReadFile("./testdata/game1/game_request.json")
	is.NoErr(err)
	req := &pb.GameRequest{}
	err = json.Unmarshal(reqjson, req)
	is.NoErr(err)

	stats, err := computeGameStats(context.Background(), hist, variantKey(req), ustore)
	is.NoErr(err)
	is.Equal(stats.PlayerOneData[1].Name, "Bingos")
	is.Equal(stats.PlayerOneData[1].List, []*entity.ListItem{
		{Word: "PARDINE", Score: 76, Probability: 1, GameId: "m5ktbp4qPVTqaAhg6HJMsb"},
		{Word: "HETAERA", Score: 91, Probability: 1, GameId: "m5ktbp4qPVTqaAhg6HJMsb"}})
}

func TestComputePlayerStats(t *testing.T) {
	is := is.New(t)
	recreateDB()
	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")

	histjson, err := ioutil.ReadFile("./testdata/game1/history.json")
	is.NoErr(err)
	hist := &macondopb.GameHistory{}
	err = json.Unmarshal(histjson, hist)
	is.NoErr(err)

	reqjson, err := ioutil.ReadFile("./testdata/game1/game_request.json")
	is.NoErr(err)
	req := &pb.GameRequest{}
	err = json.Unmarshal(reqjson, req)
	is.NoErr(err)

	_, err = computeGameStats(context.Background(), hist, variantKey(req), ustore)
	is.NoErr(err)

	p0id, p1id := hist.Players[0].UserId, hist.Players[1].UserId

	u0, err := ustore.GetByUUID(context.Background(), p0id)
	is.NoErr(err)
	u1, err := ustore.GetByUUID(context.Background(), p1id)
	is.NoErr(err)

	stats0, ok := u0.Profile.Stats.Data["CSW19.classic.ultrablitz"]
	is.True(ok)
	is.Equal(stats0.PlayerOneData[1].Name, "Bingos")
	is.Equal(stats0.PlayerOneData[1].List, []*entity.ListItem{
		{Word: "PARDINE", Score: 76, Probability: 1, GameId: "m5ktbp4qPVTqaAhg6HJMsb"},
		{Word: "HETAERA", Score: 91, Probability: 1, GameId: "m5ktbp4qPVTqaAhg6HJMsb"}})

	is.Equal(stats0.PlayerOneData[27].Name, "Wins")
	is.Equal(stats0.PlayerOneData[27].Total, 1)

	stats1, ok := u1.Profile.Stats.Data["CSW19.classic.ultrablitz"]
	is.True(ok)
	is.Equal(stats1.PlayerOneData[27].Total, 0)
}

func TestComputePlayerStatsMultipleGames(t *testing.T) {
	is := is.New(t)
	recreateDB()
	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")

	for _, g := range []string{"game1", "game2"} {
		histjson, err := ioutil.ReadFile("./testdata/" + g + "/history.json")
		is.NoErr(err)
		hist := &macondopb.GameHistory{}
		err = json.Unmarshal(histjson, hist)
		is.NoErr(err)

		reqjson, err := ioutil.ReadFile("./testdata/" + g + "/game_request.json")
		is.NoErr(err)
		req := &pb.GameRequest{}
		err = json.Unmarshal(reqjson, req)
		is.NoErr(err)

		_, err = computeGameStats(context.Background(), hist, variantKey(req), ustore)
		is.NoErr(err)
	}

	u0, err := ustore.Get(context.Background(), "Mina")
	is.NoErr(err)
	u1, err := ustore.Get(context.Background(), "cesar4")
	is.NoErr(err)

	stats0, ok := u0.Profile.Stats.Data["CSW19.classic.ultrablitz"]
	is.True(ok)
	is.Equal(stats0.PlayerOneData[27].Name, "Wins")
	is.Equal(stats0.PlayerOneData[27].Total, 1)

	is.Equal(stats0.PlayerOneData[1].Name, "Bingos")
	is.Equal(stats0.PlayerOneData[1].List, []*entity.ListItem{
		{Word: "PARDINE", Score: 76, Probability: 1, GameId: "m5ktbp4qPVTqaAhg6HJMsb"},
		{Word: "HETAERA", Score: 91, Probability: 1, GameId: "m5ktbp4qPVTqaAhg6HJMsb"},
		{Word: "UNITERS", Score: 68, Probability: 1, GameId: "ycj5de5gArFF3ap76JyiUA"},
	})

	stats1, ok := u1.Profile.Stats.Data["CSW19.classic.ultrablitz"]
	is.True(ok)
	is.Equal(stats1.PlayerOneData[1].List, []*entity.ListItem{
		{Word: "STYMING", Score: 70, Probability: 1, GameId: "ycj5de5gArFF3ap76JyiUA"},
	})
	is.Equal(stats1.PlayerOneData[27].Total, 1)

}
