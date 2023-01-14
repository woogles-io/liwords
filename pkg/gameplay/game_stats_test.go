package gameplay_test

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	pkgstats "github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/stores/common"
	"github.com/domino14/liwords/pkg/stores/mod"
	"github.com/domino14/liwords/pkg/stores/stats"
	"github.com/domino14/liwords/pkg/stores/user"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	macondoconfig "github.com/domino14/macondo/config"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

var gameReq = &pb.GameRequest{Lexicon: "CSW21",
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
	DefaultLexicon:            "CSW21",
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

func recreateDB() (*pgxpool.Pool, *user.DBStore, *stats.DBStore, *mod.DBStore) {
	// Create a database.
	err := common.RecreateTestDB()
	if err != nil {
		panic(err)
	}
	pool, err := common.OpenTestingDB()
	if err != nil {
		panic(err)
	}

	// Create a user table. Initialize the user store.
	ustore, err := user.NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	lstore, err := stats.NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	nstore, err := mod.NewDBStore(pool)
	if err != nil {
		panic(err)
	}

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
	return pool, ustore, lstore, nstore
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
	_, ustore, lstore, _ := recreateDB()

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

	ctx := context.WithValue(context.Background(), config.CtxKeyword, &config.Config{MacondoConfig: DefaultConfig})
	s, err := gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, ustore, lstore)
	is.NoErr(err)
	pkgstats.Finalize(ctx, s, lstore, []string{"m5ktbp4qPVTqaAhg6HJMsb"},
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

	ustore.Disconnect()
	lstore.Disconnect()
}

func TestComputeGameStats2(t *testing.T) {
	is := is.New(t)
	_, ustore, lstore, _ := recreateDB()

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

	ctx := context.WithValue(context.Background(), config.CtxKeyword, &config.Config{MacondoConfig: DefaultConfig})
	s, err := gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, ustore, lstore)
	is.NoErr(err)

	pkgstats.Finalize(ctx, s, lstore, []string{"ycj5de5gArFF3ap76JyiUA"},
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

	ustore.Disconnect()
	lstore.Disconnect()

}

func TestComputePlayerStats(t *testing.T) {
	is := is.New(t)
	_, ustore, lstore, _ := recreateDB()

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

	ctx := context.WithValue(context.Background(), config.CtxKeyword, &config.Config{MacondoConfig: DefaultConfig})
	_, err = gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, ustore, lstore)
	is.NoErr(err)

	p0id, p1id := hist.Players[0].UserId, hist.Players[1].UserId

	u0, err := ustore.GetByUUID(context.Background(), p0id)
	is.NoErr(err)
	u1, err := ustore.GetByUUID(context.Background(), p1id)
	is.NoErr(err)

	stats0, ok := u0.Profile.Stats.Data["CSW19.classic.ultrablitz"]
	is.True(ok)

	err = pkgstats.Finalize(ctx, stats0, lstore, []string{"m5ktbp4qPVTqaAhg6HJMsb"},
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
	ustore.Disconnect()
	lstore.Disconnect()
}

func TestComputePlayerStatsMultipleGames(t *testing.T) {
	is := is.New(t)
	_, ustore, lstore, _ := recreateDB()
	ctx := context.Background()

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

		ctx := context.WithValue(context.Background(), config.CtxKeyword, &config.Config{MacondoConfig: DefaultConfig})
		s, err := gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, ustore, lstore)
		is.NoErr(err)
		log.Debug().Interface("stats", s).Msg("computed-stats")
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
	err = pkgstats.Finalize(ctx, stats0, lstore, []string{
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

	err = pkgstats.Finalize(ctx, stats1, lstore, []string{
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
	ustore.Disconnect()
	lstore.Disconnect()
}
