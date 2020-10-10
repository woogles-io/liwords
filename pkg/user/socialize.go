package user

import (
	"context"

	"github.com/domino14/liwords/pkg/apiserver"
	pb "github.com/domino14/liwords/rpc/api/proto/user_service"
	"github.com/rs/zerolog/log"
)

type SocializeService struct {
	userStore Store
}

func NewSocializeService(u Store) *SocializeService {
	return &SocializeService{userStore: u}
}

func (ss *SocializeService) AddFollow(ctx context.Context, req *pb.AddFollowRequest) (*pb.OKResponse, error) {
	// stub
	return &pb.OKResponse{}, nil
}

func (ss *SocializeService) RemoveFollow(ctx context.Context, req *pb.RemoveFollowRequest) (*pb.OKResponse, error) {
	return &pb.OKResponse{}, nil
}

func (ss *SocializeService) GetFollows(ctx context.Context, req *pb.GetFollowsRequest) (*pb.GetFollowsResponse, error) {
	return &pb.GetFollowsResponse{}, nil
}

// blocks
func (ss *SocializeService) AddBlock(ctx context.Context, req *pb.AddBlockRequest) (*pb.OKResponse, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, err
	}

	blocked, err := ss.userStore.GetByUUID(ctx, req.Uuid)
	if err != nil {
		log.Err(err).Msg("getting-blocked")
		return nil, err
	}

	err = ss.userStore.AddBlock(ctx, blocked.ID, user.ID)
	if err != nil {
		return nil, err
	}
	return &pb.OKResponse{}, nil
}

func (ss *SocializeService) RemoveBlock(ctx context.Context, req *pb.RemoveBlockRequest) (*pb.OKResponse, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, err
	}

	unblocked, err := ss.userStore.GetByUUID(ctx, req.Uuid)
	if err != nil {
		log.Err(err).Msg("getting-unblocked")
		return nil, err
	}

	err = ss.userStore.RemoveBlock(ctx, unblocked.ID, user.ID)
	if err != nil {
		return nil, err
	}
	return &pb.OKResponse{}, nil
}

func (ss *SocializeService) GetBlocks(ctx context.Context, req *pb.GetBlocksRequest) (*pb.GetBlocksResponse, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, err
	}

	users, err := ss.userStore.GetBlocks(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	basicUsers := make([]*pb.BasicUser, len(users))
	for i, u := range users {
		basicUsers[i] = &pb.BasicUser{
			Uuid:     u.UUID,
			Username: u.Username,
		}
	}

	return &pb.GetBlocksResponse{Users: basicUsers}, nil
}

func (ss *SocializeService) GetFullBlocks(ctx context.Context, req *pb.GetFullBlocksRequest) (*pb.GetFullBlocksResponse, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, err
	}

	users, err := ss.userStore.GetFullBlocks(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	basicUsers := make([]*pb.BasicUser, len(users))
	for i, u := range users {
		basicUsers[i] = &pb.BasicUser{
			Uuid:     u.UUID,
			Username: u.Username,
		}
	}

	return &pb.GetFullBlocksResponse{Users: basicUsers}, nil
}

// func (ss *SocializeService) GetBlockedBy(ctx context.Context, req *pb.GetBlocksRequest) (*pb.GetBlockedByResponse, error) {
// 	sess, err := apiserver.GetSession(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	user, err := ss.userStore.Get(ctx, sess.Username)
// 	if err != nil {
// 		log.Err(err).Msg("getting-user")
// 		return nil, err
// 	}

// 	users, err := ss.userStore.GetBlockedBy(ctx, user.ID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	basicUsers := make([]*pb.BasicUser, len(users))
// 	for i, u := range users {
// 		basicUsers[i] = &pb.BasicUser{
// 			Uuid:     u.UUID,
// 			Username: u.Username,
// 		}
// 	}

// 	return &pb.GetBlockedByResponse{Users: basicUsers}, nil
// }
