package gameplay

import (
	"context"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"

	"github.com/woogles-io/liwords/pkg/entity"
)

// SoughtGameStore is an interface for getting a sought game.
type SoughtGameStore interface {
	Get(ctx context.Context, id string) (*entity.SoughtGame, error)
	GetBySeekerConnID(ctx context.Context, connID string) (*entity.SoughtGame, error)
	// New refuses a second open real-time match request from one seeker to
	// one receiver with entity.ErrMatchRequestOpen.
	New(context.Context, *entity.SoughtGame) error
	Delete(ctx context.Context, id string) error
	ListOpenSeeks(ctx context.Context, receiverID, tourneyID string) ([]*entity.SoughtGame, error)
	ListCorrespondenceSeeksForUser(ctx context.Context, userID string) ([]*entity.SoughtGame, error)
	ExistsForUser(ctx context.Context, userID string) (bool, error)
	DeleteForUser(ctx context.Context, userID string) ([]*entity.SoughtGame, error)
	DeleteForSeekerConnID(ctx context.Context, connID string) ([]*entity.SoughtGame, error)
	Claim(ctx context.Context, id string) (claimed *entity.SoughtGame, others []*entity.SoughtGame, err error)
	UserMatchedBy(ctx context.Context, userID, matcher string) (bool, error)
	ExpireOld(ctx context.Context) error
}

func NewSoughtGame(ctx context.Context, gameStore SoughtGameStore,
	req *pb.SeekRequest) (*entity.SoughtGame, error) {

	sg := entity.NewSoughtGame(req)
	if err := gameStore.New(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}

func CancelSoughtGame(ctx context.Context, gameStore SoughtGameStore, id string) error {
	return gameStore.Delete(ctx, id)
}
