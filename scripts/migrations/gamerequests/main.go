package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Migrate game requests from protobuf bytea (request) to JSON (game_request).
// This migrates data from the old 'request' column to the new 'game_request' column.

func migrate(cfg *config.Config, pool *pgxpool.Pool, batchSize int) error {
	ctx := context.Background()
	migrationStart := time.Now()

	log.Info().Msg("Starting game request migration from bytea to jsonb")

	// Count total rows to migrate
	// Check for both NULL and empty JSON object '{}'
	var totalRows int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM games WHERE request IS NOT NULL AND (game_request IS NULL OR game_request::text = '{}')").Scan(&totalRows)
	if err != nil {
		return fmt.Errorf("failed to count rows: %w", err)
	}

	log.Info().Int("total_rows", totalRows).Msg("Found rows to migrate")

	if totalRows == 0 {
		log.Info().Msg("No rows to migrate")
		return nil
	}

	processed := 0
	var lastID int32 = 0

	log.Info().
		Int("batch_size", batchSize).
		Int32("starting_id", lastID).
		Msg("Beginning migration loop")

	for processed < totalRows {
		log.Debug().
			Int32("last_id", lastID).
			Int("processed_so_far", processed).
			Msg("Starting new batch")

		tx, err := pool.BeginTx(ctx, common.DefaultTxOptions)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Select batch of games with null or empty game_request but non-null request
		// Use ID-based pagination for better performance
		// Using jsonb = '{}' is much faster than ::text cast
		query := `
			SELECT id, uuid, request
			FROM games
			WHERE id > $1 
			  AND request IS NOT NULL 
			  AND game_request = '{}'::jsonb
			ORDER BY id
			LIMIT $2
		`

		batchStart := time.Now()
		log.Debug().
			Int32("last_id", lastID).
			Int("batch_size", batchSize).
			Msg("Executing query")

		rows, err := tx.Query(ctx, query, lastID, batchSize)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to query games: %w", err)
		}

		log.Debug().
			Dur("query_duration", time.Since(batchStart)).
			Msg("Query completed, processing rows")

		// Collect all games to update in this batch
		type gameUpdate struct {
			id       int32
			uuid     string
			jsonData string
		}
		var updates []gameUpdate

		for rows.Next() {
			var gameID int32
			var gameUUID string
			var requestBytes []byte

			if err := rows.Scan(&gameID, &gameUUID, &requestBytes); err != nil {
				rows.Close()
				tx.Rollback(ctx)
				return fmt.Errorf("failed to scan row: %w", err)
			}

			// Unmarshal protobuf data
			gameReq := &pb.GameRequest{}
			if err := proto.Unmarshal(requestBytes, gameReq); err != nil {
				log.Warn().
					Int32("game_id", gameID).
					Str("game_uuid", gameUUID).
					Err(err).
					Msg("Failed to unmarshal protobuf, skipping game")
				continue
			}

			// Convert to JSON
			jsonBytes, err := protojson.Marshal(gameReq)
			if err != nil {
				log.Warn().
					Int32("game_id", gameID).
					Str("game_uuid", gameUUID).
					Err(err).
					Msg("Failed to marshal to JSON, skipping game")
				continue
			}

			updates = append(updates, gameUpdate{
				id:       gameID,
				uuid:     gameUUID,
				jsonData: string(jsonBytes),
			})
		}
		rows.Close()

		// Now perform all updates
		batchCount := 0
		for _, update := range updates {
			updateQuery := "UPDATE games SET game_request = $1 WHERE id = $2"
			_, err = tx.Exec(ctx, updateQuery, update.jsonData, update.id)
			if err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("failed to update game %d: %w", update.id, err)
			}
			batchCount++
			// Track the last ID processed for next batch
			if update.id > lastID {
				lastID = update.id
			}
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		processed += batchCount
		batchDuration := time.Since(batchStart)
		elapsedTotal := time.Since(migrationStart)
		rowsPerSec := float64(processed) / elapsedTotal.Seconds()
		estimatedRemaining := time.Duration(0)
		if rowsPerSec > 0 {
			estimatedRemaining = time.Duration(float64(totalRows-processed)/rowsPerSec) * time.Second
		}

		log.Info().
			Int("processed", processed).
			Int("total", totalRows).
			Int("batch_size", batchCount).
			Dur("batch_duration", batchDuration).
			Dur("elapsed_total", elapsedTotal).
			Float64("rows_per_sec", rowsPerSec).
			Dur("estimated_remaining", estimatedRemaining).
			Msg("Migrated batch")

		if batchCount == 0 {
			// No more rows to process
			break
		}
	}

	totalDuration := time.Since(migrationStart)
	log.Info().
		Int("total_migrated", processed).
		Dur("total_duration", totalDuration).
		Float64("avg_rows_per_sec", float64(processed)/totalDuration.Seconds()).
		Msg("Migration completed")
	return nil
}

func main() {
	cfg := &config.Config{}
	cfg.Load(nil)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().Msg("Starting game request migration")

	pool, err := common.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer pool.Close()

	// Use batch size of 1000 to balance memory usage and transaction size
	batchSize := 1000
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &batchSize)
	}

	log.Info().Int("batch_size", batchSize).Msg("Starting migration")

	if err := migrate(cfg, pool, batchSize); err != nil {
		log.Fatal().Err(err).Msg("Migration failed")
	}

	log.Info().Msg("Migration completed successfully")
}
