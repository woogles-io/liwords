package main

import (
	"context"

	commondb "github.com/domino14/liwords/pkg/stores/common"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/puzzles"
	gamestore "github.com/domino14/liwords/pkg/stores/game"
	puzzlesstore "github.com/domino14/liwords/pkg/stores/puzzles"
	"github.com/domino14/liwords/pkg/stores/user"
	pb "github.com/domino14/liwords/rpc/api/proto/puzzle_service"
)

func main() {
	// gf := flag.NewFlagSet("gf", flag.ContinueOnError)
	// reqString := gf.String("json", "", "JSON string puzzle generation job request")
	var req *pb.PuzzleGenerationJobRequest
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

	err = puzzles.Generate(ctx, cfg, pool, gs, ps, req, true)
	if err != nil {
		panic(err)
	}
}
