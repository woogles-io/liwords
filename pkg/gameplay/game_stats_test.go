package gameplay_test

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
	"github.com/domino14/liwords/pkg/gameplay"
	pkgstats "github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/stores/stats"
	"github.com/domino14/liwords/pkg/stores/user"
	pkguser "github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondoconfig "github.com/domino14/macondo/config"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")
var TestingDBConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"
var gameReq = &pb.GameRequest{Lexicon: "CSW19",
	Rules: &pb.GameRules{BoardLayoutName: entity.CrosswordGame,
		LetterDistributionName: "English",
		VariantName:            "classic"},

	InitialTimeSeconds: 25 * 60,
	IncrementSeconds:   0,
	ChallengeRule:      macondopb.ChallengeRule_FIVE_POINT,
	GameMode:           pb.GameMode_REAL_TIME,
	RatingMode:         pb.RatingMode_RATED,
	RequestId:          "yeet",
	OriginalRequestId:  "originalyeet",
	MaxOvertimeMinutes: 10}

var DefaultConfig = macondoconfig.Config{
	LexiconPath:               os.Getenv("LEXICON_PATH"),
	LetterDistributionPath:    os.Getenv("LETTER_DISTRIBUTION_PATH"),
	DefaultLexicon:            "CSW19",
	DefaultLetterDistribution: "English",
}

// Just dummy info to test that rating stats work
var gameEndedEventObj = &pb.GameEndedEvent{
	Scores: map[string]int32{"cesar4": 1500,
		"Mina": 1800},
	NewRatings: map[string]int32{"cesar4": 1500,
		"Mina": 1800},
	EndReason: pb.GameEndReason_STANDARD,
	Winner:    "cesar4",
	Loser:     "Mina",
	Tie:       false,
}

func userStore(dbURL string) pkguser.Store {
	ustore, err := user.NewDBStore(TestingDBConnStr + " dbname=liwords_test")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return ustore
}

func listStatStore(dbURL string) pkgstats.ListStatStore {
	s, err := stats.NewListStatStore(TestingDBConnStr + " dbname=liwords_test")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return s
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
		panic(db.Error)
	}
	db = db.Exec("CREATE DATABASE liwords_test")
	if db.Error != nil {
		panic(db.Error)
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
	ustore.(*user.DBStore).Disconnect()
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
	lstore := listStatStore(TestingDBConnStr + " dbname=liwords_test")

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

	ctx := context.WithValue(context.Background(), gameplay.ConfigCtxKey("config"), &DefaultConfig)
	s, err := gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, ustore, lstore)
	is.NoErr(err)
	pkgstats.Finalize(s, lstore, []string{"m5ktbp4qPVTqaAhg6HJMsb"},
		"qUQkST8CendYA3baHNoPjk", "xjCWug7EZtDxDHX5fRZTLo",
	)

	is.Equal(s.PlayerOneData[entity.BINGOS_STAT].List, []*entity.ListItem{
		{
			GameId:   "m5ktbp4qPVTqaAhg6HJMsb",
			PlayerId: "qUQkST8CendYA3baHNoPjk",
			Time:     0,
			Item:     entity.ListDatum{Word: "PARDINE", Score: 76, Probability: 1},
		},
		{
			GameId:   "m5ktbp4qPVTqaAhg6HJMsb",
			PlayerId: "qUQkST8CendYA3baHNoPjk",
			Time:     0,
			Item:     entity.ListDatum{Word: "HETAERA", Score: 91, Probability: 1},
		},
	})
	log.Info().Interface("ratings", s.PlayerOneData[entity.RATINGS_STAT].List).Msg("player rating")

	is.Equal(s.PlayerOneData[entity.RATINGS_STAT].List, []*entity.ListItem{
		{
			GameId:   "m5ktbp4qPVTqaAhg6HJMsb",
			PlayerId: "qUQkST8CendYA3baHNoPjk",
			Time:     0,
			Item:     entity.ListDatum{Rating: 1800, Variant: "CSW19.classic.regular"},
		},
	})

	ustore.(*user.DBStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
}

func TestComputeGameStats2(t *testing.T) {
	is := is.New(t)
	recreateDB()
	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")
	lstore := listStatStore(TestingDBConnStr + " dbname=liwords_test")

	histjson, err := ioutil.ReadFile("./testdata/game2/history.json")
	is.NoErr(err)
	hist := &macondopb.GameHistory{}
	err = json.Unmarshal(histjson, hist)
	is.NoErr(err)

	reqjson, err := ioutil.ReadFile("./testdata/game2/game_request.json")
	is.NoErr(err)
	req := &pb.GameRequest{}
	err = json.Unmarshal(reqjson, req)
	is.NoErr(err)

	ctx := context.WithValue(context.Background(), gameplay.ConfigCtxKey("config"), &DefaultConfig)
	s, err := gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, ustore, lstore)
	is.NoErr(err)

	pkgstats.Finalize(s, lstore, []string{"ycj5de5gArFF3ap76JyiUA"},
		"xjCWug7EZtDxDHX5fRZTLo", "qUQkST8CendYA3baHNoPjk",
	)
	log.Info().Interface("bingos", s.PlayerOneData[entity.BINGOS_STAT].List).Msg("player one bingos")

	is.Equal(s.PlayerOneData[entity.BINGOS_STAT].List, []*entity.ListItem{
		{
			GameId:   "ycj5de5gArFF3ap76JyiUA",
			PlayerId: "xjCWug7EZtDxDHX5fRZTLo",
			Time:     0,
			Item:     entity.ListDatum{Word: "STYMING", Score: 70, Probability: 1},
		},
	})

	is.Equal(s.PlayerTwoData[entity.BINGOS_STAT].List, []*entity.ListItem{
		{
			GameId:   "ycj5de5gArFF3ap76JyiUA",
			PlayerId: "qUQkST8CendYA3baHNoPjk",
			Time:     0,
			Item:     entity.ListDatum{Word: "UNITERS", Score: 68, Probability: 1},
		},
	})

	ustore.(*user.DBStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()

}

func TestComputePlayerStats(t *testing.T) {
	is := is.New(t)
	recreateDB()
	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")
	lstore := listStatStore(TestingDBConnStr + " dbname=liwords_test")

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

	ctx := context.WithValue(context.Background(), gameplay.ConfigCtxKey("config"), &DefaultConfig)
	_, err = gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, ustore, lstore)
	is.NoErr(err)

	p0id, p1id := hist.Players[0].UserId, hist.Players[1].UserId

	u0, err := ustore.GetByUUID(context.Background(), p0id)
	is.NoErr(err)
	u1, err := ustore.GetByUUID(context.Background(), p1id)
	is.NoErr(err)

	stats0, ok := u0.Profile.Stats.Data["CSW19.classic.ultrablitz"]
	is.True(ok)

	err = pkgstats.Finalize(stats0, lstore, []string{"m5ktbp4qPVTqaAhg6HJMsb"},
		"qUQkST8CendYA3baHNoPjk", "xjCWug7EZtDxDHX5fRZTLo",
	)
	is.NoErr(err)

	is.Equal(stats0.PlayerOneData[entity.BINGOS_STAT].List, []*entity.ListItem{
		{
			GameId:   "m5ktbp4qPVTqaAhg6HJMsb",
			PlayerId: "qUQkST8CendYA3baHNoPjk",
			Time:     0,
			Item:     entity.ListDatum{Word: "PARDINE", Score: 76, Probability: 1},
		},
		{
			GameId:   "m5ktbp4qPVTqaAhg6HJMsb",
			PlayerId: "qUQkST8CendYA3baHNoPjk",
			Time:     0,
			Item:     entity.ListDatum{Word: "HETAERA", Score: 91, Probability: 1},
		},
	})

	is.Equal(stats0.PlayerOneData[entity.WINS_STAT].Total, 1)

	stats1, ok := u1.Profile.Stats.Data["CSW19.classic.ultrablitz"]
	is.True(ok)
	is.Equal(stats1.PlayerOneData[entity.WINS_STAT].Total, 0)
	ustore.(*user.DBStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
}

func TestComputePlayerStatsMultipleGames(t *testing.T) {
	is := is.New(t)
	recreateDB()
	ustore := userStore(TestingDBConnStr + " dbname=liwords_test")
	lstore := listStatStore(TestingDBConnStr + " dbname=liwords_test")

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

		ctx := context.WithValue(context.Background(), gameplay.ConfigCtxKey("config"), &DefaultConfig)
		_, err = gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, ustore, lstore)
		is.NoErr(err)
	}

	u0, err := ustore.Get(context.Background(), "Mina")
	is.NoErr(err)
	u1, err := ustore.Get(context.Background(), "cesar4")
	is.NoErr(err)

	stats0, ok := u0.Profile.Stats.Data["CSW19.classic.ultrablitz"]
	is.True(ok)
	is.Equal(stats0.PlayerOneData[entity.GAMES_STAT].Total, 2)
	is.Equal(stats0.PlayerOneData[entity.WINS_STAT].Total, 1)

	log.Debug().Interface("li", stats0.PlayerOneData[entity.BINGOS_STAT].List).Msg("--")
	err = pkgstats.Finalize(stats0, lstore, []string{
		// Aggregate across two games
		"m5ktbp4qPVTqaAhg6HJMsb", "ycj5de5gArFF3ap76JyiUA"},
		// Mina then cesar4
		"qUQkST8CendYA3baHNoPjk", "xjCWug7EZtDxDHX5fRZTLo",
	)
	is.NoErr(err)

	is.Equal(stats0.PlayerOneData[entity.BINGOS_STAT].List, []*entity.ListItem{
		{
			GameId:   "m5ktbp4qPVTqaAhg6HJMsb",
			PlayerId: "qUQkST8CendYA3baHNoPjk",
			Time:     0,
			Item:     entity.ListDatum{Word: "PARDINE", Score: 76, Probability: 1},
		},
		{
			GameId:   "m5ktbp4qPVTqaAhg6HJMsb",
			PlayerId: "qUQkST8CendYA3baHNoPjk",
			Time:     0,
			Item:     entity.ListDatum{Word: "HETAERA", Score: 91, Probability: 1},
		},
		{
			GameId:   "ycj5de5gArFF3ap76JyiUA",
			PlayerId: "qUQkST8CendYA3baHNoPjk",
			Time:     0,
			Item:     entity.ListDatum{Word: "UNITERS", Score: 68, Probability: 1},
		},
	})

	stats1, ok := u1.Profile.Stats.Data["CSW19.classic.ultrablitz"]
	is.True(ok)

	err = pkgstats.Finalize(stats1, lstore, []string{
		"m5ktbp4qPVTqaAhg6HJMsb", "ycj5de5gArFF3ap76JyiUA",
	},
		// cesar4 then mina
		"xjCWug7EZtDxDHX5fRZTLo", "qUQkST8CendYA3baHNoPjk")

	is.Equal(stats1.PlayerOneData[entity.BINGOS_STAT].List, []*entity.ListItem{
		{
			GameId:   "ycj5de5gArFF3ap76JyiUA",
			PlayerId: "xjCWug7EZtDxDHX5fRZTLo",
			Time:     0,
			Item:     entity.ListDatum{Word: "STYMING", Score: 70, Probability: 1},
		},
	})

	is.Equal(stats1.PlayerOneData[entity.SCORE_STAT].Total, 307)
	is.Equal(stats1.PlayerOneData[entity.WINS_STAT].Total, 1)
	ustore.(*user.DBStore).Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
}
