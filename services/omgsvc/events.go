package omgsvc

import (
	"context"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
)

// MsgHandler handles all game svc related messages
func MsgHandler(ctx context.Context, topic string, data []byte, reply string) error {
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
	case pb.MessageType_CLIENT_GAMEPLAY_EVENT.String():
		evt := &pb.ClientGameplayEvent{}
		err := proto.Unmarshal(data, evt)
		if err != nil {
			return err
		}

	}
}
