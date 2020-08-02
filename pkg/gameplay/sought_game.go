package gameplay

import (
	"context"
	"errors"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"

	"github.com/domino14/liwords/pkg/entity"
)

var errAlreadyOpenReq = errors.New("you already have an open match or seek request")

// SoughtGameStore is an interface for getting a sought game.
type SoughtGameStore interface {
	Get(ctx context.Context, id string) (*entity.SoughtGame, error)
	Set(context.Context, *entity.SoughtGame) error
	Delete(ctx context.Context, id string) error
	ListOpen(ctx context.Context) ([]*entity.SoughtGame, error)
	ExistsForUser(ctx context.Context, userID string) (bool, error)
	DeleteForUser(ctx context.Context, userID string) (string, error)
}

func NewSoughtGame(ctx context.Context, gameStore SoughtGameStore,
	req *pb.SeekRequest) (*entity.SoughtGame, error) {

	exists, err := gameStore.ExistsForUser(ctx, req.User.UserId)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errAlreadyOpenReq
	}

	sg := entity.NewSoughtGame(req)
	if err := gameStore.Set(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}

func CancelSoughtGame(ctx context.Context, gameStore SoughtGameStore, id string) error {
	return gameStore.Delete(ctx, id)
}

func NewMatchRequest(ctx context.Context, gameStore SoughtGameStore,
	req *pb.MatchRequest) (*entity.SoughtGame, error) {

	sg := entity.NewMatchRequest(req)
	if err := gameStore.Set(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}
