// Command backfill-analysis-s3 moves analysis results out of
// analysis_jobs.result into the analysis bucket, in two passes:
//
//  1. (default) upload each result still held only in the column to its own
//     object and record result_s3_key. The column is left in place.
//  2. --clear-db: once that has run and been checked, drop the column's copy
//     for jobs whose object matches it (and zero-turn jobs, which have no
//     object). Needs cmd/backfill-analysis-columns to have run first; jobs
//     without summary columns are left alone.
//
// Both passes are safe to stop (Ctrl-C) and re-run; --start-after resumes from
// the cursor in the progress logs.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/analysis"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/utilities"
)

func main() {
	clearDB := flag.Bool("clear-db", false, "drop the result column's copy of results already uploaded and verified")
	workers := flag.Int("workers", 8, "parallel workers")
	batchSize := flag.Int("batch-size", 200, "jobs per batch")
	maxJobs := flag.Int("max-jobs", 0, "stop after this many jobs (0 = all; for canary runs)")
	startAfter := flag.String("start-after", "", "resume after this job id")
	dryRun := flag.Bool("dry-run", false, "read and check, but don't write anything")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var after uuid.UUID
	if *startAfter != "" {
		var err error
		if after, err = uuid.Parse(*startAfter); err != nil {
			log.Fatal().Err(err).Msg("invalid --start-after")
		}
	}

	cfg := &config.Config{}
	cfg.Load(nil)

	pool, err := pgxpool.New(ctx, cfg.DBConnUri)
	if err != nil {
		log.Fatal().Err(err).Msg("db-connect-failed")
	}
	defer pool.Close()

	bucket := os.Getenv("ANALYSIS_UPLOAD_BUCKET")
	if bucket == "" {
		log.Fatal().Msg("ANALYSIS_UPLOAD_BUCKET not set")
	}
	store := analysis.NewS3ResultStore(newS3Client(ctx), bucket)

	start := time.Now()
	opts := analysis.BackfillOptions{
		Workers:    *workers,
		BatchSize:  *batchSize,
		StartAfter: after,
		MaxJobs:    *maxJobs,
		DryRun:     *dryRun,
		Progress: func(s analysis.BackfillStats) {
			log.Info().Int64("processed", s.Processed).Int64("done", s.Done).
				Int64("skipped", s.Skipped).Int64("failed", s.Failed).
				Str("cursor", s.Cursor.String()).
				Float64("per_sec", float64(s.Processed)/time.Since(start).Seconds()).
				Msg("progress")
		},
	}

	queries := models.New(pool)
	var stats analysis.BackfillStats
	if *clearDB {
		stats, err = analysis.ClearUploadedResults(ctx, queries, store, opts)
	} else {
		stats, err = analysis.UploadResults(ctx, queries, store, opts)
	}
	l := log.Info()
	if err != nil {
		l = log.Error().Err(err)
	}
	l.Bool("clear_db", *clearDB).Bool("dry_run", *dryRun).
		Int64("processed", stats.Processed).Int64("done", stats.Done).
		Int64("skipped", stats.Skipped).Int64("failed", stats.Failed).
		Str("cursor", stats.Cursor.String()).Dur("elapsed", time.Since(start)).
		Msg("finished")
	if err != nil || stats.Failed > 0 {
		os.Exit(1)
	}
}

// newS3Client mirrors cmd/liwords-api's S3 setup, including MinIO for local dev.
func newS3Client(ctx context.Context) *s3.Client {
	var configOpts []func(*awsconfig.LoadOptions) error
	if os.Getenv("USE_MINIO_S3") == "1" {
		configOpts = append(configOpts, awsconfig.WithRegion("us-east-1"))
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		if accessKey != "" && secretKey != "" {
			configOpts = append(configOpts, awsconfig.WithCredentialsProvider(
				aws.NewCredentialsCache(aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
					return aws.Credentials{AccessKeyID: accessKey, SecretAccessKey: secretKey, Source: "Environment"}, nil
				})),
			))
		}
	}
	awscfg, err := awsconfig.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		log.Fatal().Err(err).Msg("aws-config-failed")
	}
	return s3.NewFromConfig(awscfg, utilities.CustomClientOptions)
}
