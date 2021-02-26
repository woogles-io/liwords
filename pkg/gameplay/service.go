package gameplay

import (
	"context"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/macondo/gcgio"
	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/game_service"
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
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
	return censorGameInfoResponse(ctx, gs.userStore, gs.gameStore.GetMetadata(ctx, req.GameId))

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
	return censorGameInfoResponses(ctx, gs.userStore, resp), nil
}

// GetGCG downloads a GCG for a finished game.
func (gs *GameService) GetGCG(ctx context.Context, req *pb.GCGRequest) (*pb.GCGResponse, error) {
	hist, err := gs.gameStore.GetHistory(ctx, req.GameId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	hist = censorHistory(ctx, gs.userStore, hist)
	if hist.PlayState != macondopb.PlayState_GAME_OVER {
		return nil, twirp.NewError(twirp.InvalidArgument, "please wait until the game is over to download GCG")
	}
	gcg, err := gcgio.GameHistoryToGCG(hist, true)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.GCGResponse{Gcg: gcg}, nil
}

func (gs *GameService) GetGameHistory(ctx context.Context, req *pb.GameHistoryRequest) (*pb.GameHistoryResponse, error) {
	hist, err := gs.gameStore.GetHistory(ctx, req.GameId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	hist = censorHistory(ctx, gs.userStore, hist)
	if hist.PlayState != macondopb.PlayState_GAME_OVER {
		return nil, twirp.NewError(twirp.InvalidArgument, "please wait until the game is over to download GCG")
	}
	return &pb.GameHistoryResponse{History: hist}, nil
}

func censorPlayerInfo(gir *pb.GameInfoResponse, playerIndex int) {
	gir.PlayerInfo[playerIndex].FullName = mod.CensoredUsername
	gir.PlayerInfo[playerIndex].Nickname = mod.CensoredUsername
	gir.PlayerInfo[playerIndex].CountryCode = ""
	gir.PlayerInfo[playerIndex].Title = ""
	gir.PlayerInfo[playerIndex].Rating = ""
}

func censorGameInfoResponse(ctx context.Context, us user.Store, gir *pb.GameInfoResponse) error {
	if mod.IsCensorable(ctx, us, gir.PlayerInfo[0].UserId) {
		censorPlayerInfo(gir, 0)
	}
	if mod.IsCensorable(ctx, us, gir.PlayerInfo[1].UserId) {
		censorPlayerInfo(gir, 1)
	}
}

func censorGameInfoResponses(ctx context.Context, us user.Store, girs *pb.GameInfoResponse) error {
	knownUsers := make(map[string]bool)

	for _, gir := range girs.GameInfo {
		playerOne := gir.PlayerInfo[0].UserId
		playerOne := gir.PlayerInfo[1].UserId

		_, known := knownUsers[playerOne]
		if !known {
			knownUsers[playerOne] = mod.IsCensorable(ctx, us, playerOne)
		}
		if knownUsers[playerOne] {
			censorPlayerInfo(gir, 0)
		}

		_, known = knownUsers[playerTwo]
		if !known {
			knownUsers[playerTwo] = mod.IsCensorable(ctx, us, playerTwo)
		}
		if knownUsers[playerTwo] {
			censorPlayerInfo(gir, 1)
		}
	}
}

func censorPlayerInHistory(hist *macondopb.GameHistory, playerIndex int) {
	uncensoredNickname := hist.Players[playerIndex].Nickname
	hist.Players[playerIndex].RealName = mod.CensoredUsername
	hist.Players[playerIndex].Nickname = mod.CensoredUsername
	for _, event := range hist.Events {
		if event.Nickname == uncensoredNickname {
			event.Nickname = mod.CensoredUsername
		}
	}
}

func censorHistory(ctx context.Context, us user.Store, hist *macondopb.GameHistory) *macondopb.GameHistory {
	playerOne := hist.Players[0].UserId
	playerTwo := hist.Players[1].UserId

	playerOneCensorable := mod.IsCensorable(ctx, us, playerOne)
	playerTwoCensorable := mod.IsCensorable(ctx, us, playerTwo)

	if !playerOneCensorable && !playerTwoCensorable {
		return hist
	}

	censoredHistory := proto.Clone(hist).(*macondopb.GameHistory)

	if playerOneCensorable {
		censorPlayerInHistory(censoredHistory, 0)
	}

	if playerTwoCensorable {
		censorPlayerInHistory(censoredHistory, 1)
	}
	return censoredHistory
}
