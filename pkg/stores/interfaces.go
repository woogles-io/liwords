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

// PushSender sends web push notifications to the browsers a user subscribed.
// The implementation lives in pkg/push; main sets it only when web push is
// on, so it is nil otherwise.
type PushSender interface {
	// Notify returns at once and sends in the background.
	Notify(userUUID string, payload []byte)
}
