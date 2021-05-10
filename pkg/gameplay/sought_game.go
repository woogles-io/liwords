package gameplay

import (
	"context"
	"errors"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"

	"github.com/domino14/liwords/pkg/entity"
)

var errAlreadyOpenReq = errors.New("You already have an open match or seek request")
var errMatchAlreadyExists = errors.New("The user you are trying to match has already matched you")

// SoughtGameStore is an interface for getting a sought game.
type SoughtGameStore interface {
	Get(ctx context.Context, id string) (*entity.SoughtGame, error)
	GetByConnID(ctx context.Context, connID string) (*entity.SoughtGame, error)
	Set(context.Context, *entity.SoughtGame) error
	Delete(ctx context.Context, id string) error
	ListOpenSeeks(ctx context.Context) ([]*entity.SoughtGame, error)
	ListOpenMatches(ctx context.Context, receiverID, tourneyID string) ([]*entity.SoughtGame, error)
	ExistsForUser(ctx context.Context, userID string) (bool, error)
	DeleteForUser(ctx context.Context, userID string) (*entity.SoughtGame, error)
	DeleteForConnID(ctx context.Context, connID string) (*entity.SoughtGame, error)
	UserMatchedBy(ctx context.Context, userID, matcher string) (bool, error)
	ExpireOld(ctx context.Context) error
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

	exists, err := gameStore.ExistsForUser(ctx, req.User.UserId)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errAlreadyOpenReq
	}

	// Check that the user we are matching hasn't already matched us.
	// XXX: Move to Redis store, and put this in an atomic script
	// create match only if not already matched by, to avoid
	// race conditions.
	matched, err := gameStore.UserMatchedBy(ctx, req.User.UserId, req.ReceivingUser.UserId)
	if err != nil {
		return nil, err
	}
	if matched {
		return nil, errMatchAlreadyExists
	}

	sg := entity.NewMatchRequest(req)
	if err := gameStore.Set(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}
