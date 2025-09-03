package game

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/entity/utilities"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// MigrateGameToPastGames migrates a completed game to the past_games and game_players tables
func (s *DBStore) MigrateGameToPastGames(ctx context.Context, g *entity.Game, ratingsBefore, ratingsAfter map[string]int32) error {
	// Convert game to GameDocument format
	doc, err := utilities.ToGameDocument(g, s.cfg)
	if err != nil {
		return fmt.Errorf("converting to game document: %w", err)
	}

	docJSON, err := protojson.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshaling game document: %w", err)
	}

	// Start transaction for all migration operations
	tx, err := s.dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Create queries with transaction
	txQueries := s.queries.WithTx(tx)

	// Marshal GameRequest as protojson for past_games table
	gameRequestJSON, err := MarshalGameRequestAsJSON(g.GameReq.GameRequest)
	if err != nil {
		return fmt.Errorf("marshaling game request: %w", err)
	}

	// Insert into past_games
	err = txQueries.InsertPastGame(ctx, models.InsertPastGameParams{
		Gid:            g.GameID(),
		CreatedAt:      pgtype.Timestamptz{Time: g.CreatedAt, Valid: true},
		GameEndReason:  int16(g.GameEndReason),
		WinnerIdx:      pgtype.Int2{Int16: int16(g.WinnerIdx), Valid: g.WinnerIdx >= -1},
		GameRequest:    gameRequestJSON,
		GameDocument:   docJSON,
		Stats:          *g.Stats,
		Quickdata:      *g.Quickdata,
		Type:           int16(g.Type),
		TournamentData: g.TournamentData,
	})
	if err != nil {
		return fmt.Errorf("inserting into past_games: %w", err)
	}

	// Insert game_players records (using transaction)
	err = s.insertGamePlayersWithTx(ctx, txQueries, g, ratingsBefore, ratingsAfter)
	if err != nil {
		return fmt.Errorf("inserting game players: %w", err)
	}

	// Update migration status
	err = txQueries.UpdateGameMigrationStatus(ctx, models.UpdateGameMigrationStatusParams{
		MigrationStatus: pgtype.Int2{Int16: MigrationStatusMigrated, Valid: true},
		Uuid:            common.ToPGTypeText(g.GameID()),
	})
	if err != nil {
		return fmt.Errorf("updating migration status: %w", err)
	}

	// Note: In staged migration approach, we don't clear data immediately.
	// Data remains in games table until separate cleanup phase.

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	log.Info().Str("gameID", g.GameID()).Msg("game migrated to past_games (data preserved in games table)")
	return nil
}

// insertGamePlayersWithTx inserts game_players records within a transaction
func (s *DBStore) insertGamePlayersWithTx(ctx context.Context, queries *models.Queries, g *entity.Game, ratingsBefore, ratingsAfter map[string]int32) error {
	for pidx := 0; pidx < 2; pidx++ {
		opponentIdx := 1 - pidx
		playerNick := g.History().Players[pidx].Nickname

		params := models.InsertGamePlayerParams{
			GameUuid:          g.GameID(),
			PlayerID:          int32(g.PlayerDBIDs[pidx]),
			PlayerIndex:       int16(pidx),
			Score:             int32(g.PointsFor(pidx)),
			GameEndReason:     int16(g.GameEndReason),
			CreatedAt:         pgtype.Timestamptz{Time: g.CreatedAt, Valid: true},
			GameType:          int16(g.Type),
			OpponentID:        int32(g.PlayerDBIDs[opponentIdx]),
			OpponentScore:     int32(g.PointsFor(opponentIdx)),
			OriginalRequestID: pgtype.Text{String: g.Quickdata.OriginalRequestId, Valid: g.Quickdata.OriginalRequestId != ""},
		}

		// Set win/loss/tie
		if g.WinnerIdx == pidx {
			params.Won = pgtype.Bool{Bool: true, Valid: true}
		} else if g.WinnerIdx == opponentIdx {
			params.Won = pgtype.Bool{Bool: false, Valid: true}
		}
		// Leave as NULL for ties (WinnerIdx == -1)

		// Set rating data if available
		if ratingsBefore != nil && ratingsAfter != nil {
			if before, ok := ratingsBefore[playerNick]; ok {
				params.RatingBefore = pgtype.Int4{Int32: before, Valid: true}
			}
			if after, ok := ratingsAfter[playerNick]; ok {
				params.RatingAfter = pgtype.Int4{Int32: after, Valid: true}
				if params.RatingBefore.Valid {
					delta := after - params.RatingBefore.Int32
					params.RatingDelta = pgtype.Int4{Int32: delta, Valid: true}
				}
			}
		}

		err := queries.InsertGamePlayer(ctx, params)
		if err != nil {
			return fmt.Errorf("inserting player %d: %w", pidx, err)
		}
	}
	return nil
}
