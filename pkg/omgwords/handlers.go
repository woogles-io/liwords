package omgwords

import (
	"context"
	"fmt"

	"github.com/domino14/liwords/pkg/cwgame"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/omgwords/stores"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
)

// handlers handle events from either the HTTP API or from the websocket/bus
func handleEvent(ctx context.Context, userID string, evt *ipc.ClientGameplayEvent,
	gs *stores.GameDocumentStore, evtChan chan *entity.EventWrapper) error {

	g, err := gs.GetDocument(ctx, evt.GameId, true)
	if err != nil {
		return err
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
		return err
	}

	// Now check for changes and send events accordingly.
	if len(g.Events) != oldNumEvents {
		// This will pretty much always happen if we didn't return an error.
		newEvents := g.Events[oldNumEvents:]
		for _, evt := range newEvents {
			sge := &ipc.ServerOMGWordsEvent{}
			sge.Event = evt
			sge.GameId = g.Uid
			sge.TimeRemaining = int32(g.Timers.TimeRemaining[g.PlayerOnTurn])
			sge.NewRack = g.Racks[g.PlayerOnTurn]
			sge.Playing = g.PlayState
			sge.UserId = g.Players[evt.PlayerIndex].UserId

			wrapped := entity.WrapEvent(sge, ipc.MessageType_OMGWORDS_GAMEPLAY_EVENT)
			wrapped.AddAudience(entity.AudGameTV, g.Uid)
			for _, p := range g.Players {
				wrapped.AddAudience(entity.AudUser,
					fmt.Sprintf("%s.game.%s", p.UserId, g.Uid))
			}
			evtChan <- wrapped
		}
	}

	err = gs.UpdateDocument(ctx, g)
	if err != nil {
		return err
	}

	if g.PlayState == ipc.PlayState_GAME_OVER {
		// rate the game and send such and such.
		// performendgameduties

	} else {
		// potentially send bot move request
	}
	return nil

}
