package bus

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Events need to be sanitized so that we don't send user racks to people
// who shouldn't get them. Note that sanitize only runs for events that are
// sent DIRECTLY to a player (see AudUser), and not for AudGameTv for example.
func sanitize(us user.Store, gs gameplay.GameStore, evt *entity.EventWrapper, userID string) (*entity.EventWrapper, error) {
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
		cloned.History = mod.CensorHistory(context.Background(), us, cloned.History)

		if subevt.History.PlayState == macondopb.PlayState_GAME_OVER {
			// no need to sanitize if the game is over.
			return entity.WrapEvent(cloned, pb.MessageType_GAME_HISTORY_REFRESHER), nil
		}
		myPlayerIndex := playerIndexFromUserID(userID, subevt.History.Players)
		if myPlayerIndex == -1 {
			// User not found in game - could be spectator OR auth issue
			// For correspondence games: ALWAYS sanitize (prevent tile leak)
			// For real-time games: Allow spectating (current behavior)

			// Check if this is a correspondence game
			game, err := gs.Get(context.Background(), subevt.History.Uid)
			isCorrespondence := false
			if err == nil && game != nil {
				isCorrespondence = game.IsCorrespondence()
			}

			if isCorrespondence {
				// CRITICAL: Sanitize ALL tiles for correspondence games
				// This prevents the tile leak when auth fails
				for _, evt := range cloned.History.Events {
					evt.Rack = ""
					if evt.Type == macondopb.GameEvent_EXCHANGE {
						evt.Exchanged = ""
					}
				}
				cloned.History.LastKnownRacks[0] = ""
				cloned.History.LastKnownRacks[1] = ""

				log.Warn().
					Str("userID", userID).
					Str("gameID", subevt.History.Uid).
					Interface("players", subevt.History.Players).
					Msg("correspondence-game-user-not-player-sanitizing-all-tiles")
			}

			// Return sanitized (or unsanitized for real-time spectators)
			return entity.WrapEvent(cloned, pb.MessageType_GAME_HISTORY_REFRESHER), nil
		}

		// Only sanitize if we are in the game.
		for _, evt := range cloned.History.Events {
			if evt.GetPlayerIndex() != uint32(myPlayerIndex) {
				evt.Rack = ""
				if evt.Type == macondopb.GameEvent_EXCHANGE {
					evt.Exchanged = ""
				}
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
			cloned.Event.Exchanged = ""
		}
		return entity.WrapEvent(cloned, pb.MessageType_SERVER_GAMEPLAY_EVENT), nil

	default:
		return evt, nil
	}
}

func playerIndexFromUserID(userid string, playerinfo []*macondopb.PlayerInfo) int {
	// given a user id, return the index in playerinfo for the given user.
	// If the user id is not found, return -1.
	for i, p := range playerinfo {
		if p.UserId == userid {
			return i
		}
	}
	return -1
}
