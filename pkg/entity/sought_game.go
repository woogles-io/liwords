package entity

import (
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	"github.com/lithammer/shortuuid"
)

type SoughtGameType int

const (
	TypeSeek SoughtGameType = iota
	TypeMatch
	TypeTournamentMatch
	TypeNone
)

type SoughtGame struct {
	// A sought game has either of these fields set
	SeekRequest            *pb.SeekRequest
	MatchRequest           *pb.MatchRequest
	TournamentMatchRequest *pb.TournamentMatchRequest
	Type                   SoughtGameType
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

func NewTournamentMatchRequest(tmr *pb.TournamentMatchRequest) *SoughtGame {
	sg := &SoughtGame{
		TournamentMatchRequest: tmr,
		Type:                   TypeTournamentMatch,
	}
	sg.TournamentMatchRequest.GameRequest.RequestId = shortuuid.New()
	return sg
}

func (sg *SoughtGame) ID() string {
	switch sg.Type {
	case TypeMatch:
		return sg.MatchRequest.GameRequest.RequestId
	case TypeSeek:
		return sg.SeekRequest.GameRequest.RequestId
	case TypeTournamentMatch:
		return sg.TournamentMatchRequest.GameRequest.RequestId
	}
	return ""
}

func (sg *SoughtGame) ConnID() string {
	switch sg.Type {
	case TypeSeek:
		return sg.SeekRequest.ConnectionId
	case TypeMatch:
		return sg.MatchRequest.ConnectionId
		// Tournament matches don't have connection IDs as they are sent
		// from the server to all the user's connections.
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
