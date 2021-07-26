package gameplay

import (
	"context"
	"errors"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/tournament"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrTooManyAborts = errors.New("you have made too many abort requests in this game")
	ErrTooManyNudges = errors.New("you have made too many nudges in this game")

	ErrNoMatchingEvent              = errors.New("no matching request to respond to")
	ErrTooManyTurns                 = errors.New("it is too late to abort")
	ErrPleaseWaitToEnd              = errors.New("this game is almost over; request not sent")
	ErrMetaEventExpirationIncorrect = errors.New("meta event did not expire")
	ErrAlreadyOutstandingRequest    = errors.New("you already have an outstanding request")
	ErrOutstandingRequestExists     = errors.New("please respond to existing request")
	// generic not allowed error; the front-end should disallow anything that can
	// return this error:
	ErrNotAllowed = errors.New("that action is not allowed")
)

const (
	// Per player, per game.
	MaxAllowedAbortRequests = 1
	MaxAllowedNudges        = 2
	// Disallow abort after this many turns.
	// XXX: This is purposefully somewhat high to account for people playing
	// in a club or legacy tournament oblivious to the fact that they should
	// be cancelling. We can make it lower as our chat implementation becomes
	// more obvious.
	AbortDisallowTurns = 7

	// If receiver has this many milliseconds on their clock or fewer, we don't allow
	// sending them requests.
	DisallowMsecsRemaining = 30 * 1000

	AbortTimeout = time.Second * 60
	NudgeTimeout = time.Second * 120
)

func numEvtsOfSameType(evts []*pb.GameMetaEvent, evt *pb.GameMetaEvent) int {
	log.Debug().Interface("evts", evts).Interface("evt", evt).Msg("counting-meta-evts")
	ct := 0
	for _, e := range evts {
		if e.Type == evt.Type && e.PlayerId == evt.PlayerId {
			ct++
		}
	}
	return ct
}

func intypes(t pb.GameMetaEvent_EventType, types []pb.GameMetaEvent_EventType) bool {
	for _, et := range types {
		if t == et {
			return true
		}
	}
	return false
}

func findLastMatchingEvt(evts []*pb.GameMetaEvent, evt *pb.GameMetaEvent) *pb.GameMetaEvent {

	var lookfor pb.GameMetaEvent_EventType
	var handlertypes []pb.GameMetaEvent_EventType
	switch evt.Type {
	case pb.GameMetaEvent_ABORT_ACCEPTED, pb.GameMetaEvent_ABORT_DENIED:
		lookfor = pb.GameMetaEvent_REQUEST_ABORT
		handlertypes = append(handlertypes, pb.GameMetaEvent_ABORT_ACCEPTED, pb.GameMetaEvent_ABORT_DENIED)
	case pb.GameMetaEvent_ADJUDICATION_ACCEPTED, pb.GameMetaEvent_ADJUDICATION_DENIED:
		lookfor = pb.GameMetaEvent_REQUEST_ADJUDICATION
		handlertypes = append(handlertypes, pb.GameMetaEvent_ADJUDICATION_ACCEPTED, pb.GameMetaEvent_ADJUDICATION_DENIED)

	default:
		return nil
	}

	log.Debug().Interface("evts", evts).Interface("evt", evt).Msg("looking for match")

	var lastEvt *pb.GameMetaEvent
	for _, e := range evts {
		if e.OrigEventId == evt.OrigEventId {
			if e.Type == lookfor {
				if lastEvt != nil {
					// There is already a matching event. There should only be one matching event.
					return nil
				}
				lastEvt = e
			} else if intypes(e.Type, handlertypes) {
				// This event has already been handled.
				return nil
			}
		}
	}
	return lastEvt
}

func lastEventWithId(evts []*pb.GameMetaEvent, origEvtId string) *pb.GameMetaEvent {
	var lastEvt *pb.GameMetaEvent
	for _, e := range evts {
		if e.OrigEventId == origEvtId {
			if lastEvt != nil {
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
	gameStore GameStore, userStore user.Store, notorietyStore mod.NotorietyStore,
	listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore) error {
	g, err := gameStore.Get(ctx, evt.GameId)
	if err != nil {
		return err
	}

	g.Lock()
	defer g.Unlock()

	if g.GameEndReason != pb.GameEndReason_NONE {
		// game is over. Don't actually return an error; but log the situation.
		log.Info().Msg("game-not-active")
		return nil
	}

	now := g.TimerModule().Now()
	tnow := time.Unix(0, now*int64(time.Millisecond)).UTC()

	evt.Timestamp = timestamppb.New(tnow)

	switch evt.Type {
	case pb.GameMetaEvent_REQUEST_ABORT,
		pb.GameMetaEvent_REQUEST_ADJUDICATION,
		pb.GameMetaEvent_REQUEST_UNDO,
		pb.GameMetaEvent_REQUEST_ADJOURN:

		// These are "original" events.
		n := numEvtsOfSameType(g.MetaEvents.Events, evt)
		if evt.Type == pb.GameMetaEvent_REQUEST_ABORT && n >= MaxAllowedAbortRequests {
			return ErrTooManyAborts
		}
		if evt.Type == pb.GameMetaEvent_REQUEST_ADJUDICATION && n >= MaxAllowedNudges {
			return ErrTooManyNudges
		}

		if evt.Type == pb.GameMetaEvent_REQUEST_ABORT && g.History() != nil &&
			len(g.History().Events) > AbortDisallowTurns {
			return ErrTooManyTurns
		}
		// Check if this player has another outstanding request open.
		if entity.LastOutstandingMetaRequest(g.MetaEvents.Events, evt.PlayerId, now) != nil {
			return ErrAlreadyOutstandingRequest
		}
		// Check if other player has an outstanding request open
		if entity.LastOutstandingMetaRequest(g.MetaEvents.Events, "", now) != nil {
			return ErrOutstandingRequestExists
		}
		if g.TournamentData != nil && g.TournamentData.Id != "" {
			// disallow adjudication/adjourn
			if evt.Type == pb.GameMetaEvent_REQUEST_ADJUDICATION ||
				evt.Type == pb.GameMetaEvent_REQUEST_ADJOURN {
				// note: adjourn is not implemented
				return ErrNotAllowed
			}
		}

		// XXX: Receiver may not be the one on turn, since either player may request abort.
		onTurn := g.Game.PlayerOnTurn()
		timeRemaining := g.TimeRemaining(onTurn)
		log.Debug().Int("timeRemaining", timeRemaining).Int("onturn", onTurn).Msg("timeremaining")
		// XXX: Should time remaining include overtime?
		if timeRemaining < DisallowMsecsRemaining {
			return ErrPleaseWaitToEnd
		}

		if g.Game.PlayerIDOnTurn() == evt.PlayerId && evt.Type == pb.GameMetaEvent_REQUEST_ADJUDICATION {
			// people with running clocks shouldn't be allowed to request adjudication.
			return ErrNotAllowed
		}

		// XXX: Adjust reasonably based on receiver's remaining time.
		// Add expiry time to event.
		if evt.Type == pb.GameMetaEvent_REQUEST_ABORT {
			evt.Expiry = int32(AbortTimeout.Seconds() * 1000)
		} else if evt.Type == pb.GameMetaEvent_REQUEST_ADJUDICATION {
			evt.Expiry = int32(NudgeTimeout.Seconds() * 1000)
		}

		// For this type of event, we just append it to the list and return.
		// The event will be sent via the appropriate game channel
		g.MetaEvents.Events = append(g.MetaEvents.Events, evt)
		err := gameStore.Set(ctx, g)
		if err != nil {
			return err
		}

		// Only send the event to the game channel. Observers don't need to know
		// that an abort was requested etc.
		wrapped := entity.WrapEvent(evt, pb.MessageType_GAME_META_EVENT)
		wrapped.AddAudience(entity.AudGame, evt.GameId)
		eventChan <- wrapped

	case pb.GameMetaEvent_TIMER_EXPIRED:
		// This event gets sent by the front end of the requester after
		// the time for an event has expired.
		matchingEvt := lastEventWithId(g.MetaEvents.Events, evt.OrigEventId)
		if matchingEvt == nil ||
			!(matchingEvt.Type == pb.GameMetaEvent_REQUEST_ABORT ||
				matchingEvt.Type == pb.GameMetaEvent_REQUEST_ADJUDICATION ||
				matchingEvt.Type == pb.GameMetaEvent_REQUEST_ADJOURN) {
			return ErrNoMatchingEvent
		}
		elapsed := tnow.Sub(matchingEvt.Timestamp.AsTime())
		if matchingEvt.Type == pb.GameMetaEvent_REQUEST_ABORT && elapsed >= AbortTimeout {
			// if time ran out, auto accept the abort
			// create a pseudo event.

			pseudoEvt := &pb.GameMetaEvent{
				OrigEventId: evt.OrigEventId,
				Timestamp:   evt.Timestamp,
				Type:        pb.GameMetaEvent_ABORT_ACCEPTED,
				GameId:      g.GameID(),
				// Do not add a player ID since technically this event was not accepted by the player.
			}
			g.MetaEvents.Events = append(g.MetaEvents.Events, evt)

			err = processMetaEvent(ctx, g, pseudoEvt, matchingEvt, gameStore, userStore, notorietyStore,
				listStatStore, tournamentStore)
			if err != nil {
				return err
			}

		} else if matchingEvt.Type == pb.GameMetaEvent_REQUEST_ADJUDICATION && elapsed >= NudgeTimeout {
			// if time ran out, auto adjudicate.

			pseudoEvt := &pb.GameMetaEvent{
				OrigEventId: evt.OrigEventId,
				Timestamp:   evt.Timestamp,
				Type:        pb.GameMetaEvent_ADJUDICATION_ACCEPTED,
				GameId:      g.GameID(),
				// Do not add a player ID since technically this event was not accepted by the player.
			}
			g.MetaEvents.Events = append(g.MetaEvents.Events, evt)

			err = processMetaEvent(ctx, g, pseudoEvt, matchingEvt, gameStore, userStore, notorietyStore,
				listStatStore, tournamentStore)
			if err != nil {
				return err
			}

		} else {
			return ErrMetaEventExpirationIncorrect
		}

	default:
		matchingEvt := findLastMatchingEvt(g.MetaEvents.Events, evt)
		if matchingEvt == nil {
			return ErrNoMatchingEvent
		}
		g.MetaEvents.Events = append(g.MetaEvents.Events, evt)

		err = processMetaEvent(ctx, g, evt, matchingEvt, gameStore, userStore, notorietyStore,
			listStatStore, tournamentStore)
		if err != nil {
			return err
		}

		// Send the event here as well.
		wrapped := entity.WrapEvent(evt, pb.MessageType_GAME_META_EVENT)
		wrapped.AddAudience(entity.AudGame, evt.GameId)
		wrapped.AddAudience(entity.AudGameTV, evt.GameId)
		eventChan <- wrapped
	}

	return nil
}

func cancelMetaEvent(ctx context.Context, g *entity.Game, evt *pb.GameMetaEvent) error {

	var pseudoEvt *pb.GameMetaEvent
	if evt.Type == pb.GameMetaEvent_REQUEST_ADJUDICATION {
		pseudoEvt = &pb.GameMetaEvent{
			OrigEventId: evt.OrigEventId,
			Timestamp:   evt.Timestamp,
			Type:        pb.GameMetaEvent_ADJUDICATION_DENIED,
			GameId:      g.GameID(),
			// Do not add a player ID since technically this event was not denied by the player.
		}
	} else if evt.Type == pb.GameMetaEvent_REQUEST_ABORT {
		pseudoEvt = &pb.GameMetaEvent{
			OrigEventId: evt.OrigEventId,
			Timestamp:   evt.Timestamp,
			Type:        pb.GameMetaEvent_ABORT_DENIED,
			GameId:      g.GameID(),
			// Do not add a player ID since technically this event was not denied by the player.
		}
	}
	// don't need to call processMetaEvent here as a "deny" event is essentially
	// a no-op (we only add it to the list of events).
	g.MetaEvents.Events = append(g.MetaEvents.Events, pseudoEvt)

	// send the cancellation event.

	wrapped := entity.WrapEvent(pseudoEvt, pb.MessageType_GAME_META_EVENT)
	wrapped.AddAudience(entity.AudGame, evt.GameId)
	wrapped.AddAudience(entity.AudGameTV, evt.GameId)
	g.SendChange(wrapped)

	return nil
}

func processMetaEvent(ctx context.Context, g *entity.Game, evt *pb.GameMetaEvent, matchingEvt *pb.GameMetaEvent,
	gameStore GameStore, userStore user.Store, notorietyStore mod.NotorietyStore,
	listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore) error {
	// process an event in a locked game. evt is the event that came in,
	// and matchingEvt is the event that it corresponds to.
	// evt is always going to be of a "response" type (like accept/decline),
	// matchingEvt is always going to be of the original type.

	switch evt.Type {
	case pb.GameMetaEvent_ABORT_ACCEPTED:
		// Abort the game.
		log.Info().Str("gameID", g.GameID()).Msg("abort-accepted")
		err := AbortGame(ctx, gameStore, tournamentStore, g, pb.GameEndReason_ABORTED)
		if err != nil {
			return err
		}

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
		return performEndgameDuties(ctx, g, gameStore, userStore, notorietyStore, listStatStore, tournamentStore)
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
