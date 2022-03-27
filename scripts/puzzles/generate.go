package main

import (
	"context"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lithammer/shortuuid"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/common"
	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/puzzles"
	commondb "github.com/domino14/liwords/pkg/stores/common"
	"github.com/domino14/liwords/pkg/stores/game"
	puzzlesstore "github.com/domino14/liwords/pkg/stores/puzzles"
	"github.com/domino14/liwords/pkg/stores/user"
	"github.com/domino14/macondo/alphabet"
	"github.com/domino14/macondo/cross_set"
	"github.com/domino14/macondo/gaddag"
	macondogame "github.com/domino14/macondo/game"

	"github.com/domino14/liwords/rpc/api/proto/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/macondo/automatic"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/namsral/flag"
)

func newBotvBotPuzzleGame(mcg *macondogame.Game) *entity.Game {
	g := entity.NewGame(mcg, common.DefaultGameReq)
	g.Started = true
	uuid := shortuuid.New()
	g.GameEndReason = ipc.GameEndReason_STANDARD
	g.Quickdata.FinalScores = []int32{int32(g.Game.PointsFor(0)), int32(g.Game.PointsFor(1))}
	g.Quickdata.PlayerInfo = []*ipc.PlayerInfo{&common.DefaultPlayerOneInfo, &common.DefaultPlayerTwoInfo}
	// add a fake uuid for each user
	g.Game.History().Players[0].UserId = common.DefaultPlayerOneInfo.UserId
	g.Game.History().Players[1].UserId = common.DefaultPlayerTwoInfo.UserId
	g.Game.History().Uid = uuid
	g.Game.History().PlayState = macondopb.PlayState_GAME_OVER
	g.Timers = entity.Timers{
		TimeRemaining: []int{0, 0},
		MaxOvertime:   0,
	}

	return g
}

func main() {
	gf := flag.NewFlagSet("gf", flag.ContinueOnError)
	numGames := gf.Int("i", 10, "number of bot vs bot games used to create puzzles")
	useBotVsBot := gf.Bool("b", false, "use bot vs bot games to create puzzles")
	lexicon := gf.String("lex", common.DefaultLexicon, "use lexicon to generate puzzles")
	letterDistribution := gf.String("ld", common.DefaultLetterDistribution, "letter distribution for puzzles")
	sqlLimit := gf.Int("limit", 100, "sql limit to consider")
	sqlOffset := gf.Int("offset", 0, "sql offset")
	err := gf.Parse(os.Args[1:])
	if err != nil {
		panic(err)
	}

	cfg := &config.Config{}
	// Only load config from environment variables:
	cfg.Load(nil)
	cfg.MacondoConfig.DefaultLexicon = *lexicon
	cfg.MacondoConfig.DefaultLetterDistribution = *letterDistribution

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Info().Msgf("Loaded config: %v", cfg)

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
		rows, err := db.QueryContext(ctx,
			`SELECT uuid FROM games WHERE games.id NOT IN
				(SELECT game_id FROM puzzles) AND
				(stats->'d1'->'Unchallenged Phonies'->'t')::int = 0 AND
				(stats->'d2'->'Unchallenged Phonies'->'t')::int = 0 AND
				game_end_reason != 0 LIMIT $1 OFFSET $2`, *sqlLimit, *sqlOffset)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		numGames := 0
		numPuzzles := 0

		// For non-bot-v-bot games we need to "hydrate" the game we get back
		// from the database with the right data structures in order for it
		// to generate moves properly.
		gd, err := gaddag.Get(&cfg.MacondoConfig, *lexicon)
		if err != nil {
			panic(err)
		}
		dist, err := alphabet.Get(&cfg.MacondoConfig, *letterDistribution)
		if err != nil {
			panic(err)
		}
		csgen := cross_set.GaddagCrossSetGenerator{Dist: dist, Gaddag: gd}

		for rows.Next() {
			var UUID string
			if err := rows.Scan(&UUID); err != nil {
				log.Err(err).Msg("games-scan")
			}
			entGame, err := gameStore.Get(ctx, UUID)
			if err != nil {
				log.Err(err).Msg("games-store")
			}
			if entGame.GameReq.Lexicon != *lexicon {
				continue
			}
			if entGame.GameReq.Rules.LetterDistributionName != *letterDistribution {
				continue
			}
			_, variant, err := entity.VariantFromGameReq(entGame.GameReq)
			if err != nil {
				panic(err)
			}
			if variant != macondogame.VarClassic {
				continue
			}
			// Set cross-set generator so that it can actually generate moves.
			entGame.Game.SetCrossSetGen(csgen)
			pzls, err := puzzles.CreatePuzzlesFromGame(ctx, gameStore, puzzlesStore, entGame, "", pb.GameType_NATIVE)
			for _, pzl := range pzls {
				numPuzzles += 1
				fmt.Printf("liwords.localhost/game/%s?turn=%d\n", pzl.GetGameId(), pzl.GetTurnNumber()+1)
			}
			if len(pzls) > 0 {
				numGames += 1
			}
			if err != nil {
				fmt.Println(err.Error())
				log.Err(err).Msg("create-puzzles-from-game")
			}
		}
		log.Info().Msgf("created %d puzzles from %d games", numPuzzles, numGames)
	} else {
		for i := 0; i < *numGames; i++ {
			r := automatic.NewGameRunner(nil, &cfg.MacondoConfig)
			err := r.CompVsCompStatic(true)
			if err != nil {
				log.Err(err).Msg("game-runner")
				continue
			}
			g := newBotvBotPuzzleGame(r.Game())
			pzls, err := puzzles.CreatePuzzlesFromGame(ctx, gameStore, puzzlesStore, g, "", pb.GameType_BOT_VS_BOT)
			for _, pzl := range pzls {
				fmt.Printf("liwords.localhost/game/%s?turn=%d\n", pzl.GetGameId(), pzl.GetTurnNumber()+1)
			}
			if err != nil {
				log.Err(err).Msg("create-puzzles-from-game")
			}
		}
	}
}
