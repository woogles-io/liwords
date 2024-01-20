package main

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	pkgmod "github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/mod"
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
	notorietyStore, err := mod.NewDBStore(pool)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	ids, err := userStore.ListAllIDs(ctx)
	if err != nil {
		panic(err)
	}
	log.Info().Interface("ids", ids).Msg("listed-user-ids")

	for _, uid := range ids {
		err = pkgmod.ResetNotoriety(context.Background(), userStore, notorietyStore, uid)
		if err != nil {
			log.Err(err).Str("uid", uid).Msg("stats-reset-failure")
		}
	}
}
