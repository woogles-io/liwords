package bus

import (
	"context"
	"errors"

	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/stores/tournament"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Events need to be sanitized so that we don't send user racks to people
// who shouldn't get them. Note that sanitize only runs for events that are
// sent DIRECTLY to a player (see AudUser), and not for AudGameTv for example.
func sanitize(us user.Store, gs gameplay.GameStore, ts *tournament.Cache, evt *entity.EventWrapper, userID string) (*entity.EventWrapper, error) {
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
			// User not found in game — spectator or auth issue.
			// Censor racks for league games and private-analysis tournaments.
			game, err := gs.Get(context.Background(), subevt.History.Uid)
			shouldCensor := false
			if err != nil {
				// Can't load game — censor to be safe.
				shouldCensor = true
			} else if game != nil {
				if game.IsCorrespondence() || game.LeagueID != nil {
					shouldCensor = true
				} else if game.TournamentData != nil && game.TournamentData.Id != "" && ts != nil {
					t, err := ts.Get(context.Background(), game.TournamentData.Id)
					if err != nil {
						shouldCensor = true // Can't verify — censor to be safe.
					} else if t.ExtraMeta != nil && t.ExtraMeta.PrivateAnalysis {
						shouldCensor = true
					}
				}
			}

			if shouldCensor {
				for _, evt := range cloned.History.Events {
					evt.Rack = ""
					if evt.Type == macondopb.GameEvent_EXCHANGE {
						evt.Exchanged = ""
					}
				}
				cloned.History.LastKnownRacks[0] = ""
				cloned.History.LastKnownRacks[1] = ""
			}

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
		// When sent to AudUser, we sanitize opponent racks here.
		// AudGameTV events are pre-censored at the source (gameplay/game.go)
		// for league games and tournaments with private analysis.
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
		// Otherwise clone and censor opponent's rack.
		cloned := proto.Clone(subevt).(*pb.ServerGameplayEvent)
		entity.CensorRacks(cloned)
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
