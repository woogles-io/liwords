package sockets

import (
	"context"
	"errors"
	"fmt"

	"github.com/domino14/crosswords/pkg/entity"
	"github.com/domino14/crosswords/pkg/gameplay"
	pb "github.com/domino14/crosswords/rpc/api/proto"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
)

func (h *Hub) parseAndExecuteMessage(msg []byte, sender string) error {
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
			return errors.New("unexpected typing error")
		}
		sg, err := gameplay.NewSoughtGame(context.Background(), h.soughtGameStore, evt)
		if err != nil {
			return err
		}

		h.NewSeekRequest(sg.SeekRequest)

	case pb.MessageType_GAME_ACCEPTED_EVENT:
		ctx := context.Background()
		evt, ok := ew.Event.(*pb.GameAcceptedEvent)
		if !ok {
			return errors.New("unexpected typing error")
		}

		sg, err := h.soughtGameStore.Get(ctx, evt.RequestId)
		if err != nil {
			return err
		}
		requester := sg.SeekRequest.User.Username
		if requester == sender {
			return errors.New("cannot accept own seek")
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

		// Now, send a start game event.
		err = gameplay.StartGame(ctx, h.gameStore, h.eventChan, g.GameID())
		if err != nil {
			return err
		}

	case pb.MessageType_CLIENT_GAMEPLAY_EVENT:
		evt, ok := ew.Event.(*pb.ClientGameplayEvent)
		if !ok {
			// This really shouldn't happen
			return errors.New("unexpected typing error")
		}
		err := gameplay.PlayMove(context.Background(), h.gameStore, sender, evt)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("message type %v not yet handled", ew.Type)

	}

	return nil
}
