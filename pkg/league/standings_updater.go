package league

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/woogles-io/liwords/pkg/stores/league"
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

	return nil
}
