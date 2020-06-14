package sockets

import (
	"context"
	"errors"
	"fmt"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	pb "github.com/domino14/liwords/rpc/api/proto"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
)

func (h *Hub) parseAndExecuteMessage(ctx context.Context, msg []byte, sender string) error {
	// All socket messages are encoded entity.Events.
	// (or they better be)

	ew, err := entity.EventFromByteArray(msg)
	if err != nil {
		return err
	}
	switch ew.Type {
	case pb.MessageType_SEEK_REQUEST:
		evt, ok := ew.Event.(*pb.SeekRequest)
		if !ok {
			return errors.New("sr unexpected typing error")
		}
		sg, err := gameplay.NewSoughtGame(ctx, h.soughtGameStore, evt)
		if err != nil {
			return err
		}

		h.NewSeekRequest(sg.SeekRequest)

	case pb.MessageType_GAME_ACCEPTED_EVENT:
		evt, ok := ew.Event.(*pb.GameAcceptedEvent)
		if !ok {
			return errors.New("gae unexpected typing error")
		}

		sg, err := h.soughtGameStore.Get(ctx, evt.RequestId)
		if err != nil {
			return err
		}
		requester := sg.SeekRequest.User.Username
		if requester == sender {
			log.Info().Str("sender", sender).Msg("canceling seek")
			err := gameplay.CancelSoughtGame(ctx, h.soughtGameStore, evt.RequestId)
			if err != nil {
				return err
			}
			// broadcast a seek deletion.
			err = h.DeleteSeek(evt.RequestId)
			if err != nil {
				return err
			}
			return err
		}

		players := []*macondopb.PlayerInfo{
			{Nickname: sender, RealName: sender},
			{Nickname: requester, RealName: requester},
		}
		log.Debug().Interface("seekreq", sg.SeekRequest).Msg("seek-request-accepted")

		g, err := gameplay.InstantiateNewGame(ctx, h.gameStore, h.config,
			players, sg.SeekRequest.GameRequest)
		if err != nil {
			return err
		}
		// Broadcast a seek delete event, and send both parties a game redirect.
		h.soughtGameStore.Delete(ctx, evt.RequestId)
		err = h.DeleteSeek(evt.RequestId)
		if err != nil {
			return err
		}
		// This event will result in a redirect.
		ngevt := entity.WrapEvent(&pb.NewGameEvent{
			GameId: g.GameID(),
		}, pb.MessageType_NEW_GAME_EVENT, "")

		h.sendToClient(sender, ngevt)
		h.sendToClient(requester, ngevt)
		// Create a new realm to put these players in.
		realm := h.addNewRealm(g.GameID())
		h.addToRealm(realm, sender)
		h.addToRealm(realm, requester)
		log.Info().Str("newgameid", g.History().Uid).
			Str("sender", sender).
			Str("requester", requester).
			Str("onturn", g.NickOnTurn()).Msg("game-accepted")

		// Now, reset the timer and register the event change hook.
		err = gameplay.StartGame(ctx, h.gameStore, h.eventChan, g.GameID())
		if err != nil {
			return err
		}

	case pb.MessageType_CLIENT_GAMEPLAY_EVENT:
		evt, ok := ew.Event.(*pb.ClientGameplayEvent)
		if !ok {
			// This really shouldn't happen
			return errors.New("cge unexpected typing error")
		}
		err := gameplay.PlayMove(ctx, h.gameStore, sender, evt)
		if err != nil {
			return err
		}

	case pb.MessageType_REGISTER_REALM:
		evt, ok := ew.Event.(*pb.RegisterRealm)
		if !ok {
			// This really shouldn't happen
			return errors.New("rr unexpected typing error")
		}
		h.addToRealm(Realm(evt.Realm), sender)
		h.sendRealmData(ctx, Realm(evt.Realm), sender)

	case pb.MessageType_DEREGISTER_REALM:
		evt, ok := ew.Event.(*pb.DeregisterRealm)
		if !ok {
			// This really shouldn't happen
			return errors.New("dr unexpected typing error")
		}
		h.removeFromRealm(Realm(evt.Realm), sender)

	case pb.MessageType_TIMED_OUT:
		evt, ok := ew.Event.(*pb.TimedOut)
		if !ok {
			return errors.New("to unexpected typing error")
		}
		// Verify the timeout.
		err := gameplay.TimedOut(ctx, h.gameStore, sender, evt.Username, evt.GameId)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("message type %v not yet handled", ew.Type)

	}

	return nil
}
