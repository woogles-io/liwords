package omgwords

import (
	"context"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/user"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/omgwords_service"
)

type OMGWordsService struct {
	userStore user.Store
	cfg       *config.Config
}

// NewGameService creates a Twirp GameService
func NewOMGWordsService(u user.Store, cfg *config.Config) *OMGWordsService {
	return &OMGWordsService{u, cfg}
}

func (gs *OMGWordsService) CreateBroadcastGame(ctx context.Context, req *pb.CreateBroadcastGameRequest) (
	*pb.CreateBroadcastGameResponse, error) {
	/*
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
		}, nil */
	return nil, nil
}

func (gs *OMGWordsService) SendGameEvent(ctx context.Context, req *ipc.ClientGameplayEvent) (
	*pb.GameEventResponse, error) {

	/*
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
	*/
	return nil, nil
}

func (gs *OMGWordsService) SendTimePenaltyEvent(ctx context.Context, req *pb.TimePenaltyEvent) (
	*pb.GameEventResponse, error) {

	return nil, nil
}

func (gs *OMGWordsService) SendChallengeBonusEvent(ctx context.Context, req *pb.ChallengeBonusPointsEvent) (
	*pb.GameEventResponse, error) {

	return nil, nil
}

func (gs *OMGWordsService) SetBroadcastGamePrivacy(ctx context.Context, req *pb.BroadcastGamePrivacy) (
	*pb.GameEventResponse, error) {

	return nil, nil
}

func (gs *OMGWordsService) GetGamesForEditor(ctx context.Context, req *pb.GetGamesForEditorRequest) (
	*pb.BroadcastGamesResponse, error) {

	return nil, nil
}
