package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/domino14/liwords/pkg/apiserver"
	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	pb "github.com/domino14/liwords/rpc/api/proto/user_service"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"
)

type SocializeService struct {
	userStore Store
	chatStore ChatStore
}

func NewSocializeService(u Store, c ChatStore) *SocializeService {
	return &SocializeService{userStore: u, chatStore: c}
}

func (ss *SocializeService) AddFollow(ctx context.Context, req *pb.AddFollowRequest) (*pb.OKResponse, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, twirp.InternalErrorWith(err)
	}

	followed, err := ss.userStore.GetByUUID(ctx, req.Uuid)
	if err != nil {
		log.Err(err).Msg("getting-followed")
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	err = ss.userStore.AddFollower(ctx, followed.ID, user.ID)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.OKResponse{}, nil
}

func (ss *SocializeService) RemoveFollow(ctx context.Context, req *pb.RemoveFollowRequest) (*pb.OKResponse, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, twirp.InternalErrorWith(err)
	}

	unfollowed, err := ss.userStore.GetByUUID(ctx, req.Uuid)
	if err != nil {
		log.Err(err).Msg("getting-unfollowed")
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	err = ss.userStore.RemoveFollower(ctx, unfollowed.ID, user.ID)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.OKResponse{}, nil
}

func (ss *SocializeService) GetFollows(ctx context.Context, req *pb.GetFollowsRequest) (*pb.GetFollowsResponse, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, twirp.InternalErrorWith(err)
	}

	users, err := ss.userStore.GetFollows(ctx, user.ID)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	basicUsers := make([]*pb.BasicUser, len(users))
	for i, u := range users {
		basicUsers[i] = &pb.BasicUser{
			Uuid:     u.UUID,
			Username: u.Username,
		}
	}

	return &pb.GetFollowsResponse{Users: basicUsers}, nil
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
		return nil, twirp.InternalErrorWith(err)
	}

	blocked, err := ss.userStore.GetByUUID(ctx, req.Uuid)
	if err != nil {
		log.Err(err).Msg("getting-blocked")
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if blocked.IsAdmin || blocked.IsMod {
		log.Err(err).Msg("blocking-admin")
		return nil, twirp.NewError(twirp.InvalidArgument, "you cannot block that user")
	}

	err = ss.userStore.AddBlock(ctx, blocked.ID, user.ID)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
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
		return nil, twirp.InternalErrorWith(err)
	}

	unblocked, err := ss.userStore.GetByUUID(ctx, req.Uuid)
	if err != nil {
		log.Err(err).Msg("getting-unblocked")
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	err = ss.userStore.RemoveBlock(ctx, unblocked.ID, user.ID)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
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
		return nil, twirp.InternalErrorWith(err)
	}

	users, err := ss.userStore.GetBlocks(ctx, user.ID)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
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
		return nil, twirp.InternalErrorWith(err)
	}

	users, err := ss.userStore.GetFullBlocks(ctx, user.ID)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	basicUsers := make([]string, len(users))
	for i, u := range users {
		basicUsers[i] = u.UUID
	}

	return &pb.GetFullBlocksResponse{UserIds: basicUsers}, nil
}

func (ss *SocializeService) GetActiveChatChannels(ctx context.Context, req *pb.GetActiveChatChannelsRequest) (*pb.ActiveChatChannels, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	return ss.chatStore.LatestChannels(ctx, int(req.Number), int(req.Offset), sess.UserUUID, req.TournamentId)
}

func ChatChannelReceiver(uid, name string) (string, error) {
	users := strings.Split(strings.TrimPrefix(name, "chat.pm."), "_")
	if len(users) != 2 {
		return "", fmt.Errorf("malformed pm chat channel: %v", name)
	}
	foundus := false
	receiver := ""
	for _, user := range users {
		if user == uid {
			foundus = true
		} else {
			receiver = user
		}
	}
	if !foundus {
		return "", errors.New("cannot access chat in a channel you are not part of")
	}
	if receiver == "" {
		return "", errors.New("bad channel")
	}
	return receiver, nil
}

func (ss *SocializeService) GetChatsForChannel(ctx context.Context, req *pb.GetChatsRequest) (*realtime.ChatMessages, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("get-session-get-chats-for-channel")
		// Don't exit on error. We should allow unauthenticated users to get
		// chats, just not private ones.
	}
	if strings.HasPrefix(req.Channel, "chat.pm.") {
		// Verify that this chat channel is well formed and we have access to it.
		if sess == nil {
			return nil, err
		}
		_, err := ChatChannelReceiver(sess.UserUUID, req.Channel)
		if err != nil {
			return nil, err
		}
	}
	chats, err := ss.chatStore.OldChats(ctx, req.Channel, 100)
	if err != nil {
		return nil, err
	}
	return &realtime.ChatMessages{Messages: chats}, err
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
