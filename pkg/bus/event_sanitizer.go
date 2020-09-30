package bus

import (
	"errors"
	"strconv"

	"github.com/golang/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/domino14/liwords/pkg/entity"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
)

// Events need to be sanitized so that we don't send user racks to people
// who shouldn't get them. Note that sanitize only runs for events that are
// sent DIRECTLY to a player (see AudUser), and not for AudGameTv for example.
func sanitize(evt *entity.EventWrapper, userID string) (*entity.EventWrapper, error) {
	// Depending on the event type and even the state of the game, we return a
	// sanitized event (or not).
	switch evt.Type {
	case pb.MessageType_GAME_HISTORY_REFRESHER:
		// When sent to AudUser, we should sanitize ONLY if we are someone
		// who is playing in the game. This is because observers can also
		// receive these events directly (through AudUser).
		subevt, ok := evt.Event.(*pb.GameHistoryRefresher)
		if !ok {
			return nil, errors.New("subevt-wrong-format")
		}
		if subevt.History.PlayState == macondopb.PlayState_GAME_OVER {
			// no need to sanitize if the game is over.
			return evt, nil
		}
		mynick := nicknameFromUserID(userID, subevt.History.Players)
		if mynick == "" {
			// No need to sanitize if we don't have a nickname IN THE GAME;
			// this only happens if we are not playing the game.
			return evt, nil
		}

		cloned := proto.Clone(subevt).(*pb.GameHistoryRefresher)
		// Only sanitize if the nickname is not empty. The nickname is
		// empty if they are not playing in this game.
		for _, evt := range cloned.History.Events {
			if evt.Nickname != mynick {
				evt.Rack = ""
			}
			if evt.Type == macondopb.GameEvent_EXCHANGE {
				evt.Exchanged = strconv.Itoa(len(evt.Exchanged))
			}
		}

		if cloned.History.Players[0].UserId == userID {
			cloned.History.LastKnownRacks[1] = ""
		} else if cloned.History.Players[1].UserId == userID {
			cloned.History.LastKnownRacks[0] = ""
		}
		return entity.WrapEvent(cloned, pb.MessageType_GAME_HISTORY_REFRESHER), nil

	case pb.MessageType_SERVER_GAMEPLAY_EVENT:
		// Server gameplay events
		// When sent to AudUser, we need to sanitize them here. When sent to
		// an AudGameTV, they are unsanitized, and handled elsewhere.
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
		return entity.WrapEvent(cloned, pb.MessageType_SERVER_GAMEPLAY_EVENT), nil

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
