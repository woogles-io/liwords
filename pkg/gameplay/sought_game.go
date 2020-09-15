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
	Set(context.Context, *entity.SoughtGame) error
	Delete(ctx context.Context, id string) error
	ListOpenSeeks(ctx context.Context) ([]*entity.SoughtGame, error)
	ListOpenMatches(ctx context.Context, receiverID string) ([]*entity.SoughtGame, error)
	ExistsForUser(ctx context.Context, userID string) (bool, error)
	DeleteForUser(ctx context.Context, userID string) (string, error)
	UserMatchedBy(ctx context.Context, userID, matcher string) (bool, error)
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

// ValidateSoughtGame validates the seek request.
func ValidateSoughtGame(ctx context.Context, req *pb.GameRequest) error {
	if req.InitialTimeSeconds < 15 {
		return errors.New("the initial time must be at least 15 seconds")
	}
	if req.MaxOvertimeMinutes < 0 || req.MaxOvertimeMinutes > 5 {
		return errors.New("overtime minutes must be between 0 and 5")
	}
	if req.IncrementSeconds < 0 {
		return errors.New("you cannot have a negative time increment")
	}
	if req.MaxOvertimeMinutes > 0 && req.IncrementSeconds > 0 {
		return errors.New("you can have increments or max overtime, but not both")
	}
	if req.PlayerVsBot && entity.TotalTimeEstimate(req) < 45 {
		return errors.New("this time control is too fast for our poor bot")
	}
	return nil
}
