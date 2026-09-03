package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/analysis"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// Requeue all analyses completed with v0.12.3
// This properly handles league standings by using the RequeueAnalysis service method

func main() {
	dryRun := flag.Bool("dry-run", true, "Only show what would be requeued, don't actually requeue")
	verbose := flag.Bool("v", false, "Verbose logging")
	flag.Parse()

	// Setup logging
	if *verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	ctx := context.Background()

	// Load config
	cfg := &config.Config{}
	cfg.Load(nil)

	// Connect to database
	pool, err := pgxpool.New(ctx, cfg.DBConnDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	queries := models.New(pool)

	// Find all completed jobs with v0.12.3
	// Note: analysisVersion is an integer (1, 2, 3, etc.), analyzerVersion is the macondo version string
	rows, err := pool.Query(ctx, `
		SELECT
			aj.id,
			aj.game_id,
			aj.completed_at,
			aj.result->>'analyzerVersion' as version
		FROM analysis_jobs aj
		WHERE aj.status = 'completed'
		  AND aj.result->>'analyzerVersion' = 'v0.12.3'
		ORDER BY aj.completed_at DESC
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to query jobs")
	}
	defer rows.Close()

	type jobInfo struct {
		ID          string
		GameID      string
		CompletedAt time.Time
		Version     string
	}

	var jobs []jobInfo
	for rows.Next() {
		var j jobInfo
		if err := rows.Scan(&j.ID, &j.GameID, &j.CompletedAt, &j.Version); err != nil {
			log.Fatal().Err(err).Msg("failed to scan row")
		}
		jobs = append(jobs, j)
	}

	if len(jobs) == 0 {
		fmt.Println("No jobs found with v0.12.3")
		return
	}

	fmt.Printf("Found %d jobs with v0.12.3\n", len(jobs))

	if *dryRun {
		fmt.Println("\nDRY RUN - showing sample of jobs that would be requeued:")
		for i, j := range jobs {
			if i >= 10 {
				break
			}
			fmt.Printf("  %s - game %s - completed %s\n", j.ID, j.GameID, j.CompletedAt.Format(time.RFC3339))
		}
		fmt.Println("\nRun with -dry-run=false to actually requeue")
		fmt.Println("This will properly handle league standings by subtracting old mistake index before requeueing.")
		return
	}

	// Actually requeue using the helper function (which properly handles league standings)
	fmt.Println("\nRequeueing jobs (this will properly handle league standings)...")
	requeued := 0
	failed := 0

	for i, j := range jobs {
		log.Debug().
			Str("game_id", j.GameID).
			Msg("requeueing job")

		// Call the helper function with priority -1 (below league games at 0)
		err := analysis.RequeueJobByGameID(ctx, queries, j.GameID, -1)
		if err != nil {
			log.Error().Err(err).Str("game_id", j.GameID).Msg("failed to requeue")
			failed++
			continue
		}

		requeued++
		if (i+1)%100 == 0 {
			fmt.Printf("Progress: %d/%d jobs processed (%d succeeded, %d failed)\n", i+1, len(jobs), requeued, failed)
		}
	}

	fmt.Printf("\nDone! Requeued %d jobs, %d failed\n", requeued, failed)
}
