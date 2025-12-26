package stores

import (
	"context"

	"github.com/woogles-io/liwords/pkg/entity"
)

// LeagueStandingsUpdater is the interface for updating league standings when a game ends.
// This interface allows pkg/gameplay to update league standings without importing pkg/league,
// thus avoiding circular dependencies.
//
// The implementation lives in pkg/league, and is injected into Stores during initialization.
type LeagueStandingsUpdater interface {
	// UpdateGameStandings updates league standings for a completed game.
	// Returns nil if the game is not a league game.
	UpdateGameStandings(ctx context.Context, g *entity.Game) error
}
