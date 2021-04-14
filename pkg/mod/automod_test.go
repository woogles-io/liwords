package mod_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/mod"
	pkgstats "github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/stores/stats"
	ts "github.com/domino14/liwords/pkg/stores/tournament"
	"github.com/domino14/liwords/pkg/stores/user"
	"github.com/domino14/liwords/pkg/tournament"
	pkguser "github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/alphabet"
	macondoconfig "github.com/domino14/macondo/config"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/jinzhu/gorm"
	"github.com/matryer/is"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

var TestDBHost = os.Getenv("TEST_DB_HOST")
var TestingDBConnStr = "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"
var gameReq = &pb.GameRequest{Lexicon: "CSW19",
	Rules: &pb.GameRules{BoardLayoutName: entity.CrosswordGame,
		LetterDistributionName: "English",
		VariantName:            "classic"},

	InitialTimeSeconds: 5 * 60,
	IncrementSeconds:   0,
	ChallengeRule:      macondopb.ChallengeRule_TRIPLE,
	GameMode:           pb.GameMode_REAL_TIME,
	RatingMode:         pb.RatingMode_RATED,
	RequestId:          "yeet",
	OriginalRequestId:  "originalyeet",
	MaxOvertimeMinutes: 0}

var DefaultConfig = macondoconfig.Config{
	LexiconPath:               os.Getenv("LEXICON_PATH"),
	LetterDistributionPath:    os.Getenv("LETTER_DISTRIBUTION_PATH"),
	DefaultLexicon:            "CSW19",
	DefaultLetterDistribution: "English",
}

var playerIds = []string{"xjCWug7EZtDxDHX5fRZTLo", "qUQkST8CendYA3baHNoPjk"}

func gameStore(dbURL string, userStore pkguser.Store) (*config.Config, gameplay.GameStore) {
	cfg := &config.Config{}
	cfg.MacondoConfig = DefaultConfig
	cfg.DBConnString = dbURL

	tmp, err := game.NewDBStore(cfg, userStore)
	if err != nil {
		panic(err)
	}
	gameStore := game.NewCache(tmp)
	return cfg, gameStore
}

func tournamentStore(cfg *config.Config, gs gameplay.GameStore) tournament.TournamentStore {
	tmp, err := ts.NewDBStore(cfg, gs)
	if err != nil {
		panic(err)
	}
	tournamentStore := ts.NewCache(tmp)
	return tournamentStore
}

type evtConsumer struct {
	evts []*entity.EventWrapper
}

func (ec *evtConsumer) consumeEventChan(ctx context.Context,
	ch chan *entity.EventWrapper,
	done chan bool) {
	defer func() { done <- true }()
	for {
		select {
		case msg := <-ch:
			ec.evts = append(ec.evts, msg)
		case <-ctx.Done():
			return
		}
	}
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
		{Username: "player1", Email: "cesar4@woogles.io", UUID: playerIds[0]},
		{Username: "player2", Email: "mina@gmail.com", UUID: playerIds[1]},
	} {
		err = ustore.New(context.Background(), u)
		if err != nil {
			log.Fatal().Err(err).Msg("error")
		}
	}
	ustore.(*user.DBStore).Disconnect()
}

func makeGame(cfg *config.Config, ustore pkguser.Store, gstore gameplay.GameStore, longGame bool) (
	*entity.Game, *entity.FakeNower, context.CancelFunc, chan bool, *evtConsumer) {

	ctx := context.Background()
	cesar, _ := ustore.Get(ctx, "player1")
	jesse, _ := ustore.Get(ctx, "player2")
	// see the gameReq in game_test.go in this package
	gr := proto.Clone(gameReq).(*pb.GameRequest)

	if longGame {
		gr.InitialTimeSeconds = 60 * 60
	}
	g, _ := gameplay.InstantiateNewGame(ctx, gstore, cfg, [2]*entity.User{cesar, jesse},
		1, gr, nil)

	ch := make(chan *entity.EventWrapper)
	donechan := make(chan bool)
	consumer := &evtConsumer{}

	cctx, cancel := context.WithCancel(ctx)
	go consumer.consumeEventChan(cctx, ch, donechan)

	nower := entity.NewFakeNower(1234)
	g.SetTimerModule(nower)

	gameplay.StartGame(ctx, gstore, ch, g.GameID())

	return g, nower, cancel, donechan, consumer
}

func playGame(g *entity.Game,
	ustore pkguser.Store,
	lstore pkgstats.ListStatStore,
	tstore tournament.TournamentStore,
	gstore gameplay.GameStore,
	turns []*pb.ClientGameplayEvent,
	loserIndex int,
	gameEndReason pb.GameEndReason) error {

	ctx := context.WithValue(context.Background(), gameplay.ConfigCtxKey("config"), &DefaultConfig)
	nower := entity.NewFakeNower(int64(g.GameReq.InitialTimeSeconds))
	g.SetTimerModule(nower)
	gid := ""
	for i := 0; i < len(turns); i++ {
		// Let each turn take a minute
		nower.Sleep(60)
		turn := turns[i]
		turn.GameId = g.GameID()
		playerIdx := 1 - (i % 2)

		g.SetRackFor(playerIdx, alphabet.RackFromString(turn.Tiles, g.Alphabet()))

		_, err := gameplay.HandleEvent(ctx, gstore, ustore, lstore, tstore,
			playerIds[playerIdx], turn)
		gid = turn.GameId
		if err != nil {
			return err
		}
	}

	if gameEndReason == pb.GameEndReason_RESIGNED {
		_, err := gameplay.HandleEvent(ctx, gstore, ustore, lstore, tstore,
			playerIds[loserIndex], &pb.ClientGameplayEvent{Type: pb.ClientGameplayEvent_RESIGN, GameId: g.GameID()})
		if err != nil {
			return err
		}
	} else if gameEndReason == pb.GameEndReason_TIME {
		g.SetPlayerOnTurn(loserIndex)
		nower.Sleep(int64(g.GameReq.InitialTimeSeconds * 2))
		err := gameplay.TimedOut(ctx, gstore, ustore, lstore, tstore, playerIds[loserIndex], gid)
		if err != nil {
			return err
		}
	} else {
		// End the game with a triple challenge
		_, err := gameplay.HandleEvent(ctx, gstore, ustore, lstore, tstore,
			playerIds[loserIndex], &pb.ClientGameplayEvent{Type: pb.ClientGameplayEvent_CHALLENGE_PLAY, GameId: g.GameID()})
		if err != nil {
			return err
		}
	}

	return nil
}

func printPlayerNotorieties(ustore pkguser.Store) {
	for _, playerId := range playerIds {
		score, games, err := mod.GetNotorietyReport(context.Background(), ustore, playerId)
		if err != nil {
			panic(err)
		}
		gamesString := "[]*ms.NotoriousGames{"
		for _, game := range games {
			gamesString += fmt.Sprintf("{Type: ms.NotoriousGameType_%s},", game.Type.String())
		}
		gamesString += "}"
		fmt.Printf("&ms.NotorietyReport{Score: %d, Games: %s}\n", score, gamesString)
	}
}

func TestNotoriety(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	is := is.New(t)
	recreateDB()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	ustore := userStore(cstr)
	lstore := listStatStore(cstr)
	cfg, gstore := gameStore(cstr, ustore)
	tstore := tournamentStore(cfg, gstore)

	defaultTurns := []*pb.ClientGameplayEvent{
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "8D",
			Tiles:          "BANJO",
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "7H",
			Tiles:          "BUSUUTI",
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "O1",
			Tiles:          "MAYPOPS",
		},
	}

	// player1ng := []*ms.NotoriousGame{}
	// player2ng := []*ms.NotoriousGame{}
	// Main tests
	// No play
	fmt.Println("NO PLAY")
	g, _, _, _, _ := makeGame(cfg, ustore, gstore, true)
	err := playGame(g, ustore, lstore, tstore, gstore, nil, 0, pb.GameEndReason_RESIGNED)
	is.NoErr(err)
	printPlayerNotorieties(ustore)
	// Lost on time, reasonable
	fmt.Println("LOST ON TIME, REASONABLE")
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, false)
	err = playGame(g, ustore, lstore, tstore, gstore, defaultTurns, 0, pb.GameEndReason_TIME)
	is.NoErr(err)
	printPlayerNotorieties(ustore)
	// Lost on time, unreasonable
	fmt.Println("LOST ON TIME, UNREASONABLE")
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, true)
	err = playGame(g, ustore, lstore, tstore, gstore, defaultTurns, 0, pb.GameEndReason_TIME)
	is.NoErr(err)
	printPlayerNotorieties(ustore)
	// Resigned, unrated game, unreasonable
	// Resigned, rated game, reasonable
	// Resigned, rated game, unreasonable
	// Triple Challenge

	// Test in parallel
	// Increase notoriety, under threshold
	// Decrease notoriety
	// Increase notoriety, under to over
	// Increase notoriety, over to over

}
