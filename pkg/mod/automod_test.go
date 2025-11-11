package mod_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lithammer/shortuuid/v4"
	"github.com/matryer/is"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	pkgmod "github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
	"google.golang.org/protobuf/proto"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

var pkg = "mod_test"

var gameReq = &pb.GameRequest{Lexicon: "CSW21",
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

var playerIds = []string{"xjCWug7EZtDxDHX5fRZTLo", "qUQkST8CendYA3baHNoPjk"}

var DefaultConfig = config.DefaultConfig()

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

func recreateDB() (*pgxpool.Pool, *stores.Stores, *config.Config) {
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

	// Insert a couple of users into the table.

	for _, u := range []*entity.User{
		{Username: "cesar", Email: os.Getenv("TEST_EMAIL_USERNAME") + "+spammer@woogles.io", UUID: playerIds[0]},
		{Username: "jesse", Email: os.Getenv("TEST_EMAIL_USERNAME") + "@woogles.io", UUID: playerIds[1]},
	} {
		err = stores.UserStore.New(context.Background(), u)
		if err != nil {
			log.Fatal().Err(err).Msg("error")
		}
	}
	return pool, stores, cfg
}

func makeGame(cfg *config.Config, stores *stores.Stores, initialTime int, ratingMode pb.RatingMode) (
	*entity.Game, *entity.FakeNower, context.CancelFunc, chan bool, *evtConsumer) {

	ctx := context.Background()
	cesar, err := stores.UserStore.Get(ctx, "cesar")
	if err != nil {
		panic(err)
	}
	jesse, err := stores.UserStore.Get(ctx, "jesse")
	if err != nil {
		panic(err)
	}

	gr := proto.Clone(gameReq).(*pb.GameRequest)

	gr.InitialTimeSeconds = int32(initialTime * 60)
	gr.RatingMode = ratingMode
	g, err := gameplay.InstantiateNewGame(ctx, stores.GameStore, cfg, [2]*entity.User{cesar, jesse},
		gr, nil)
	if err != nil {
		panic(err)
	}

	ch := make(chan *entity.EventWrapper)
	donechan := make(chan bool)
	consumer := &evtConsumer{}
	stores.GameStore.SetGameEventChan(ch)

	cctx, cancel := context.WithCancel(ctx)
	go consumer.consumeEventChan(cctx, ch, donechan)

	nower := entity.NewFakeNower(1234)
	g.SetTimerModule(nower)

	err = gameplay.StartGame(ctx, stores, ch, g)
	if err != nil {
		panic(err)
	}
	return g, nower, cancel, donechan, consumer
}

func playGame(ctx context.Context,
	g *entity.Game,
	stores *stores.Stores,
	turns []*pb.ClientGameplayEvent,
	loserIndex int,
	gameEndReason pb.GameEndReason,
	sitResign bool) error {

	fmt.Println("turns", turns)
	nower := entity.NewFakeNower(1234)
	g.SetTimerModule(nower)
	g.ResetTimersAndStart()
	gid := ""
	for i := 0; i < len(turns); i++ {
		// Let each turn take a minute
		nower.Sleep(60 * 1000)
		turn := turns[i]
		turn.GameId = g.GameID()
		playerIdx := i % 2
		fmt.Println("on turn now", g.NickOnTurn())
		r := tilemapping.NewRack(g.Alphabet())
		r.Set(tilemapping.FromByteArr(turn.MachineLetters))
		g.SetRackFor(playerIdx, r)

		_, err := gameplay.HandleEvent(ctx, stores,
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
		_, err := gameplay.HandleEvent(ctx, stores,
			playerIds[loserIndex], &pb.ClientGameplayEvent{Type: pb.ClientGameplayEvent_RESIGN, GameId: g.GameID()})
		if err != nil {
			return err
		}
	} else if gameEndReason == pb.GameEndReason_TIME {
		g.SetPlayerOnTurn(loserIndex)
		nower.Sleep(int64(g.GameReq.InitialTimeSeconds * 2 * 1000))
		err := gameplay.TimedOut(ctx, stores, playerIds[loserIndex], gid)
		if err != nil {
			return err
		}
	} else {
		// End the game with a triple challenge
		_, err := gameplay.HandleEvent(ctx, stores,
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

func printPlayerNotorieties(stores *stores.Stores) {
	notorietyString := "err = comparePlayerNotorieties([]*ms.NotorietyReport{"
	for _, playerId := range playerIds {
		fmt.Println(playerId)
		score, games, err := pkgmod.GetNotorietyReport(context.Background(), stores.UserStore, stores.NotorietyStore, playerId, 100)
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
	notorietyString += "}, stores)\nis.NoErr(err)"
	fmt.Printf("%s\n", notorietyString)
}

func comparePlayerNotorieties(pnrs []*ms.NotorietyReport, stores *stores.Stores) error {
	for idx, playerId := range playerIds {
		score, games, err := pkgmod.GetNotorietyReport(context.Background(), stores.UserStore, stores.NotorietyStore, playerId, 100)
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

func englishBytes(tiles string) []byte {
	ld, err := tilemapping.GetDistribution(DefaultConfig.WGLConfig(), "english")
	if err != nil {
		panic(err)
	}
	mw, err := tilemapping.ToMachineWord(tiles, ld.TileMapping())
	if err != nil {
		panic(err)
	}
	return mw.ToByteArr()
}

func TestNotoriety(t *testing.T) {
	//zerolog.SetGlobalLevel(zerolog.Disabled)
	is := is.New(t)
	_, stores, cfg := recreateDB()
	defer stores.Disconnect()
	ctx := DefaultConfig.WithContext(context.Background())

	defaultTurns := []*pb.ClientGameplayEvent{
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "8D",
			MachineLetters: englishBytes("BANJO"),
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "7H",
			MachineLetters: englishBytes("BUSUUTI"),
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "O1",
			MachineLetters: englishBytes("MAYPOPS"),
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "9H",
			MachineLetters: englishBytes("RETINAS"),
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "10B",
			MachineLetters: englishBytes("RETINAS"),
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "11H",
			MachineLetters: englishBytes("ZI"),
		},
	}

	sandbagTurns := []*pb.ClientGameplayEvent{
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "8D",
			MachineLetters: englishBytes("BANJO"),
		},
		{
			Type: pb.ClientGameplayEvent_PASS,
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "7H",
			MachineLetters: englishBytes("BUSUUTI"),
		},
		{
			Type: pb.ClientGameplayEvent_PASS,
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "O1",
			MachineLetters: englishBytes("MAYPOPS"),
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "9H",
			MachineLetters: englishBytes("RETINAS"),
		},
		{
			Type:           pb.ClientGameplayEvent_TILE_PLACEMENT,
			PositionCoords: "10B",
			MachineLetters: englishBytes("RETINAS"),
		},
		{
			Type: pb.ClientGameplayEvent_PASS,
		},
	}
	// No play
	g, _, _, _, _ := makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err := playGame(ctx, g, stores, defaultTurns[:1], 1, pb.GameEndReason_TIME, false)
	is.NoErr(err)
	// printPlayerNotorieties(stores)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 6, Games: []*ms.NotoriousGame{{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	// Play two good games to bring down notoriety
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns[:1], 1, pb.GameEndReason_TRIPLE_CHALLENGE, false)
	is.NoErr(err)
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns[:1], 1, pb.GameEndReason_TRIPLE_CHALLENGE, false)
	is.NoErr(err)

	// Lost on time, reasonable
	g, _, _, _, _ = makeGame(cfg, stores, 7, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns, 1, pb.GameEndReason_TIME, false)
	is.NoErr(err)
	// printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 3, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	// Lost on time, unreasonable
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns, 1, pb.GameEndReason_TIME, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 7, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	// Resigned, unrated game, unreasonable
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_CASUAL)
	err = playGame(ctx, g, stores, defaultTurns, 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 7, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	// Resigned, rated game, reasonable
	g, _, _, _, _ = makeGame(cfg, stores, 6, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns, 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 6, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	// Resigned, rated game, unreasonable sitresign
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns[:2], 1, pb.GameEndReason_RESIGNED, true)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 10, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	// Make sure no action exists
	_, err = pkgmod.ActionExists(context.Background(), stores.UserStore, playerIds[1], false, []ms.ModActionType{ms.ModActionType_SUSPEND_RATED_GAMES})
	is.NoErr(err)

	// Check the DB as well
	_, err = pkgmod.ActionExists(context.Background(), stores.UserStore, playerIds[1], false, []ms.ModActionType{ms.ModActionType_SUSPEND_RATED_GAMES})
	is.NoErr(err)

	// Add these additional misbehaved games bring the user over the threshold
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, nil, 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 16, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	// Check mod actions here
	_, err = pkgmod.ActionExists(context.Background(), stores.UserStore, playerIds[1], false, []ms.ModActionType{ms.ModActionType_SUSPEND_RATED_GAMES})
	is.True(err != nil)

	_, err = pkgmod.ActionExists(context.Background(), stores.UserStore, playerIds[1], false, []ms.ModActionType{ms.ModActionType_SUSPEND_RATED_GAMES})
	is.True(err != nil)

	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, nil, 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 22, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	// Check mod actions here again
	// There should be an action in the action history
	actionGames := &ms.ModAction{UserId: playerIds[1], Type: ms.ModActionType_SUSPEND_RATED_GAMES, Duration: 60 * 60 * 24 * 6}
	_, err = pkgmod.ActionExists(context.Background(), stores.UserStore, playerIds[1], false, []ms.ModActionType{ms.ModActionType_SUSPEND_RATED_GAMES})
	is.True(err != nil)
	_, err = pkgmod.ActionExists(context.Background(), stores.UserStore, playerIds[1], false, []ms.ModActionType{ms.ModActionType_SUSPEND_RATED_GAMES})
	is.True(err != nil)
	history, err := pkgmod.GetActionHistory(context.Background(), stores.UserStore, playerIds[1])
	is.NoErr(err)
	is.NoErr(equalActionHistories(history, []*ms.ModAction{actionGames}))

	history, err = stores.UserStore.GetActionHistory(context.Background(), playerIds[1])
	is.NoErr(err)
	is.NoErr(equalActionHistories(history, []*ms.ModAction{actionGames}))

	// Triple Challenge
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns[:1], 1, pb.GameEndReason_TRIPLE_CHALLENGE, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 21, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	// The other player has now misbehaved
	// Now both players have a nonzero notoriety
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, nil, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 6, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 20, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	// One player's notoriety should increase, the other's should decrease
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, nil, 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 5, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 26, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	actionGames1 := &ms.ModAction{UserId: playerIds[1], Type: ms.ModActionType_SUSPEND_RATED_GAMES, Duration: 60 * 60 * 24 * 6}
	actionGames2 := &ms.ModAction{UserId: playerIds[1], Type: ms.ModActionType_SUSPEND_RATED_GAMES, Duration: 60 * 60 * 24 * 12}
	_, err = pkgmod.ActionExists(context.Background(), stores.UserStore, playerIds[1], false, []ms.ModActionType{ms.ModActionType_SUSPEND_RATED_GAMES})
	is.True(err != nil)

	history, err = pkgmod.GetActionHistory(context.Background(), stores.UserStore, playerIds[1])
	is.NoErr(err)
	is.NoErr(equalActionHistories(history, []*ms.ModAction{actionGames1, actionGames2}))

	history, err = stores.UserStore.GetActionHistory(context.Background(), playerIds[1])
	is.NoErr(err)
	is.NoErr(equalActionHistories(history, []*ms.ModAction{actionGames1, actionGames2}))

	// Both players' notorieties should decrease
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns[:1], 1, pb.GameEndReason_TRIPLE_CHALLENGE, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 4, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 25, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	g, _, _, _, consumer := makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	is.NoErr(err)

	evtID := shortuuid.New()

	metaEvt := &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_REQUEST_ABORT,
		PlayerId:    g.Quickdata.PlayerInfo[0].UserId,
		GameId:      g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt, consumer.ch, stores)

	is.NoErr(err)

	metaEvt = &pb.GameMetaEvent{
		Timestamp:   timestamppb.New(time.Now()),
		Type:        pb.GameMetaEvent_ABORT_DENIED,
		PlayerId:    g.Quickdata.PlayerInfo[1].UserId,
		GameId:      g.GameID(),
		OrigEventId: evtID,
	}

	err = gameplay.HandleMetaEvent(context.Background(), metaEvt,
		consumer.ch, stores)
	is.NoErr(err)

	// Update context so notifications are sent for this game
	session := &entity.Session{
		ID:       "abcdef",
		Username: "Moderator",
		UserUUID: "Moderator",
		Expiry:   time.Now().Add(time.Second * 100)}
	ctx = apiserver.PlaceInContext(ctx, session)

	cfg.MailgunKey = os.Getenv("TEST_MAILGUN_KEY")
	cfg.DiscordToken = os.Getenv("TEST_DISCORD_TOKEN")

	err = playGame(ctx, g, stores, nil, 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)

	// Set the context back so the tests do not give excessive notifications
	cfg.MailgunKey = ""
	cfg.DiscordToken = ""

	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 3, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY}}},
		{Score: 35, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_NO_PLAY_DENIED_NUDGE},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_NO_PLAY},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_SITTING},
			{Type: ms.NotoriousGameType_NO_PLAY}}},
	}, stores)
	is.NoErr(err)

	// Test resetting the notorieties
	err = pkgmod.ResetNotoriety(context.Background(), stores.UserStore, stores.NotorietyStore, playerIds[0])
	is.NoErr(err)
	err = pkgmod.ResetNotoriety(context.Background(), stores.UserStore, stores.NotorietyStore, playerIds[1])
	is.NoErr(err)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, stores)
	is.NoErr(err)

	// Test Sitresigning
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns, 1, pb.GameEndReason_RESIGNED, true)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 4, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SITTING}}},
	}, stores)
	is.NoErr(err)

	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns, 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	//printPlayerNotorieties(ustore)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 3, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SITTING}}},
	}, stores)
	is.NoErr(err)

	// Test sandbag

	err = pkgmod.ResetNotoriety(context.Background(), stores.UserStore, stores.NotorietyStore, playerIds[0])
	is.NoErr(err)
	err = pkgmod.ResetNotoriety(context.Background(), stores.UserStore, stores.NotorietyStore, playerIds[1])
	is.NoErr(err)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 0, Games: []*ms.NotoriousGame{}}}, stores)
	is.NoErr(err)

	// Sandbagging
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns[:2], 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 4, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SANDBAG}}},
	}, stores)
	is.NoErr(err)

	// Not sandbagging
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns, 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)

	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 3, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SANDBAG}}},
	}, stores)
	is.NoErr(err)

	// Sandbagging because of passes
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, sandbagTurns, 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)

	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 7, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_SANDBAG},
			{Type: ms.NotoriousGameType_SANDBAG}}},
	}, stores)
	is.NoErr(err)

	// Reset notorieties
	err = pkgmod.ResetNotoriety(context.Background(), stores.UserStore, stores.NotorietyStore, playerIds[0])
	is.NoErr(err)
	err = pkgmod.ResetNotoriety(context.Background(), stores.UserStore, stores.NotorietyStore, playerIds[1])
	is.NoErr(err)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 0, Games: []*ms.NotoriousGame{}},
		{Score: 0, Games: []*ms.NotoriousGame{}},
	}, stores)
	is.NoErr(err)

	// Excessive phonies
	defaultTurns[0].MachineLetters = englishBytes("ABNJO")
	defaultTurns[2].MachineLetters = englishBytes("MAYPPOS")
	defaultTurns[4].MachineLetters = englishBytes("RETIANS")

	// Winner and loser should not matter
	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns, 1, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)
	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 8, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_EXCESSIVE_PHONIES}}},
		{Score: 0, Games: []*ms.NotoriousGame{}},
	}, stores)
	is.NoErr(err)

	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)

	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 16, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_EXCESSIVE_PHONIES},
			{Type: ms.NotoriousGameType_EXCESSIVE_PHONIES}}},
		{Score: 0, Games: []*ms.NotoriousGame{}},
	}, stores)
	is.NoErr(err)

	// Now the other player phonies too much
	defaultTurns[0].MachineLetters = englishBytes("BANJO")
	defaultTurns[2].MachineLetters = englishBytes("MAYPOPS")
	defaultTurns[4].MachineLetters = englishBytes("RETINAS")
	defaultTurns[1].MachineLetters = englishBytes("BUSUTUI")
	defaultTurns[3].MachineLetters = englishBytes("RETIANS")

	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)

	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 15, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_EXCESSIVE_PHONIES},
			{Type: ms.NotoriousGameType_EXCESSIVE_PHONIES}}},
		{Score: 8, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_EXCESSIVE_PHONIES}}},
	}, stores)
	is.NoErr(err)

	defaultTurns[1].MachineLetters = englishBytes("BUSUUTI")

	g, _, _, _, _ = makeGame(cfg, stores, 60, pb.RatingMode_RATED)
	err = playGame(ctx, g, stores, defaultTurns, 0, pb.GameEndReason_RESIGNED, false)
	is.NoErr(err)

	err = comparePlayerNotorieties([]*ms.NotorietyReport{
		{Score: 14, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_EXCESSIVE_PHONIES},
			{Type: ms.NotoriousGameType_EXCESSIVE_PHONIES}}},
		{Score: 7, Games: []*ms.NotoriousGame{
			{Type: ms.NotoriousGameType_EXCESSIVE_PHONIES}}},
	}, stores)
	is.NoErr(err)

}
