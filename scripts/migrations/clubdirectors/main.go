package main

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/game"
	"github.com/woogles-io/liwords/pkg/stores/tournament"
	"github.com/woogles-io/liwords/pkg/stores/user"

	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func main() {
	// Migrate all club director users to be of the form uuid:username in the backend.
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

	tmp, err := game.NewDBStore(cfg, userStore, pool)
	if err != nil {
		panic(err)
	}
	gameStore := game.NewCache(tmp)

	ctx := context.Background()

	tournamentStore, err := tournament.NewDBStore(pool, gameStore)
	if err != nil {
		panic(err)
	}

	ids, err := tournamentStore.ListAllIDs(ctx)
	if err != nil {
		panic(err)
	}

	for _, tid := range ids {
		// A one-off script: the user lookups below run under the lock, which
		// only matters for the API server's pool.
		t, lk, err := tournamentStore.GetForUpdate(ctx, tid)
		if err != nil {
			log.Err(err).Str("tid", tid).Msg("bug")
			continue
		}
		log.Info().Str("tid", tid).Msg("migrating")
		directors := t.Directors
		newDirectors := &ipc.TournamentPersons{
			Persons: []*ipc.TournamentPerson{},
		}
		for _, person := range directors.Persons {
			uuid := person.Id
			player, err := userStore.GetByUUID(ctx, uuid)
			if err != nil {
				log.Err(err).Str("uuid", uuid).Msg("err-userstore-get")
				lk.Release()
				panic(err)
			}
			person.Id = uuid + ":" + player.Username
		}
		t.Directors = newDirectors
		err = lk.Save(ctx)
		if err != nil {
			panic(err)
		}
		log.Info().Str("tid", tid).
			Interface("newDirectors", newDirectors).Msg("migrating")
	}
}
