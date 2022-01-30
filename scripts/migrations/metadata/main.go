package main

import (
	"context"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/game"
	"github.com/domino14/liwords/pkg/stores/user"
	ipc "github.com/domino14/liwords/rpc/api/proto/ipc"
)

func main() {
	// Populate every game with its quickdata
	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	userStore, err := user.NewDBStore(cfg.DBConnString)
	if err != nil {
		panic(err)
	}

	gameStore, err := game.NewDBStore(cfg, userStore)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	ids, err := gameStore.ListAllIDs(ctx)
	if err != nil {
		panic(err)
	}
	log.Info().Interface("ids", ids).Msg("listed-game-ids")

	for _, gid := range ids {
		g, err := gameStore.Get(ctx, gid)
		if err != nil {
			log.Err(err).Str("gid", gid).Msg("bug")
			continue
		}

		// Currently, Quickdata contains the following:
		//   OriginalRequestId
		//   FinalScores
		//   UserInfo
		//
		// Technically, the OriginalRequestId is not the same
		// as the RequestId in the GameRequest, but we set
		// it here just so it's not null and it doesn't matter
		// because it's only used to obtain current rematch streaks.

		playerinfos := make([]*ipc.PlayerInfo, 2)

		for idx, u := range g.History().Players {
			first := idx == 0
			if g.History().SecondWentFirst {
				first = !first
			}

			playerinfos[idx] = &ipc.PlayerInfo{
				Nickname: u.Nickname,
				UserId:   u.UserId,
				// This migration script will only be run once.
				IsBot: strings.HasPrefix(u.Nickname, "MacondoBot"),
				First: first,
			}
		}

		quickdata := &entity.Quickdata{OriginalRequestId: g.GameReq.RequestId,
			FinalScores: g.History().FinalScores,
			PlayerInfo:  playerinfos,
		}

		g.Quickdata = quickdata

		// Write the game back to the database
		gameStore.Set(ctx, g)
	}
}
