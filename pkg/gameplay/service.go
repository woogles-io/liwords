package gameplay

import (
	"context"

	"github.com/domino14/macondo/gcgio"
	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/game_service"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

// GameService is a Twirp service that contains functions relevant to a game's
// metadata, stats, etc. All real-time functionality is handled in
// gameplay/game.go and related files.
type GameService struct {
	userStore user.Store
	gameStore GameStore
}

// NewGameService creates a Twirp GameService
func NewGameService(u user.Store, gs GameStore) *GameService {
	return &GameService{u, gs}
}

// GetMetadata gets metadata for the given game.
func (gs *GameService) GetMetadata(ctx context.Context, req *pb.GameInfoRequest) (*pb.GameInfoResponse, error) {
	return gs.gameStore.GetMetadata(ctx, req.GameId)

}

//  GetRematchStreak gets quickdata for the given rematch streak.
func (gs *GameService) GetRematchStreak(ctx context.Context, req *pb.RematchStreakRequest) (*pb.StreakInfoResponse, error) {
	resp, err := gs.gameStore.GetRematchStreak(ctx, req.OriginalRequestId)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return resp, nil
}

//  GetRecentGames gets quickdata for the numGames most recent games of the player
// offset by offset.
func (gs *GameService) GetRecentGames(ctx context.Context, req *pb.RecentGamesRequest) (*pb.GameInfoResponses, error) {
	resp, err := gs.gameStore.GetRecentGames(ctx, req.Username, int(req.NumGames), int(req.Offset))
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return resp, nil
}

// GetGCG downloads a GCG for a finished game.
func (gs *GameService) GetGCG(ctx context.Context, req *pb.GCGRequest) (*pb.GCGResponse, error) {
	entGame, err := gs.gameStore.Get(ctx, req.GameId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if entGame.Playing() != macondopb.PlayState_GAME_OVER {
		return nil, twirp.NewError(twirp.InvalidArgument, "please wait until the game is over to download GCG")
	}
	gcg, err := gcgio.GameHistoryToGCG(entGame.History(), true)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.GCGResponse{Gcg: gcg}, nil
}

func (gs *GameService) GetGameHistory(ctx context.Context, req *pb.GameHistoryRequest) (*pb.GameHistoryResponse, error) {
	entGame, err := gs.gameStore.Get(ctx, req.GameId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if entGame.Playing() != macondopb.PlayState_GAME_OVER {
		return nil, twirp.NewError(twirp.InvalidArgument, "please wait until the game is over to download game history")
	}
	return &pb.GameHistoryResponse{History: entGame.History()}, nil
}
