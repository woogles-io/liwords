package omgsvc

import (
	"context"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

func handleGameInstantiation(ctx context.Context, b ipc.Publisher, data []byte,
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

	ngevt := ipc.WrapEvent(&pb.NewGameEvent{
		GameId: resmsg.Id,
		// Doesn't matter who's accepter/requester; consider renaming these.
		AccepterCid:  evt.ConnIds[0],
		RequesterCid: evt.ConnIds[1],
	}, pb.MessageType_NEW_GAME_EVENT)

	// We have a game ID. Send this to all participants.
	for _, u := range evt.UserIds {
		ngevt.AddAudience(ipc.AudUser, u)
	}
	err = ngevt.Publish(b)
	if err != nil {
		return err
	}

	// Also broadcast the game creation to the lobby/whoever needs to know
	ongoingGameEvt := ipc.WrapEvent(resmsg.GameInfo, pb.MessageType_ONGOING_GAME_EVENT)
	ongoingGameEvt.AddAudience(ipc.AudLobby, "newLiveGame")

	// Also publish to tournament channel if this is a tournament game.
	if resmsg.GameInfo.TournamentId != "" {
		ongoingGameEvt.AddAudience(ipc.AudTournament, resmsg.GameInfo.TournamentId+".newLiveGame")
	}
	err = ongoingGameEvt.Publish(b)
	if err != nil {
		return err
	}

	return nil
}

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
	err = ipc.RequestProto("storesvc.omgwords.readygame", b, ready, serverReady)
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
	err = ipc.RequestProto("storesvc.omgwords.startgame", b, start, startResponse)
	if err != nil {
		return err
	}
	log.Info().Interface("stresp", startResponse).Msg("reset-and-start")
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
