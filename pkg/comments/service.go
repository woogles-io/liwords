package comments

import (
	"context"

	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/stores/comments"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/comments_service"
)

type CommentsService struct {
	userStore     user.Store
	gameStore     gameplay.GameStore
	commentsStore *comments.DBStore
}

func NewCommentsService(u user.Store, g gameplay.GameStore, c *comments.DBStore) *CommentsService {
	return &CommentsService{u, g, c}
}

func (cs *CommentsService) AddGameComment(ctx context.Context, req *pb.AddCommentRequest) (*pb.AddCommentResponse, error) {
	return nil, nil
}

func (cs *CommentsService) GetGameComments(ctx context.Context, req *pb.GetCommentsRequest) (*pb.GetCommentsResponse, error) {
	return nil, nil
}

func (cs *CommentsService) EditGameComment(ctx context.Context, req *pb.EditCommentRequest) (*pb.EditCommentResponse, error) {
	return nil, nil
}

func (cs *CommentsService) DeleteGameComment(ctx context.Context, req *pb.DeleteCommentRequest) (*pb.DeleteCommentResponse, error) {
	return nil, nil
}
