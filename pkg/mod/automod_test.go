package mod_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	pkgmod "github.com/domino14/liwords/pkg/mod"
	pkgstats "github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/stores/mod"
	"github.com/domino14/liwords/pkg/stores/stats"
	ts "github.com/domino14/liwords/pkg/stores/tournament"
	"github.com/domino14/liwords/pkg/stores/user"
	"github.com/domino14/liwords/pkg/tournament"
	pkguser "github.com/domino14/liwords/pkg/user"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/domino14/macondo/alphabet"
	macondoconfig "github.com/domino14/macondo/config"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/jinzhu/gorm"
	"github.com/lithammer/shortuuid"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
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

func notorietyStore(dbURL string) pkgmod.NotorietyStore {
	n, err := mod.NewNotorietyStore(TestingDBConnStr + " dbname=liwords_test")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return n
}

type evtConsumer struct {
	evts []*entity.EventWrapper
	ch   chan *entity.EventWrapper
}

func (ec *evtConsumer) consumeEventChan(ctx context.Context,
	ch chan *entity.EventWrapper,
	done chan bool) {

	ec.ch = ch

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

func userStore(dbURL string) (pkguser.Store, *user.DBStore) {
	tmp, err := user.NewDBStore(TestingDBConnStr + " dbname=liwords_test")
	if err != nil {
		log.Fatal().Err(err).Msg("error")
	}
	return user.NewCache(tmp), tmp
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
	ustore, _ := userStore(TestingDBConnStr + " dbname=liwords_test")

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
}

func makeGame(cfg *config.Config, ustore pkguser.Store, gstore gameplay.GameStore, initialTime int, ratingMode pb.RatingMode) (
	*entity.Game, *entity.FakeNower, context.CancelFunc, chan bool, *evtConsumer) {

	ctx := context.Background()
	cesar, _ := ustore.Get(ctx, "player1")
	jesse, _ := ustore.Get(ctx, "player2")
	// see the gameReq in game_test.go in this package
	gr := proto.Clone(gameReq).(*pb.GameRequest)

	gr.InitialTimeSeconds = int32(initialTime * 60)
	gr.RatingMode = ratingMode
	g, _ := gameplay.InstantiateNewGame(ctx, gstore, cfg, [2]*entity.User{cesar, jesse},
		1, gr, nil)

	ch := make(chan *entity.EventWrapper)
	donechan := make(chan bool)
	consumer := &evtConsumer{}
	gstore.SetGameEventChan(ch)

	cctx, cancel := context.WithCancel(ctx)
	go consumer.consumeEventChan(cctx, ch, donechan)

	nower := entity.NewFakeNower(1234)
	g.SetTimerModule(nower)

	gameplay.StartGame(ctx, gstore, ustore, ch, g.GameID())

	return g, nower, cancel, donechan, consumer
}

func playGame(g *entity.Game,
	ustore pkguser.Store,
	lstore pkgstats.ListStatStore,
	nstore pkgmod.NotorietyStore,
	tstore tournament.TournamentStore,
	gstore gameplay.GameStore,
	turns []*pb.ClientGameplayEvent,
	loserIndex int,
	gameEndReason pb.GameEndReason,
	sitResign bool) error {

	ctx := context.WithValue(context.Background(), gameplay.ConfigCtxKey("config"), &DefaultConfig)
	nower := entity.NewFakeNower(1234)
	g.SetTimerModule(nower)
	g.ResetTimersAndStart()
	gid := ""
	for i := 0; i < len(turns); i++ {
		// Let each turn take a minute
		nower.Sleep(60 * 1000)
		turn := turns[i]
		turn.GameId = g.GameID()
		playerIdx := 1 - (i % 2)

		g.SetRackFor(playerIdx, alphabet.RackFromString(turn.Tiles, g.Alphabet()))

		_, err := gameplay.HandleEvent(ctx, gstore, ustore, nstore, lstore, tstore,
			playerIds[playerIdx], turn)

		gid = turn.GameId
		if err != nil {
			return err
		}
	}

	if gameEndReason == pb.GameEndReason_RESIGNED {
		if sitResign {
			g.SetPlayerOnTurn(loserIndex)
			nower.Sleep(int64(g.GameReq.InitialTimeSeconds * 2 * 1000))
		}
		_, err := gameplay.HandleEvent(ctx, gstore, ustore, nstore, lstore, tstore,
			playerIds[loserIndex], &pb.ClientGameplayEvent{Type: pb.ClientGameplayEvent_RESIGN, GameId: g.GameID()})
		if err != nil {
			return err
		}
	} else if gameEndReason == pb.GameEndReason_TIME {
		g.SetPlayerOnTurn(loserIndex)
		nower.Sleep(int64(g.GameReq.InitialTimeSeconds * 2 * 1000))
		err := gameplay.TimedOut(ctx, gstore, ustore, nstore, lstore, tstore, playerIds[loserIndex], gid)
		if err != nil {
			return err
		}
	} else {
		// End the game with a triple challenge
		_, err := gameplay.HandleEvent(ctx, gstore, ustore, nstore, lstore, tstore,
			playerIds[loserIndex], &pb.ClientGameplayEvent{Type: pb.ClientGameplayEvent_CHALLENGE_PLAY, GameId: g.GameID()})
		if err != nil {
			return err
		}
	}

	return nil
}

func equalActions(a1 *ms.ModAction, a2 *ms.ModAction) bool {
	return a1.UserId == a2.UserId &&
		a1.Type == a2.Type &&
		a1.Duration == a2.Duration
}

func equalActionHistories(ah1 []*ms.ModAction, ah2 []*ms.ModAction) error {
	if len(ah1) != len(ah2) {
		return errors.New("history lengths are not the same")
	}
	for i := 0; i < len(ah1); i++ {
		a1 := ah1[i]
		a2 := ah2[i]
		if !equalActions(a1, a2) {
			return fmt.Errorf("actions are not equal:\n  a1.UserId: %s a1.Type: %s, a1.Duration: %d\n"+
				"  a1.UserId: %s a1.Type: %s, a1.Duration: %d\n", a1.UserId, a1.Type, a1.Duration,
				a2.UserId, a2.Type, a2.Duration)
		}
	}
	return nil
}

func printPlayerNotorieties(ustore pkguser.Store, nstore pkgmod.NotorietyStore) {
	notorietyString := "err = comparePlayerNotorieties([]*ms.NotorietyReport{"
	for _, playerId := range playerIds {
		score, games, err := pkgmod.GetNotorietyReport(context.Background(), ustore, nstore, playerId)
		if err != nil {
			panic(err)
		}
		gamesString := "[]*ms.NotoriousGame{\n"
		for idx, game := range games {
			gamesString += fmt.Sprintf("                       {Type: ms.NotoriousGameType_%s},", game.Type.String())
			if idx != len(games)-1 {
				gamesString += "\n"
			}
		}
		gamesString += "}"
		notorietyString += fmt.Sprintf("\n                       {Score: %d, Games: %s},", score, gamesString)
	}
	notorietyString += "}, ustore)\nis.NoErr(err)"
	fmt.Printf("%s\n", notorietyString)
}

func comparePlayerNotorieties(pnrs []*ms.NotorietyReport, ustore pkguser.Store, nstore pkgmod.NotorietyStore) error {
	for idx, playerId := range playerIds {
		score, games, err := pkgmod.GetNotorietyReport(context.Background(), ustore, nstore, playerId)
		if err != nil {
			return err
		}
		if int(pnrs[idx].Score) != score {
			return fmt.Errorf("scores are not equal for player %d: %d != %d\n", idx, pnrs[idx].Score, score)
		}
		if len(pnrs[idx].Games) != len(games) {
			return fmt.Errorf("games length are not equal for player %d: %d != %d", idx, len(pnrs[idx].Games), len(games))
		}
		for gameIndex := range pnrs[idx].Games {
			ge := pnrs[idx].Games[gameIndex]
			ga := games[gameIndex]
			if ge.Type != ga.Type {
				return fmt.Errorf("game arrays do not match at index %d: %s != %s", gameIndex, ge.Type.String(), ga.Type.String())
			}
		}
	}
	return nil
}

func TestNotoriety(t *testing.T) {
	//zerolog.SetGlobalLevel(zerolog.Disabled)
	is := is.New(t)
	recreateDB()
	cstr := TestingDBConnStr + " dbname=liwords_test"

	ustore, uDBstore := userStore(cstr)
	lstore := listStatStore(cstr)
	nstore := notorietyStore(cstr)
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
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "9H",
			Tiles:          "RETINAS",
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "10B",
			Tiles:          "RETINAS",
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "11H",
			Tiles:          "ZI",
		},
	}

	// No play
	g, _, _, _, _ := makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err := playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns[:1], 0, pb.GameEndReason_TIME, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 6, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	// Play two good games to bring down notoriety
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns[:1], 0, pb.GameEndReason_TRIPLE_CHALLENGE, false)
	is.NoErr(err)
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns[:1], 0, pb.GameEndReason_TRIPLE_CHALLENGE, false)
	is.NoErr(err)

	// Lost on time, reasonable
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 7, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns, 0, pb.GameEndReason_TIME, false)
	is.NoErr(err)
	// printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 3, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	// Lost on time, unreasonable
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns, 0, pb.GameEndReason_TIME, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 7, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING}}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	// Resigned, unrated game, unreasonable
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_CASUAL)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 7, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING}}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	// Resigned, rated game, reasonable
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 6, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 6, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING}}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	// Resigned, rated game, unreasonable sitresign
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns[:2], 0, pb.GameEndReason_RESIGNED, true)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 10, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING}}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	// Make sure no action exists
	_, err = pkgmod.ActionExists(context.Background(), ustore, playerIds[0], false, []ms.ModActionType{ms.ModActionType_SUSPEND_GAMES})
	is.NoErr(err)

	// Add these additional misbehaved games bring the user over the threshold
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, nil, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 16, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	// Check mod actions here
	_, err = pkgmod.ActionExists(context.Background(), ustore, playerIds[0], false, []ms.ModActionType{ms.ModActionType_SUSPEND_GAMES})
	is.True(err != nil)

	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, nil, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 22, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	// Check mod actions here again
	// There should be an action in the action history
	actionGames := &ms.ModAction{UserId: playerIds[0], Type: ms.ModActionType_SUSPEND_GAMES, Duration: 60 * 60 * 24 * 6}
	_, err = pkgmod.ActionExists(context.Background(), ustore, playerIds[0], false, []ms.ModActionType{ms.ModActionType_SUSPEND_GAMES})
	is.True(err != nil)
	history, err := pkgmod.GetActionHistory(context.Background(), ustore, playerIds[0])
	is.NoErr(err)
	is.NoErr(equalActionHistories(history, []*ms.ModAction{actionGames}))

	// Triple Challenge
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns[:1], 0, pb.GameEndReason_TRIPLE_CHALLENGE, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 21, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	// The other play has now misbehaved
	// Now both plays have a nonzero notoriety
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, nil, 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 20, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 6, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY}}}}, ustore, nstore)
	is.NoErr(err)

	// One player's notoriety should increase, the other's should decrease
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, nil, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 26, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 5, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY}}}}, ustore, nstore)
	is.NoErr(err)

	actionGames1 := &ms.ModAction{UserId: playerIds[0], Type: ms.ModActionType_SUSPEND_GAMES, Duration: 60 * 60 * 24 * 6}
	actionGames2 := &ms.ModAction{UserId: playerIds[0], Type: ms.ModActionType_SUSPEND_GAMES, Duration: 60 * 60 * 24 * 12}
	_, err = pkgmod.ActionExists(context.Background(), ustore, playerIds[0], false, []ms.ModActionType{ms.ModActionType_SUSPEND_GAMES})
	is.True(err != nil)
	history, err = pkgmod.GetActionHistory(context.Background(), ustore, playerIds[0])
	is.NoErr(err)
	is.NoErr(equalActionHistories(history, []*ms.ModAction{actionGames1, actionGames2}))

	// Both players' notorieties should decrease
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns[:1], 0, pb.GameEndReason_TRIPLE_CHALLENGE, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 25, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 4, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY}}}}, ustore, nstore)
	is.NoErr(err)

	g, _, _, _, consumer := makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	is.NoErr(err)

	evtID := shortuuid.New()

	metaEvt := &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    g.Quickdata.PlayerInfo[1].UserId,
		GameId:      g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		consumer.ch, gstore, ustore, nstore, lstore,
		tstore)

	is.NoErr(err)

	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_ABORT_DENIED,
		PlayerId:    g.Quickdata.PlayerInfo[0].UserId,
		GameId:      g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		consumer.ch, gstore, ustore, nstore, lstore,
		tstore)
	is.NoErr(err)

	err = playGame(g, ustore, lstore, nstore, tstore, gstore, nil, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)

	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 35, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY_IGNORE_NUDGE}}},
		{Score: 3, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY}}}}, ustore, nstore)
	is.NoErr(err)

	// Test resetting the notorieties
	err = pkgmod.ResetNotoriety(context.Background(), ustore, nstore, playerIds[0])
	is.NoErr(err)
	err = pkgmod.ResetNotoriety(context.Background(), ustore, nstore, playerIds[1])
	is.NoErr(err)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	// Test Sitresigning
	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns, 0, pb.GameEndReason_RESIGNED, true)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 4, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SITTING}}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	g, _, _, _, _ = makeGame(cfg, ustore, gstore, 60, pb.RatingMode_RATED)
	err = playGame(g, ustore, lstore, nstore, tstore, gstore, defaultTurns, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 3, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SITTING}}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, ustore, nstore)
	is.NoErr(err)

	uDBstore.Disconnect()
	lstore.(*stats.ListStatStore).Disconnect()
	nstore.(*mod.NotorietyStore).Disconnect()
	gstore.(*game.Cache).Disconnect()
	tstore.(*ts.Cache).Disconnect()
	// Test sandbag
}
