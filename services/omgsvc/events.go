package omgsvc

import (
	"context"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
)

// MsgHandler handles all game svc related messages
func MsgHandler(ctx context.Context, b *ipc.Bus, topic string, data []byte, reply string) error {
	// MessageType_GAME_META_EVENT
	// MessageType_CLIENT_GAMEPLAY_EVENT
	// MessageType_TIMED_OUT
	// MessageType_READY_FOR_GAME

	log.Debug().Interface("topic", topic).Msg("msghandler")
	subtopics := strings.Split("topic", ".")

	msgType := subtopics[0]
	auth := ""
	userID := ""
	if len(subtopics) > 2 {
		auth = subtopics[1]
		userID = subtopics[2]
	}
	wsConnID := ""
	if len(subtopics) > 3 {
		wsConnID = subtopics[3]
	}
	pnum, err := strconv.Atoi(msgType)
	if err != nil {
		return err
	}
	msgType = pb.MessageType(pnum).String()

	switch msgType {
	case pb.MessageType_GAME_INSTANTIATION.String():
		return handleGameInstantiation(ctx, b, data, reply)

	case pb.MessageType_CLIENT_GAMEPLAY_EVENT.String():
		// evt := &pb.ClientGameplayEvent{}
		// err := proto.Unmarshal(data, evt)
		// if err != nil {
		// 	return err
		// }

	}

	return nil
}

func handleGameInstantiation(ctx context.Context, b *ipc.Bus, data []byte,
	reply string) error {

	evt := &pb.InstantiateGame{}
	err := proto.Unmarshal(data, evt)
	if err != nil {
		return err
	}
	// immediately forward this event onto the store.
	// return the ID.
	res, err := b.Request("storesvc.omgwords.newgame", data)
	if err != nil {
		return err
	}
	// Respond back to caller with ID.
	err = b.PublishToTopic(reply, res)
	if err != nil {
		return err
	}
	resmsg := &pb.InstantiateGameResponse{}
	err = proto.Unmarshal(res, resmsg)
	if err != nil {
		return err
	}

	ngevt := entity.WrapEvent(&pb.NewGameEvent{
		GameId: resmsg.Id,
		// Doesn't matter who's accepter/requester; consider renaming these.
		AccepterCid:  evt.ConnIds[0],
		RequesterCid: evt.ConnIds[1],
	}, pb.MessageType_NEW_GAME_EVENT)

	// We have a game ID. Send this to all participants.
	ser, err := ngevt.Serialize()
	if err != nil {
		return err
	}
	for _, u := range evt.UserIds {
		b.PublishToUser(u, ser, "")
	}
	// Also broadcast the game creation to the lobby/whoever needs to know

	ongoingGameEvt := entity.WrapEvent(resmsg.GameInfo, pb.MessageType_ONGOING_GAME_EVENT)
	ogData, err := ongoingGameEvt.Serialize()
	if err != nil {
		return err
	}
	err = b.PublishToTopic("lobby.newLiveGame", ogData)
	if err != nil {
		return err
	}

	// Also publish to tournament channel if this is a tournament game.
	if resmsg.GameInfo.TournamentId != "" {
		channelName := "tournament." + resmsg.GameInfo.TournamentId + ".newLiveGame"
		err = b.PublishToTopic(channelName, ogData)
		if err != nil {
			return err
		}
	}
	return nil
}
