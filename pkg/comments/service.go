package comments

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores/comments"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/comments_service"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type CommentsService struct {
	userStore     user.Store
	gameStore     gameplay.GameStore
	commentsStore *comments.DBStore
	queries       *models.Queries
}

const MaxCommentLength = 2048
const MaxCommentsPerReq = 25

func NewCommentsService(u user.Store, g gameplay.GameStore, c *comments.DBStore, q *models.Queries) *CommentsService {
	return &CommentsService{u, g, c, q}
}

func (cs *CommentsService) AddGameComment(ctx context.Context, req *connect.Request[pb.AddCommentRequest]) (*connect.Response[pb.AddCommentResponse], error) {
	u, err := apiserver.AuthUser(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if len(req.Msg.Comment) > MaxCommentLength {
		return nil, apiserver.InvalidArg("your comment is too long")
	}

	cid, err := cs.commentsStore.AddComment(ctx, req.Msg.GameId, int(u.ID), int(req.Msg.EventNumber), req.Msg.Comment)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.AddCommentResponse{
		CommentId: cid,
	}), nil
}

func (cs *CommentsService) GetGameComments(ctx context.Context, req *connect.Request[pb.GetCommentsRequest]) (*connect.Response[pb.GetCommentsResponse], error) {
	comments, err := cs.commentsStore.GetComments(ctx, req.Msg.GameId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.GetCommentsResponse{
		Comments: lo.Map(comments, func(c models.GetCommentsForGameRow, idx int) *pb.GameComment {
			return &pb.GameComment{
				CommentId:   c.ID.String(),
				GameId:      c.GameUuid.String,
				UserId:      c.UserUuid.String,
				Username:    c.Username.String,
				EventNumber: uint32(c.EventNumber),
				Comment:     c.Comment,
				LastEdited:  timestamppb.New(c.EditedAt.Time),
			}
		}),
	}), nil
}

func (cs *CommentsService) EditGameComment(ctx context.Context, req *connect.Request[pb.EditCommentRequest]) (*connect.Response[pb.EditCommentResponse], error) {
	u, err := apiserver.AuthUser(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if len(req.Msg.Comment) > MaxCommentLength {
		return nil, apiserver.InvalidArg("your new comment is too long")
	}

	err = cs.commentsStore.UpdateComment(ctx, int(u.ID), req.Msg.CommentId, req.Msg.Comment)
	if err != nil {
		return nil, apiserver.Unauthenticated(err.Error()) // This seems like it could be a different error, such as a server error or bad request depending on the nature of `err`.
	}
	return connect.NewResponse(&pb.EditCommentResponse{}), nil
}

func (cs *CommentsService) DeleteGameComment(ctx context.Context, req *connect.Request[pb.DeleteCommentRequest]) (*connect.Response[pb.DeleteCommentResponse], error) {
	u, err := apiserver.AuthUser(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	privilegedUser, err := cs.queries.HasPermission(ctx, models.HasPermissionParams{
		UserID:     int32(u.ID),
		Permission: string(rbac.CanModerateUsers),
	})

	if privilegedUser {
		err = cs.commentsStore.DeleteComment(ctx, req.Msg.CommentId, -1)
	} else {
		err = cs.commentsStore.DeleteComment(ctx, req.Msg.CommentId, int(u.ID))
	}
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.DeleteCommentResponse{}), nil
}

func (cs *CommentsService) GetCommentsForAllGames(ctx context.Context, req *connect.Request[pb.GetCommentsAllGamesRequest]) (*connect.Response[pb.GetCommentsResponse], error) {
	if req.Msg.Limit > MaxCommentsPerReq {
		return nil, apiserver.InvalidArg("too many comments")
	}
	comments, err := cs.commentsStore.GetCommentsForAllGames(ctx, int(req.Msg.Limit), int(req.Msg.Offset))
	if err != nil {
		return nil, err // Consider using a more specific error wrapper here, depending on what `err` typically represents.
	}

	return connect.NewResponse(&pb.GetCommentsResponse{
		Comments: lo.Map(comments, func(c models.GetCommentsForAllGamesRow, idx int) *pb.GameComment {
			gameMeta := map[string]string{}
			if c.Quickdata.PlayerInfo != nil {
				playerNames := lo.Map(c.Quickdata.PlayerInfo, func(p *ipc.PlayerInfo, idx int) string {
					return p.FullName
				})
				gameMeta["players"] = strings.Join(playerNames, " vs ")
			}
			return &pb.GameComment{
				CommentId:   c.ID.String(),
				GameId:      c.GameUuid.String,
				UserId:      c.UserUuid.String,
				Username:    c.Username.String,
				EventNumber: uint32(c.EventNumber),
				Comment:     c.Comment,
				LastEdited:  timestamppb.New(c.EditedAt.Time),
				GameMeta:    gameMeta,
			}
		}),
	}), nil
}

func (cs *CommentsService) GetCollectionComments(ctx context.Context, req *connect.Request[pb.GetCollectionCommentsRequest]) (*connect.Response[pb.GetCommentsResponse], error) {
	if req.Msg.Limit > MaxCommentsPerReq {
		return nil, apiserver.InvalidArg("too many comments")
	}

	comments, err := cs.commentsStore.GetCommentsForCollectionGames(ctx, req.Msg.CollectionUuid, int(req.Msg.Limit), int(req.Msg.Offset))
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.GetCommentsResponse{
		Comments: lo.Map(comments, func(c models.GetCommentsForCollectionGamesRow, idx int) *pb.GameComment {
			gameMeta := map[string]string{}
			if c.Quickdata.PlayerInfo != nil {
				playerNames := lo.Map(c.Quickdata.PlayerInfo, func(p *ipc.PlayerInfo, idx int) string {
					return p.FullName
				})
				gameMeta["players"] = strings.Join(playerNames, " vs ")
			}
			return &pb.GameComment{
				CommentId:   c.ID.String(),
				GameId:      c.GameUuid.String,
				UserId:      c.UserUuid.String,
				Username:    c.Username.String,
				EventNumber: uint32(c.EventNumber),
				Comment:     c.Comment,
				LastEdited:  timestamppb.New(c.EditedAt.Time),
				GameMeta:    gameMeta,
			}
		}),
	}), nil
}
