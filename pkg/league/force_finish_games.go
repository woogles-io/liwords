package league

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// GameStore defines the minimal interface needed from a game store
// This avoids circular imports with pkg/gameplay
type GameStore interface {
	Get(ctx context.Context, id string) (*entity.Game, error)
}

// ForceFinishManager handles force-finishing unfinished games
type ForceFinishManager struct {
	store     league.Store
	gameStore GameStore
}

// NewForceFinishManager creates a new force-finish manager
func NewForceFinishManager(store league.Store, gameStore GameStore) *ForceFinishManager {
	return &ForceFinishManager{
		store:     store,
		gameStore: gameStore,
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
	unfinishedGames, err := ffm.store.GetUnfinishedLeagueGames(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unfinished games: %w", err)
	}

	result.TotalGames = len(unfinishedGames)

	// Force-finish each game
	for _, gameRow := range unfinishedGames {
		var winnerIdx, loserIdx int32

		// Load the game entity to get current scores
		gameEntity, err := ffm.gameStore.Get(ctx, gameRow.GameID.String)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to load game %s: %v", gameRow.GameID.String, err))
			continue
		}

		// Get scores from the game entity
		player0Score := gameEntity.PointsFor(0)
		player1Score := gameEntity.PointsFor(1)

		// Determine winner/loser based on current scores
		if player0Score > player1Score {
			winnerIdx = 0
			loserIdx = 1
		} else if player1Score > player0Score {
			winnerIdx = 1
			loserIdx = 0
		} else {
			// Tied game - mark player 0 as winner (arbitrary choice)
			// In practice, this shouldn't happen often
			winnerIdx = 0
			loserIdx = 1
		}

		// Force-finish the game with FORCE_FORFEIT reason
		err = ffm.store.ForceFinishGame(ctx, models.ForceFinishGameParams{
			Uuid:      gameRow.GameID,
			WinnerIdx: pgtype.Int4{Int32: winnerIdx, Valid: true},
			LoserIdx:  pgtype.Int4{Int32: loserIdx, Valid: true},
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
	unfinishedGames, err := ffm.store.GetUnfinishedLeagueGames(ctx, seasonID)
	if err != nil {
		return 0, fmt.Errorf("failed to get unfinished games: %w", err)
	}
	return len(unfinishedGames), nil
}
