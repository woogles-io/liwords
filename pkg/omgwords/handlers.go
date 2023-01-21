package omgwords

import (
	"context"
	"fmt"

	"github.com/domino14/liwords/pkg/cwgame"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/omgwords/stores"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/twitchtv/twirp"
)

// handlers handle events from either the HTTP API or from the websocket/bus
// The return boolean indicates if the game ended as a result of this event.
func handleEvent(ctx context.Context, userID string, evt *ipc.ClientGameplayEvent,
	amendment bool, evtIndex uint32, gs *stores.GameDocumentStore,
	evtChan chan *entity.EventWrapper) (bool, error) {

	g, err := gs.GetDocument(ctx, evt.GameId, true)
	if err != nil {
		return false, err
	}

	// amendment is sent when we try to edit an already existing game event
	// in the past. This can only be done for annotated games.
	if amendment {
		if g.Type != ipc.GameType_ANNOTATED {
			gs.UnlockDocument(ctx, g)
			return false, twirp.NewError(twirp.InvalidArgument, "you can only amend annotated games")
		}
		if len(g.Events)-1 < int(evtIndex) {
			gs.UnlockDocument(ctx, g)
			return false, twirp.NewError(twirp.InvalidArgument, "tried to amend a rack for a non-existing event")
		}
		rack := g.Events[evtIndex].Rack
		pidx := g.Events[evtIndex].PlayerIndex
		// Truncate the document; we are editing an old event.
		evts := g.Events[:evtIndex]
		err = cwgame.ReplayEvents(ctx, g.GameDocument, evts)
		if err != nil {
			gs.UnlockDocument(ctx, g)
			return false, err
		}
		// Remember the rack we just saved. We need to re-assign it.
		racks := make([][]byte, len(g.Players))
		racks[pidx] = rack
		err = cwgame.AssignRacks(g.GameDocument, racks, true)
		if err != nil {
			gs.UnlockDocument(ctx, g)
			return false, err
		}

		// Now we have a truncated document ready to be modified
	}

	// Pretty much anything in the game document can change after the event
	// is processed, but we need to keep track of the changes that should cause
	// messages to be sent out.
	// The most important changes, in terms of messages that need to be sent out:
	// 1) The events array
	// 2) The racks
	// 3) The play_state (and end_reason, usually)
	// 4) The winner - this only has meaning once the play_state is GAME_OVER
	// 5) The timers
	// 6) The player on turn

	// Save the old values
	oldNumEvents := len(g.Events)

	err = cwgame.ProcessGameplayEvent(ctx, evt, userID, g.GameDocument)
	if err != nil {
		gs.UnlockDocument(ctx, g)
		return false, err
	}

	// REMOVE ME BEFORE DEPLOY
	// err = cwgame.ReconcileAllTiles(ctx, g.GameDocument)
	// if err != nil {
	// 	gs.UnlockDocument(ctx, g)
	// 	err = fmt.Errorf("failed-to-reconcile-handleevent: %w", err)
	// 	return false, twirp.NewError(twirp.InvalidArgument, err.Error())
	// }

	if amendment {
		// Send an entire document event.
		evt := &ipc.GameDocumentEvent{
			Doc: g.GameDocument,
		}
		wrapped := entity.WrapEvent(evt, ipc.MessageType_OMGWORDS_GAMEDOCUMENT)
		wrapped.AddAudience(entity.AudChannel, AnnotatedChannelName(g.Uid))
		evtChan <- wrapped
	} else {
		// Now check for changes and send events accordingly.
		if len(g.Events) != oldNumEvents {
			// This will pretty much always happen if we didn't return an error.
			newEvents := g.Events[oldNumEvents:]
			for _, evt := range newEvents {
				sge := &ipc.ServerOMGWordsEvent{}
				sge.Event = evt
				sge.GameId = g.Uid
				sge.TimeRemaining = int32(g.Timers.TimeRemaining[g.PlayerOnTurn])
				sge.NewRack = g.Racks[evt.PlayerIndex]
				sge.Playing = g.PlayState
				sge.UserId = g.Players[evt.PlayerIndex].UserId

				wrapped := entity.WrapEvent(sge, ipc.MessageType_OMGWORDS_GAMEPLAY_EVENT)
				if g.Type == ipc.GameType_ANNOTATED {
					wrapped.AddAudience(entity.AudChannel, AnnotatedChannelName(g.Uid))
				} else {
					wrapped.AddAudience(entity.AudGameTV, g.Uid)
					for _, p := range g.Players {
						wrapped.AddAudience(entity.AudUser,
							fmt.Sprintf("%s.game.%s", p.UserId, g.Uid))
					}
				}
				evtChan <- wrapped
			}
		}
	}

	err = gs.UpdateDocument(ctx, g)
	if err != nil {
		return false, err
	}
	gameEnded := false
	if g.PlayState == ipc.PlayState_GAME_OVER {
		// rate the game and send such and such.
		// performendgameduties
		gameEnded = true
	} else {
		// potentially send bot move request
	}
	return gameEnded, nil

}
