package gameplay_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	pkgstats "github.com/woogles-io/liwords/pkg/stats"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
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

var DefaultConfig = config.DefaultConfig()

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

func recreateDB() (*pgxpool.Pool, *stores.Stores, *config.Config) {
	// Create a database.
	err := common.RecreateTestDB(pkg)
	if err != nil {
		panic(err)
	}
	pool, err := common.OpenTestingDB(pkg)
	if err != nil {
		panic(err)
	}

	cfg := DefaultConfig
	cfg.DBConnDSN = common.TestingPostgresConnDSN(pkg) // for gorm stores
	stores, err := stores.NewInitializedStores(pool, nil, cfg)
	if err != nil {
		panic(err)
	}
	// Insert a couple of users into the table.

	for _, u := range []*entity.User{
		{Username: "cesar4", Email: "cesar4@woogles.io", UUID: "xjCWug7EZtDxDHX5fRZTLo"},
		{Username: "Mina", Email: "mina@gmail.com", UUID: "qUQkST8CendYA3baHNoPjk"},
		{Username: "jesse", Email: "jesse@woogles.io", UUID: "3xpEkpRAy3AizbVmDg3kdi"},
	} {
		err = stores.UserStore.New(context.Background(), u)
		if err != nil {
			log.Fatal().Err(err).Msg("error")
		}
	}
	return pool, stores, cfg
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
	_, stores, _ := recreateDB()
	defer stores.Disconnect()

	histjson, err := os.ReadFile("./testdata/game1/history.json")
	is.NoErr(err)
	hist := &macondopb.GameHistory{}
	err = json.Unmarshal(histjson, hist)
	is.NoErr(err)

	reqjson, err := os.ReadFile("./testdata/game1/game_request.json")
	is.NoErr(err)
	req := &pb.GameRequest{}
	err = json.Unmarshal(reqjson, req)
	is.NoErr(err)
	ctx := DefaultConfig.WithContext(context.Background())
	s, err := gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, stores)
	is.NoErr(err)
	pkgstats.Finalize(ctx, s, stores.ListStatStore, []string{"m5ktbp4qPVTqaAhg6HJMsb"},
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

}

func TestComputeGameStats2(t *testing.T) {
	is := is.New(t)
	_, stores, _ := recreateDB()
	defer stores.Disconnect()
	histjson, err := os.ReadFile("./testdata/game2/history.json")
	is.NoErr(err)
	hist := &macondopb.GameHistory{}
	err = json.Unmarshal(histjson, hist)
	is.NoErr(err)

	reqjson, err := os.ReadFile("./testdata/game2/game_request.json")
	is.NoErr(err)
	req := &pb.GameRequest{}
	err = json.Unmarshal(reqjson, req)
	is.NoErr(err)

	ctx := DefaultConfig.WithContext(context.Background())

	s, err := gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, stores)
	is.NoErr(err)

	pkgstats.Finalize(ctx, s, stores.ListStatStore, []string{"ycj5de5gArFF3ap76JyiUA"},
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
}

func TestComputePlayerStats(t *testing.T) {
	is := is.New(t)
	_, stores, _ := recreateDB()
	defer stores.Disconnect()
	histjson, err := os.ReadFile("./testdata/game1/history.json")
	is.NoErr(err)
	hist := &macondopb.GameHistory{}
	err = json.Unmarshal(histjson, hist)
	is.NoErr(err)

	reqjson, err := os.ReadFile("./testdata/game1/game_request.json")
	is.NoErr(err)
	req := &pb.GameRequest{}
	err = json.Unmarshal(reqjson, req)
	is.NoErr(err)

	ctx := DefaultConfig.WithContext(context.Background())
	_, err = gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, stores)
	is.NoErr(err)

	p0id, p1id := hist.Players[0].UserId, hist.Players[1].UserId

	u0, err := stores.UserStore.GetByUUID(context.Background(), p0id)
	is.NoErr(err)
	u1, err := stores.UserStore.GetByUUID(context.Background(), p1id)
	is.NoErr(err)

	stats0, ok := u0.Profile.Stats.Data["CSW19.classic.ultrablitz"]
	is.True(ok)

	err = pkgstats.Finalize(ctx, stats0, stores.ListStatStore, []string{"m5ktbp4qPVTqaAhg6HJMsb"},
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
}

func TestComputePlayerStatsMultipleGames(t *testing.T) {
	is := is.New(t)
	_, stores, _ := recreateDB()
	ctx := context.Background()
	defer stores.Disconnect()
	for _, g := range []string{"game1", "game2"} {
		histjson, err := os.ReadFile("./testdata/" + g + "/history.json")
		is.NoErr(err)
		hist := &macondopb.GameHistory{}
		err = json.Unmarshal(histjson, hist)
		is.NoErr(err)

		reqjson, err := os.ReadFile("./testdata/" + g + "/game_request.json")
		is.NoErr(err)
		req := &pb.GameRequest{}
		err = json.Unmarshal(reqjson, req)
		is.NoErr(err)

		ctx := DefaultConfig.WithContext(context.Background())
		s, err := gameplay.ComputeGameStats(ctx, hist, gameReq, variantKey(req), gameEndedEventObj, stores)
		is.NoErr(err)
		log.Debug().Interface("stats", s).Msg("computed-stats")
	}

	u0, err := stores.UserStore.Get(context.Background(), "Mina")
	is.NoErr(err)
	u1, err := stores.UserStore.Get(context.Background(), "cesar4")
	is.NoErr(err)

	stats0, ok := u0.Profile.Stats.Data["CSW19.classic.ultrablitz"]
	is.True(ok)
	is.Equal(stats0.PlayerOneData[entity.GAMES_STAT].Total, 2)
	is.Equal(stats0.PlayerOneData[entity.WINS_STAT].Total, 1)

	log.Debug().Interface("li", stats0.PlayerOneData[entity.BINGOS_STAT].List).Msg("--")
	err = pkgstats.Finalize(ctx, stats0, stores.ListStatStore, []string{
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

	err = pkgstats.Finalize(ctx, stats1, stores.ListStatStore, []string{
		"m5ktbp4qPVTqaAhg6HJMsb", "ycj5de5gArFF3ap76JyiUA",
	},
		// cesar4 then mina
		"xjCWug7EZtDxDHX5fRZTLo", "qUQkST8CendYA3baHNoPjk")
	is.NoErr(err)
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

}
