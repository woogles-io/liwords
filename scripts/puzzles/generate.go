package main

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/puzzles"
	commondb "github.com/domino14/liwords/pkg/stores/common"
	"github.com/domino14/liwords/pkg/stores/game"
	puzzlesstore "github.com/domino14/liwords/pkg/stores/puzzles"
	"github.com/domino14/liwords/pkg/stores/user"

	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/macondo/automatic"
)

func main() {
	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)
	cfg.MacondoConfig.DefaultLexicon = common.DefaultLexicon
	zerolog.SetGlobalLevel(zerolog.Disabled)

	TestDBHost := os.Getenv("TEST_DB_HOST")
	TestingConnStr := "host=" + TestDBHost + " port=5432 user=postgres password=pass sslmode=disable"
	TestingDBConnStr := TestingConnStr + " database=liwords_test"

	// Recreate the test database
	err := commondb.RecreateDB()
	if err != nil {
		panic(err)
	}

	// Reconnect to the new test database
	db, err := commondb.OpenDB()
	if err != nil {
		panic(err)
	}

	userStore, err := user.NewDBStore(TestingDBConnStr)
	if err != nil {
		panic(err)
	}

	cfg.DBConnString = TestingDBConnStr
	gameStore, err := game.NewDBStore(cfg, userStore)
	if err != nil {
		panic(err)
	}

	err = commondb.RecreatePuzzleTables(db)
	if err != nil {
		panic(err)
	}

	puzzlesStore, err := puzzlesstore.NewDBStore(db)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	for i := 0; i < 10; i++ {
		r := automatic.NewGameRunner(nil, &cfg.MacondoConfig)
		err := r.CompVsCompStatic(true)
		if err != nil {
			log.Err(err).Msg("game-runner")
			continue
		}
		mcg := r.Game()
		_, err = puzzles.CreatePuzzlesFromGame(ctx, gameStore, puzzlesStore, mcg, "", pb.GameType_BOT_VS_BOT)
		// pzls, err := puzzles.CreatePuzzlesFromGame(ctx, gameStore, puzzlesStore, mcg, "", entity.BotVsBot)
		// for _, pzl := range pzls {
		// 	fmt.Printf("liwords.localhost/game/%s?turn=%d\n", pzl.GetGameId(), pzl.GetTurnNumber()+1)
		// }
		if err != nil {
			log.Err(err).Msg("create-puzzles-from-game")
		}
	}
}
