package main

import (
	"context"
	"fmt"
	"os"

	commondb "github.com/domino14/liwords/pkg/stores/common"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/puzzles"
	gamestore "github.com/domino14/liwords/pkg/stores/game"
	puzzlesstore "github.com/domino14/liwords/pkg/stores/puzzles"
	"github.com/domino14/liwords/pkg/stores/user"
	pb "github.com/domino14/liwords/rpc/api/proto/puzzle_service"
)

// Example:
// go run . '{"bot_vs_bot":true,"lexicon":"CSW21","letter_distribution":"english","sql_offset":0,"game_consideration_limit":1000000,"game_creation_limit":100,"request":{"buckets":[{"size":50,"includes":["EQUITY"],"excludes":[]}]}}'

func main() {
	req := &pb.PuzzleGenerationJobRequest{}
	err := protojson.Unmarshal([]byte(os.Args[1]), req)
	if err != nil {
		panic(err)
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
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

	tempgs, err := gamestore.NewDBStore(cfg, us)
	if err != nil {
		panic(err)
	}

	gs := gamestore.NewCache(tempgs)
	if err != nil {
		panic(err)
	}

	ps, err := puzzlesstore.NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	id, err := puzzles.Generate(ctx, cfg, gs, ps, req)
	if err != nil {
		panic(err)
	}
	info, err := puzzles.GetJobInfoString(ctx, ps, id)
	if err != nil {
		panic(err)
	}
	fmt.Println(info)
}
