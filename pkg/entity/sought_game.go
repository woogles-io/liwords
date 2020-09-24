package entity

import (
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
	// A sought game has either of these two fields set
	SeekRequest  *pb.SeekRequest
	MatchRequest *pb.MatchRequest
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

func NewMatchRequest(matchRequest *pb.MatchRequest) *SoughtGame {
	sg := &SoughtGame{
		MatchRequest: matchRequest,
	}
	sg.MatchRequest.GameRequest.RequestId = shortuuid.New()
	if sg.MatchRequest.GameRequest.OriginalRequestId == "" {
		sg.MatchRequest.GameRequest.OriginalRequestId =
		  sg.MatchRequest.GameRequest.RequestId
	}
	return sg
}

func (sg *SoughtGame) ID() string {
	if sg.SeekRequest != nil {
		return sg.SeekRequest.GameRequest.RequestId
	} else if sg.MatchRequest != nil {
		return sg.MatchRequest.GameRequest.RequestId
	}
	return ""
}

func (sg *SoughtGame) Type() SoughtGameType {
	if sg.SeekRequest != nil {
		return TypeSeek
	} else if sg.MatchRequest != nil {
		return TypeMatch
	}
	return TypeNone
}

func (sg *SoughtGame) Seeker() string {
	if sg.SeekRequest != nil {
		return sg.SeekRequest.User.UserId
	} else if sg.MatchRequest != nil {
		return sg.MatchRequest.User.UserId
	}
	return ""
}
