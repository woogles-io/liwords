package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/analysis"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Print games that would be enqueued without actually doing it")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	ctx := context.Background()

	cfg := &config.Config{}
	cfg.Load(nil)

	pool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	queries := models.New(pool)

	// Find all completed league games that don't already have an analysis job.
	// game_end_reason NOT IN (0, 5, 7) excludes NONE (ongoing), ABORTED, CANCELLED.
	rows, err := pool.Query(ctx, `
		SELECT g.uuid
		FROM games g
		WHERE g.league_division_id IS NOT NULL
		  AND g.game_end_reason NOT IN (0, 5, 7)
		  AND NOT EXISTS (
		      SELECT 1 FROM analysis_jobs aj WHERE aj.game_id = g.uuid
		  )
		ORDER BY g.created_at DESC
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to query league games")
	}
	defer rows.Close()

	var gameIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Fatal().Err(err).Msg("failed to scan row")
		}
		gameIDs = append(gameIDs, id)
	}
	if err := rows.Err(); err != nil {
		log.Fatal().Err(err).Msg("error iterating rows")
	}

	log.Info().Int("count", len(gameIDs)).Bool("dry_run", *dryRun).Msg("league games without analysis jobs")

	if *dryRun {
		const preview = 10
		for i, id := range gameIDs {
			if i >= preview {
				fmt.Printf("... and %d more\n", len(gameIDs)-preview)
				break
			}
			fmt.Println(id)
		}
		return
	}

	enqueued := 0
	for _, id := range gameIDs {
		if err := analysis.EnqueueGameForAnalysis(ctx, queries, id, 0); err != nil {
			log.Error().Err(err).Str("game_id", id).Msg("failed to enqueue game")
		} else {
			enqueued++
		}
	}

	log.Info().Int("enqueued", enqueued).Int("total", len(gameIDs)).Msg("done")
}
