package gameplay

import (
	"context"
	"time"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	entityutils "github.com/domino14/liwords/pkg/entity/utilities"
	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/stats"
	"github.com/domino14/liwords/pkg/tournament"
	"github.com/domino14/liwords/pkg/utilities"
	"github.com/domino14/macondo/game"
	"github.com/domino14/macondo/gcgio"
	"github.com/domino14/macondo/runner"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/game_service"
	ipc "github.com/domino14/liwords/rpc/api/proto/ipc"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
)

// GameService is a Twirp service that contains functions relevant to a game's
// metadata, stats, etc. All real-time functionality is handled in
// gameplay/game.go and related files.
type GameService struct {
	userStore       user.Store
	gameStore       GameStore
	notorietyStore  mod.NotorietyStore
	listStatStore   stats.ListStatStore
	tournamentStore tournament.TournamentStore
	cfg             *config.Config
}

// NewGameService creates a Twirp GameService
func NewGameService(u user.Store, gs GameStore, ns mod.NotorietyStore,
	lss stats.ListStatStore, ts tournament.TournamentStore,
	cfg *config.Config) *GameService {
	return &GameService{u, gs, ns, lss, ts, cfg}
}

// GetMetadata gets metadata for the given game.
func (gs *GameService) GetMetadata(ctx context.Context, req *pb.GameInfoRequest) (*ipc.GameInfoResponse, error) {
	gir, err := gs.gameStore.GetMetadata(ctx, req.GameId)
	if err != nil {
		return nil, err
	}
	// Censors the response in-place
	if gir.Type == ipc.GameType_NATIVE {
		censorGameInfoResponse(ctx, gs.userStore, gir)
	}
	return gir, nil
}

// GetRematchStreak gets quickdata for the given rematch streak.
func (gs *GameService) GetRematchStreak(ctx context.Context, req *pb.RematchStreakRequest) (*pb.StreakInfoResponse, error) {
	resp, err := gs.gameStore.GetRematchStreak(ctx, req.OriginalRequestId)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	// Censors the response in-place
	censorStreakInfoResponse(ctx, gs.userStore, resp)
	return resp, nil
}

//	GetRecentGames gets quickdata for the numGames most recent games of the player
//
// offset by offset.
func (gs *GameService) GetRecentGames(ctx context.Context, req *pb.RecentGamesRequest) (*ipc.GameInfoResponses, error) {
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
			return &ipc.GameInfoResponses{}, nil
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

func (gs *GameService) GetGameDocument(ctx context.Context, req *pb.GameDocumentRequest) (*pb.GameDocumentResponse, error) {
	g, err := gs.gameStore.Get(ctx, req.GameId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if g.History().PlayState != macondopb.PlayState_GAME_OVER {
		return nil, twirp.NewError(twirp.InvalidArgument, "please wait until the game is over to download GCG")
	}
	gdoc, err := entityutils.ToGameDocument(g, gs.cfg)
	if err != nil {
		return nil, twirp.NewError(twirp.Internal, err.Error())
	}
	return &pb.GameDocumentResponse{Document: gdoc}, nil
}

func (gs *GameService) CreateBroadcastGame(ctx context.Context, req *pb.CreateBroadcastGameRequest) (
	*pb.CreateBroadcastGameResponse, error) {

	players := req.PlayersInfo
	if len(players) != 2 {
		return nil, twirp.NewError(twirp.InvalidArgument, "need two players")
	}
	if players[0].Nickname == players[1].Nickname {
		return nil, twirp.NewError(twirp.InvalidArgument, "player nicknames must be unique")
	}
	if players[0].Nickname == "" || players[1].Nickname == "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "player nicknames must not be blank")
	}
	if req.Rules == nil {
		return nil, twirp.NewError(twirp.InvalidArgument, "no rules")
	}
	if req.Lexicon == "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "lexicon is empty")
	}
	// We can just make the macondo user ID the same as the nickname, as it
	// doesn't matter.
	mcplayers := []*macondopb.PlayerInfo{
		{Nickname: players[0].Nickname, RealName: players[0].RealName, UserId: players[0].Nickname},
		{Nickname: players[1].Nickname, RealName: players[1].RealName, UserId: players[1].Nickname},
	}
	rules, err := game.NewBasicGameRules(
		&gs.cfg.MacondoConfig, req.Lexicon, req.Rules.BoardLayoutName,
		req.Rules.LetterDistributionName, game.CrossScoreOnly,
		game.Variant(req.Rules.VariantName))
	if err != nil {
		return nil, err
	}
	var gameRunner *runner.GameRunner
	gameRunner, err = runner.NewGameRunnerFromRules(&runner.GameOptions{
		ChallengeRule: req.ChallengeRule,
	}, mcplayers, rules)
	if err != nil {
		return nil, err
	}
	// Use a full shortuuid for broadcast games.
	gameRunner.Game.History().Uid = shortuuid.New()
	gameRunner.Game.History().IdAuth = IdentificationAuthority

	// we don't have a game request for these types of games, as they are created
	// via the API.
	entGame := entity.NewGame(&gameRunner.Game, nil)
	entGame.Type = ipc.GameType_ANNOTATED

	// Create player info in entGame.Quickdata
	playerinfos := make([]*ipc.PlayerInfo, len(players))

	for idx, u := range players {
		playerinfos[idx] = &ipc.PlayerInfo{
			Nickname: u.Nickname,
			UserId:   u.Nickname,
			First:    idx == 0,
		}
	}
	entGame.Quickdata = &entity.Quickdata{
		PlayerInfo: playerinfos,
	}

	entGame.CreatedAt = time.Now()
	if err = gs.gameStore.Create(ctx, entGame); err != nil {
		return nil, err
	}
	return &pb.CreateBroadcastGameResponse{
		GameId: gameRunner.Uid(),
	}, nil
}

func (gs *GameService) SendGameEvent(ctx context.Context, req *ipc.ClientGameplayEvent) (
	*pb.GameEventResponse, error) {

	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := gs.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, twirp.InternalErrorWith(err)
	}

	err = handleEventFromAPI(ctx, user.UUID, req, gs.gameStore,
		gs.userStore, gs.notorietyStore,
		gs.listStatStore, gs.tournamentStore)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.GameEventResponse{}, nil
}

func (gs *GameService) SendTimePenaltyEvent(ctx context.Context, req *pb.TimePenaltyEvent) (
	*pb.GameEventResponse, error) {

	return nil, nil
}

func (gs *GameService) SendChallengeBonusEvent(ctx context.Context, req *pb.ChallengeBonusPointsEvent) (
	*pb.GameEventResponse, error) {

	return nil, nil
}

func (gs *GameService) SetBroadcastGamePrivacy(ctx context.Context, req *pb.BroadcastGamePrivacy) (
	*pb.GameEventResponse, error) {

	return nil, nil
}

func censorPlayer(gir *ipc.GameInfoResponse, playerIndex int, censoredUsername string) {
	gir.Players[playerIndex].UserId = censoredUsername
	gir.Players[playerIndex].FullName = censoredUsername
	gir.Players[playerIndex].Nickname = censoredUsername
	gir.Players[playerIndex].CountryCode = ""
	gir.Players[playerIndex].Title = ""
	gir.Players[playerIndex].Rating = ""
}

func censorGameInfoResponse(ctx context.Context, us user.Store, gir *ipc.GameInfoResponse) {
	playerCensored := false
	if mod.IsCensorable(ctx, us, gir.Players[0].UserId) {
		censorPlayer(gir, 0, utilities.CensoredUsername)
		playerCensored = true
	}
	if mod.IsCensorable(ctx, us, gir.Players[1].UserId) {
		censoredUsername := utilities.CensoredUsername
		if playerCensored {
			censoredUsername = utilities.AnotherCensoredUsername
		}
		censorPlayer(gir, 1, censoredUsername)
	}
}

func censorStreakInfoResponse(ctx context.Context, us user.Store, sir *pb.StreakInfoResponse) {
	// This assumes up to two players
	playerCensored := false
	for _, pi := range sir.PlayersInfo {
		if mod.IsCensorable(ctx, us, pi.Uuid) {
			pi.Nickname = utilities.CensoredUsername
			pi.Uuid = utilities.CensoredUsername
			if playerCensored {
				pi.Nickname = utilities.AnotherCensoredUsername
				pi.Uuid = utilities.AnotherCensoredUsername
			}
			playerCensored = true
		}
	}
}

func censorGameInfoResponses(ctx context.Context, us user.Store, girs *ipc.GameInfoResponses) {
	knownUsers := make(map[string]bool)

	for _, gir := range girs.GameInfo {
		playerOne := gir.Players[0].UserId
		playerTwo := gir.Players[1].UserId

		_, known := knownUsers[playerOne]
		if !known {
			knownUsers[playerOne] = mod.IsCensorable(ctx, us, playerOne)
		}
		if knownUsers[playerOne] {
			censorPlayer(gir, 0, utilities.CensoredUsername)
		}

		_, known = knownUsers[playerTwo]
		if !known {
			knownUsers[playerTwo] = mod.IsCensorable(ctx, us, playerTwo)
		}
		if knownUsers[playerTwo] {
			censoredUsername := utilities.CensoredUsername
			if knownUsers[playerOne] {
				censoredUsername = utilities.AnotherCensoredUsername
			}
			censorPlayer(gir, 1, censoredUsername)
		}
	}
}
