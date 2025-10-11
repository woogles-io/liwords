package collections

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/collections_service"
	commentspb "github.com/woogles-io/liwords/rpc/api/proto/comments_service"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type CollectionsService struct {
	userStore user.Store
	queries   *models.Queries
	dbPool    *pgxpool.Pool
}

func NewCollectionsService(u user.Store, q *models.Queries, db *pgxpool.Pool) *CollectionsService {
	return &CollectionsService{
		userStore: u,
		queries:   q,
		dbPool:    db,
	}
}

func (cs *CollectionsService) CreateCollection(ctx context.Context, req *connect.Request[pb.CreateCollectionRequest]) (*connect.Response[pb.CreateCollectionResponse], error) {
	u, err := apiserver.AuthUser(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}

	if req.Msg.Title == "" {
		return nil, apiserver.InvalidArg("title cannot be empty")
	}

	collectionUUID := uuid.New()

	result, err := cs.queries.CreateCollection(ctx, models.CreateCollectionParams{
		Uuid:        collectionUUID,
		Title:       req.Msg.Title,
		Description: pgtype.Text{String: req.Msg.Description, Valid: req.Msg.Description != ""},
		CreatorID:   int32(u.ID),
		Public:      pgtype.Bool{Bool: req.Msg.Public, Valid: true},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create collection")
		return nil, apiserver.InternalErr(err)
	}

	return connect.NewResponse(&pb.CreateCollectionResponse{
		CollectionUuid: result.Uuid.String(),
	}), nil
}

func (cs *CollectionsService) GetCollection(ctx context.Context, req *connect.Request[pb.GetCollectionRequest]) (*connect.Response[pb.GetCollectionResponse], error) {
	collectionUUID, err := uuid.Parse(req.Msg.CollectionUuid)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid collection UUID")
	}

	// Get collection with creator username
	collection, err := cs.queries.GetCollectionWithGames(ctx, collectionUUID)
	if err != nil {
		log.Error().Err(err).Str("uuid", req.Msg.CollectionUuid).Msg("failed to get collection")
		return nil, apiserver.NotFound("collection not found")
	}

	// Get games in collection
	games, err := cs.queries.GetCollectionGames(ctx, collection.ID)
	if err != nil {
		log.Error().Err(err).Int32("collection_id", collection.ID).Msg("failed to get collection games")
		return nil, apiserver.InternalErr(err)
	}

	// Convert to protobuf
	pbCollection := &pb.Collection{
		Uuid:            collection.Uuid.String(),
		Title:           collection.Title,
		Description:     collection.Description.String,
		CreatorUuid:     collection.CreatorUuid.String,
		CreatorUsername: collection.CreatorUsername.String,
		Public:          collection.Public.Bool,
		CreatedAt:       timestamppb.New(collection.CreatedAt.Time),
		UpdatedAt:       timestamppb.New(collection.UpdatedAt.Time),
		Games:           make([]*pb.CollectionGame, len(games)),
	}

	for i, game := range games {
		pbCollection.Games[i] = &pb.CollectionGame{
			GameId:        game.GameID,
			ChapterNumber: uint32(game.ChapterNumber),
			ChapterTitle:  game.ChapterTitle.String,
			IsAnnotated:   game.IsAnnotated.Bool,
			AddedAt:       timestamppb.New(game.AddedAt.Time),
		}
	}

	return connect.NewResponse(&pb.GetCollectionResponse{
		Collection: pbCollection,
	}), nil
}

func (cs *CollectionsService) UpdateCollection(ctx context.Context, req *connect.Request[pb.UpdateCollectionRequest]) (*connect.Response[pb.UpdateCollectionResponse], error) {
	u, err := apiserver.AuthUser(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}

	collectionUUID, err := uuid.Parse(req.Msg.CollectionUuid)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid collection UUID")
	}

	// Check ownership
	owns, err := cs.queries.CheckCollectionOwnership(ctx, models.CheckCollectionOwnershipParams{
		Uuid:      collectionUUID,
		CreatorID: int32(u.ID),
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	if !owns {
		return nil, apiserver.PermissionDenied("you don't own this collection")
	}

	if req.Msg.Title == "" {
		return nil, apiserver.InvalidArg("title cannot be empty")
	}

	err = cs.queries.UpdateCollection(ctx, models.UpdateCollectionParams{
		Uuid:        collectionUUID,
		Title:       req.Msg.Title,
		Description: pgtype.Text{String: req.Msg.Description, Valid: req.Msg.Description != ""},
		Public:      pgtype.Bool{Bool: req.Msg.Public, Valid: true},
	})
	if err != nil {
		log.Error().Err(err).Str("uuid", req.Msg.CollectionUuid).Msg("failed to update collection")
		return nil, apiserver.InternalErr(err)
	}

	return connect.NewResponse(&pb.UpdateCollectionResponse{}), nil
}

func (cs *CollectionsService) DeleteCollection(ctx context.Context, req *connect.Request[pb.DeleteCollectionRequest]) (*connect.Response[pb.DeleteCollectionResponse], error) {
	u, err := apiserver.AuthUser(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}

	collectionUUID, err := uuid.Parse(req.Msg.CollectionUuid)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid collection UUID")
	}

	// Check ownership
	owns, err := cs.queries.CheckCollectionOwnership(ctx, models.CheckCollectionOwnershipParams{
		Uuid:      collectionUUID,
		CreatorID: int32(u.ID),
	})
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	if !owns {
		return nil, apiserver.PermissionDenied("you don't own this collection")
	}

	err = cs.queries.DeleteCollection(ctx, collectionUUID)
	if err != nil {
		log.Error().Err(err).Str("uuid", req.Msg.CollectionUuid).Msg("failed to delete collection")
		return nil, apiserver.InternalErr(err)
	}

	return connect.NewResponse(&pb.DeleteCollectionResponse{}), nil
}

func (cs *CollectionsService) AddGameToCollection(ctx context.Context, req *connect.Request[pb.AddGameToCollectionRequest]) (*connect.Response[pb.AddGameToCollectionResponse], error) {
	u, err := apiserver.AuthUser(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}

	collectionUUID, err := uuid.Parse(req.Msg.CollectionUuid)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid collection UUID")
	}

	// Get collection to check ownership and get ID
	collection, err := cs.queries.GetCollectionByUUID(ctx, collectionUUID)
	if err != nil {
		return nil, apiserver.NotFound("collection not found")
	}

	if collection.CreatorID != int32(u.ID) {
		return nil, apiserver.PermissionDenied("you don't own this collection")
	}

	// Get next chapter number
	maxChapter, err := cs.queries.GetMaxChapterNumber(ctx, collection.ID)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	nextChapterNumber := int32(1)
	if maxChapter != nil {
		if maxInt, ok := maxChapter.(int32); ok {
			nextChapterNumber = maxInt + 1
		}
	}

	err = cs.queries.AddGameToCollection(ctx, models.AddGameToCollectionParams{
		CollectionID:  collection.ID,
		GameID:        req.Msg.GameId,
		ChapterNumber: nextChapterNumber,
		ChapterTitle:  pgtype.Text{String: req.Msg.ChapterTitle, Valid: req.Msg.ChapterTitle != ""},
		IsAnnotated:   pgtype.Bool{Bool: req.Msg.IsAnnotated, Valid: true},
	})
	if err != nil {
		log.Error().Err(err).Str("collection_uuid", req.Msg.CollectionUuid).Str("game_id", req.Msg.GameId).Msg("failed to add game to collection")
		return nil, apiserver.InternalErr(err)
	}

	// Update collection timestamp
	err = cs.queries.UpdateCollectionTimestamp(ctx, collection.ID)
	if err != nil {
		log.Error().Err(err).Str("collection_uuid", req.Msg.CollectionUuid).Msg("failed to update collection timestamp")
		// Don't fail the operation, just log the error
	}

	return connect.NewResponse(&pb.AddGameToCollectionResponse{}), nil
}

func (cs *CollectionsService) RemoveGameFromCollection(ctx context.Context, req *connect.Request[pb.RemoveGameFromCollectionRequest]) (*connect.Response[pb.RemoveGameFromCollectionResponse], error) {
	u, err := apiserver.AuthUser(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}

	collectionUUID, err := uuid.Parse(req.Msg.CollectionUuid)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid collection UUID")
	}

	// Get collection to check ownership and get ID
	collection, err := cs.queries.GetCollectionByUUID(ctx, collectionUUID)
	if err != nil {
		return nil, apiserver.NotFound("collection not found")
	}

	if collection.CreatorID != int32(u.ID) {
		return nil, apiserver.PermissionDenied("you don't own this collection")
	}

	err = cs.queries.RemoveGameFromCollection(ctx, models.RemoveGameFromCollectionParams{
		CollectionID: collection.ID,
		GameID:       req.Msg.GameId,
	})
	if err != nil {
		log.Error().Err(err).Str("collection_uuid", req.Msg.CollectionUuid).Str("game_id", req.Msg.GameId).Msg("failed to remove game from collection")
		return nil, apiserver.InternalErr(err)
	}

	return connect.NewResponse(&pb.RemoveGameFromCollectionResponse{}), nil
}

func (cs *CollectionsService) ReorderGames(ctx context.Context, req *connect.Request[pb.ReorderGamesRequest]) (*connect.Response[pb.ReorderGamesResponse], error) {
	u, err := apiserver.AuthUser(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}

	collectionUUID, err := uuid.Parse(req.Msg.CollectionUuid)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid collection UUID")
	}

	// Get collection to check ownership and get ID
	collection, err := cs.queries.GetCollectionByUUID(ctx, collectionUUID)
	if err != nil {
		return nil, apiserver.NotFound("collection not found")
	}

	if collection.CreatorID != int32(u.ID) {
		return nil, apiserver.PermissionDenied("you don't own this collection")
	}

	// Use transaction to ensure atomicity and avoid unique constraint violations
	tx, err := cs.dbPool.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Str("collection_uuid", req.Msg.CollectionUuid).Msg("failed to begin transaction")
		return nil, apiserver.InternalErr(err)
	}
	defer tx.Rollback(ctx)

	qtx := cs.queries.WithTx(tx)

	// Step 1: Set all chapter numbers to negative values to avoid conflicts
	err = qtx.SetTempChapterNumbers(ctx, collection.ID)
	if err != nil {
		log.Error().Err(err).Str("collection_uuid", req.Msg.CollectionUuid).Msg("failed to set temporary chapter numbers")
		return nil, apiserver.InternalErr(err)
	}

	// Step 2: Update chapter numbers for each game in the desired order
	for i, gameID := range req.Msg.GameIds {
		err = qtx.ReorderCollectionGames(ctx, models.ReorderCollectionGamesParams{
			CollectionID:  collection.ID,
			ChapterNumber: int32(i + 1), // 1-indexed
			GameID:        gameID,
		})
		if err != nil {
			log.Error().Err(err).Str("collection_uuid", req.Msg.CollectionUuid).Str("game_id", gameID).Msg("failed to reorder game")
			return nil, apiserver.InternalErr(err)
		}
	}

	// Update collection timestamp
	err = qtx.UpdateCollectionTimestamp(ctx, collection.ID)
	if err != nil {
		log.Error().Err(err).Str("collection_uuid", req.Msg.CollectionUuid).Msg("failed to update collection timestamp")
		// Don't fail the operation, just log the error
	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Error().Err(err).Str("collection_uuid", req.Msg.CollectionUuid).Msg("failed to commit transaction")
		return nil, apiserver.InternalErr(err)
	}

	return connect.NewResponse(&pb.ReorderGamesResponse{}), nil
}

func (cs *CollectionsService) UpdateChapterTitle(ctx context.Context, req *connect.Request[pb.UpdateChapterTitleRequest]) (*connect.Response[pb.UpdateChapterTitleResponse], error) {
	u, err := apiserver.AuthUser(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}

	collectionUUID, err := uuid.Parse(req.Msg.CollectionUuid)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid collection UUID")
	}

	// Get collection to check ownership and get ID
	collection, err := cs.queries.GetCollectionByUUID(ctx, collectionUUID)
	if err != nil {
		return nil, apiserver.NotFound("collection not found")
	}

	if collection.CreatorID != int32(u.ID) {
		return nil, apiserver.PermissionDenied("you don't own this collection")
	}

	err = cs.queries.UpdateChapterTitle(ctx, models.UpdateChapterTitleParams{
		CollectionID: collection.ID,
		GameID:       req.Msg.GameId,
		ChapterTitle: pgtype.Text{String: req.Msg.ChapterTitle, Valid: req.Msg.ChapterTitle != ""},
	})
	if err != nil {
		log.Error().Err(err).Str("collection_uuid", req.Msg.CollectionUuid).Str("game_id", req.Msg.GameId).Msg("failed to update chapter title")
		return nil, apiserver.InternalErr(err)
	}

	// Update collection timestamp
	err = cs.queries.UpdateCollectionTimestamp(ctx, collection.ID)
	if err != nil {
		log.Error().Err(err).Str("collection_uuid", req.Msg.CollectionUuid).Msg("failed to update collection timestamp")
		// Don't fail the operation, just log the error
	}

	return connect.NewResponse(&pb.UpdateChapterTitleResponse{}), nil
}

func (cs *CollectionsService) GetUserCollections(ctx context.Context, req *connect.Request[pb.GetUserCollectionsRequest]) (*connect.Response[pb.GetUserCollectionsResponse], error) {
	// Parse user UUID - could be either requesting own collections or someone else's public ones
	var userUUID string
	if req.Msg.UserUuid == "" {
		// Get own collections
		u, err := apiserver.AuthUser(ctx, cs.userStore)
		if err != nil {
			return nil, err
		}
		userUUID = u.UUID
	} else {
		userUUID = req.Msg.UserUuid
	}

	limit := req.Msg.Limit
	if limit == 0 || limit > 50 {
		limit = 20
	}

	collections, err := cs.queries.GetUserCollections(ctx, models.GetUserCollectionsParams{
		Uuid:   pgtype.Text{String: userUUID, Valid: userUUID != ""},
		Limit:  int32(limit),
		Offset: int32(req.Msg.Offset),
	})
	if err != nil {
		log.Error().Err(err).Str("user_uuid", userUUID).Msg("failed to get user collections")
		return nil, apiserver.InternalErr(err)
	}

	pbCollections := make([]*pb.Collection, len(collections))
	for i, collection := range collections {
		// Convert GameCount from interface{} to uint32
		var gameCount uint32
		if gc, ok := collection.GameCount.(int64); ok {
			gameCount = uint32(gc)
		} else if gc, ok := collection.GameCount.(int32); ok {
			gameCount = uint32(gc)
		}

		pbCollections[i] = &pb.Collection{
			Uuid:            collection.Uuid.String(),
			Title:           collection.Title,
			Description:     collection.Description.String,
			CreatorUuid:     collection.CreatorUuid.String,
			CreatorUsername: collection.CreatorUsername.String,
			Public:          collection.Public.Bool,
			CreatedAt:       timestamppb.New(collection.CreatedAt.Time),
			UpdatedAt:       timestamppb.New(collection.UpdatedAt.Time),
			GameCount:       gameCount,
		}
	}

	return connect.NewResponse(&pb.GetUserCollectionsResponse{
		Collections: pbCollections,
	}), nil
}

func (cs *CollectionsService) GetPublicCollections(ctx context.Context, req *connect.Request[pb.GetPublicCollectionsRequest]) (*connect.Response[pb.GetPublicCollectionsResponse], error) {
	limit := req.Msg.Limit
	if limit == 0 || limit > 50 {
		limit = 20
	}

	collections, err := cs.queries.GetPublicCollections(ctx, models.GetPublicCollectionsParams{
		Limit:  int32(limit),
		Offset: int32(req.Msg.Offset),
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get public collections")
		return nil, apiserver.InternalErr(err)
	}

	pbCollections := make([]*pb.Collection, len(collections))
	for i, collection := range collections {
		pbCollections[i] = &pb.Collection{
			Uuid:            collection.Uuid.String(),
			Title:           collection.Title,
			Description:     collection.Description.String,
			CreatorUuid:     collection.CreatorUuid.String,
			CreatorUsername: collection.CreatorUsername.String,
			Public:          collection.Public.Bool,
			CreatedAt:       timestamppb.New(collection.CreatedAt.Time),
			UpdatedAt:       timestamppb.New(collection.UpdatedAt.Time),
			GameCount:       uint32(collection.GameCount),
		}
	}

	return connect.NewResponse(&pb.GetPublicCollectionsResponse{
		Collections: pbCollections,
	}), nil
}

func (cs *CollectionsService) GetCollectionsForGame(ctx context.Context, req *connect.Request[pb.GetCollectionsForGameRequest]) (*connect.Response[pb.GetCollectionsForGameResponse], error) {
	if req.Msg.GameId == "" {
		return nil, apiserver.InvalidArg("game ID cannot be empty")
	}

	collections, err := cs.queries.GetCollectionsForGame(ctx, req.Msg.GameId)
	if err != nil {
		log.Error().Err(err).Str("game_id", req.Msg.GameId).Msg("failed to get collections for game")
		return nil, apiserver.InternalErr(err)
	}

	// Group by collection UUID to avoid duplicates
	collectionMap := make(map[string]*pb.Collection)
	for _, collection := range collections {
		uuid := collection.Uuid.String()
		if _, exists := collectionMap[uuid]; !exists {
			collectionMap[uuid] = &pb.Collection{
				Uuid:            uuid,
				Title:           collection.Title,
				Description:     collection.Description.String,
				CreatorUuid:     collection.CreatorUuid.String,
				CreatorUsername: collection.CreatorUsername.String,
				Public:          collection.Public.Bool,
				Games: []*pb.CollectionGame{{
					GameId:        req.Msg.GameId,
					ChapterNumber: uint32(collection.ChapterNumber),
					ChapterTitle:  collection.ChapterTitle.String,
				}},
			}
		}
	}

	// Convert map to slice
	pbCollections := make([]*pb.Collection, 0, len(collectionMap))
	for _, collection := range collectionMap {
		pbCollections = append(pbCollections, collection)
	}

	return connect.NewResponse(&pb.GetCollectionsForGameResponse{
		Collections: pbCollections,
	}), nil
}

func (cs *CollectionsService) GetRecentlyUpdatedCollections(ctx context.Context, req *connect.Request[pb.GetRecentlyUpdatedCollectionsRequest]) (*connect.Response[pb.GetRecentlyUpdatedCollectionsResponse], error) {
	limit := req.Msg.Limit
	if limit == 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Build parameters for the query
	params := models.GetRecentlyUpdatedCollectionsParams{
		Limit:  int32(limit),
		Offset: int32(req.Msg.Offset),
	}

	// If user UUID is provided, include it for private collections
	if req.Msg.UserUuid != "" {
		// The userUuid might be in short UUID format (base57 encoded) which is valid
		// We'll pass it through to the database as-is since the users table stores
		// UUIDs in the short format
		params.UserUuid = pgtype.Text{String: req.Msg.UserUuid, Valid: true}
	}

	collections, err := cs.queries.GetRecentlyUpdatedCollections(ctx, params)
	if err != nil {
		log.Error().Err(err).Msg("failed to get recently updated collections")
		return nil, apiserver.InternalErr(err)
	}

	pbCollections := make([]*pb.Collection, len(collections))
	for i, collection := range collections {
		// Convert GameCount from interface{} to uint32
		var gameCount uint32
		if gc, ok := collection.GameCount.(int64); ok {
			gameCount = uint32(gc)
		} else if gc, ok := collection.GameCount.(int32); ok {
			gameCount = uint32(gc)
		}

		pbCollections[i] = &pb.Collection{
			Uuid:            collection.Uuid.String(),
			Title:           collection.Title,
			Description:     collection.Description.String,
			CreatorUuid:     collection.CreatorUuid.String,
			CreatorUsername: collection.CreatorUsername.String,
			Public:          collection.Public.Bool,
			CreatedAt:       timestamppb.New(collection.CreatedAt.Time),
			UpdatedAt:       timestamppb.New(collection.UpdatedAt.Time),
			GameCount:       gameCount,
		}
	}

	return connect.NewResponse(&pb.GetRecentlyUpdatedCollectionsResponse{
		Collections: pbCollections,
	}), nil
}

func (cs *CollectionsService) GetCommentsForCollectionGames(ctx context.Context, collectionUUID string, limit, offset int) ([]*commentspb.GameComment, error) {
	comments, err := cs.queries.GetCommentsForCollectionGames(ctx, models.GetCommentsForCollectionGamesParams{
		Uuid:   uuid.MustParse(collectionUUID),
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	return lo.Map(comments, func(c models.GetCommentsForCollectionGamesRow, idx int) *commentspb.GameComment {
		gameMeta := map[string]string{}
		if c.Quickdata.PlayerInfo != nil {
			playerNames := lo.Map(c.Quickdata.PlayerInfo, func(p *ipc.PlayerInfo, idx int) string {
				return p.FullName
			})
			gameMeta["players"] = strings.Join(playerNames, " vs ")
		}
		return &commentspb.GameComment{
			CommentId:   c.ID.String(),
			GameId:      c.GameUuid.String,
			UserId:      c.UserUuid.String,
			Username:    c.Username.String,
			EventNumber: uint32(c.EventNumber),
			Comment:     c.Comment,
			LastEdited:  timestamppb.New(c.EditedAt.Time),
			GameMeta:    gameMeta,
		}
	}), nil
}
