package main

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/stores/tournament"
	"github.com/domino14/liwords/pkg/stores/user"
	
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
)

func main() {
	// Migrate all club director users to be of the form uuid:username in the backend.
	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	userStore, err := user.NewDBStore(cfg.DBConnString)
	if err != nil {
		panic(err)
	}

	tmp, err := game.NewDBStore(cfg, userStore)
	if err != nil {
		panic(err)
	}
	gameStore := game.NewCache(tmp)

	ctx := context.Background()

	tournamentStore, err := tournament.NewDBStore(cfg, gameStore)
	if err != nil {
		panic(err)
	}

	ids, err := tournamentStore.ListAllIDs(ctx)
	if err != nil {
		panic(err)
	}

	for _, tid := range ids {
		t, err := tournamentStore.Get(ctx, tid)
		if err != nil {
			log.Err(err).Str("tid", tid).Msg("bug")
			continue
		}
		log.Info().Str("tid", tid).Msg("migrating")
		directors := t.Directors
		newDirectors := &realtime.TournamentPersons{
			Persons: []*realtime.TournamentPerson{},
		}
		for uuid, person := range directors.Persons {
			uuid := person.Id
			player, err := userStore.GetByUUID(ctx, uuid)
			if err != nil {
				log.Err(err).Str("uuid", uuid).Msg("err-userstore-get")
				panic(err)
			}
			person.Id = uuid+":"+player.Username
		}
		t.Directors = newDirectors
		err = tournamentStore.Set(ctx, t)
		if err != nil {
			panic(err)
		}
		log.Info().Str("tid", tid).
			Interface("newDirectors", newDirectors).Msg("migrating")
	}
}
