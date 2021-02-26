package gameplay

import (
	"context"
	"google.golang.org/protobuf/proto"

	"github.com/domino14/liwords/pkg/mod"
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
	gir, err := gs.gameStore.GetMetadata(ctx, req.GameId)
	if err != nil {
		return nil, err
	}
	// Censors the response in-place
	censorGameInfoResponse(ctx, gs.userStore, gir)
	return gir, nil
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
	// Censors the responses in-place
	censorGameInfoResponses(ctx, gs.userStore, resp)
	return resp, nil
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

func censorPlayer(gir *pb.GameInfoResponse, playerIndex int) {
	gir.Players[playerIndex].FullName = mod.CensoredUsername
	gir.Players[playerIndex].Nickname = mod.CensoredUsername
	gir.Players[playerIndex].CountryCode = ""
	gir.Players[playerIndex].Title = ""
	gir.Players[playerIndex].Rating = ""
}

func censorGameInfoResponse(ctx context.Context, us user.Store, gir *pb.GameInfoResponse) {
	if mod.IsCensorable(ctx, us, gir.Players[0].UserId) {
		censorPlayer(gir, 0)
	}
	if mod.IsCensorable(ctx, us, gir.Players[1].UserId) {
		censorPlayer(gir, 1)
	}
}

func censorGameInfoResponses(ctx context.Context, us user.Store, girs *pb.GameInfoResponses) {
	knownUsers := make(map[string]bool)

	for _, gir := range girs.GameInfo {
		playerOne := gir.Players[0].UserId
		playerTwo := gir.Players[1].UserId

		_, known := knownUsers[playerOne]
		if !known {
			knownUsers[playerOne] = mod.IsCensorable(ctx, us, playerOne)
		}
		if knownUsers[playerOne] {
			censorPlayer(gir, 0)
		}

		_, known = knownUsers[playerTwo]
		if !known {
			knownUsers[playerTwo] = mod.IsCensorable(ctx, us, playerTwo)
		}
		if knownUsers[playerTwo] {
			censorPlayer(gir, 1)
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
