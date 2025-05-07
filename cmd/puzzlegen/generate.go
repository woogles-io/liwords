package main

import (
	"context"
	"fmt"
	"os"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	commondb "github.com/woogles-io/liwords/pkg/stores/common"
	"google.golang.org/protobuf/encoding/protojson"

	macondoconfig "github.com/domino14/macondo/config"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/puzzles"
	gamestore "github.com/woogles-io/liwords/pkg/stores/game"
	puzzlesstore "github.com/woogles-io/liwords/pkg/stores/puzzles"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/puzzle_service"
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
	log.Info().Interface("req", req).Msg("requested-job")
	cfg := &config.Config{}
	// Only load config from environment variables:
	cfg.Load(nil)
	cfg.MacondoConfig().Set(macondoconfig.ConfigDefaultLexicon, req.Lexicon)
	cfg.MacondoConfig().Set(macondoconfig.ConfigDefaultLetterDistribution, req.LetterDistribution)
	ctx := context.Background()

	pool, err := commondb.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		panic(err)
	}

	us, err := user.NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	tempgs, err := gamestore.NewDBAndS3Store(cfg, us, pool, nil, "")
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
