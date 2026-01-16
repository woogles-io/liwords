package league

import (
	"context"
	"fmt"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// ForceFinishManager handles force-finishing and repair operations for league games
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

// ForceFinishUnfinishedGames finds all unfinished games for a season and adjudicates them.
// This is called when the ongoing season is closed.
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

	// Adjudicate each game using the gameplay package
	for _, gameRow := range unfinishedGames {
		// Load the game entity
		gameEntity, err := ffm.stores.GameStore.Get(ctx, gameRow.GameID.String)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to load game %s: %v", gameRow.GameID.String, err))
			continue
		}

		// Adjudicate the game - gameplay.AdjudicateGame handles most of the work:
		// - Sets game end reason to ADJUDICATED
		// - Determines winner/loser based on current score
		// - Adds final scores to history
		// - Sets PlayState to GAME_OVER (stops timer)
		// - Computes and updates ratings/stats
		// - Saves game to DB
		// - Inserts game_players rows
		// - Updates league standings (via injected LeagueStandingsUpdater)
		// - Sends events to clients
		err = gameplay.AdjudicateGame(ctx, gameEntity, ffm.stores)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to adjudicate game %s: %v", gameRow.GameID.String, err))
			continue
		}

		result.ForceForfeitGames++
	}

	return result, nil
}

// RepairResult tracks the outcome of repairing force-finished games
type RepairResult struct {
	TotalGamesFound   int
	GamesRepaired     int
	AlreadyHadPlayers int
	Errors            []string
}

// RepairForceFinishedGames finds force-finished or adjudicated games missing game_players rows
// and creates them, then recalculates standings for the season.
// This function is idempotent - safe to run multiple times.
func (ffm *ForceFinishManager) RepairForceFinishedGames(
	ctx context.Context,
	seasonID uuid.UUID,
) (*RepairResult, error) {
	result := &RepairResult{
		Errors: []string{},
	}

	// Find all force-finished/adjudicated games missing game_players rows
	brokenGames, err := ffm.stores.LeagueStore.GetForceFinishedGamesMissingPlayers(ctx, pgtype.UUID{Bytes: seasonID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get broken games: %w", err)
	}

	result.TotalGamesFound = len(brokenGames)

	// Repair each broken game
	for _, gameRow := range brokenGames {
		// Load the game entity
		gameEntity, err := ffm.stores.GameStore.Get(ctx, gameRow.GameID.String)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to load game %s: %v", gameRow.GameID.String, err))
			continue
		}

		// Lock the game for modification
		gameEntity.Lock()

		// Migrate old FORCE_FORFEIT (8) to ADJUDICATED (9)
		if gameEntity.GameEndReason == pb.GameEndReason_FORCE_FORFEIT {
			gameEntity.SetGameEndReason(pb.GameEndReason_ADJUDICATED)
		}

		// Ensure winner/loser is set based on scores if not already set
		if !gameEntity.WinnerWasSet() {
			player0Score := gameEntity.PointsFor(0)
			player1Score := gameEntity.PointsFor(1)
			if player0Score > player1Score {
				gameEntity.SetWinnerIdx(0)
				gameEntity.SetLoserIdx(1)
			} else if player1Score > player0Score {
				gameEntity.SetWinnerIdx(1)
				gameEntity.SetLoserIdx(0)
			} else {
				gameEntity.SetWinnerIdx(-1)
				gameEntity.SetLoserIdx(-1)
			}
		}

		// Ensure final scores are in history
		if len(gameEntity.History().FinalScores) == 0 {
			gameEntity.AddFinalScoresToHistory()
		}

		// Ensure PlayState is GAME_OVER (fixes timer ticking issue)
		gameEntity.History().PlayState = macondopb.PlayState_GAME_OVER
		gameEntity.Game.SetPlaying(macondopb.PlayState_GAME_OVER)

		// Ensure winner in history matches reality
		gameEntity.History().Winner = int32(gameEntity.WinnerIdx)

		// Save the updated game
		err = ffm.stores.GameStore.Set(ctx, gameEntity)
		gameEntity.Unlock()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to save game %s: %v", gameRow.GameID.String, err))
			continue
		}

		// Insert game_players rows (idempotent - has ON CONFLICT DO NOTHING)
		err = ffm.stores.GameStore.InsertGamePlayers(ctx, gameEntity)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to insert game_players for game %s: %v", gameRow.GameID.String, err))
			continue
		}

		result.GamesRepaired++
	}

	// Recalculate standings for the entire season to fix counts
	standingsMgr := NewStandingsManager(ffm.stores.LeagueStore)
	err = standingsMgr.RecalculateAndSaveStandings(ctx, seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to recalculate standings: %w", err)
	}

	return result, nil
}
