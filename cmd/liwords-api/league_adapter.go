package main

import (
	"context"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// GameCreatorAdapter adapts the gameplay package functions to the league.GameCreator interface
type GameCreatorAdapter struct {
	stores    *stores.Stores
	cfg       *config.Config
	eventChan chan<- *entity.EventWrapper
}

func (a *GameCreatorAdapter) InstantiateNewGame(ctx context.Context, users [2]*entity.User,
	req *pb.GameRequest, tdata *entity.TournamentData) (*entity.Game, error) {
	return gameplay.InstantiateNewGame(ctx, a.stores.GameStore, a.cfg, users, req, tdata)
}

func (a *GameCreatorAdapter) StartGame(ctx context.Context, game *entity.Game) error {
	return gameplay.StartGame(ctx, a.stores, a.eventChan, game)
}
