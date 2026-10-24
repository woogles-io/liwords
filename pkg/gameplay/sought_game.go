package gameplay

import (
	"context"
	"errors"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"

	"github.com/woogles-io/liwords/pkg/entity"
)

var (
	errAlreadyOpenReq = errors.New("You already have an open match or seek request")
)

// SoughtGameStore is an interface for getting a sought game.
type SoughtGameStore interface {
	Get(ctx context.Context, id string) (*entity.SoughtGame, error)
	GetBySeekerConnID(ctx context.Context, connID string) (*entity.SoughtGame, error)
	New(context.Context, *entity.SoughtGame) error
	Delete(ctx context.Context, id string) error
	ListOpenSeeks(ctx context.Context, receiverID, tourneyID string) ([]*entity.SoughtGame, error)
	ListCorrespondenceSeeksForUser(ctx context.Context, userID string) ([]*entity.SoughtGame, error)
	ExistsForUser(ctx context.Context, userID string) (bool, error)
	CanCreateSeek(ctx context.Context, userID string, gameMode pb.GameMode, receiverID string) (bool, string, error)
	DeleteForUser(ctx context.Context, userID string) ([]*entity.SoughtGame, error)
	UpdateForReceiver(ctx context.Context, userID string) (*entity.SoughtGame, error)
	DeleteForSeekerConnID(ctx context.Context, connID string) ([]*entity.SoughtGame, error)
	Claim(ctx context.Context, id string) (claimed *entity.SoughtGame, others []*entity.SoughtGame, err error)
	UpdateForReceiverConnID(ctx context.Context, connID string) (*entity.SoughtGame, error)
	UserMatchedBy(ctx context.Context, userID, matcher string) (bool, error)
	ExpireOld(ctx context.Context) error
}

func NewSoughtGame(ctx context.Context, gameStore SoughtGameStore,
	req *pb.SeekRequest) (*entity.SoughtGame, error) {

	// Determine receiver ID (empty string for open seeks)
	receiverID := ""
	if req.ReceivingUser != nil {
		receiverID = req.ReceivingUser.UserId
	}

	// Determine game mode (default to REAL_TIME if not specified)
	gameMode := pb.GameMode_REAL_TIME
	if req.GameRequest != nil {
		gameMode = req.GameRequest.GameMode
	}

	// Check if user can create this seek/match
	canCreate, _, err := gameStore.CanCreateSeek(ctx, req.User.UserId, gameMode, receiverID)
	if err != nil {
		return nil, err
	}
	if !canCreate {
		return nil, errAlreadyOpenReq
	}

	sg := entity.NewSoughtGame(req)
	if err := gameStore.New(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}

func CancelSoughtGame(ctx context.Context, gameStore SoughtGameStore, id string) error {
	return gameStore.Delete(ctx, id)
}
