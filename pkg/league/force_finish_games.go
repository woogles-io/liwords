package league

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// GameStore defines the minimal interface needed from a game store
// This avoids circular imports with pkg/gameplay
type GameStore interface {
	Get(ctx context.Context, id string) (*entity.Game, error)
}

// ForceFinishManager handles force-finishing unfinished games
type ForceFinishManager struct {
	stores *stores.Stores
}

// NewForceFinishManager creates a new force-finish manager
func NewForceFinishManager(allStores *stores.Stores) *ForceFinishManager {
	return &ForceFinishManager{
		stores: allStores,
	}
}

// ForceFinishResult tracks the outcome of force-finishing games
type ForceFinishResult struct {
	TotalGames        int
	ForceForfeitGames int
	Errors            []string
}

// ForceFinishUnfinishedGames finds all unfinished games for a season and marks them
// as complete with FORCE_FORFEIT reason. The player with the lower score is marked
// as the loser.
//
// This is called on Day 20 at midnight when the season is force-closed.
func (ffm *ForceFinishManager) ForceFinishUnfinishedGames(
	ctx context.Context,
	seasonID uuid.UUID,
) (*ForceFinishResult, error) {
	result := &ForceFinishResult{
		Errors: []string{},
	}

	// Get all unfinished games for this season
	unfinishedGames, err := ffm.stores.LeagueStore.GetUnfinishedLeagueGames(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unfinished games: %w", err)
	}

	result.TotalGames = len(unfinishedGames)

	// Force-finish each game
	for _, gameRow := range unfinishedGames {
		// Load the game entity to get current scores
		gameEntity, err := ffm.stores.GameStore.Get(ctx, gameRow.GameID.String)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to load game %s: %v", gameRow.GameID.String, err))
			continue
		}

		// Get scores from the game entity
		player0Score := gameEntity.PointsFor(0)
		player1Score := gameEntity.PointsFor(1)

		// Determine winner/loser based on current scores
		var winnerIdx, loserIdx pgtype.Int4
		if player0Score > player1Score {
			winnerIdx = pgtype.Int4{Int32: 0, Valid: true}
			loserIdx = pgtype.Int4{Int32: 1, Valid: true}
		} else if player1Score > player0Score {
			winnerIdx = pgtype.Int4{Int32: 1, Valid: true}
			loserIdx = pgtype.Int4{Int32: 0, Valid: true}
		} else {
			// Tied game - both winner and loser are NULL to indicate tie
			winnerIdx = pgtype.Int4{Valid: false}
			loserIdx = pgtype.Int4{Valid: false}
		}

		// Force-finish the game with FORCE_FORFEIT reason
		err = ffm.stores.LeagueStore.ForceFinishGame(ctx, models.ForceFinishGameParams{
			Uuid:      gameRow.GameID,
			WinnerIdx: winnerIdx,
			LoserIdx:  loserIdx,
		})
		if err != nil {
			// Record error but continue processing other games
			result.Errors = append(result.Errors, fmt.Sprintf("failed to force-finish game %s: %v", gameRow.GameID.String, err))
			continue
		}

		result.ForceForfeitGames++
	}

	return result, nil
}

// GetUnfinishedGameCount returns the number of unfinished games for a season
// This can be used to check if a season is ready to close
func (ffm *ForceFinishManager) GetUnfinishedGameCount(
	ctx context.Context,
	seasonID uuid.UUID,
) (int, error) {
	unfinishedGames, err := ffm.stores.LeagueStore.GetUnfinishedLeagueGames(ctx, seasonID)
	if err != nil {
		return 0, fmt.Errorf("failed to get unfinished games: %w", err)
	}
	return len(unfinishedGames), nil
}
