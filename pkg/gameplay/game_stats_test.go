package gameplay_test

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
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

// testRedisPool dials REDIS_URL, which CI sets. A challenge posts its result
// to the game's chats there. With no REDIS_URL every dial fails, and so does
// the post, which the game then does without.
func testRedisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(os.Getenv("REDIS_URL")) },
	}
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
	stores, err := stores.NewInitializedStores(pool, testRedisPool(), cfg)
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
	// PARDINE and HETAERA.
	is.Equal(s.PlayerOneData[entity.BINGOS_STAT].Total, 2)
	is.Equal(s.PlayerOneData[entity.RATINGS_STAT].Total, 1)
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

	// STYMING, and UNITERS.
	is.Equal(s.PlayerOneData[entity.BINGOS_STAT].Total, 1)
	is.Equal(s.PlayerTwoData[entity.BINGOS_STAT].Total, 1)
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

	stats0, err := stores.UserStore.GetProfileVariantStats(context.Background(), u0.UUID, "CSW19.classic.ultrablitz")
	is.NoErr(err)
	is.True(stats0 != nil)

	// PARDINE and HETAERA.
	is.Equal(stats0.PlayerOneData[entity.BINGOS_STAT].Total, 2)

	is.Equal(stats0.PlayerOneData[entity.WINS_STAT].Total, 1)

	stats1, err := stores.UserStore.GetProfileVariantStats(context.Background(), u1.UUID, "CSW19.classic.ultrablitz")
	is.NoErr(err)
	is.True(stats1 != nil)
	is.Equal(stats1.PlayerOneData[entity.WINS_STAT].Total, 0)
}

func TestComputePlayerStatsMultipleGames(t *testing.T) {
	is := is.New(t)
	_, stores, _ := recreateDB()
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

	stats0, err := stores.UserStore.GetProfileVariantStats(context.Background(), u0.UUID, "CSW19.classic.ultrablitz")
	is.NoErr(err)
	is.True(stats0 != nil)
	is.Equal(stats0.PlayerOneData[entity.GAMES_STAT].Total, 2)
	is.Equal(stats0.PlayerOneData[entity.WINS_STAT].Total, 1)

	// PARDINE and HETAERA, then UNITERS.
	is.Equal(stats0.PlayerOneData[entity.BINGOS_STAT].Total, 3)

	stats1, err := stores.UserStore.GetProfileVariantStats(context.Background(), u1.UUID, "CSW19.classic.ultrablitz")
	is.NoErr(err)
	is.True(stats1 != nil)

	// STYMING.
	is.Equal(stats1.PlayerOneData[entity.BINGOS_STAT].Total, 1)
	is.Equal(stats1.PlayerOneData[entity.SCORE_STAT].Total, 307)
	is.Equal(stats1.PlayerOneData[entity.WINS_STAT].Total, 1)

}

// Two games between the same players, ending at the same moment, both count
// in their profile stats.
func TestComputePlayerStatsGamesEndingAtOnce(t *testing.T) {
	is := is.New(t)
	_, stores, _ := recreateDB()
	defer stores.Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())

	type game struct {
		hist *macondopb.GameHistory
		req  *pb.GameRequest
	}
	var games []game
	for _, g := range []string{"game1", "game2"} {
		histjson, err := os.ReadFile("./testdata/" + g + "/history.json")
		is.NoErr(err)
		hist := &macondopb.GameHistory{}
		is.NoErr(json.Unmarshal(histjson, hist))
		reqjson, err := os.ReadFile("./testdata/" + g + "/game_request.json")
		is.NoErr(err)
		req := &pb.GameRequest{}
		is.NoErr(json.Unmarshal(reqjson, req))
		games = append(games, game{hist, req})
	}

	// Several rounds, as one might not happen to overlap.
	const rounds = 5
	var wg sync.WaitGroup
	errs := make(chan error, rounds*len(games))
	for r := 0; r < rounds; r++ {
		for _, g := range games {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := gameplay.ComputeGameStats(ctx, g.hist, gameReq, variantKey(g.req), gameEndedEventObj, stores)
				errs <- err
			}()
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		is.NoErr(err)
	}

	for _, name := range []string{"Mina", "cesar4"} {
		u, err := stores.UserStore.Get(context.Background(), name)
		is.NoErr(err)
		st, err := stores.UserStore.GetProfileVariantStats(context.Background(), u.UUID, "CSW19.classic.ultrablitz")
		is.NoErr(err)
		is.Equal(st.PlayerOneData[entity.GAMES_STAT].Total, rounds*len(games))
	}
}
