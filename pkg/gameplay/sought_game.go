package gameplay

import (
	"context"

	pb "github.com/domino14/crosswords/rpc/api/proto"

	"github.com/domino14/crosswords/pkg/entity"
)

// SoughtGameStore is an interface for getting a sought game.
type SoughtGameStore interface {
	Get(ctx context.Context, id string) (*entity.SoughtGame, error)
	Set(context.Context, *entity.SoughtGame) error
	Delete(ctx context.Context, id string) error
}

func NewSoughtGame(ctx context.Context, gameStore SoughtGameStore,
	req *pb.SeekRequest) (*entity.SoughtGame, error) {

	sg := entity.NewSoughtGame(req)
	if err := gameStore.Set(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}

func NewMatchRequest(ctx context.Context, gameStore SoughtGameStore,
	req *pb.MatchRequest) (*entity.SoughtGame, error) {

	sg := entity.NewMatchRequest(req)
	if err := gameStore.Set(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}
