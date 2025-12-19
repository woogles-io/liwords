package omgwords

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	wglconfig "github.com/domino14/word-golib/config"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/cwgame"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/omgwords/stores"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/proto"
)

// handlers handle events from either the HTTP API or from the websocket/bus
// The return boolean indicates if the game ended as a result of this event.
func handleEvent(ctx context.Context, cfg *wglconfig.Config, userID string, evt *ipc.ClientGameplayEvent,
	amendment bool, evtIndex uint32, gs *stores.GameDocumentStore,
	evtChan chan *entity.EventWrapper) (bool, error) {

	g, err := gs.GetDocument(ctx, evt.GameId, true)
	if err != nil {
		return false, err
	}

	// amendment is sent when we try to edit an already existing game event
	// in the past. This can only be done for annotated games.
	if amendment {
		gameEnded, err := handleAmendment(ctx, cfg, userID, evt, evtIndex, g, gs, evtChan)
		// handleAmendment unlocks the document, so we can return directly
		return gameEnded, err
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

	err = cwgame.ProcessGameplayEvent(ctx, cfg, evt, userID, g.GameDocument)
	if err != nil {
		gs.UnlockDocument(ctx, g)
		return false, apiserver.InvalidArg(err.Error())
	}

	// Validate tile counts after processing the event
	if g.Type == ipc.GameType_ANNOTATED {
		dist, distErr := cwgame.GetLetterDistribution(cfg, g.LetterDistribution)
		if distErr == nil {
			expectedTotal := int(dist.NumTotalLetters())
			if validationErr := cwgame.ValidateTotalTiles(g.GameDocument, expectedTotal); validationErr != nil {
				zerolog.Ctx(ctx).Error().
					Err(validationErr).
					Str("game_id", evt.GameId).
					Str("editor_op", "send_game_event").
					Int("expected", expectedTotal).
					Int("bag", len(g.Bag.Tiles)).
					Int("rack0", len(g.Racks[0])).
					Int("rack1", len(g.Racks[1])).
					Msg("editor-tile-mismatch")
				panic(fmt.Sprintf("TILE CORRUPTION DETECTED: %v", validationErr))
			}
			// Also validate per-letter distribution
			if validationErr := cwgame.ValidateTileDistribution(g.GameDocument, dist); validationErr != nil {
				zerolog.Ctx(ctx).Error().
					Err(validationErr).
					Str("game_id", evt.GameId).
					Str("editor_op", "send_game_event").
					Msg("editor-tile-distribution-mismatch")
				panic(fmt.Sprintf("TILE DISTRIBUTION CORRUPTION DETECTED: %v", validationErr))
			}
		}
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

// handleAmendment handles editing a past event in a non-destructive way.
// It preserves subsequent events and attempts to re-apply them.
func handleAmendment(ctx context.Context, cfg *wglconfig.Config, userID string,
	evt *ipc.ClientGameplayEvent, evtIndex uint32, g *stores.MaybeLockedDocument,
	gs *stores.GameDocumentStore, evtChan chan *entity.EventWrapper) (bool, error) {

	if g.Type != ipc.GameType_ANNOTATED {
		gs.UnlockDocument(ctx, g)
		return false, apiserver.InvalidArg("you can only amend annotated games")
	}
	if len(g.Events)-1 < int(evtIndex) {
		gs.UnlockDocument(ctx, g)
		return false, apiserver.InvalidArg("tried to amend a rack for a non-existing event")
	}

	rack := g.Events[evtIndex].Rack
	pidx := g.Events[evtIndex].PlayerIndex

	// Clone the document to work on - we'll only update the real document if everything succeeds
	gdocClone := proto.Clone(g.GameDocument).(*ipc.GameDocument)

	// NON-DESTRUCTIVE: Save events after the edit point
	var eventsAfter []*ipc.GameEvent
	if int(evtIndex)+1 < len(g.Events) {
		eventsAfter = make([]*ipc.GameEvent, len(g.Events)-int(evtIndex)-1)
		copy(eventsAfter, g.Events[evtIndex+1:])
	}

	// Replay events up to (but not including) the event we're editing
	evts := gdocClone.Events[:evtIndex]
	cwgame.LogTileState(gdocClone, "before-replay")
	err := cwgame.ReplayEvents(ctx, cfg, gdocClone, evts, false)
	if err != nil {
		gs.UnlockDocument(ctx, g)
		return false, apiserver.InvalidArg(err.Error())
	}
	cwgame.LogTileState(gdocClone, "after-replay")

	// Remember the rack we just saved. We need to re-assign it.
	racks := make([][]byte, len(g.Players))
	racks[pidx] = rack
	err = cwgame.AssignRacks(cfg, gdocClone, racks, cwgame.AssignEmptyIfUnambiguous)
	if err != nil {
		gs.UnlockDocument(ctx, g)
		return false, apiserver.InvalidArg(err.Error())
	}
	cwgame.LogTileState(gdocClone, "after-assign-racks")

	// Process the new/edited event
	err = cwgame.ProcessGameplayEvent(ctx, cfg, evt, userID, gdocClone)
	if err != nil {
		gs.UnlockDocument(ctx, g)
		return false, apiserver.InvalidArg(err.Error())
	}
	cwgame.LogTileState(gdocClone, "after-process-event")

	// Re-apply subsequent events in editor mode
	// If any event fails to apply, truncate at that point
	// The visual feedback (subsequent events disappearing) shows the truncation
	log := zerolog.Ctx(ctx)
	for i, savedEvt := range eventsAfter {
		cwgame.LogTileState(gdocClone, fmt.Sprintf("before-reapply-event-%d", i))
		err := cwgame.ApplyEventInEditorMode(ctx, cfg, gdocClone, savedEvt)
		if err != nil {
			log.Warn().
				Err(err).
				Str("game_id", g.Uid).
				Int("event_index", int(evtIndex)+1+i).
				Msg("game truncated: subsequent event could not be re-applied after amendment")
			cwgame.LogTileState(gdocClone, fmt.Sprintf("after-failed-reapply-event-%d", i))
			break
		}
		cwgame.LogTileState(gdocClone, fmt.Sprintf("after-reapply-event-%d", i))
	}

	// Ensure racks are replenished after truncation/replay
	err = cwgame.AssignRacks(cfg, gdocClone, [][]byte{nil, nil}, cwgame.AlwaysAssignEmpty)
	if err != nil {
		gs.UnlockDocument(ctx, g)
		return false, apiserver.InvalidArg(err.Error())
	}
	cwgame.LogTileState(gdocClone, "after-replenish-racks")

	// All operations succeeded - copy the clone back to the real document
	g.GameDocument = gdocClone

	// Send entire document event for amendments
	docEvt := &ipc.GameDocumentEvent{
		Doc: proto.Clone(g.GameDocument).(*ipc.GameDocument),
	}
	wrapped := entity.WrapEvent(docEvt, ipc.MessageType_OMGWORDS_GAMEDOCUMENT)
	wrapped.AddAudience(entity.AudChannel, AnnotatedChannelName(g.Uid))
	evtChan <- wrapped

	err = gs.UpdateDocument(ctx, g)
	if err != nil {
		gs.UnlockDocument(ctx, g)
		return false, err
	}

	gs.UnlockDocument(ctx, g)

	gameEnded := g.PlayState == ipc.PlayState_GAME_OVER
	return gameEnded, nil
}
