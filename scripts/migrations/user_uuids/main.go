package main

import (
	"context"
	"os"

	"github.com/lithammer/shortuuid/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/user"
)

func main() {
	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	pool, err := common.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		panic(err)
	}

	userStore, err := user.NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	ids, err := userStore.ListAllIDs(ctx)
	if err != nil {
		panic(err)
	}
	log.Info().Int("ids", len(ids)).Msg("count-user-ids")

	tx, err := pool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)

	for _, id := range ids {
		uuid, err := shortuuid.DefaultEncoder.Decode(id)
		if err != nil {
			panic(err)
		}

		_, err = tx.Exec(ctx, "UPDATE users SET entity_uuid = $1 WHERE id = $2", uuid, id)
		if err != nil {
			panic(err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		panic(err)
	}
}
