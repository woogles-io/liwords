package stats

import (
	"context"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
)

// GameStore is an interface for getting a full game.

func GetStats(ctx context.Context, gameStore gameplay.GameStore, id string) (*entity.Stats, error) {

	entGame, err := gameStore.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	stats := entity.InstantiateNewStats()
	stats.AddGameToStats(entGame.Game.History(), id)
	return stats, nil
}