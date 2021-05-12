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
	TypeMatch
	TypeNone
)

type SoughtGame struct {
	// A sought game has either of these fields set
	SeekRequest  *pb.SeekRequest
	MatchRequest *pb.MatchRequest
	Type         SoughtGameType
}

func NewSoughtGame(seekRequest *pb.SeekRequest) *SoughtGame {
	sg := &SoughtGame{
		SeekRequest: seekRequest,
		Type:        TypeSeek,
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

func NewMatchRequest(matchRequest *pb.MatchRequest) *SoughtGame {
	sg := &SoughtGame{
		MatchRequest: matchRequest,
		Type:         TypeMatch,
	}
	sg.MatchRequest.GameRequest.RequestId = shortuuid.New()
	if sg.MatchRequest.GameRequest.OriginalRequestId == "" {
		sg.MatchRequest.GameRequest.OriginalRequestId =
			sg.MatchRequest.GameRequest.RequestId
	}
	return sg
}

func (sg *SoughtGame) ID() string {
	switch sg.Type {
	case TypeMatch:
		return sg.MatchRequest.GameRequest.RequestId
	case TypeSeek:
		return sg.SeekRequest.GameRequest.RequestId
	}
	return ""
}

func (sg *SoughtGame) ConnID() string {
	switch sg.Type {
	case TypeSeek:
		return sg.SeekRequest.ConnectionId
	case TypeMatch:
		return sg.MatchRequest.ConnectionId
	}
	return ""
}

func (sg *SoughtGame) Seeker() string {
	switch sg.Type {
	case TypeSeek:
		return sg.SeekRequest.User.UserId
	case TypeMatch:
		return sg.MatchRequest.User.UserId
	}
	return ""
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
	if req.PlayerVsBot && TotalTimeEstimate(req) < 45 {
		return errors.New("this time control is too fast for our poor bot")
	}
	return nil
}
