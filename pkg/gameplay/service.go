package gameplay

import (
	"context"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"

	"github.com/domino14/macondo/gcgio"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/config"
	entityutils "github.com/woogles-io/liwords/pkg/entity/utilities"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/omgwords/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	"github.com/woogles-io/liwords/pkg/utilities"
	pb "github.com/woogles-io/liwords/rpc/api/proto/game_service"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// GameService is a service that contains functions relevant to a game's
// metadata, stats, etc. All real-time functionality is handled in
// gameplay/game.go and related files.
type GameService struct {
	userStore user.Store
	gameStore GameStore
	cfg       *config.Config
	// New stores. These will replace the game store eventually.
	gameDocumentStore *stores.GameDocumentStore
	queries           *models.Queries
}

// NewGameService creates a GameService
func NewGameService(u user.Store, gs GameStore, gds *stores.GameDocumentStore,
	cfg *config.Config, q *models.Queries) *GameService {
	return &GameService{u, gs, cfg, gds, q}
}

// GetMetadata gets metadata for the given game.
func (gs *GameService) GetMetadata(ctx context.Context, req *connect.Request[pb.GameInfoRequest],
) (*connect.Response[ipc.GameInfoResponse], error) {
	log.Debug().Str("id", req.Msg.GameId).Msg("get-metadata-svc")
	gir, err := gs.gameStore.GetMetadata(ctx, req.Msg.GameId)
	if err != nil {
		return nil, err
	}
	// Censors the response in-place
	if gir.Type == ipc.GameType_NATIVE {
		censorGameInfoResponse(ctx, gs.userStore, gir)
	}
	return connect.NewResponse(gir), nil
}

// GetRematchStreak gets quickdata for the given rematch streak.
func (gs *GameService) GetRematchStreak(ctx context.Context, req *connect.Request[pb.RematchStreakRequest],
) (*connect.Response[pb.StreakInfoResponse], error) {
	resp, err := gs.gameStore.GetRematchStreak(ctx, req.Msg.OriginalRequestId)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	// Censors the response in-place
	censorStreakInfoResponse(ctx, gs.userStore, resp)
	return connect.NewResponse(resp), nil
}

//	GetRecentGames gets quickdata for the numGames most recent games of the player
//
// offset by offset.
func (gs *GameService) GetRecentGames(ctx context.Context, req *connect.Request[pb.RecentGamesRequest],
) (*connect.Response[ipc.GameInfoResponses], error) {
	resp, err := gs.gameStore.GetRecentGames(ctx, req.Msg.Username, int(req.Msg.NumGames), int(req.Msg.Offset))
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	user, err := gs.userStore.Get(ctx, req.Msg.Username)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
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
			return nil, apiserver.InternalErr(err)
		}

		privilegedViewer, err := gs.queries.HasPermission(ctx, models.HasPermissionParams{
			UserID:     int32(viewer.ID),
			Permission: string(rbac.CanModerateUsers),
		})

		if !privilegedViewer {
			return connect.NewResponse(&ipc.GameInfoResponses{}), nil
		}
	}
	// Censors the responses in-place
	censorGameInfoResponses(ctx, gs.userStore, resp)
	return connect.NewResponse(resp), nil
}

// GetGCG downloads a GCG for a full native game, or a partial GCG
// for an annotated game.
func (gs *GameService) GetGCG(ctx context.Context, req *connect.Request[pb.GCGRequest],
) (*connect.Response[pb.GCGResponse], error) {
	hist, err := gs.gameStore.GetHistory(ctx, req.Msg.GameId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	anno := false
	if hist.Version == 0 {
		// A shortcut for a blank history. Look in the game document store.
		gdoc, err := gs.gameDocumentStore.GetDocument(ctx, req.Msg.GameId, false)
		if err != nil {
			return nil, err
		}
		hist, err = entityutils.ToGameHistory(gdoc.GameDocument, gs.cfg)
		if err != nil {
			return nil, err
		}
		anno = true
	}

	hist = mod.CensorHistory(ctx, gs.userStore, hist)
	if hist.PlayState != macondopb.PlayState_GAME_OVER && !anno {
		return nil, apiserver.InvalidArg("please wait until the game is over to download GCG")
	}
	gcg, err := gcgio.GameHistoryToGCG(hist, true)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.GCGResponse{Gcg: gcg}), nil
}

func (gs *GameService) GetGameHistory(ctx context.Context, req *connect.Request[pb.GameHistoryRequest],
) (*connect.Response[pb.GameHistoryResponse], error) {
	hist, err := gs.gameStore.GetHistory(ctx, req.Msg.GameId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	hist = mod.CensorHistory(ctx, gs.userStore, hist)
	if hist.PlayState != macondopb.PlayState_GAME_OVER {
		return nil, apiserver.InvalidArg("please wait until the game is over to download GCG")
	}
	return connect.NewResponse(&pb.GameHistoryResponse{History: hist}), nil
}

// XXX: GetGameDocument should be moved to omgwords service eventually, once
// we get rid of GameHistory and game entities etc.
func (gs *GameService) GetGameDocument(ctx context.Context, req *connect.Request[pb.GameDocumentRequest],
) (*connect.Response[pb.GameDocumentResponse], error) {
	g, err := gs.gameStore.Get(ctx, req.Msg.GameId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	if g.History().PlayState != macondopb.PlayState_GAME_OVER {
		return nil, apiserver.InvalidArg("please wait until the game is over to download GCG")
	}
	gdoc, err := entityutils.ToGameDocument(g, gs.cfg)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.GameDocumentResponse{Document: gdoc}), nil
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

func (gs *GameService) UnfreezeBot(ctx context.Context, req *connect.Request[pb.UnfreezeBotRequest]) (
	*connect.Response[pb.UnfreezeBotResponse], error) {

	_, err := apiserver.AuthenticateWithPermission(ctx, gs.userStore, gs.queries, rbac.AdminAllAccess)
	if err != nil {
		return nil, err
	}

	var gameIDs []string

	switch req.Msg.Mode {
	case pb.UnfreezeBotMode_UNFREEZE_BOT_MODE_ALL_CORRESPONDENCE:
		rows, err := gs.queries.ListActiveCorrespondenceGamesWithBotOnTurn(ctx)
		if err != nil {
			return nil, apiserver.InternalErr(err)
		}
		gameIDs = make([]string, len(rows))
		for i, row := range rows {
			gameIDs[i] = row.String
		}
		log.Info().Int("count", len(gameIDs)).Msg("found active correspondence games with bot on turn")

	case pb.UnfreezeBotMode_UNFREEZE_BOT_MODE_ALL_REALTIME:
		rows, err := gs.queries.ListActiveRealtimeGamesWithBotOnTurn(ctx)
		if err != nil {
			return nil, apiserver.InternalErr(err)
		}
		gameIDs = make([]string, len(rows))
		for i, row := range rows {
			gameIDs[i] = row.String
		}
		log.Info().Int("count", len(gameIDs)).Msg("found active realtime games with bot on turn")

	case pb.UnfreezeBotMode_UNFREEZE_BOT_MODE_SPECIFIC_GAME:
		if req.Msg.GameId == "" {
			return nil, apiserver.InvalidArg("game_id is required for SPECIFIC_GAME mode")
		}
		gameIDs = []string{req.Msg.GameId}
		log.Info().Str("gameID", req.Msg.GameId).Msg("processing specific game")

	default:
		return nil, apiserver.InvalidArg("invalid mode specified")
	}

	gamesProcessed := int32(0)
	requestsSent := int32(0)
	errors := int32(0)

	for _, gameID := range gameIDs {
		gamesProcessed++

		// Load game
		game, err := gs.gameStore.Get(ctx, gameID)
		if err != nil {
			log.Err(err).Str("gameID", gameID).Msg("failed to load game")
			errors++
			continue
		}

		err = PotentiallySendBotMoveRequest(ctx, gs.userStore, game)
		if err != nil {
			log.Err(err).Str("gameID", gameID).Msg("failed to send bot move request")
			errors++
			continue
		}

		requestsSent++
		log.Info().Str("gameID", gameID).Msg("sent bot request")
	}

	log.Info().
		Int32("games_processed", gamesProcessed).
		Int32("requests_sent", requestsSent).
		Int32("errors", errors).
		Msg("unfreeze bot completed")

	return connect.NewResponse(&pb.UnfreezeBotResponse{
		GamesProcessed: gamesProcessed,
		RequestsSent:   requestsSent,
		Errors:         errors,
	}), nil
}
