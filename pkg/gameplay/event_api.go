package gameplay

import (
	"context"
	"fmt"

	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/tournament"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/rs/zerolog"
)

// This file should contain helper methods for our Game Event API
// see GameEventService in the game_service.proto
// The Game Event API should provide a way to play an OMGWords game with an HTTP
// API (from the sending perspective)
// See also bus/event_api.go, which itself publishes game and other events to an
// http endpoint.
// A bot could therefore subscribe to the bus/event_api.go http endpoint and receive
// game moves from there, while sending moves via the Game Event API in this package.

func handleEventFromAPI(ctx context.Context, callingUserId string, cge *pb.ClientGameplayEvent,
	gameStore GameStore, userStore user.Store, notorietyStore mod.NotorietyStore,
	listStatStore stats.ListStatStore, tournamentStore tournament.TournamentStore) error {
	entGame, err := gameStore.Get(ctx, cge.GameId)
	if err != nil {
		return err
	}
	entGame.Lock()
	defer entGame.Unlock()
	log := zerolog.Ctx(ctx).With().Str("gameID", entGame.GameID()).Logger()

	var userID string
	// figure out the userID to use here.
	switch entGame.Type {
	case pb.GameType_NATIVE:
		userID = callingUserId
	case pb.GameType_ANNOTATED:
		userID = entGame.PlayerIDOnTurn()
	default:
		return fmt.Errorf("game type %s not handled", entGame.Type.String())
	}

	_, err = handleEventAfterLockingGame(
		log.WithContext(ctx),
		gameStore,
		userStore, listStatStore, notorietyStore, tournamentStore,
		userID, cge, entGame,
	)
	// XXX: let's unload the game from the cache after this. (if annotated)

	return err
}
