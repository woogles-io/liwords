package omgwords

import (
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
)

// functions for dealing with a message bus

func announceGameCreation(g *ipc.GameDocument, playerInfo []*ipc.PlayerInfo,
	evtChan chan *entity.EventWrapper) error {

	gameInfo := &ipc.GameInfoResponse{
		Players: playerInfo,
		GameId:  g.Uid,
		Type:    g.Type,
	}
	toSend := entity.WrapEvent(gameInfo, ipc.MessageType_ONGOING_GAME_EVENT)
	toSend.AddAudience(entity.AudLobby, "newLiveGame")
	evtChan <- toSend

	return nil
}
