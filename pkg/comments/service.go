package comments

import (
	"context"

	"github.com/samber/lo"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/stores/comments"
	"github.com/domino14/liwords/pkg/stores/models"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/comments_service"
)

type CommentsService struct {
	userStore     user.Store
	gameStore     gameplay.GameStore
	commentsStore *comments.DBStore
}

const MaxCommentLength = 2048

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
	err = cs.commentsStore.DeleteComment(ctx, req.CommentId, int(u.ID))
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.DeleteCommentResponse{}, nil
}
