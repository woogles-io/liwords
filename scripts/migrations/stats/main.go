package main

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/stores/user"
)

func main() {
	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	userStore, err := user.NewDBStore(cfg.DBConnString)
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
		err = userStore.ResetStatsAndRatings(ctx, uid)
		if err != nil {
			log.Err(err).Str("uid", uid).Msg("stats-reset-failure")
		}
	}
}
