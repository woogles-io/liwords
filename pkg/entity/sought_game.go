package entity

import (
	"context"
	"errors"

	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/lithammer/shortuuid"
)

type SoughtGameType int

const (
	TypeSeek SoughtGameType = iota
	TypeNone
)

type SoughtGame struct {
	SeekRequest *pb.SeekRequest
}

func NewSoughtGame(seekRequest *pb.SeekRequest) *SoughtGame {
	sg := &SoughtGame{
		SeekRequest: seekRequest,
	}

	sg.SeekRequest.GameRequest.RequestId = shortuuid.New()
	// Note that even though sought games are never rematches,
	// we must set the OriginalRequestId since they can start
	// a streak of rematches, from which an OriginalRequestId
	// is needed.
	if sg.SeekRequest.GameRequest.OriginalRequestId == "" {
		sg.SeekRequest.GameRequest.OriginalRequestId =
			sg.SeekRequest.GameRequest.RequestId
	}
	return sg
}

func (sg *SoughtGame) ID() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	if sr.GameRequest == nil {
		return "", errors.New("nil game request on seek request")
	}
	return sr.GameRequest.RequestId, nil
}

func (sg *SoughtGame) SeekerConnID() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	return sr.SeekerConnectionId, nil
}

func (sg *SoughtGame) ReceiverConnID() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	return sr.ReceiverConnectionId, nil
}

func (sg *SoughtGame) SeekerUserID() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	if sr.User == nil {
		return "", errors.New("nil user for seek request")
	}
	return sr.User.UserId, nil
}

func (sg *SoughtGame) ReceiverUserID() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	if sr.ReceivingUser == nil {
		return "", errors.New("nil receiving user on seek request")
	}
	return sr.ReceivingUser.UserId, nil
}

func (sg *SoughtGame) ReceiverDisplayName() (string, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return "", err
	}
	if sr.ReceivingUser == nil {
		return "", errors.New("nil receiving user on seek request")
	}
	return sr.ReceivingUser.DisplayName, nil
}

func (sg *SoughtGame) ReceiverIsPermanent() (bool, error) {
	sr, err := getSeekRequest(sg)
	if err != nil {
		return false, err
	}
	return sr.ReceiverIsPermanent, nil
}

func getSeekRequest(sg *SoughtGame) (*pb.SeekRequest, error) {
	if sg.SeekRequest != nil {
		return sg.SeekRequest, nil
	}
	return nil, errors.New("nil seek request on sought game")
}

// ValidateGameRequest validates a generic game request.
func ValidateGameRequest(ctx context.Context, req *pb.GameRequest) error {
	if req == nil {
		return errors.New("game request is missing")
	}
	if req.InitialTimeSeconds < 15 {
		return errors.New("the initial time must be at least 15 seconds")
	}
	if req.MaxOvertimeMinutes < 0 || req.MaxOvertimeMinutes > 10 {
		return errors.New("overtime minutes must be between 0 and 10")
	}
	if req.IncrementSeconds < 0 {
		return errors.New("you cannot have a negative time increment")
	}
	if req.IncrementSeconds > 60 {
		return errors.New("time increment must be at most 60 seconds")
	}
	if req.MaxOvertimeMinutes > 0 && req.IncrementSeconds > 0 {
		return errors.New("you can have increments or max overtime, but not both")
	}
	return nil
}
