package sockets

import (
	"context"
	"errors"
	"fmt"

	"github.com/domino14/crosswords/pkg/entity"
	"github.com/domino14/crosswords/pkg/game"
	pb "github.com/domino14/crosswords/rpc/api/proto"
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
		h.NewSeekRequest(evt)

	// case "crosswords.GameAcceptedEvent":
	// 	evt, ok := ew.Event.(*pb.GameAcceptedEvent)
	// 	if !ok {
	// 		return errors.New("unexpected typing error")
	// 	}
	// 	// XXX: We're going to want to fetch these player IDs from the database later.
	// 	players := []*macondopb.PlayerInfo{
	// 		{Nickname: evt.Acceptor, RealName: evt.Acceptor},
	// 		{Nickname: evt.Requester, RealName: evt.Requester},
	// 	}

	// 	g, err := game.InstantiateNewGame(context.Background(), h.gameStore, h.config,
	// 		players, evt.GameRequest)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	// Create a "realm" for these users, and add both of them to the realm.
	// 	// Use the id of the game as the id of the realm.
	// 	realm := h.addNewRealm(g.GameID())
	// 	h.addToRealm(realm, evt.Acceptor)
	// 	h.addToRealm(realm, evt.Requester)
	// 	// Now, we start the timer and register the event hook.
	// 	err = game.StartGameInstance(g, h.eventChan)
	// 	if err != nil {
	// 		return err
	// 	}

	case pb.MessageType_CLIENT_GAMEPLAY_EVENT:
		evt, ok := ew.Event.(*pb.ClientGameplayEvent)
		if !ok {
			// This really shouldn't happen
			return errors.New("unexpected typing error")
		}
		err := game.PlayMove(context.Background(), h.gameStore, sender, evt)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("message type %v not yet handled", ew.Type)

	}

	return nil
}
