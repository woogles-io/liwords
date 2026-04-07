package broadcasts

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/sync/singleflight"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/omgwords"
	omgstores "github.com/woogles-io/liwords/pkg/omgwords/stores"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/broadcast_service"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

const feedCacheTTL = 30 * time.Minute

// feedCacheEntry holds all divisions for a single broadcast URL fetch.
// One HTTP request populates all divisions at once.
type feedCacheEntry struct {
	divisions map[string]*FeedData // division name → FeedData (e.g. "A", "B")
	expiresAt time.Time
}

// BroadcastService implements the BroadcastService ConnectRPC service.
type BroadcastService struct {
	userStore     user.Store
	queries       *models.Queries
	cfg           *config.Config
	gameStore     *omgstores.GameDocumentStore
	metadataStore *omgstores.DBStore
	gameEventChan chan *entity.EventWrapper
	natsConn      *nats.Conn

	feedCacheMu sync.RWMutex
	feedCache   map[string]*feedCacheEntry // keyed by slug
	sfGroup     singleflight.Group
}

func NewBroadcastService(
	u user.Store,
	q *models.Queries,
	cfg *config.Config,
	gameStore *omgstores.GameDocumentStore,
	metadataStore *omgstores.DBStore,
) *BroadcastService {
	return &BroadcastService{
		userStore:     u,
		queries:       q,
		cfg:           cfg,
		gameStore:     gameStore,
		metadataStore: metadataStore,
		feedCache:     make(map[string]*feedCacheEntry),
	}
}

func (bs *BroadcastService) SetEventChannel(c chan *entity.EventWrapper) {
	bs.gameEventChan = c
}

func (bs *BroadcastService) SetNatsConn(nc *nats.Conn) {
	bs.natsConn = nc
}

// requireDirector checks that the authenticated user is a director for the broadcast.
func (bs *BroadcastService) requireDirector(ctx context.Context, broadcastID int32) (*entity.User, error) {
	u, err := apiserver.AuthUser(ctx, bs.userStore)
	if err != nil {
		return nil, err
	}
	isDir, err := bs.queries.IsBroadcastDirector(ctx, models.IsBroadcastDirectorParams{
		BroadcastID: broadcastID,
		UserID:      int32(u.ID),
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	if !isDir {
		return nil, apiserver.PermissionDenied("not a director of this broadcast")
	}
	return u, nil
}

func (bs *BroadcastService) CreateBroadcast(ctx context.Context, req *connect.Request[pb.CreateBroadcastRequest]) (
	*connect.Response[pb.CreateBroadcastResponse], error) {

	u, err := apiserver.AuthUser(ctx, bs.userStore)
	if err != nil {
		return nil, err
	}
	allowed, err := rbac.HasPermission(ctx, bs.queries, u.ID, rbac.CanCreateBroadcasts)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	if !allowed {
		return nil, apiserver.PermissionDenied("not permitted to create broadcasts")
	}
	if req.Msg.Slug == "" || req.Msg.Name == "" {
		return nil, apiserver.InvalidArg("slug and name are required")
	}
	if req.Msg.BroadcastUrl == "" {
		return nil, apiserver.InvalidArg("broadcast_url is required")
	}
	format := req.Msg.BroadcastUrlFormat
	if format == "" {
		format = "tsh_newt_json"
	}
	if _, err := NewFeedParser(format); err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	interval := req.Msg.PollIntervalSeconds
	if interval <= 0 {
		interval = 300
	}
	lexicon := req.Msg.Lexicon
	if lexicon == "" {
		lexicon = "CSW24"
	}
	boardLayout := req.Msg.BoardLayout
	if boardLayout == "" {
		boardLayout = "CrosswordGame"
	}
	letterDist := req.Msg.LetterDistribution
	if letterDist == "" {
		letterDist = "english"
	}

	broadcastUUID := uuid.New()
	result, err := bs.queries.CreateBroadcast(ctx, models.CreateBroadcastParams{
		Uuid:                broadcastUUID,
		Slug:                req.Msg.Slug,
		Name:                req.Msg.Name,
		Description:         pgtype.Text{String: req.Msg.Description, Valid: req.Msg.Description != ""},
		BroadcastUrl:        req.Msg.BroadcastUrl,
		BroadcastUrlFormat:  format,
		PollIntervalSeconds: interval,
		PollStartTime:       pgTimestamptz(req.Msg.PollStartTime),
		PollEndTime:         pgTimestamptz(req.Msg.PollEndTime),
		Lexicon:             lexicon,
		BoardLayout:         boardLayout,
		LetterDistribution:  letterDist,
		ChallengeRule:       req.Msg.ChallengeRule,
		CreatorID:           int32(u.ID),
	})
	if err != nil {
		log.Err(err).Str("slug", req.Msg.Slug).Msg("create-broadcast-failed")
		return nil, apiserver.InternalErr(err)
	}

	// Auto-add creator as director.
	if err := bs.queries.AddBroadcastDirector(ctx, models.AddBroadcastDirectorParams{
		BroadcastID: result.ID,
		UserID:      int32(u.ID),
	}); err != nil {
		log.Err(err).Msg("add-broadcast-director-failed")
	}

	return connect.NewResponse(&pb.CreateBroadcastResponse{
		Uuid: result.Uuid.String(),
		Slug: req.Msg.Slug,
	}), nil
}

func (bs *BroadcastService) GetBroadcast(ctx context.Context, req *connect.Request[pb.GetBroadcastRequest]) (
	*connect.Response[pb.GetBroadcastResponse], error) {

	row, err := bs.queries.GetBroadcastBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, apiserver.NotFound("broadcast not found")
	}

	broadcast := broadcastRowToProto(row.ID, row.Uuid, row.Slug, row.Name,
		row.Description, row.BroadcastUrl, row.BroadcastUrlFormat,
		row.PollIntervalSeconds, row.PollStartTime, row.PollEndTime,
		row.Lexicon, row.BoardLayout, row.LetterDistribution, row.ChallengeRule,
		row.Active, row.CreatedAt, row.CreatorUsername)

	division := req.Msg.Division
	divisionNames := bs.getCachedDivisionNames(row.Slug, row.BroadcastUrl, row.BroadcastUrlFormat)
	// Default to first division if not specified.
	if division == "" && len(divisionNames) > 0 {
		division = divisionNames[0]
	}

	if fd := bs.getCachedFeed(row.Slug, division, row.BroadcastUrl, row.BroadcastUrlFormat); fd != nil {
		broadcast.CurrentRound = int32(fd.CurrentRound)
	}

	directors, err := bs.queries.GetBroadcastDirectors(ctx, row.ID)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	annotators, err := bs.queries.GetBroadcastAnnotators(ctx, row.ID)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	dirUsernames := make([]string, 0, len(directors))
	for _, d := range directors {
		if d.Username.Valid {
			dirUsernames = append(dirUsernames, d.Username.String)
		}
	}
	annUsernames := make([]string, 0, len(annotators))
	for _, a := range annotators {
		if a.Username.Valid {
			annUsernames = append(annUsernames, a.Username.String)
		}
	}

	var players []*pb.BroadcastPlayer
	var totalRounds int32
	if fd := bs.getCachedFeed(row.Slug, division, row.BroadcastUrl, row.BroadcastUrlFormat); fd != nil {
		players, totalRounds = feedDataToProto(fd)
	}

	return connect.NewResponse(&pb.GetBroadcastResponse{
		Broadcast:          broadcast,
		Players:            players,
		DirectorUsernames:  dirUsernames,
		AnnotatorUsernames: annUsernames,
		TotalRounds:        totalRounds,
		Divisions:          divisionNames,
	}), nil
}

func (bs *BroadcastService) GetBroadcastGames(ctx context.Context, req *connect.Request[pb.GetBroadcastGamesRequest]) (
	*connect.Response[pb.GetBroadcastGamesResponse], error) {

	row, err := bs.queries.GetBroadcastBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, apiserver.NotFound("broadcast not found")
	}

	division := req.Msg.Division
	// Default to first available division if not specified.
	if division == "" {
		if names := bs.getCachedDivisionNames(row.Slug, row.BroadcastUrl, row.BroadcastUrlFormat); len(names) > 0 {
			division = names[0]
		}
	}

	fd := bs.getCachedFeed(row.Slug, division, row.BroadcastUrl, row.BroadcastUrlFormat)
	var totalRounds int32
	if fd != nil {
		totalRounds = int32(fd.TotalRounds)
	}

	round := req.Msg.Round
	if round == 0 {
		if fd != nil && fd.CurrentRound > 0 {
			round = int32(fd.CurrentRound)
		} else {
			round = 1
		}
	}

	// Get claimed games for this round from DB.
	dbGames, err := bs.queries.GetBroadcastGamesForRound(ctx, models.GetBroadcastGamesForRoundParams{
		BroadcastID: row.ID,
		Division:    division,
		Round:       round,
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	// Index claimed games by table number for quick lookup.
	claimedByTable := make(map[int32]models.GetBroadcastGamesForRoundRow)
	for _, g := range dbGames {
		claimedByTable[g.TableNumber] = g
	}

	// Build game list from feed pairings, merged with claimed status.
	var games []*pb.BroadcastRoundGame
	if fd != nil {
		pairings := GetRoundPairings(fd, int(round))
		for _, p := range pairings {
			g := &pb.BroadcastRoundGame{
				Round:           round,
				TableNumber:     int32(p.TableNumber),
				Player1Name:     p.Player1Name,
				Player2Name:     p.Player2Name,
				Player1Score:    int32(p.Player1Score),
				Player2Score:    int32(p.Player2Score),
				ScoresFinalized: p.Finalized,
				Division:        division,
			}
			if claimed, ok := claimedByTable[int32(p.TableNumber)]; ok {
				g.GameUuid = claimed.GameUuid
				if claimed.AnnotatorUsername.Valid {
					g.AnnotatorUsername = claimed.AnnotatorUsername.String
				}
				done, err := bs.metadataStore.GameIsDone(ctx, claimed.GameUuid)
				if err == nil {
					g.AnnotationDone = done
				}
			}
			games = append(games, g)
		}
	} else {
		// No feed data yet — return only claimed games.
		for _, g := range dbGames {
			game := &pb.BroadcastRoundGame{
				Round:       round,
				TableNumber: g.TableNumber,
				Player1Name: g.Player1Name,
				Player2Name: g.Player2Name,
				GameUuid:    g.GameUuid,
				Division:    g.Division,
			}
			if g.AnnotatorUsername.Valid {
				game.AnnotatorUsername = g.AnnotatorUsername.String
			}
			games = append(games, game)
		}
	}

	return connect.NewResponse(&pb.GetBroadcastGamesResponse{
		Games:       games,
		Round:       round,
		TotalRounds: totalRounds,
	}), nil
}

func (bs *BroadcastService) ClaimGame(ctx context.Context, req *connect.Request[pb.ClaimGameRequest]) (
	*connect.Response[pb.ClaimGameResponse], error) {

	u, err := apiserver.AuthUser(ctx, bs.userStore)
	if err != nil {
		return nil, err
	}

	broadcast, err := bs.queries.GetBroadcastBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, apiserver.NotFound("broadcast not found")
	}

	// Verify user is an annotator.
	isAnnotator, err := bs.queries.IsBroadcastAnnotator(ctx, models.IsBroadcastAnnotatorParams{
		BroadcastID: broadcast.ID,
		UserID:      int32(u.ID),
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	// Also allow directors to claim games.
	if !isAnnotator {
		isDir, err := bs.queries.IsBroadcastDirector(ctx, models.IsBroadcastDirectorParams{
			BroadcastID: broadcast.ID,
			UserID:      int32(u.ID),
		})
		if err != nil {
			return nil, apiserver.InternalErr(err)
		}
		if !isDir {
			return nil, apiserver.PermissionDenied("not an annotator or director of this broadcast")
		}
	}

	// Check this table/round/division isn't already claimed.
	existing, err := bs.queries.GetBroadcastGameByTableRound(ctx, models.GetBroadcastGameByTableRoundParams{
		BroadcastID: broadcast.ID,
		Division:    req.Msg.Division,
		Round:       req.Msg.Round,
		TableNumber: req.Msg.TableNumber,
	})
	if err == nil && existing.GameUuid != "" {
		// Already claimed — return existing game ID so the annotator can continue.
		return connect.NewResponse(&pb.ClaimGameResponse{GameId: existing.GameUuid}), nil
	}

	// Find the pairing from the feed to get player names.
	player1Name, player2Name, err := bs.playerNamesForTable(req.Msg.Slug, req.Msg.Division, broadcast.BroadcastUrl, broadcast.BroadcastUrlFormat,
		int(req.Msg.Round), int(req.Msg.TableNumber))
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	// Build PlayerInfo for game creation.
	playersInfo := []*ipc.PlayerInfo{
		{Nickname: player1Name, FullName: player1Name, First: true},
		{Nickname: player2Name, FullName: player2Name, First: false},
	}

	// Create the annotated game document, bypassing the outstanding-games check.
	gameID, err := omgwords.CreateAnnotatedGameDocForBroadcast(
		ctx,
		u.UUID,
		playersInfo,
		broadcast.Lexicon,
		broadcast.BoardLayout,
		broadcast.LetterDistribution,
		ipc.ChallengeRule(broadcast.ChallengeRule),
		bs.cfg,
		bs.gameStore,
		bs.metadataStore,
		bs.gameEventChan,
	)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	// Record the claim in broadcast_games.
	if _, err := bs.queries.CreateBroadcastGame(ctx, models.CreateBroadcastGameParams{
		BroadcastID:     broadcast.ID,
		GameUuid:        gameID,
		Division:        req.Msg.Division,
		Round:           req.Msg.Round,
		TableNumber:     req.Msg.TableNumber,
		Player1Name:     player1Name,
		Player2Name:     player2Name,
		AnnotatorUserID: pgtype.Int4{Int32: int32(u.ID), Valid: true},
	}); err != nil {
		// Best effort: game was created, log the error but still return the game ID.
		log.Err(err).Str("gameID", gameID).Msg("record-broadcast-game-failed")
	}

	bs.notifyBroadcastGamesUpdated(broadcast.Uuid.String(), broadcast.Slug)
	return connect.NewResponse(&pb.ClaimGameResponse{GameId: gameID}), nil
}

func (bs *BroadcastService) UnclaimGame(ctx context.Context, req *connect.Request[pb.UnclaimGameRequest]) (
	*connect.Response[pb.UnclaimGameResponse], error) {

	u, err := apiserver.AuthUser(ctx, bs.userStore)
	if err != nil {
		return nil, err
	}

	broadcast, err := bs.queries.GetBroadcastBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, apiserver.NotFound("broadcast not found")
	}

	// Check permission: must be a director, OR the annotator of this specific game.
	isDir, err := bs.queries.IsBroadcastDirector(ctx, models.IsBroadcastDirectorParams{
		BroadcastID: broadcast.ID,
		UserID:      int32(u.ID),
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	if !isDir {
		info, err := bs.queries.GetBroadcastGameAnnotatorInfo(ctx, models.GetBroadcastGameAnnotatorInfoParams{
			BroadcastID: broadcast.ID,
			Division:    req.Msg.Division,
			Round:       req.Msg.Round,
			TableNumber: req.Msg.TableNumber,
		})
		if err != nil {
			return nil, apiserver.NotFound("game not found")
		}
		if !info.AnnotatorUserID.Valid || int32(u.ID) != info.AnnotatorUserID.Int32 {
			return nil, apiserver.PermissionDenied("not authorized to unclaim this game")
		}
	}

	gameUUID, err := bs.queries.UnclaimBroadcastGame(ctx, models.UnclaimBroadcastGameParams{
		BroadcastID: broadcast.ID,
		Division:    req.Msg.Division,
		Round:       req.Msg.Round,
		TableNumber: req.Msg.TableNumber,
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	// Clean up the game document if one was created.
	if gameUUID != "" {
		if err := bs.metadataStore.DeleteAnnotatedGame(ctx, gameUUID); err != nil {
			log.Err(err).Str("gameUUID", gameUUID).Msg("unclaim: failed to delete annotated game metadata")
		}
		if err := bs.metadataStore.RemoveGameFromAllCollections(ctx, gameUUID); err != nil {
			log.Err(err).Str("gameUUID", gameUUID).Msg("unclaim: failed to remove game from collections")
		}
		if err := bs.gameStore.DeleteDocument(ctx, gameUUID); err != nil {
			log.Err(err).Str("gameUUID", gameUUID).Msg("unclaim: failed to delete game document")
		}
	}

	bs.notifyBroadcastGamesUpdated(broadcast.Uuid.String(), broadcast.Slug)
	return connect.NewResponse(&pb.UnclaimGameResponse{}), nil
}

func (bs *BroadcastService) GetMyClaimedGames(ctx context.Context, req *connect.Request[pb.GetMyClaimedGamesRequest]) (
	*connect.Response[pb.GetMyClaimedGamesResponse], error) {

	u, err := apiserver.AuthUser(ctx, bs.userStore)
	if err != nil {
		return nil, err
	}

	broadcast, err := bs.queries.GetBroadcastBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, apiserver.NotFound("broadcast not found")
	}

	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 10
	}

	rows, err := bs.queries.GetMyClaimedGames(ctx, models.GetMyClaimedGamesParams{
		BroadcastID:     broadcast.ID,
		AnnotatorUserID: pgtype.Int4{Int32: int32(u.ID), Valid: true},
		Limit:           limit,
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	games := make([]*pb.BroadcastRoundGame, 0, len(rows))
	for _, row := range rows {
		annotationDone := bs.isAnnotatedGameDone(ctx, row.GameUuid)
		games = append(games, &pb.BroadcastRoundGame{
			Round:             row.Round,
			TableNumber:       row.TableNumber,
			Player1Name:       row.Player1Name,
			Player2Name:       row.Player2Name,
			GameUuid:          row.GameUuid,
			Division:          row.Division,
			AnnotatorUsername: u.Username,
			AnnotationDone:    annotationDone,
		})
	}

	return connect.NewResponse(&pb.GetMyClaimedGamesResponse{Games: games}), nil
}

func (bs *BroadcastService) GetBroadcastGameContext(ctx context.Context, req *connect.Request[pb.GetBroadcastGameContextRequest]) (
	*connect.Response[pb.GetBroadcastGameContextResponse], error) {

	row, err := bs.queries.GetBroadcastGameByUUID(ctx, req.Msg.GameUuid)
	if err != nil {
		return nil, apiserver.NotFound("broadcast game not found")
	}

	return connect.NewResponse(&pb.GetBroadcastGameContextResponse{
		BroadcastSlug: row.BroadcastSlug,
		BroadcastName: row.BroadcastName,
		Round:         row.Round,
		TableNumber:   row.TableNumber,
		Division:      row.Division,
	}), nil
}

func (bs *BroadcastService) UpdateBroadcast(ctx context.Context, req *connect.Request[pb.UpdateBroadcastRequest]) (
	*connect.Response[pb.UpdateBroadcastResponse], error) {

	broadcast, err := bs.queries.GetBroadcastBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, apiserver.NotFound("broadcast not found")
	}

	if _, err := bs.requireDirector(ctx, broadcast.ID); err != nil {
		return nil, err
	}

	if err := bs.queries.UpdateBroadcast(ctx, models.UpdateBroadcastParams{
		Slug:                req.Msg.Slug,
		Name:                req.Msg.Name,
		Description:         pgtype.Text{String: req.Msg.Description, Valid: req.Msg.Description != ""},
		BroadcastUrl:        req.Msg.BroadcastUrl,
		BroadcastUrlFormat:  req.Msg.BroadcastUrlFormat,
		PollIntervalSeconds: req.Msg.PollIntervalSeconds,
		PollStartTime:       pgTimestamptz(req.Msg.PollStartTime),
		PollEndTime:         pgTimestamptz(req.Msg.PollEndTime),
		Lexicon:             req.Msg.Lexicon,
		BoardLayout:         req.Msg.BoardLayout,
		LetterDistribution:  req.Msg.LetterDistribution,
		ChallengeRule:       req.Msg.ChallengeRule,
		Active:              req.Msg.Active,
	}); err != nil {
		return nil, apiserver.InternalErr(err)
	}

	return connect.NewResponse(&pb.UpdateBroadcastResponse{}), nil
}

func (bs *BroadcastService) AddBroadcastDirectors(ctx context.Context, req *connect.Request[pb.AddBroadcastDirectorsRequest]) (
	*connect.Response[pb.AddBroadcastDirectorsResponse], error) {

	broadcast, err := bs.queries.GetBroadcastBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, apiserver.NotFound("broadcast not found")
	}
	if _, err := bs.requireDirector(ctx, broadcast.ID); err != nil {
		return nil, err
	}

	for _, username := range req.Msg.Usernames {
		target, err := bs.userStore.Get(ctx, username)
		if err != nil {
			return nil, apiserver.InvalidArg("user not found: " + username)
		}
		if err := bs.queries.AddBroadcastDirector(ctx, models.AddBroadcastDirectorParams{
			BroadcastID: broadcast.ID,
			UserID:      int32(target.ID),
		}); err != nil {
			return nil, apiserver.InternalErr(err)
		}
	}
	return connect.NewResponse(&pb.AddBroadcastDirectorsResponse{}), nil
}

func (bs *BroadcastService) RemoveBroadcastDirectors(ctx context.Context, req *connect.Request[pb.RemoveBroadcastDirectorsRequest]) (
	*connect.Response[pb.RemoveBroadcastDirectorsResponse], error) {

	broadcast, err := bs.queries.GetBroadcastBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, apiserver.NotFound("broadcast not found")
	}
	if _, err := bs.requireDirector(ctx, broadcast.ID); err != nil {
		return nil, err
	}

	for _, username := range req.Msg.Usernames {
		target, err := bs.userStore.Get(ctx, username)
		if err != nil {
			continue // ignore not found — idempotent
		}
		bs.queries.RemoveBroadcastDirector(ctx, models.RemoveBroadcastDirectorParams{
			BroadcastID: broadcast.ID,
			UserID:      int32(target.ID),
		})
	}
	return connect.NewResponse(&pb.RemoveBroadcastDirectorsResponse{}), nil
}

func (bs *BroadcastService) AddBroadcastAnnotators(ctx context.Context, req *connect.Request[pb.AddBroadcastAnnotatorsRequest]) (
	*connect.Response[pb.AddBroadcastAnnotatorsResponse], error) {

	broadcast, err := bs.queries.GetBroadcastBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, apiserver.NotFound("broadcast not found")
	}
	if _, err := bs.requireDirector(ctx, broadcast.ID); err != nil {
		return nil, err
	}

	for _, username := range req.Msg.Usernames {
		target, err := bs.userStore.Get(ctx, username)
		if err != nil {
			return nil, apiserver.InvalidArg("user not found: " + username)
		}
		if err := bs.queries.AddBroadcastAnnotator(ctx, models.AddBroadcastAnnotatorParams{
			BroadcastID: broadcast.ID,
			UserID:      int32(target.ID),
		}); err != nil {
			return nil, apiserver.InternalErr(err)
		}
	}
	return connect.NewResponse(&pb.AddBroadcastAnnotatorsResponse{}), nil
}

func (bs *BroadcastService) RemoveBroadcastAnnotators(ctx context.Context, req *connect.Request[pb.RemoveBroadcastAnnotatorsRequest]) (
	*connect.Response[pb.RemoveBroadcastAnnotatorsResponse], error) {

	broadcast, err := bs.queries.GetBroadcastBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, apiserver.NotFound("broadcast not found")
	}
	if _, err := bs.requireDirector(ctx, broadcast.ID); err != nil {
		return nil, err
	}

	for _, username := range req.Msg.Usernames {
		target, err := bs.userStore.Get(ctx, username)
		if err != nil {
			continue
		}
		bs.queries.RemoveBroadcastAnnotator(ctx, models.RemoveBroadcastAnnotatorParams{
			BroadcastID: broadcast.ID,
			UserID:      int32(target.ID),
		})
	}
	return connect.NewResponse(&pb.RemoveBroadcastAnnotatorsResponse{}), nil
}

func (bs *BroadcastService) GetActiveBroadcasts(ctx context.Context, req *connect.Request[pb.GetActiveBroadcastsRequest]) (
	*connect.Response[pb.GetActiveBroadcastsResponse], error) {

	rows, err := bs.queries.GetActiveBroadcasts(ctx)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	broadcasts := make([]*pb.Broadcast, 0, len(rows))
	for _, row := range rows {
		b := broadcastRowToProto(
			row.ID, row.Uuid, row.Slug, row.Name,
			row.Description, row.BroadcastUrl, row.BroadcastUrlFormat,
			row.PollIntervalSeconds, row.PollStartTime, row.PollEndTime,
			row.Lexicon, row.BoardLayout, row.LetterDistribution, row.ChallengeRule,
			row.Active, row.CreatedAt, row.CreatorUsername,
		)
		if entry := bs.getCachedEntry(row.Slug, row.BroadcastUrl, row.BroadcastUrlFormat); entry != nil {
			// Use first division's current round for the broadcast list.
			div := firstDivision(entry)
			if fd := entry.divisions[div]; fd != nil {
				b.CurrentRound = int32(fd.CurrentRound)
			}
		}
		broadcasts = append(broadcasts, b)
	}

	return connect.NewResponse(&pb.GetActiveBroadcastsResponse{Broadcasts: broadcasts}), nil
}

func (bs *BroadcastService) TriggerPoll(ctx context.Context, req *connect.Request[pb.TriggerPollRequest]) (
	*connect.Response[pb.TriggerPollResponse], error) {

	broadcast, err := bs.queries.GetBroadcastBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, apiserver.NotFound("broadcast not found")
	}
	if _, err := bs.requireDirector(ctx, broadcast.ID); err != nil {
		return nil, err
	}

	if err := bs.pollBroadcast(ctx, broadcast.ID, broadcast.Uuid.String(), broadcast.Slug, broadcast.BroadcastUrl, broadcast.BroadcastUrlFormat); err != nil {
		return nil, apiserver.InternalErr(err)
	}

	return connect.NewResponse(&pb.TriggerPollResponse{DataChanged: true}), nil
}

// ---- helpers ----

func (bs *BroadcastService) playerNamesForTable(slug, division, broadcastURL, format string, round, tableNumber int) (string, string, error) {
	fd := bs.getCachedFeed(slug, division, broadcastURL, format)
	if fd == nil {
		return "", "", fmt.Errorf("feed data not available")
	}
	pairings := GetRoundPairings(fd, round)
	for _, p := range pairings {
		if p.TableNumber == tableNumber {
			return p.Player1Name, p.Player2Name, nil
		}
	}
	return "", "", fmt.Errorf("no pairing found for round %d table %d", round, tableNumber)
}

// notifyBroadcastGamesUpdated publishes a BROADCAST_GAMES_UPDATED NATS event
// when annotation state changes (claim/unclaim), so viewers refetch game rows.
func (bs *BroadcastService) notifyBroadcastGamesUpdated(broadcastUUID, slug string) {
	if bs.natsConn == nil {
		return
	}
	evt := entity.WrapEvent(&ipc.BroadcastGamesUpdatedEvent{
		Slug: slug,
	}, ipc.MessageType_BROADCAST_GAMES_UPDATED)
	bts, err := evt.Serialize()
	if err != nil {
		log.Err(err).Str("slug", slug).Msg("broadcast-games-updated-serialize-failed")
		return
	}
	if err := bs.natsConn.Publish(NatsBroadcastSubjectPrefix+broadcastUUID, bts); err != nil {
		log.Err(err).Str("slug", slug).Msg("broadcast-games-updated-publish-failed")
	}
}

// setCachedFeed stores all parsed divisions for a broadcast. Called by the
// poller after each successful fetch.
func (bs *BroadcastService) setCachedFeed(slug string, divisions map[string]*FeedData) {
	bs.feedCacheMu.Lock()
	bs.feedCache[slug] = &feedCacheEntry{divisions: divisions, expiresAt: time.Now().Add(feedCacheTTL)}
	bs.feedCacheMu.Unlock()
}

// fetchAndCacheAllDivisions fetches the broadcast URL, parses all divisions,
// stores them in the cache, and returns the division map.
func (bs *BroadcastService) fetchAndCacheAllDivisions(slug, broadcastURL, format string) (map[string]*FeedData, error) {
	rawData, err := fetchURL(broadcastURL)
	if err != nil {
		return nil, err
	}
	p, err := NewFeedParser(format)
	if err != nil {
		return nil, err
	}
	tp, ok := p.(*TSHNewtParser)
	if !ok {
		// Non-TSH parser: fall back to single-division parse under empty key.
		fd, err := p.Parse(rawData)
		if err != nil {
			return nil, err
		}
		divs := map[string]*FeedData{"": fd}
		bs.setCachedFeed(slug, divs)
		return divs, nil
	}
	divNames, err := tp.ListDivisions(rawData)
	if err != nil {
		return nil, err
	}
	divs := make(map[string]*FeedData, len(divNames))
	for _, name := range divNames {
		fd, err := tp.ParseDivision(rawData, name)
		if err != nil {
			log.Err(err).Str("slug", slug).Str("division", name).Msg("feed: failed to parse division")
			continue
		}
		divs[name] = fd
	}
	bs.setCachedFeed(slug, divs)
	return divs, nil
}

// getCachedEntry returns the cache entry for a slug, fetching on miss.
func (bs *BroadcastService) getCachedEntry(slug, broadcastURL, format string) *feedCacheEntry {
	bs.feedCacheMu.RLock()
	entry, ok := bs.feedCache[slug]
	bs.feedCacheMu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry
	}

	result, err, _ := bs.sfGroup.Do(slug, func() (any, error) {
		return bs.fetchAndCacheAllDivisions(slug, broadcastURL, format)
	})
	if err != nil {
		log.Err(err).Str("slug", slug).Msg("feed-cache-miss-fetch-failed")
		return nil
	}
	divs := result.(map[string]*FeedData)
	bs.feedCacheMu.RLock()
	entry = bs.feedCache[slug]
	bs.feedCacheMu.RUnlock()
	if entry != nil {
		return entry
	}
	// Fallback: construct an ephemeral entry (shouldn't normally happen).
	return &feedCacheEntry{divisions: divs, expiresAt: time.Now().Add(feedCacheTTL)}
}

// getCachedFeed returns the FeedData for a specific division. If division is
// empty, the first available division is returned.
func (bs *BroadcastService) getCachedFeed(slug, division, broadcastURL, format string) *FeedData {
	entry := bs.getCachedEntry(slug, broadcastURL, format)
	if entry == nil {
		return nil
	}
	if division == "" {
		division = firstDivision(entry)
		if division == "" {
			return nil
		}
	}
	return entry.divisions[division]
}

// getCachedDivisionNames returns the sorted list of division names from the cache.
func (bs *BroadcastService) getCachedDivisionNames(slug, broadcastURL, format string) []string {
	entry := bs.getCachedEntry(slug, broadcastURL, format)
	if entry == nil {
		return nil
	}
	names := make([]string, 0, len(entry.divisions))
	for name := range entry.divisions {
		names = append(names, name)
	}
	// Sort for stable ordering.
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	return names
}

// firstDivision returns the first (lowest alphabetically) division name from an entry.
func firstDivision(entry *feedCacheEntry) string {
	if entry == nil {
		return ""
	}
	best := ""
	for name := range entry.divisions {
		if best == "" || name < best {
			best = name
		}
	}
	return best
}

func feedDataToProto(fd *FeedData) ([]*pb.BroadcastPlayer, int32) {
	players := make([]*pb.BroadcastPlayer, 0, len(fd.Players))
	for _, p := range fd.Players {
		stats := computePlayerStats(p, fd.Players)
		players = append(players, &pb.BroadcastPlayer{
			PlayerId:    int32(p.ID),
			Name:        p.Name,
			Rating:      int32(p.Rating),
			Wins:        stats.wins,
			Losses:      stats.losses,
			Spread:      int32(stats.spread),
			GamesPlayed: int32(stats.gamesPlayed),
		})
	}
	// Sort by wins desc, then spread desc.
	for i := 1; i < len(players); i++ {
		for j := i; j > 0; j-- {
			a, b := players[j-1], players[j]
			if b.Wins > a.Wins || (b.Wins == a.Wins && b.Spread > a.Spread) {
				players[j-1], players[j] = players[j], players[j-1]
			} else {
				break
			}
		}
	}

	return players, int32(fd.TotalRounds)
}

type playerStats struct {
	wins        float64
	losses      float64
	spread      int
	gamesPlayed int
}

func computePlayerStats(p FeedPlayer, allPlayers []FeedPlayer) playerStats {
	var s playerStats
	for r := 0; r < len(p.Scores); r++ {
		if r >= len(p.Pairings) {
			continue
		}
		score := p.Scores[r]
		oppID := p.Pairings[r]
		if oppID == 0 && score == 0 {
			continue // unplayed
		}
		s.gamesPlayed++
		if oppID == 0 {
			if score < 0 {
				s.losses++
				s.spread += score
			} else {
				s.wins++
				s.spread += score
			}
			continue
		}
		opp := findPlayer(allPlayers, oppID)
		if opp == nil || r >= len(opp.Scores) {
			continue
		}
		oppScore := opp.Scores[r]
		s.spread += score - oppScore
		switch {
		case score > oppScore:
			s.wins++
		case score < oppScore:
			s.losses++
		default:
			s.wins += 0.5
			s.losses += 0.5
		}
	}
	return s
}

func broadcastRowToProto(
	id int32, uid uuid.UUID, slug, name string,
	description pgtype.Text,
	broadcastURL, broadcastURLFormat string,
	pollInterval int32,
	pollStart, pollEnd pgtype.Timestamptz,
	lexicon, boardLayout, letterDist string,
	challengeRule int32,
	active bool,
	createdAt pgtype.Timestamptz,
	creatorUsername pgtype.Text,
) *pb.Broadcast {
	_ = id // used only for DB lookups
	b := &pb.Broadcast{
		Uuid:                uid.String(),
		Slug:                slug,
		Name:                name,
		BroadcastUrl:        broadcastURL,
		BroadcastUrlFormat:  broadcastURLFormat,
		PollIntervalSeconds: pollInterval,
		Lexicon:             lexicon,
		BoardLayout:         boardLayout,
		LetterDistribution:  letterDist,
		ChallengeRule:       challengeRule,
		Active:              active,
	}
	if description.Valid {
		b.Description = description.String
	}
	if creatorUsername.Valid {
		b.CreatorUsername = creatorUsername.String
	}
	if pollStart.Valid {
		b.PollStartTime = timestamppb.New(pollStart.Time)
	}
	if pollEnd.Valid {
		b.PollEndTime = timestamppb.New(pollEnd.Time)
	}
	if createdAt.Valid {
		b.CreatedAt = timestamppb.New(createdAt.Time)
	}
	return b
}

func pgTimestamptz(ts *timestamppb.Timestamp) pgtype.Timestamptz {
	if ts == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: ts.AsTime(), Valid: true}
}

// GameIsDone is a helper used by GetBroadcastGames to check annotation status.
// It wraps the metadataStore call so service.go stays clean.
func (bs *BroadcastService) isAnnotatedGameDone(ctx context.Context, gameUUID string) bool {
	done, err := bs.metadataStore.GameIsDone(ctx, gameUUID)
	if err != nil {
		return false
	}
	return done
}

