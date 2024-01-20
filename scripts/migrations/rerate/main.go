package main

import (
	"context"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/game"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func main() {
	// Rerate all games in game store. Assumes that ratings are blanked out.
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
		var winner string
		winnerIdx := g.GetWinnerIdx()
		if winnerIdx == 0 || winnerIdx == -1 {
			winner = g.History().Players[0].Nickname
		} else if winnerIdx == 1 {
			winner = g.History().Players[1].Nickname
		}

		scores := map[string]int32{
			g.History().Players[0].Nickname: int32(g.PointsFor(0)),
			g.History().Players[1].Nickname: int32(g.PointsFor(1))}
		ratings := map[string][2]int32{}
		if g.CreationRequest().RatingMode == pb.RatingMode_RATED {
			timeStarted := g.Timers.TimeStarted / 1000
			ratings, err = gameplay.Rate(ctx, scores, g, winner, userStore, timeStarted)
			if err != nil {
				panic(err)
			}
			log.Info().Interface("ratings", ratings).Str("gameID", gid).Msg("rated")
		}
	}
}
