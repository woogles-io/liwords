package omgsvc

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/proto"
)

// consider eliminating this signal entirely once the app becomes more of a SPA
// (i.e. when it doesn't reconnect socket going from lobby to game)
func handleReadyForGame(ctx context.Context, b ipc.Publisher, userID, connID string,
	data []byte, reply string) error {
	// User is ready for game.
	// Communicate this to the store service, and optionally start the game.
	ready := &pb.ReadyForGame{}
	err := proto.Unmarshal(data, ready)
	if err != nil {
		return err
	}
	gid := ready.GameId

	// Populate the ready event with the user id and conn id. These don't come
	// with the event normally as the user id/conn id are part of the topic,
	// when they come via the socket.
	ready.UserId = userID
	ready.ConnId = connID

	// Forward event onto the store.
	serverReady := &pb.ReadyForGameResponse{}
	err = b.RequestProto("storesvc.omgwords.readygame", ready, serverReady)
	if err != nil {
		return err
	}

	if !serverReady.ReadyToStart {
		// Game is not yet ready to start.
		return nil
	}
	// game is ready to start.
	// XXX: send activegameentry
	// Once the game is ready, tell the store to reset the timers.
	// Then just send history refresher event to both players.

	start := &pb.ResetTimersAndStart{GameId: gid}
	startResponse := &pb.ResetTimersAndStartResponse{}
	err = b.RequestProto("storesvc.omgwords.startgame", start, startResponse)
	if err != nil {
		return err
	}
	// need to remove user racks for the relevant players.
	startEvt := ipc.WrapEvent(startResponse.GameHistoryRefresher, pb.MessageType_GAME_HISTORY_REFRESHER)
	startEvt.AddAudience(entity.AudGameTV, gid)
	// XXX: should add null checks for startResponse sub-fields.
	for _, p := range startResponse.GameHistoryRefresher.History.Players {
		startEvt.AddAudience(entity.AudUser, p.UserId+".game."+gid)
	}

	err = publishWithSanitization(b, startEvt)
	if err != nil {
		return err
	}

	return nil
}

func handleGameplayEvent(ctx context.Context, b ipc.Publisher, data []byte, reply string) error {

	return nil
}
