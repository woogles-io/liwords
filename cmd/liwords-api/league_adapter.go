package main

import (
	"context"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// GameCreatorAdapter adapts the gameplay package functions to the league.GameCreator interface
type GameCreatorAdapter struct {
	stores    *stores.Stores
	cfg       *config.Config
	eventChan chan<- *entity.EventWrapper
}

func (a *GameCreatorAdapter) InstantiateNewGame(ctx context.Context, users [2]*entity.User,
	req *pb.GameRequest, tdata *entity.TournamentData, opts ...gameplay.GameOption) (*entity.Game, error) {
	return gameplay.InstantiateNewGame(ctx, a.stores.GameStore, a.cfg, users, req, tdata, opts...)
}

func (a *GameCreatorAdapter) StartGame(ctx context.Context, game *entity.Game, users user.Store) error {
	return gameplay.StartGameWithUsers(ctx, a.stores, users, a.eventChan, game)
}

func (a *GameCreatorAdapter) GetGame(ctx context.Context, id string) (*entity.Game, error) {
	return a.stores.GameStore.Get(ctx, id)
}
