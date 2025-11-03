package league

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// UpdateGameStandings updates standings for a completed league game
// This is called from gameplay/end.go when a game completes
func UpdateGameStandings(ctx context.Context, store league.Store, gameUUID string) error {
	// Get game info
	gameInfo, err := store.GetGameLeagueInfo(ctx, gameUUID)
	if err != nil {
		return fmt.Errorf("failed to get game league info: %w", err)
	}

	// Check if this is a league game
	if !gameInfo.LeagueDivisionID.Valid {
		// Not a league game, nothing to do
		return nil
	}

	divisionID, err := uuid.FromBytes(gameInfo.LeagueDivisionID.Bytes[:])
	if err != nil {
		return fmt.Errorf("failed to parse division UUID: %w", err)
	}

	// Determine winner index (-1 for tie, 0 or 1 for winner) from won column
	// won = true means won, false means lost, null means tie
	winnerIdx := int32(-1)
	if gameInfo.Player0Won.Valid {
		if gameInfo.Player0Won.Bool {
			winnerIdx = 0
		} else {
			winnerIdx = 1
		}
	}

	// Update standings using StandingsManager
	standingsMgr := NewStandingsManager(store)
	err = standingsMgr.UpdateStandingsIncremental(
		ctx,
		divisionID,
		gameInfo.Player0ID.Int32,
		gameInfo.Player1ID.Int32,
		winnerIdx,
		gameInfo.Player0Score.Int32,
		gameInfo.Player1Score.Int32,
	)
	if err != nil {
		return fmt.Errorf("failed to update incremental standings: %w", err)
	}

	// Check if all games in the season are now complete
	// If so, automatically mark the season as COMPLETED
	err = checkAndCompleteSeasonIfDone(ctx, store, divisionID)
	if err != nil {
		// Log error but don't fail the standings update
		// Season completion is a nice-to-have, not critical
		log.Warn().Err(err).Str("divisionID", divisionID.String()).Msg("failed to check season completion")
	}

	return nil
}

// checkAndCompleteSeasonIfDone checks if all games in a season are complete
// and automatically marks the season as COMPLETED if so
func checkAndCompleteSeasonIfDone(ctx context.Context, store league.Store, divisionID uuid.UUID) error {
	// Get the division to find the season
	division, err := store.GetDivision(ctx, divisionID)
	if err != nil {
		return fmt.Errorf("failed to get division: %w", err)
	}

	seasonID := division.SeasonID

	// Get the season
	season, err := store.GetSeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get season: %w", err)
	}

	// Only check active seasons
	if season.Status != int32(ipc.SeasonStatus_SEASON_ACTIVE) {
		return nil
	}

	// Get all divisions for this season
	divisions, err := store.GetDivisionsBySeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get divisions: %w", err)
	}

	// Check if all divisions have all games complete
	for _, div := range divisions {
		divisionUUID, err := uuid.FromBytes(div.Uuid[:])
		if err != nil {
			return fmt.Errorf("failed to parse division UUID: %w", err)
		}

		totalGames, err := store.CountDivisionGamesTotal(ctx, divisionUUID)
		if err != nil {
			return fmt.Errorf("failed to count total games: %w", err)
		}

		completeGames, err := store.CountDivisionGamesComplete(ctx, divisionUUID)
		if err != nil {
			return fmt.Errorf("failed to count complete games: %w", err)
		}

		if totalGames != completeGames {
			// Not all games are complete yet
			return nil
		}
	}

	// All games are complete! Mark the season as COMPLETED
	err = store.UpdateSeasonStatus(ctx, models.UpdateSeasonStatusParams{
		Uuid:   seasonID,
		Status: int32(ipc.SeasonStatus_SEASON_COMPLETED),
	})
	if err != nil {
		return fmt.Errorf("failed to update season status: %w", err)
	}

	log.Info().
		Str("seasonID", seasonID.String()).
		Str("leagueID", season.LeagueID.String()).
		Msg("season-automatically-completed-all-games-finished")

	return nil
}
