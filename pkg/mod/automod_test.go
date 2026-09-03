package mod_test

import (
	"context"
	"errors"
	"fmt"
	"os"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	pkgmod "github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
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
