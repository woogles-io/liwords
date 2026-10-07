// Command backfill-analysis-columns copies each analysis job's player mistake
// indexes and analysis version out of the stored result into the summary
// columns added in migration 202610060002. Jobs completed after that migration
// already have them; this fills in the older ones. It works in small batches so
// it never holds many job rows locked at once, and is safe to stop and re-run.
package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

func main() {
	batchSize := flag.Int("batch-size", 500, "jobs to update per batch")
	pause := flag.Duration("pause", 200*time.Millisecond, "pause between batches")
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

	var total int64
	start := time.Now()
	for {
		n, err := queries.BackfillAnalysisSummaryColumns(ctx, int32(*batchSize))
		if err != nil {
			log.Fatal().Err(err).Int64("updated_so_far", total).Msg("batch failed")
		}
		if n == 0 {
			break
		}
		total += n
		log.Info().Int64("batch", n).Int64("total", total).Msg("backfilled")
		time.Sleep(*pause)
	}
	log.Info().Int64("total", total).Dur("elapsed", time.Since(start)).Msg("done")
}
