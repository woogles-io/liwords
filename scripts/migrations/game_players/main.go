package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/common"
)

// Migrate existing games to the game_players table.
// This populates game_players from the games table for all completed games.

func migrate(cfg *config.Config, pool *pgxpool.Pool, batchSize int) error {
	ctx := context.Background()
	migrationStart := time.Now()

	log.Info().Msg("Starting game_players migration from existing games")

	// Count total games to migrate (exclude ongoing and cancelled games)
	// Include ABORTED games (5) as they represent actual gameplay
	var totalRows int
	countQuery := `
		SELECT COUNT(*)
		FROM games
		WHERE game_end_reason NOT IN (0, 7) -- NONE (ongoing), CANCELLED
		  AND created_at IS NOT NULL
		  AND quickdata IS NOT NULL
		  AND quickdata->'pi'->0->>'user_id' IS NOT NULL
		  AND quickdata->'pi'->1->>'user_id' IS NOT NULL
	`
	err := pool.QueryRow(ctx, countQuery).Scan(&totalRows)
	if err != nil {
		return fmt.Errorf("failed to count rows: %w", err)
	}

	log.Info().Int("total_rows", totalRows).Msg("Found games to migrate")

	if totalRows == 0 {
		log.Info().Msg("No games to migrate")
		return nil
	}

	// Check if any data already exists in game_players
	var existingCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM game_players").Scan(&existingCount)
	if err != nil {
		return fmt.Errorf("failed to count existing game_players: %w", err)
	}

	if existingCount > 0 {
		log.Info().Int("existing_count", existingCount).Msg("Found existing data in game_players")
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

		// Select batch of games to migrate
		// Use quickdata PlayerInfo to determine correct turn order
		query := `
			SELECT
				id, uuid,
				game_end_reason, created_at, type,
				winner_idx, loser_idx,
				(quickdata->'s'->0)::int as first_player_score,
				(quickdata->'s'->1)::int as second_player_score,
				COALESCE(quickdata->>'o', '') as original_request_id,
				quickdata->'pi'->0->>'user_id' as first_player_uuid,
				quickdata->'pi'->1->>'user_id' as second_player_uuid
			FROM games
			WHERE id > $1
			  AND game_end_reason NOT IN (0, 7) -- NONE (ongoing), CANCELLED
			  AND created_at IS NOT NULL
			  AND quickdata IS NOT NULL
			  AND quickdata->'pi'->0->>'user_id' IS NOT NULL
			  AND quickdata->'pi'->1->>'user_id' IS NOT NULL
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

		// Collect all games to insert
		type gameData struct {
			ID                int32
			UUID              string
			GameEndReason     int32
			CreatedAt         *time.Time
			GameType          int32
			WinnerIdx         *int32
			LoserIdx          *int32
			FirstPlayerScore  *int32
			SecondPlayerScore *int32
			OriginalRequestID string
			FirstPlayerUUID   *string
			SecondPlayerUUID  *string
		}
		var games []gameData

		for rows.Next() {
			var g gameData
			if err := rows.Scan(
				&g.ID, &g.UUID,
				&g.GameEndReason, &g.CreatedAt, &g.GameType,
				&g.WinnerIdx, &g.LoserIdx,
				&g.FirstPlayerScore, &g.SecondPlayerScore, &g.OriginalRequestID,
				&g.FirstPlayerUUID, &g.SecondPlayerUUID,
			); err != nil {
				rows.Close()
				tx.Rollback(ctx)
				return fmt.Errorf("failed to scan row: %w", err)
			}

			games = append(games, g)
		}
		rows.Close()

		// Insert into game_players
		batchCount := 0
		for _, g := range games {
			// Skip games without proper PlayerInfo data or created_at
			if g.FirstPlayerUUID == nil || g.SecondPlayerUUID == nil || g.CreatedAt == nil {
				log.Debug().Int32("game_id", g.ID).Msg("skipping game due to missing data")
				continue
			}

			// Resolve UUIDs to user IDs
			var firstPlayerID, secondPlayerID int32
			err = tx.QueryRow(ctx, "SELECT id FROM users WHERE uuid = $1", *g.FirstPlayerUUID).Scan(&firstPlayerID)
			if err != nil {
				log.Warn().Str("uuid", *g.FirstPlayerUUID).Int32("game_id", g.ID).Msg("could not find first player UUID")
				continue
			}

			err = tx.QueryRow(ctx, "SELECT id FROM users WHERE uuid = $1", *g.SecondPlayerUUID).Scan(&secondPlayerID)
			if err != nil {
				log.Warn().Str("uuid", *g.SecondPlayerUUID).Int32("game_id", g.ID).Msg("could not find second player UUID")
				continue
			}

			// Use the scores directly extracted from the query
			var firstPlayerScore, secondPlayerScore int32
			if g.FirstPlayerScore != nil {
				firstPlayerScore = *g.FirstPlayerScore
			}
			if g.SecondPlayerScore != nil {
				secondPlayerScore = *g.SecondPlayerScore
			}

			// Determine won status for each player based on winner_idx
			// winner_idx still refers to the macondo game indices (0 = first player, 1 = second player)
			var firstPlayerWon, secondPlayerWon *bool
			if g.WinnerIdx != nil {
				if *g.WinnerIdx == 0 {
					t, f := true, false
					firstPlayerWon, secondPlayerWon = &t, &f
				} else if *g.WinnerIdx == 1 {
					t, f := true, false
					firstPlayerWon, secondPlayerWon = &f, &t
				}
				// If winner_idx is neither 0 nor 1, leave both as nil (tie)
			}

			// Insert for both players with correct ordering
			insertQuery := `
				INSERT INTO game_players (
					game_uuid, player_id, player_index, score, won,
					game_end_reason, created_at, game_type,
					opponent_id, opponent_score, original_request_id
				) VALUES
					($1, $2, 0, $3, $4, $5, $6, $7, $8, $9, $10),
					($1, $11, 1, $12, $13, $5, $6, $7, $2, $3, $10)
				ON CONFLICT (game_uuid, player_id) DO NOTHING
			`

			_, err = tx.Exec(ctx, insertQuery,
				g.UUID, firstPlayerID, firstPlayerScore, firstPlayerWon, g.GameEndReason,
				*g.CreatedAt, g.GameType, secondPlayerID, secondPlayerScore, g.OriginalRequestID,
				secondPlayerID, secondPlayerScore, secondPlayerWon,
			)
			if err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("failed to insert game_players for game %d: %w", g.ID, err)
			}

			batchCount++
			if g.ID > lastID {
				lastID = g.ID
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

	// Verify the migration
	var finalCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM game_players").Scan(&finalCount)
	if err != nil {
		return fmt.Errorf("failed to count final game_players: %w", err)
	}

	log.Info().Int("final_count", finalCount).Msg("Final game_players count")

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

	log.Info().Msg("Starting game_players migration")

	pool, err := common.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer pool.Close()

	// Use batch size of 500 to balance memory usage and transaction size
	batchSize := 500
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &batchSize)
	}

	log.Info().Int("batch_size", batchSize).Msg("Starting migration")

	if err := migrate(cfg, pool, batchSize); err != nil {
		log.Fatal().Err(err).Msg("Migration failed")
	}

	log.Info().Msg("Migration completed successfully")
}
