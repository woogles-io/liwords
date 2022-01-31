package omgsvc

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
)

func publishWithSanitization(b ipc.Publisher, evt *ipc.EventWrapper) error {
	aud := evt.Audience()
	data, err := evt.Serialize()
	if err != nil {
		return err
	}
	for _, r := range aud {
		if r.Type() == ipc.AudUser {
			// possibly sanitize any data going directly to a user.
			components := strings.SplitN(r.Receiver(), ".", 2)
			user := components[0]
			// The receiver can have a suffix, like userid.game.gameid

			sanitized, err := sanitize(evt, user)
			if err != nil {
				return err
			}
			ser, err := sanitized.Serialize()
			if err != nil {
				return err
			}
			err = b.PublishToTopic(r.RecipientTopic(evt.Type), ser)
			if err != nil {
				return err
			}

		} else {
			err := b.PublishToTopic(r.RecipientTopic(evt.Type), data)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Events need to be sanitized so that we don't send user racks to people
// who shouldn't get them. Note that sanitize only runs for events that are
// sent DIRECTLY to a player (see AudUser), and not for AudGameTv for example.
func sanitize(evt *ipc.EventWrapper, userID string) (*ipc.EventWrapper, error) {
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
		// Possibly censors users
		cloned := proto.Clone(subevt).(*pb.GameHistoryRefresher)
		// XXX: re-implement history censorship.

		// cloned.History = mod.CensorHistory(context.Background(), us, cloned.History)

		if subevt.History.PlayState == macondopb.PlayState_GAME_OVER {
			// no need to sanitize if the game is over.
			return ipc.WrapEvent(cloned, pb.MessageType_GAME_HISTORY_REFRESHER), nil
		}
		ourIndex := pindex(userID, subevt.History.Players)
		if ourIndex == -1 {
			// No need to sanitize if we don't have a nickname IN THE GAME;
			// this only happens if we are not playing the game.
			return ipc.WrapEvent(cloned, pb.MessageType_GAME_HISTORY_REFRESHER), nil
		}
		// If we're here, userID is currently playing this game.
		for _, evt := range cloned.History.Events {
			if int(evt.PlayerIndex) != ourIndex {
				evt.Rack = ""
				if evt.Type == macondopb.GameEvent_EXCHANGE {
					evt.Exchanged = strconv.Itoa(utf8.RuneCountInString(evt.Exchanged))
				}
			}
		}
		// clear the last known racks for all players not us.
		for i := range cloned.History.LastKnownRacks {
			if i != ourIndex {
				cloned.History.LastKnownRacks[i] = ""
			}
		}

		return ipc.WrapEvent(cloned, pb.MessageType_GAME_HISTORY_REFRESHER), nil

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
			cloned.Event.Exchanged = strconv.Itoa(utf8.RuneCountInString(cloned.Event.Exchanged))
		}
		return ipc.WrapEvent(cloned, pb.MessageType_SERVER_GAMEPLAY_EVENT), nil

	default:
		return evt, nil
	}
}

// pindex returns the index of the userid in playerinfo, or -1 if not found.
func pindex(userid string, playerinfo []*macondopb.PlayerInfo) int {
	for idx, p := range playerinfo {
		if p.UserId == userid {
			return idx
		}
	}
	return -1
}
