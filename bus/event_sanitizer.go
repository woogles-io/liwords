package bus

import (
	"errors"
	"strconv"

	"github.com/golang/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/domino14/liwords/pkg/entity"
	pb "github.com/domino14/liwords/rpc/api/proto"
)

// Events need to be sanitized so that we don't send user racks to people
// who shouldn't get them.

func sanitize(evt *entity.EventWrapper, userID string) (*entity.EventWrapper, error) {
	// Depending on the event type and even the state of the game, we return a
	// sanitized event (or not).
	switch evt.Type {
	case pb.MessageType_GAME_HISTORY_REFRESHER:
		subevt, ok := evt.Event.(*pb.GameHistoryRefresher)
		if !ok {
			return nil, errors.New("subevt-wrong-format")
		}
		if subevt.History.PlayState == macondopb.PlayState_GAME_OVER {
			// no need to sanitize if the game is over.
			return evt, nil
		}
		cloned := proto.Clone(subevt).(*pb.GameHistoryRefresher)
		mynick := nicknameFromUserID(userID, cloned.History.Players)
		for _, turn := range cloned.History.Turns {
			for _, evt := range turn.Events {
				if evt.Nickname != mynick {
					evt.Rack = ""
				}
				if evt.Type == macondopb.GameEvent_EXCHANGE {
					evt.Exchanged = strconv.Itoa(len(evt.Exchanged))
				}
			}
		}
		if cloned.History.Players[0].UserId == userID {
			cloned.History.LastKnownRacks[1] = ""
		} else if cloned.History.Players[1].UserId == userID {
			cloned.History.LastKnownRacks[0] = ""
		}
		return entity.WrapEvent(cloned, pb.MessageType_GAME_HISTORY_REFRESHER, evt.GameID()), nil

	case pb.MessageType_GAME_TURNS_REFRESHER:
		subevt, ok := evt.Event.(*pb.GameTurnsRefresher)
		if !ok {
			return nil, errors.New("subevt-wrong-format")
		}
		if subevt.PlayState == macondopb.PlayState_GAME_OVER {
			return evt, nil
		}
		cloned := proto.Clone(subevt).(*pb.GameTurnsRefresher)
		mynick := nicknameFromUserID(userID, cloned.Players)
		// We know this contains no exchanges, because of the way challenges
		// work, so we only need to blank event racks.
		for _, turn := range cloned.Turns {
			for _, evt := range turn.Events {
				if evt.Nickname != mynick {
					evt.Rack = ""
				}
			}
		}
		return entity.WrapEvent(cloned, pb.MessageType_GAME_TURNS_REFRESHER, evt.GameID()), nil

	case pb.MessageType_SERVER_GAMEPLAY_EVENT:
		subevt, ok := evt.Event.(*pb.ServerGameplayEvent)
		if !ok {
			return nil, errors.New("subevt-wrong-format")
		}
		if subevt.Playing == macondopb.PlayState_GAME_OVER {
			return evt, nil
		}
		if subevt.UserId == userID {
			return evt, nil
		}
		// Otherwise clone it.
		cloned := proto.Clone(subevt).(*pb.ServerGameplayEvent)
		cloned.NewRack = ""
		cloned.Event.Rack = ""
		if cloned.Event.Type == macondopb.GameEvent_EXCHANGE {
			cloned.Event.Exchanged = strconv.Itoa(len(cloned.Event.Exchanged))
		}
		return entity.WrapEvent(cloned, pb.MessageType_SERVER_GAMEPLAY_EVENT, evt.GameID()), nil

	default:
		return evt, nil
	}
}

func nicknameFromUserID(userid string, playerinfo []*macondopb.PlayerInfo) string {
	// given a user id, return the nickname of the given user.
	nick := ""
	for _, p := range playerinfo {
		if p.UserId == userid {
			return p.Nickname
		}
	}
	return nick
}
