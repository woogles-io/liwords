package gameplay

import (
	"context"
	"errors"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/tournament"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
)

var (
	errTooManyAborts = errors.New("you have made too many abort requests in this game")
	errTooManyNudges = errors.New("you have made too many nudges in this game")

	errNoMatchingEvent = errors.New("no matching request to respond to")
)

const (
	// Per player, per game.
	MaxAllowedAbortRequests = 1
	MaxAllowedNudges        = 2

	AbortTimeout = time.Second * 60
	NudgeTimeout = time.Second * 120
)

func numEvtsOfSameType(evts []*pb.GameMetaEvent, evt *pb.GameMetaEvent) int {
	ct := 0
	for _, e := range evts {
		if e.Type == evt.Type && e.PlayerId == evt.PlayerId {
			ct++
		}
	}
	return ct
}

func findLastEvtOfMatchingType(evts []*pb.GameMetaEvent, evt *pb.GameMetaEvent) *pb.GameMetaEvent {

	var lookfor pb.GameMetaEvent_EventType

	switch evt.Type {
	case pb.GameMetaEvent_ABORT_ACCEPTED, pb.GameMetaEvent_ABORT_DENIED:
		lookfor = pb.GameMetaEvent_REQUEST_ABORT
	case pb.GameMetaEvent_ADJUDICATION_ACCEPTED, pb.GameMetaEvent_ADJUDICATION_DENIED:
		lookfor = pb.GameMetaEvent_REQUEST_ADJUDICATION
	default:
		return nil
	}

	var lastEvt *pb.GameMetaEvent
	for _, e := range evts {
		if e.Type == lookfor && e.OrigEventId == evt.OrigEventId && e.PlayerId != evt.PlayerId {
			if lastEvt != nil {
				// There is already a matching event. There should only be one matching event.
				return nil
			}
			lastEvt = e
		}
	}
	return lastEvt
}

// Meta Events are events such as abort requests, adding time,
// adjudication requests, etc. Not so much for the actual gameplay.

// HandleMetaEvent processes a passed-in Meta Event, returning an error if
// it is not applicable.
func HandleMetaEvent(ctx context.Context, evt *pb.GameMetaEvent, eventChan chan<- *entity.EventWrapper,
	gameStore GameStore, userStore user.Store,
	listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore) error {
	g, err := gameStore.Get(ctx, evt.GameId)
	if err != nil {
		return err
	}
	g.Lock()
	defer g.Unlock()

	switch evt.Type {
	case pb.GameMetaEvent_REQUEST_ABORT,
		pb.GameMetaEvent_REQUEST_ADJUDICATION,
		pb.GameMetaEvent_REQUEST_UNDO,
		pb.GameMetaEvent_REQUEST_ADJOURN:

		// These are "original" events.
		n := numEvtsOfSameType(g.MetaEvents.Events, evt)
		if evt.Type == pb.GameMetaEvent_REQUEST_ABORT && n > MaxAllowedAbortRequests {
			return errTooManyAborts
		}
		if evt.Type == pb.GameMetaEvent_REQUEST_ADJUDICATION && n > MaxAllowedNudges {
			return errTooManyNudges
		}
		// For this type of event, we just append it to the list and return.
		// The event will be sent via the appropriate game channel
		g.MetaEvents.Events = append(g.MetaEvents.Events, evt)
		err := gameStore.Set(ctx, g)
		if err != nil {
			return err
		}

		wrapped := entity.WrapEvent(evt, pb.MessageType_GAME_META_EVENT)
		wrapped.AddAudience(entity.AudGame, evt.GameId)
		wrapped.AddAudience(entity.AudGameTV, evt.GameId)
		eventChan <- wrapped

	default:
		matchingEvt := findLastEvtOfMatchingType(g.MetaEvents.Events, evt)
		if matchingEvt == nil {
			return errNoMatchingEvent
		}
		g.MetaEvents.Events = append(g.MetaEvents.Events, evt)

		err = processMetaEvent(ctx, g, evt, matchingEvt, gameStore, userStore,
			listStatStore, tournamentStore)
		if err != nil {
			return err
		}

		// Send the event here as well. XXX can move outside of switch?
		// XXX only send it for denies. If game is aborted / adjudicated we don't
		// need to send the event for that.
		wrapped := entity.WrapEvent(evt, pb.MessageType_GAME_META_EVENT)
		wrapped.AddAudience(entity.AudGame, evt.GameId)
		wrapped.AddAudience(entity.AudGameTV, evt.GameId)
		eventChan <- wrapped
	}

	return nil
}

func processMetaEvent(ctx context.Context, g *entity.Game, evt *pb.GameMetaEvent, matchingEvt *pb.GameMetaEvent,
	gameStore GameStore, userStore user.Store,
	listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore) error {
	// process an event in a locked game. evt is the event that came in,
	// and matchingEvt is the event that it corresponds to.
	// evt is always going to be of a "response" type (like accept/decline),
	// matchingEvt is always going to be of the original type.

	switch evt.Type {
	case pb.GameMetaEvent_ABORT_ACCEPTED:
		// Abort the game.
		log.Info().Str("gameID", g.GameID()).Msg("abort-accepted")
		g.SetGameEndReason(pb.GameEndReason_ABORTED)
		g.History().PlayState = macondopb.PlayState_GAME_OVER
		err := gameStore.Set(ctx, g)
		if err != nil {
			return err
		}
		gameStore.Unload(ctx, g.GameID())

		// XXX: Send a game ended event here like the performEndgameDuties does
	case pb.GameMetaEvent_ABORT_DENIED:
		log.Info().Str("gameID", g.GameID()).Msg("abort-denied")
		err := gameStore.Set(ctx, g)
		if err != nil {
			return err
		}
	case pb.GameMetaEvent_ADJUDICATION_ACCEPTED:
		log.Info().Str("gameID", g.GameID()).Msg("adjudication-accepted")
		g.SetGameEndReason(pb.GameEndReason_FORCE_FORFEIT)
		g.History().PlayState = macondopb.PlayState_GAME_OVER
		// The playerid in the original event is the player who initiated the adjudication.
		// So they should be the winner of this game.
		hist := g.Game.History()
		winner := 0
		if matchingEvt.PlayerId == hist.Players[0].UserId {
			// winner already set to 0
		} else if matchingEvt.PlayerId == hist.Players[1].UserId {
			winner = 1
		} else {
			return errors.New("matching-evt-player-id-not-found")
		}

		g.SetWinnerIdx(winner)
		g.SetLoserIdx(1 - winner)
		// performEndgameDuties Sets the game back to the store, so no need to do it again here,
		// unlike in the other cases.
		return performEndgameDuties(ctx, g, gameStore, userStore, listStatStore, tournamentStore)
	case pb.GameMetaEvent_ADJUDICATION_DENIED:
		log.Info().Str("gameID", g.GameID()).Msg("adjudication-denied")
		err := gameStore.Set(ctx, g)
		if err != nil {
			return err
		}
	default:
		return errors.New("event not handled")
	}
	return nil
}
