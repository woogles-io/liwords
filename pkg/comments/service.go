package comments

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/samber/lo"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
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
}

const MaxCommentLength = 2048
const MaxCommentsPerReq = 25

func NewCommentsService(u user.Store, g gameplay.GameStore, c *comments.DBStore) *CommentsService {
	return &CommentsService{u, g, c}
}

func (cs *CommentsService) AddGameComment(ctx context.Context, req *pb.AddCommentRequest) (*pb.AddCommentResponse, error) {
	u, err := apiserver.AuthUser(ctx, apiserver.CookieFirst, cs.userStore)
	if err != nil {
		return nil, twirp.NewError(twirp.Unauthenticated, err.Error())
	}
	if len(req.Comment) > MaxCommentLength {
		return nil, twirp.NewError(twirp.InvalidArgument, "your comment is too long")
	}

	cid, err := cs.commentsStore.AddComment(ctx, req.GameId, int(u.ID), int(req.EventNumber), req.Comment)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.AddCommentResponse{
		CommentId: cid,
	}, nil
}

func (cs *CommentsService) GetGameComments(ctx context.Context, req *pb.GetCommentsRequest) (*pb.GetCommentsResponse, error) {
	comments, err := cs.commentsStore.GetComments(ctx, req.GameId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.GetCommentsResponse{
		Comments: lo.Map(comments, func(c models.GetCommentsForGameRow, idx int) *pb.GameComment {
			return &pb.GameComment{
				CommentId:   c.ID.String(),
				GameId:      c.GameUuid.String,
				UserId:      c.UserUuid.String,
				Username:    c.Username.String,
				EventNumber: uint32(c.EventNumber),
				Comment:     c.Comment,
				LastEdited:  timestamppb.New(c.EditedAt),
			}
		})}, nil
}

func (cs *CommentsService) EditGameComment(ctx context.Context, req *pb.EditCommentRequest) (*pb.EditCommentResponse, error) {
	u, err := apiserver.AuthUser(ctx, apiserver.CookieFirst, cs.userStore)
	if err != nil {
		return nil, twirp.NewError(twirp.Unauthenticated, err.Error())
	}
	if len(req.Comment) > MaxCommentLength {
		return nil, twirp.NewError(twirp.InvalidArgument, "your new comment is too long")
	}

	err = cs.commentsStore.UpdateComment(ctx, int(u.ID), req.CommentId, req.Comment)
	if err != nil {
		return nil, twirp.NewError(twirp.Unauthenticated, err.Error())
	}
	return &pb.EditCommentResponse{}, nil
}

func (cs *CommentsService) DeleteGameComment(ctx context.Context, req *pb.DeleteCommentRequest) (*pb.DeleteCommentResponse, error) {
	u, err := apiserver.AuthUser(ctx, apiserver.CookieFirst, cs.userStore)
	if err != nil {
		return nil, twirp.NewError(twirp.Unauthenticated, err.Error())
	}
	// Allow admins to delete comments.

	if u.IsAdmin || u.IsMod {
		err = cs.commentsStore.DeleteComment(ctx, req.CommentId, -1)
	} else {
		err = cs.commentsStore.DeleteComment(ctx, req.CommentId, int(u.ID))
	}
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.DeleteCommentResponse{}, nil
}

func (cs *CommentsService) GetCommentsForAllGames(ctx context.Context, req *pb.GetCommentsAllGamesRequest) (*pb.GetCommentsResponse, error) {
	if req.Limit > MaxCommentsPerReq {
		return nil, twirp.NewError(twirp.InvalidArgument, "too many comments")
	}
	comments, err := cs.commentsStore.GetCommentsForAllGames(ctx, int(req.Limit), int(req.Offset))

	if err != nil {
		return nil, err
	}

	return &pb.GetCommentsResponse{
		Comments: lo.Map(comments, func(c models.GetCommentsForAllGamesRow, idx int) *pb.GameComment {
			qd := &entity.Quickdata{}
			err := json.Unmarshal(c.Quickdata.Bytes, qd)
			gameMeta := map[string]string{}
			if err == nil {
				playerNames := lo.Map(qd.PlayerInfo, func(p *ipc.PlayerInfo, idx int) string {
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
				LastEdited:  timestamppb.New(c.EditedAt),
				GameMeta:    gameMeta,
			}
		})}, nil
}
