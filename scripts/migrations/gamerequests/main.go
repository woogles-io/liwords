package main

import (
	"context"
	"fmt"
	"os"

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

	log.Info().Msg("Starting game request migration from bytea to jsonb")

	// Count total rows to migrate
	var totalRows int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM games WHERE request IS NOT NULL AND game_request IS NULL").Scan(&totalRows)
	if err != nil {
		return fmt.Errorf("failed to count rows: %w", err)
	}

	log.Info().Int("total_rows", totalRows).Msg("Found rows to migrate")

	if totalRows == 0 {
		log.Info().Msg("No rows to migrate")
		return nil
	}

	processed := 0
	offset := 0

	for processed < totalRows {
		tx, err := pool.BeginTx(ctx, common.DefaultTxOptions)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Select batch of games with null game_request but non-null request
		query := `
			SELECT id, uuid, request
			FROM games
			WHERE request IS NOT NULL AND game_request IS NULL
			ORDER BY id
			LIMIT $1 OFFSET $2
			FOR UPDATE
		`

		rows, err := tx.Query(ctx, query, batchSize, offset)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to query games: %w", err)
		}

		batchCount := 0
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
				batchCount++
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
				batchCount++
				continue
			}

			// Update the game_request column
			updateQuery := "UPDATE games SET game_request = $1 WHERE id = $2"
			_, err = tx.Exec(ctx, updateQuery, string(jsonBytes), gameID)
			if err != nil {
				rows.Close()
				tx.Rollback(ctx)
				return fmt.Errorf("failed to update game %d: %w", gameID, err)
			}

			batchCount++
		}
		rows.Close()

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		processed += batchCount
		offset += batchSize

		log.Info().
			Int("processed", processed).
			Int("total", totalRows).
			Int("batch_size", batchCount).
			Msg("Migrated batch")

		if batchCount == 0 {
			// No more rows to process
			break
		}
	}

	log.Info().Int("total_migrated", processed).Msg("Migration completed")
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
