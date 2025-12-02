package league

import (
	"context"
	"fmt"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/league"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// GameStats contains per-player stats extracted from a completed game
type GameStats struct {
	Score        int32
	Bingos       int32
	Turns        int32
	HighTurn     int32
	HighGame     int32
	BlanksPlayed int32
	TilesPlayed  int32
	TimedOut     bool // true if this player timed out
}

// extractGameStats extracts stats from a game entity for both players
func extractGameStats(g *entity.Game) (p0Stats, p1Stats GameStats) {
	// Extract scores from final scores
	if len(g.History().FinalScores) >= 2 {
		p0Stats.Score = g.History().FinalScores[0]
		p1Stats.Score = g.History().FinalScores[1]
		p0Stats.HighGame = g.History().FinalScores[0]
		p1Stats.HighGame = g.History().FinalScores[1]
	}

	// Check for timeout - the loser is the one who timed out
	if g.GameEndReason == pb.GameEndReason_TIME {
		if g.LoserIdx == 0 {
			p0Stats.TimedOut = true
		} else if g.LoserIdx == 1 {
			p1Stats.TimedOut = true
		}
	}

	// Extract stats from computed game stats if available
	if g.Stats != nil {
		// Player 0 stats (PlayerOneData = player who went first = player index 0)
		if bingoStat, ok := g.Stats.PlayerOneData[entity.BINGOS_STAT]; ok {
			p0Stats.Bingos = int32(bingoStat.Total)
		}
		if turnsStat, ok := g.Stats.PlayerOneData[entity.TURNS_STAT]; ok {
			p0Stats.Turns = int32(turnsStat.Total)
		}
		if highTurnStat, ok := g.Stats.PlayerOneData[entity.HIGH_TURN_STAT]; ok {
			p0Stats.HighTurn = int32(highTurnStat.Total)
		}
		if tilesStat, ok := g.Stats.PlayerOneData[entity.TILES_PLAYED_STAT]; ok {
			p0Stats.TilesPlayed = int32(tilesStat.Total)
			// Blanks are tracked in Subitems["?"]
			if tilesStat.Subitems != nil {
				p0Stats.BlanksPlayed = int32(tilesStat.Subitems["?"])
			}
		}

		// Player 1 stats (PlayerTwoData = player who went second = player index 1)
		if bingoStat, ok := g.Stats.PlayerTwoData[entity.BINGOS_STAT]; ok {
			p1Stats.Bingos = int32(bingoStat.Total)
		}
		if turnsStat, ok := g.Stats.PlayerTwoData[entity.TURNS_STAT]; ok {
			p1Stats.Turns = int32(turnsStat.Total)
		}
		if highTurnStat, ok := g.Stats.PlayerTwoData[entity.HIGH_TURN_STAT]; ok {
			p1Stats.HighTurn = int32(highTurnStat.Total)
		}
		if tilesStat, ok := g.Stats.PlayerTwoData[entity.TILES_PLAYED_STAT]; ok {
			p1Stats.TilesPlayed = int32(tilesStat.Total)
			// Blanks are tracked in Subitems["?"]
			if tilesStat.Subitems != nil {
				p1Stats.BlanksPlayed = int32(tilesStat.Subitems["?"])
			}
		}
	}

	return p0Stats, p1Stats
}

// UpdateGameStandingsWithGame updates standings for a completed league game using the game entity
// This is called from gameplay/end.go when a game completes
func UpdateGameStandingsWithGame(ctx context.Context, store league.Store, g *entity.Game) error {
	// Check if this is a league game
	if g.LeagueDivisionID == nil {
		// Not a league game, nothing to do
		return nil
	}

	// Check if standings have already been processed for this game
	// This prevents double-counting when performEndgameDuties is called multiple times
	if g.LeagueStandingsProcessed {
		return nil
	}

	divisionID := *g.LeagueDivisionID

	// Determine winner index (-1 for tie, 0 or 1 for winner)
	winnerIdx := int32(g.WinnerIdx)

	// Get player database IDs
	player0ID := int32(g.PlayerDBIDs[0])
	player1ID := int32(g.PlayerDBIDs[1])

	// Extract stats from game
	p0Stats, p1Stats := extractGameStats(g)

	// Update standings using StandingsManager
	standingsMgr := NewStandingsManager(store)
	err := standingsMgr.UpdateStandingsIncremental(
		ctx,
		divisionID,
		player0ID,
		player1ID,
		winnerIdx,
		p0Stats,
		p1Stats,
	)
	if err != nil {
		return fmt.Errorf("failed to update incremental standings: %w", err)
	}

	// Mark standings as processed to prevent double-counting
	g.LeagueStandingsProcessed = true

	return nil
}
