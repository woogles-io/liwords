package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/common"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const GameChunkSize = 1000

var migrateCount int

type GameFields struct {
	mdata               entity.Quickdata
	tdata               entity.Timers
	history             entity.GameHistory
	winnerIdx, loserIdx sql.NullInt32
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

		gh, migrated := common.MigrateGameHistory(&hist.GameHistory)
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
