package main

import (
	"context"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/puzzles"
	commondb "github.com/domino14/liwords/pkg/stores/common"
	"github.com/domino14/liwords/pkg/stores/game"
	puzzlesstore "github.com/domino14/liwords/pkg/stores/puzzles"
	"github.com/domino14/liwords/pkg/stores/user"
	macondogame "github.com/domino14/macondo/game"

	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/macondo/automatic"
	"github.com/namsral/flag"
)

func main() {
	gf := flag.NewFlagSet("gf", flag.ContinueOnError)
	numGames := gf.Int("i", 10, "number of bot vs bot games used to create puzzles")
	useBotVsBot := gf.Bool("b", false, "use bot vs bot games to create puzzles")
	err := gf.Parse(os.Args[1:])
	if err != nil {
		panic(err)
	}

	cfg := &config.Config{}
	cfg.Load(nil)
	log.Info().Msgf("Loaded config: %v", cfg)
	cfg.MacondoConfig.DefaultLexicon = common.DefaultLexicon
	zerolog.SetGlobalLevel(zerolog.Disabled)

	// Reconnect to the new test database
	db, err := commondb.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		panic(err)
	}

	cfg.DBConnString = commondb.PostgresConnString(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	userStore, err := user.NewDBStore(cfg.DBConnString)
	if err != nil {
		panic(err)
	}

	gameStore, err := game.NewDBStore(cfg, userStore)
	if err != nil {
		panic(err)
	}

	m, err := migrate.New(commondb.MigrationFile, commondb.MigrationConnString(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode))
	if err != nil {
		panic(err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		panic(err)
	}

	puzzlesStore, err := puzzlesstore.NewDBStore(db)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	if !*useBotVsBot {
		rows, err := db.QueryContext(ctx, `SELECT uuid FROM games WHERE games.id NOT IN (SELECT game_id FROM puzzles) AND (stats->'d1'->'Unchallenged Phonies'->'t')::int = 0 AND (stats->'d2'->'Unchallenged Phonies'->'t')::int = 0 AND game_end_reason != 0`)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		for rows.Next() {
			var UUID string
			if err := rows.Scan(&UUID); err != nil {
				log.Err(err).Msg("games-scan")
			}
			fmt.Printf("uuid: %s\n", UUID)
			entGame, err := gameStore.Get(ctx, UUID)
			if err != nil {
				log.Err(err).Msg("games-store")
			}
			mcg := &entGame.Game
			pzls, err := puzzles.CreatePuzzlesFromGame(ctx, gameStore, puzzlesStore, mcg, "", pb.GameType_NATIVE)
			for _, pzl := range pzls {
				fmt.Printf("liwords.localhost/game/%s?turn=%d\n", pzl.GetGameId(), pzl.GetTurnNumber()+1)
			}
			if err != nil {
				fmt.Println(err.Error())
				log.Err(err).Msg("create-puzzles-from-game")
			}
		}
	} else {
		for i := 0; i < *numGames; i++ {
			var mcg *macondogame.Game
			r := automatic.NewGameRunner(nil, &cfg.MacondoConfig)
			err := r.CompVsCompStatic(true)
			if err != nil {
				log.Err(err).Msg("game-runner")
				continue
			}
			mcg = r.Game()
			pzls, err := puzzles.CreatePuzzlesFromGame(ctx, gameStore, puzzlesStore, mcg, "", pb.GameType_BOT_VS_BOT)
			for _, pzl := range pzls {
				fmt.Printf("liwords.localhost/game/%s?turn=%d\n", pzl.GetGameId(), pzl.GetTurnNumber()+1)
			}
			if err != nil {
				log.Err(err).Msg("create-puzzles-from-game")
			}
		}
	}
}
