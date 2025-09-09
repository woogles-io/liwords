package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	macondogame "github.com/domino14/macondo/game"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/entity/utilities"
	"github.com/woogles-io/liwords/pkg/stores/common"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// GameRequest utility functions for handling both proto and protojson formats

// ParseGameRequest parses GameRequest from bytes, trying proto format first, then protojson
func ParseGameRequest(data []byte) (*pb.GameRequest, error) {
	if len(data) == 0 {
		return &pb.GameRequest{}, nil
	}

	gr := &pb.GameRequest{}
	
	// Try proto format first (binary data from live games)
	err := proto.Unmarshal(data, gr)
	if err == nil {
		return gr, nil
	}
	
	// Fall back to protojson format (from past games)
	err = protojson.Unmarshal(data, gr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GameRequest as both proto and protojson: %w", err)
	}
	
	return gr, nil
}

// MarshalGameRequestAsJSON marshals GameRequest as protojson for past games table
func MarshalGameRequestAsJSON(gr *pb.GameRequest) ([]byte, error) {
	if gr == nil {
		return nil, fmt.Errorf("GameRequest is nil")
	}
	return protojson.Marshal(gr)
}

type GameRow struct {
	UUID           string
	CreatedAt      time.Time
	GameEndReason  int
	WinnerIdx      int
	GameRequest    []byte // Binary proto data from games.request
	History        []byte
	Stats          json.RawMessage
	Quickdata      json.RawMessage
	Type           int
	TournamentData json.RawMessage
	Player0ID      sql.NullInt32
	Player1ID      sql.NullInt32
}

func main() {
	var (
		configFile  = flag.String("config", "", "Config file path")
		batchSize   = flag.Int("batch", 100, "Batch size for processing games")
		startOffset = flag.Int("offset", 0, "Starting offset for processing")
		limit       = flag.Int("limit", 0, "Limit number of games to process (0 = no limit)")
		dryRun      = flag.Bool("dry-run", false, "Dry run mode - don't actually migrate")
		verbose     = flag.Bool("verbose", false, "Verbose logging")
	)
	flag.Parse()

	// Load config
	cfg := &config.Config{}
	if *configFile != "" {
		cfg.Load([]string{"-config", *configFile})
	} else {
		cfg.Load(os.Args[1:])
	}

	ctx := context.Background()

	// Set up database connection
	dbPool, err := common.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Get total count of games to migrate
	var totalCount int
	err = dbPool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM games
		WHERE game_end_reason != 0
		AND (migration_status IS NULL OR migration_status = 0)
	`).Scan(&totalCount)
	if err != nil {
		log.Fatalf("Failed to get game count: %v", err)
	}

	log.Printf("Found %d games to migrate", totalCount)
	if *limit > 0 && *limit < totalCount {
		totalCount = *limit
		log.Printf("Limiting to %d games", totalCount)
	}

	processed := 0
	errors := 0
	offset := *startOffset

	for processed < totalCount {
		// Get batch of games
		rows, err := dbPool.Query(ctx, `
			SELECT uuid, created_at, game_end_reason, winner_idx,
				   request, history, stats, quickdata, type, tournament_data,
				   player0_id, player1_id
			FROM games
			WHERE game_end_reason != 0
			AND (migration_status IS NULL OR migration_status = 0)
			ORDER BY created_at
			LIMIT $1 OFFSET $2
		`, *batchSize, offset)
		if err != nil {
			log.Fatalf("Failed to query games: %v", err)
		}

		games := []GameRow{}
		for rows.Next() {
			var g GameRow
			err := rows.Scan(&g.UUID, &g.CreatedAt, &g.GameEndReason, &g.WinnerIdx,
				&g.GameRequest, &g.History, &g.Stats, &g.Quickdata, &g.Type,
				&g.TournamentData, &g.Player0ID, &g.Player1ID)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				errors++
				continue
			}
			games = append(games, g)
		}
		rows.Close()

		if len(games) == 0 {
			break
		}

		// Process each game in the batch
		for _, game := range games {
			if *verbose {
				log.Printf("Processing game %s...", game.UUID)
			}

			err := migrateGame(ctx, dbPool, cfg, game, *dryRun)
			if err != nil {
				log.Printf("Error migrating game %s: %v", game.UUID, err)
				errors++
			} else {
				processed++
				if processed%100 == 0 {
					log.Printf("Progress: %d/%d games migrated", processed, totalCount)
				}
			}
		}

		offset += *batchSize

		// Small delay to avoid overloading the database
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Migration complete. Processed: %d, Errors: %d", processed, errors)
}

func migrateGame(ctx context.Context, db *pgxpool.Pool, cfg *config.Config, game GameRow, dryRun bool) error {
	// Parse history to get game details
	hist := &macondopb.GameHistory{}
	err := proto.Unmarshal(game.History, hist)
	if err != nil {
		return fmt.Errorf("failed to unmarshal history: %v", err)
	}

	// Parse game request using utility function
	grproto, err := ParseGameRequest(game.GameRequest)
	if err != nil {
		return fmt.Errorf("failed to parse game request: %v", err)
	}
	
	// Create entity.GameRequest wrapper for compatibility
	gameReq := entity.GameRequest{GameRequest: grproto}
	
	// Marshal as JSON for past_games table
	grprotoJson, err := MarshalGameRequestAsJSON(grproto)
	if err != nil {
		return fmt.Errorf("failed to marshal game request: %v", err)
	}

	// Parse other data
	var quickdata entity.Quickdata
	if err := json.Unmarshal(game.Quickdata, &quickdata); err != nil {
		return fmt.Errorf("failed to parse quickdata: %v", err)
	}

	var stats entity.Stats
	if err := json.Unmarshal(game.Stats, &stats); err != nil {
		return fmt.Errorf("failed to parse stats: %v", err)
	}

	var tournamentData entity.TournamentData
	if len(game.TournamentData) > 0 {
		if err := json.Unmarshal(game.TournamentData, &tournamentData); err != nil {
			return fmt.Errorf("failed to parse tournament data: %v", err)
		}
	}

	// Create macondo rules from game request
	lexicon := hist.Lexicon
	if lexicon == "" {
		lexicon = gameReq.GameRequest.Lexicon
	}

	rules, err := macondogame.NewBasicGameRules(
		cfg.MacondoConfig(), lexicon, gameReq.GameRequest.Rules.BoardLayoutName,
		gameReq.GameRequest.Rules.LetterDistributionName, macondogame.CrossScoreOnly,
		macondogame.Variant(gameReq.GameRequest.Rules.VariantName))
	if err != nil {
		return fmt.Errorf("failed to create game rules: %v", err)
	}

	// Create macondo game from history
	mcg, err := macondogame.NewFromHistory(hist, rules, len(hist.Events))
	if err != nil {
		return fmt.Errorf("failed to create game from history: %v", err)
	}

	// Create an entity.Game to use the proper conversion function
	entGame := &entity.Game{
		Game:           *mcg,
		GameReq:        &gameReq,
		Stats:          &stats,
		Quickdata:      &quickdata,
		Type:           pb.GameType(game.Type),
		TournamentData: &tournamentData,
		PlayerDBIDs:    [2]uint{uint(game.Player0ID.Int32), uint(game.Player1ID.Int32)},
		CreatedAt:      game.CreatedAt,
		GameEndReason:  pb.GameEndReason(game.GameEndReason),
		WinnerIdx:      game.WinnerIdx,
	}

	// Convert to GameDocument using the proper utility function
	doc, err := utilities.ToGameDocument(entGame, cfg)
	if err != nil {
		return fmt.Errorf("failed to convert to game document: %v", err)
	}

	docBytes, err := protojson.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal game document: %v", err)
	}

	// Get final scores
	finalScores := quickdata.FinalScores
	if len(finalScores) == 0 && len(hist.FinalScores) > 0 {
		finalScores = hist.FinalScores
	}

	if dryRun {
		log.Printf("DRY RUN: Would migrate game %s with scores %v", game.UUID, finalScores)
		return nil
	}

	// Start transaction
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Insert into game_metadata first
	_, err = tx.Exec(ctx, `
		INSERT INTO game_metadata (
			game_uuid, created_at, game_request, tournament_data
		) VALUES ($1, $2, $3, $4)
	`, game.UUID, game.CreatedAt, grprotoJson, game.TournamentData)
	if err != nil {
		return fmt.Errorf("failed to insert into game_metadata: %v", err)
	}

	// Insert into past_games (without game_request and tournament_data)
	_, err = tx.Exec(ctx, `
		INSERT INTO past_games (
			gid, created_at, game_end_reason, winner_idx,
			game_document, stats, quickdata, type
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, game.UUID, game.CreatedAt, game.GameEndReason, game.WinnerIdx,
		docBytes, game.Stats, game.Quickdata, game.Type)
	if err != nil {
		return fmt.Errorf("failed to insert into past_games: %v", err)
	}

	// Insert into game_players for each player
	if game.Player0ID.Valid && game.Player1ID.Valid {
		// Extract common data for both players
		originalRequestID := extractOriginalRequestID(game.Quickdata)
		
		// Player 0
		ratingBefore0, ratingAfter0, ratingDelta0 := extractRatingData(game.Quickdata, 0)
		_, err = tx.Exec(ctx, `
			INSERT INTO game_players (
				game_uuid, player_id, player_index, score, won, game_end_reason,
				rating_before, rating_after, rating_delta, created_at, game_type,
				opponent_id, opponent_score, original_request_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		`, game.UUID, game.Player0ID.Int32, 0,
			getScore(finalScores, 0), getWon(game.WinnerIdx, 0), game.GameEndReason,
			ratingBefore0, ratingAfter0, ratingDelta0, game.CreatedAt, game.Type,
			game.Player1ID.Int32, getScore(finalScores, 1), originalRequestID)
		if err != nil {
			return fmt.Errorf("failed to insert player 0 into game_players: %v", err)
		}

		// Player 1
		ratingBefore1, ratingAfter1, ratingDelta1 := extractRatingData(game.Quickdata, 1)
		_, err = tx.Exec(ctx, `
			INSERT INTO game_players (
				game_uuid, player_id, player_index, score, won, game_end_reason,
				rating_before, rating_after, rating_delta, created_at, game_type,
				opponent_id, opponent_score, original_request_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		`, game.UUID, game.Player1ID.Int32, 1,
			getScore(finalScores, 1), getWon(game.WinnerIdx, 1), game.GameEndReason,
			ratingBefore1, ratingAfter1, ratingDelta1, game.CreatedAt, game.Type,
			game.Player0ID.Int32, getScore(finalScores, 0), originalRequestID)
		if err != nil {
			return fmt.Errorf("failed to insert player 1 into game_players: %v", err)
		}
	}

	// Update migration status
	_, err = tx.Exec(ctx, `
		UPDATE games
		SET migration_status = 1, updated_at = NOW()
		WHERE uuid = $1
	`, game.UUID)
	if err != nil {
		return fmt.Errorf("failed to update migration status: %v", err)
	}

	// Optionally clear the data from games table to save space
	// Uncomment this if you want to clear data after migration
	/*
		_, err = tx.Exec(ctx, `
			UPDATE games
			SET history = NULL,
				stats = NULL,
				quickdata = NULL,
				timers = NULL,
				meta_events = NULL,
				request = NULL,
				tournament_data = NULL,
				player0_id = NULL,
				player1_id = NULL,
				migration_status = 2,
				updated_at = NOW()
			WHERE uuid = $1
		`, game.UUID)
		if err != nil {
			return fmt.Errorf("failed to clear game data: %v", err)
		}
	*/

	return tx.Commit(ctx)
}

func getScore(scores []int32, playerIdx int) int32 {
	if playerIdx < len(scores) {
		return scores[playerIdx]
	}
	return 0
}

func getWon(winnerIdx, playerIdx int) sql.NullBool {
	if winnerIdx == -1 {
		// Tie
		return sql.NullBool{Valid: false}
	}
	return sql.NullBool{Bool: winnerIdx == playerIdx, Valid: true}
}

// extractRatingData extracts rating information from quickdata
func extractRatingData(quickdataJSON json.RawMessage, playerIdx int) (before, after sql.NullInt32, delta sql.NullInt32) {
	var quickdata struct {
		OriginalRatings []float64 `json:"OriginalRatings"`
		NewRatings      []float64 `json:"NewRatings"`
	}

	if err := json.Unmarshal(quickdataJSON, &quickdata); err != nil {
		return sql.NullInt32{}, sql.NullInt32{}, sql.NullInt32{}
	}

	// Check if we have rating data for this player
	if playerIdx < len(quickdata.OriginalRatings) && playerIdx < len(quickdata.NewRatings) {
		beforeRating := int32(quickdata.OriginalRatings[playerIdx])
		afterRating := int32(quickdata.NewRatings[playerIdx])
		ratingDelta := afterRating - beforeRating

		return sql.NullInt32{Int32: beforeRating, Valid: true},
			sql.NullInt32{Int32: afterRating, Valid: true},
			sql.NullInt32{Int32: ratingDelta, Valid: true}
	}

	return sql.NullInt32{}, sql.NullInt32{}, sql.NullInt32{}
}

// extractOriginalRequestID extracts the original request ID from quickdata
func extractOriginalRequestID(quickdataJSON json.RawMessage) sql.NullString {
	var quickdata struct {
		OriginalRequestID string `json:"o"`
	}

	if err := json.Unmarshal(quickdataJSON, &quickdata); err != nil {
		return sql.NullString{}
	}

	if quickdata.OriginalRequestID != "" {
		return sql.NullString{String: quickdata.OriginalRequestID, Valid: true}
	}

	return sql.NullString{}
}
