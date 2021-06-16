package gameplay

import (
	"context"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/macondo/gcgio"
	"github.com/rs/zerolog/log"
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
	// Censors the response in-place
	censorStreakInfoResponse(ctx, gs.userStore, resp)
	return resp, nil
}

//  GetRecentGames gets quickdata for the numGames most recent games of the player
// offset by offset.
func (gs *GameService) GetRecentGames(ctx context.Context, req *pb.RecentGamesRequest) (*pb.GameInfoResponses, error) {
	resp, err := gs.gameStore.GetRecentGames(ctx, req.Username, int(req.NumGames), int(req.Offset))
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	user, err := gs.userStore.Get(ctx, req.Username)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if mod.IsCensorable(ctx, gs.userStore, user.UUID) {
		// This view requires authentication.
		sess, err := apiserver.GetSession(ctx)
		if err != nil {
			return nil, err
		}

		viewer, err := gs.userStore.Get(ctx, sess.Username)
		if err != nil {
			log.Err(err).Msg("getting-user")
			return nil, twirp.InternalErrorWith(err)
		}
		if !viewer.IsMod && !viewer.IsAdmin {
			return &pb.GameInfoResponses{}, nil
		}
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
	hist = mod.CensorHistory(ctx, gs.userStore, hist)
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
	hist = mod.CensorHistory(ctx, gs.userStore, hist)
	if hist.PlayState != macondopb.PlayState_GAME_OVER {
		return nil, twirp.NewError(twirp.InvalidArgument, "please wait until the game is over to download GCG")
	}
	return &pb.GameHistoryResponse{History: hist}, nil
}

func censorPlayer(gir *pb.GameInfoResponse, playerIndex int) {
	gir.Players[playerIndex].UserId = mod.CensoredUsername
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

func censorStreakInfoResponse(ctx context.Context, us user.Store, sir *pb.StreakInfoResponse) {
	for _, pi := range sir.PlayersInfo {
		if mod.IsCensorable(ctx, us, pi.Uuid) {
			pi.Nickname = mod.CensoredUsername
			pi.Uuid = mod.CensoredUsername
		}
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
