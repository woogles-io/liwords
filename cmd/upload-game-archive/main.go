// upload-game-archive archives finished games to S3 by rebuilding their
// GameHistory from game_turns rows. Use this to retry failed archives or to
// process a batch of games that still have pending turn rows.
//
// Requires the same env vars as liwords-api: DB_CONN_DSN, USE_MINIO_S3,
// MINIO_S3_ENDPOINT, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY,
// GAMEHISTORY_UPLOAD_BUCKET.
package main

import (
	"context"
	"flag"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

	"github.com/woogles-io/liwords/pkg/config"
	gamestore "github.com/woogles-io/liwords/pkg/stores/game"
	"github.com/woogles-io/liwords/pkg/stores/models"
	userstore "github.com/woogles-io/liwords/pkg/stores/user"
	"github.com/woogles-io/liwords/pkg/utilities"
)

func main() {
	gameID := flag.String("game-id", "", "Archive a single game by UUID")
	allPending := flag.Bool("all-pending", false, "Archive all finished games with pending turn rows")
	flag.Parse()

	if *gameID == "" && !*allPending {
		log.Fatal().Msg("specify --game-id <uuid> or --all-pending")
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	ctx := context.Background()

	cfg := &config.Config{}
	cfg.Load(nil)

	pool, err := pgxpool.New(ctx, cfg.DBConnUri)
	if err != nil {
		log.Fatal().Err(err).Msg("db-connect-failed")
	}
	defer pool.Close()

	bucket := os.Getenv("GAMEHISTORY_UPLOAD_BUCKET")
	if bucket == "" {
		log.Fatal().Msg("GAMEHISTORY_UPLOAD_BUCKET not set")
	}

	// Build S3 client (same logic as liwords-api/main.go).
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
	otelaws.AppendMiddlewares(&awscfg.APIOptions)
	s3Client := s3.NewFromConfig(awscfg, utilities.CustomClientOptions)

	userStore, err := userstore.NewDBStore(pool)
	if err != nil {
		log.Fatal().Err(err).Msg("user-store-failed")
	}
	dbStore, err := gamestore.NewDBStore(cfg, userStore, pool)
	if err != nil {
		log.Fatal().Err(err).Msg("game-store-failed")
	}
	cache := gamestore.NewCache(dbStore)

	archiver := gamestore.NewHistoryArchiver(bucket, s3Client, cache)
	queries := models.New(pool)

	var gameIDs []string
	if *gameID != "" {
		gameIDs = []string{*gameID}
	} else {
		rows, err := queries.ListPendingArchival(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("list-pending-failed")
		}
		for _, r := range rows {
			gameIDs = append(gameIDs, r.String)
		}
		log.Info().Int("count", len(gameIDs)).Msg("pending-games-found")
	}

	ok, failed := 0, 0
	for _, id := range gameIDs {
		g, err := cache.Get(ctx, id)
		if err != nil {
			log.Error().Err(err).Str("gameID", id).Msg("load-failed")
			failed++
			continue
		}
		if err := archiver.ArchiveAndCleanup(ctx, g); err != nil {
			log.Error().Err(err).Str("gameID", id).Msg("archive-failed")
			failed++
			continue
		}
		ok++
	}
	log.Info().Int("ok", ok).Int("failed", failed).Msg("done")
}
