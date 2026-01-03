package league

import (
	"context"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/league"
)

// StandingsUpdaterImpl implements the stores.LeagueStandingsUpdater interface.
// This allows it to be injected into pkg/gameplay to update league standings
// without creating circular dependencies.
type StandingsUpdaterImpl struct {
	store league.Store
}

// NewStandingsUpdaterImpl creates a new standings updater implementation.
func NewStandingsUpdaterImpl(store league.Store) *StandingsUpdaterImpl {
	return &StandingsUpdaterImpl{
		store: store,
	}
}

// UpdateGameStandings updates league standings for a completed game.
// This implements the stores.LeagueStandingsUpdater interface.
func (su *StandingsUpdaterImpl) UpdateGameStandings(ctx context.Context, g *entity.Game) error {
	// Delegate to the existing UpdateGameStandingsWithGame function
	return UpdateGameStandingsWithGame(ctx, su.store, g)
}
