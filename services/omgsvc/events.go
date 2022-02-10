package omgsvc

import (
	"context"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/domino14/liwords/pkg/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
)

// MsgHandler handles all game svc related messages
// omgsvc should subscribe to omgsvc.>
func MsgHandler(ctx context.Context, b ipc.Publisher, topic string, data []byte, reply string) error {
	// MessageType_GAME_META_EVENT
	// MessageType_CLIENT_GAMEPLAY_EVENT
	// MessageType_TIMED_OUT
	// MessageType_READY_FOR_GAME

	// example topic: omgsvc.pb.3.auth.userid.wsconnid

	log.Debug().Interface("topic", topic).Msg("msghandler")
	subtopics := strings.Split(topic, ".")

	msgType := subtopics[2]
	//auth := ""
	userID := ""
	if len(subtopics) > 4 {
		//	auth = subtopics[1]
		userID = subtopics[4]
	}
	wsConnID := ""
	if len(subtopics) > 5 {
		wsConnID = subtopics[5]
	}
	pnum, err := strconv.Atoi(msgType)
	if err != nil {
		return err
	}
	msgType = pb.MessageType(pnum).String()

	switch msgType {
	case pb.MessageType_GAME_INSTANTIATION.String():
		return handleGameInstantiation(ctx, b, data, reply)

	case pb.MessageType_READY_FOR_GAME.String():
		return handleReadyForGame(ctx, b, userID, wsConnID, data, reply)

	case pb.MessageType_CLIENT_GAMEPLAY_EVENT.String():
		return handleGameplayEvent(ctx, b, data, reply)
		// evt := &pb.ClientGameplayEvent{}
		// err := proto.Unmarshal(data, evt)
		// if err != nil {
		// 	return err
		// }

	}

	return nil
}
