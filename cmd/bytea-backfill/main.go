// bytea-backfill uploads GameHistory protos stored as bytea in games.history
// directly to S3, without going through the game_turns assembly path.
// Use this to archive the ~11M historical games that predate the dual-write
// schema.
//
// Requires the same env vars as liwords-api: DB_CONN_DSN, USE_MINIO_S3,
// MINIO_S3_ENDPOINT, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY,
// GAMEHISTORY_UPLOAD_BUCKET.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	gamestore "github.com/woogles-io/liwords/pkg/stores/game"
	"github.com/woogles-io/liwords/pkg/stores/models"
	userstore "github.com/woogles-io/liwords/pkg/stores/user"
	"github.com/woogles-io/liwords/pkg/utilities"
)

func main() {
	workers := flag.Int("workers", 16, "Number of parallel upload workers")
	batchSize := flag.Int("batch-size", 500, "Number of rows to fetch per DB query")
	maxGames := flag.Int("max-games", 0, "Stop after this many games (0 = unlimited; for canary runs)")
	startAfter := flag.String("start-after", "", "Resume from this UUID (exclusive keyset cursor)")
	dryRun := flag.Bool("dry-run", false, "Simulate without uploading or writing to DB")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := &config.Config{}
	cfg.Load(nil)

	poolCfg, err := pgxpool.ParseConfig(cfg.DBConnUri)
	if err != nil {
		log.Fatal().Err(err).Msg("db-parse-config-failed")
	}
	poolCfg.MaxConns = 32
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("db-connect-failed")
	}
	defer pool.Close()

	bucket := os.Getenv("GAMEHISTORY_UPLOAD_BUCKET")
	if bucket == "" {
		log.Fatal().Msg("GAMEHISTORY_UPLOAD_BUCKET not set")
	}

	var configOpts []func(*awsconfig.LoadOptions) error
	if os.Getenv("USE_MINIO_S3") == "1" {
		configOpts = append(configOpts, awsconfig.WithRegion("us-east-1"))
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		if accessKey != "" && secretKey != "" {
			configOpts = append(configOpts, awsconfig.WithCredentialsProvider(
				aws.NewCredentialsCache(aws.CredentialsProviderFunc(func(_ context.Context) (aws.Credentials, error) {
					return aws.Credentials{AccessKeyID: accessKey, SecretAccessKey: secretKey, Source: "Environment"}, nil
				})),
			))
		}
	}
	awscfg, err := awsconfig.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		log.Fatal().Err(err).Msg("aws-config-failed")
	}
	s3Client := s3.NewFromConfig(awscfg, utilities.CustomClientOptions)

	userStore, err := userstore.NewDBStore(pool)
	if err != nil {
		log.Fatal().Err(err).Msg("user-store-failed")
	}
	dbStore, err := gamestore.NewDBStore(cfg, userStore, pool)
	if err != nil {
		log.Fatal().Err(err).Msg("game-store-failed")
	}

	archiver := gamestore.NewHistoryArchiver(bucket, s3Client, dbStore)
	queries := models.New(pool)

	var (
		processed atomic.Int64
		succeeded atomic.Int64
		failed    atomic.Int64
	)
	afterUUID := *startAfter
	startTime := time.Now()

	// Progress reporter: logs stats every 60s until ctx is cancelled.
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				p := processed.Load()
				elapsed := time.Since(startTime).Seconds()
				rate := float64(p) / elapsed
				log.Info().
					Int64("processed", p).
					Int64("ok", succeeded.Load()).
					Int64("failed", failed.Load()).
					Str("cursor", afterUUID).
					Float64("rate_per_sec", rate).
					Msg("bytea-backfill-progress")
			case <-ctx.Done():
				return
			}
		}
	}()

	sem := make(chan struct{}, *workers)
	var wg sync.WaitGroup

	for {
		if *maxGames > 0 && processed.Load() >= int64(*maxGames) {
			break
		}
		if ctx.Err() != nil {
			log.Info().Msg("bytea-backfill-interrupted")
			break
		}

		rows, err := queries.ListByteaBackfillBatch(ctx, models.ListByteaBackfillBatchParams{
			AfterUuid: pgtype.Text{String: afterUUID, Valid: true},
			Lim:       int32(*batchSize),
		})
		if err != nil {
			log.Fatal().Err(err).Str("cursor", afterUUID).Msg("bytea-backfill-list-failed")
		}
		if len(rows) == 0 {
			break
		}

		for _, r := range rows {
			row := r
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				gameID := row.Uuid.String
				if *dryRun {
					succeeded.Add(1)
					processed.Add(1)
					return
				}
				_, uerr := archiver.ArchiveBytea(ctx, gameID, row.CreatedAt.Time, row.History)
				if uerr != nil {
					log.Error().Err(uerr).Str("gameID", gameID).Msg("bytea-backfill-row-error")
					failed.Add(1)
				} else {
					succeeded.Add(1)
				}
				processed.Add(1)
			}()
		}
		wg.Wait()

		// Advance the keyset cursor past the last row in the batch, whether or
		// not it succeeded. Failures stay in the partial index and are picked up
		// on the next invocation.
		afterUUID = rows[len(rows)-1].Uuid.String
	}

	stop() // cancel progress reporter context

	log.Info().
		Int64("processed", processed.Load()).
		Int64("ok", succeeded.Load()).
		Int64("failed", failed.Load()).
		Str("last_uuid", afterUUID).
		Bool("dry_run", *dryRun).
		Msg("bytea-backfill-done")
}
