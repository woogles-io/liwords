package main

import (
	"context"
	"fmt"

	commondb "github.com/domino14/liwords/pkg/stores/common"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/puzzles"
	gamestore "github.com/domino14/liwords/pkg/stores/game"
	puzzlesstore "github.com/domino14/liwords/pkg/stores/puzzles"
	"github.com/domino14/liwords/pkg/stores/user"
	pb "github.com/domino14/liwords/rpc/api/proto/puzzle_service"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

// Example arg that kind of works
// '{"BotVsBot":true,"Lexicon":"CSW21","LetterDistribution":"english","SqlOffset":0,"GameConsiderationLimit":1000000,"GameCreationLimit":100,"Request":{"Buckets":[{"Size":50,"Includes":[0],"Excludes":[]}]}}'

func main() {
	req := &pb.PuzzleGenerationJobRequest{
		BotVsBot:               true,
		Lexicon:                "CSW21",
		LetterDistribution:     "english",
		SqlOffset:              0,
		GameConsiderationLimit: 1000,
		GameCreationLimit:      100,
		Request: &macondopb.PuzzleGenerationRequest{
			Buckets: []*macondopb.PuzzleBucket{
				{
					Size:     50,
					Includes: []macondopb.PuzzleTag{macondopb.PuzzleTag_EQUITY},
					Excludes: []macondopb.PuzzleTag{},
				},
			},
		},
	}

	cfg := &config.Config{}
	// Only load config from environment variables:
	cfg.Load(nil)
	cfg.MacondoConfig.DefaultLexicon = req.Lexicon
	cfg.MacondoConfig.DefaultLetterDistribution = req.LetterDistribution
	ctx := context.Background()

	pool, err := commondb.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		panic(err)
	}

	us, err := user.NewDBStore(commondb.PostgresConnDSN(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode))
	if err != nil {
		panic(err)
	}

	gs, err := gamestore.NewDBStore(cfg, us)
	if err != nil {
		panic(err)
	}

	ps, err := puzzlesstore.NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	id, err := puzzles.Generate(ctx, cfg, pool, gs, ps, req)
	if err != nil {
		panic(err)
	}
	info, err := puzzles.GetJobInfoString(ctx, ps, id)
	if err != nil {
		panic(err)
	}
	fmt.Println(info)
}
