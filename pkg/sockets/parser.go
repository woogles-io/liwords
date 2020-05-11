package sockets

import (
	"context"
	"errors"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"

	"github.com/domino14/crosswords/pkg/entity"
	"github.com/domino14/crosswords/pkg/game"
	pb "github.com/domino14/crosswords/rpc/api/proto"
)

func (h *Hub) registerAndRunEvtQueue(evtChan <-chan *entity.EventWrapper, realm Realm) {
	// This function is meant to run inside a goroutine.
	// XXX: if this function stops running we will have a deadlock during gameplay!
	// this must be thoroughly tested.
	// At the same time we must figure out how to kill it so we don't have a bunch
	// of these running uselessly! think this through carefully.
forloop:
	for {
		select {
		case w := <-evtChan:
			err := h.sendToRealm(realm, w)
			if err != nil {
				log.Err(err).Str("realm", string(realm)).Msg("sending to realm")
				break forloop
			}
		}
	}
	log.Debug().Str("realm", string(realm)).Msg("Exiting event queue for realm")

}

func (h *Hub) parseAndExecuteMessage(msg []byte, sender string) error {
	// All socket messages are encoded entity.Events.
	// (or they better be)

	ew, err := entity.EventFromByteArray(msg)
	if err != nil {
		return err
	}
	switch ew.Name {

	case "crosswords.GameAcceptedEvent":
		evt, ok := ew.Event.(*pb.GameAcceptedEvent)
		if !ok {
			return errors.New("unexpected typing error")
		}
		// XXX: We're going to want to fetch these player IDs from the database later.
		players := []*macondopb.PlayerInfo{
			&macondopb.PlayerInfo{Nickname: evt.Acceptor, RealName: evt.Acceptor},
			&macondopb.PlayerInfo{Nickname: evt.Requester, RealName: evt.Requester},
		}
		// Create a "realm" for these users, and add both of them to the realm.
		realm := h.addNewRealm()
		h.addToRealm(realm, evt.Acceptor)
		h.addToRealm(realm, evt.Requester)

		// Create a channel to pass to the game. This channel is only written
		// to by the game, so we must only read from it here.
		evtChan := make(chan *entity.EventWrapper)
		go h.registerAndRunEvtQueue(evtChan, realm)

		_, err := game.StartNewGame(context.Background(), h.gameStore, h.config,
			players, evt.GameRequest, evtChan)
		if err != nil {
			return err
		}

	case "crosswords.UserGameplayEvent":
		evt, ok := ew.Event.(*pb.UserGameplayEvent)
		if !ok {
			// This really shouldn't happen
			return errors.New("unexpected typing error")
		}
		err := game.PlayMove(context.Background(), h.gameStore, sender, evt)
		if err != nil {
			return err
		}

	default:
		return errors.New("evt " + ew.Name + " not yet handled")

	}

	return nil
}
