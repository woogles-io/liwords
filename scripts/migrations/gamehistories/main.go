package main

import (
	"context"
	"os"

	"github.com/woogles-io/liwords/rpc/api/proto/vendored/macondo"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
)

const GameChunkSize = 1000

const (
	CurrentGameHistoryVersion = 2
)

// This contains functions for migrating a game in-place.
// The Macondo GameHistory will occasionally, hopefully very rarely,
// change schemas, and thus we need to be able to migrate when these
// change.

func MigrateGameHistory(gh *macondo.GameHistory) (*macondo.GameHistory, bool) {
	if gh.Version < 2 {
		// Either 0 (unspecified) or 1
		// Migrate to v2.
		return migrateToV2(gh), true
	}
	// Otherwise, return the history as is.
	return gh, false
}

func migrateToV2(gh *macondo.GameHistory) *macondo.GameHistory {
	// Version 2 of a macondo GameHistory works as such:
	// - id_auth should be set to `io.woogles`
	// - second_went_first is deprecated. If it is true, we need to flip
	// the order of the players instead and set it to false.
	//   -- this will also affect the `winner`
	// - the `nickname` field in each GameEvent of the history is deprecated.
	//    we should instead use player_index

	gh2 := proto.Clone(gh).(*macondo.GameHistory)

	if len(gh.Players) != 2 {
		log.Error().Interface("players", gh.Players).Str("gid", gh.Uid).
			Msg("bad-gh-players")
		return gh
	}

	gh2.Version = 2
	gh2.IdAuth = "io.woogles"

	if gh2.SecondWentFirst {
		gh2.SecondWentFirst = false
		gh2.Players[0], gh2.Players[1] = gh2.Players[1], gh2.Players[0]
		if gh2.Winner == 0 {
			gh2.Winner = 1
		} else if gh2.Winner == 1 {
			gh2.Winner = 0
		} // otherwise it's a tie
		if len(gh2.FinalScores) == 2 {
			gh2.FinalScores[0], gh2.FinalScores[1] = gh2.FinalScores[1], gh2.FinalScores[0]
		}
		if len(gh2.LastKnownRacks) == 2 {
			gh2.LastKnownRacks[0], gh2.LastKnownRacks[1] = gh2.LastKnownRacks[1], gh2.LastKnownRacks[0]
		}
	}

	nickname0, nickname1 := gh2.Players[0].Nickname, gh2.Players[1].Nickname
	for _, evt := range gh2.Events {
		if evt.Nickname == nickname1 {
			evt.PlayerIndex = 1
		} else if evt.Nickname == nickname0 {
			evt.PlayerIndex = 0 // technically not necessary because of protobuf but let's be explicit
		}
		evt.Nickname = ""
	}

	return gh2
}

var migrateCount int

type GameFields struct {
	mdata               entity.Quickdata
	tdata               entity.Timers
	history             entity.GameHistory
	winnerIdx, loserIdx pgtype.Int4
	id                  int
}

func migrateGames(ctx context.Context, pool *pgxpool.Pool, games []*GameFields) error {
	query := `
	UPDATE games
	SET quickdata = $1, timers = $2, winner_idx = $3, loser_idx = $4, history = $5
	WHERE id = $6
	`
	tx, err := pool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	batch := &pgx.Batch{}

	for _, game := range games {
		hist := game.history

		if hist.SecondWentFirst {
			// Flip some relevant fields.
			if len(game.mdata.PlayerInfo) == 2 {
				game.mdata.PlayerInfo[0], game.mdata.PlayerInfo[1] =
					game.mdata.PlayerInfo[1], game.mdata.PlayerInfo[0]
			}
			if len(game.mdata.FinalScores) == 2 {
				game.mdata.FinalScores[0], game.mdata.FinalScores[1] =
					game.mdata.FinalScores[1], game.mdata.FinalScores[0]
			}
			if len(game.mdata.NewRatings) == 2 {
				game.mdata.NewRatings[0], game.mdata.NewRatings[1] =
					game.mdata.NewRatings[1], game.mdata.NewRatings[0]
			}
			if len(game.mdata.OriginalRatings) == 2 {
				game.mdata.OriginalRatings[0], game.mdata.OriginalRatings[1] =
					game.mdata.OriginalRatings[1], game.mdata.OriginalRatings[0]
			}
			if len(game.tdata.TimeRemaining) == 2 {
				game.tdata.TimeRemaining[0], game.tdata.TimeRemaining[1] =
					game.tdata.TimeRemaining[1], game.tdata.TimeRemaining[0]
			}
			game.winnerIdx, game.loserIdx = game.loserIdx, game.winnerIdx
		}

		gh, migrated := MigrateGameHistory(&hist.GameHistory)
		if migrated {
			hist.GameHistory = *gh
			migrateCount++
		}
		if gh.SecondWentFirst {
			log.Fatal().Msg("migration-failed?")
		}
		batch.Queue(query, game.mdata, game.tdata, game.winnerIdx, game.loserIdx, &hist,
			game.id)
	}

	br := tx.SendBatch(ctx, batch)
	ct, err := br.Exec()
	if err != nil {
		log.Err(err).Msg("?")
		return err
	}
	// must close batch before reusing connection for the commit.
	err = br.Close()
	if err != nil {
		log.Err(err).Msg("??")
	}
	if err := tx.Commit(ctx); err != nil {
		log.Err(err).Msg("???")

		return err
	}

	log.Info().Int64("rows-affected", ct.RowsAffected()).Msg("batch-rows")

	return nil
}

func migrateBatches(pool *pgxpool.Pool) error {
	ctx := context.Background()

	// Only select games that have already ended
	query := `
	SELECT quickdata, timers, winner_idx, loser_idx, history, id
	FROM games
	WHERE game_end_reason != 0
	ORDER BY created_at DESC
	`

	rows, err := pool.Query(ctx, query)

	if err != nil {
		return err
	}
	defer rows.Close()

	ct := 0

	games := []*GameFields{}

	for rows.Next() {
		gf := &GameFields{}

		if err := rows.Scan(&gf.mdata, &gf.tdata, &gf.winnerIdx,
			&gf.loserIdx, &gf.history, &gf.id); err != nil {

			return err
		}
		// log.Info().Interface("history", history).Interface("mdata", mdata).Interface("tdata", tdata).Msg("test")
		ct += 1
		games = append(games, gf)
		if len(games) == GameChunkSize {
			err = migrateGames(ctx, pool, games)
			if err != nil {
				return err
			}
			games = []*GameFields{}
			log.Info().Int("ct", ct).Msg("...")
		}
	}
	if len(games) > 0 {
		err = migrateGames(ctx, pool, games)
		if err != nil {
			return err
		}
	}

	log.Info().Int("total-ct", ct).Int("migrated-ct", migrateCount).Msg("count")
	return nil
}

func main() {
	// Migrate all game histories to V2.
	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	pool, err := common.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		panic(err)
	}

	err = migrateBatches(pool)
	if err != nil {
		panic(err)
	}
}
