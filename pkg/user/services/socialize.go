package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

type SocializeService struct {
	userStore     user.Store
	chatStore     user.ChatStore
	presenceStore user.PresenceStore
	queries       *models.Queries
}

func NewSocializeService(u user.Store, c user.ChatStore, p user.PresenceStore, q *models.Queries) *SocializeService {
	return &SocializeService{userStore: u, chatStore: c, presenceStore: p, queries: q}
}

func (ss *SocializeService) AddFollow(ctx context.Context, req *connect.Request[pb.AddFollowRequest],
) (*connect.Response[pb.OKResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, apiserver.InternalErr(err)
	}

	followed, err := ss.userStore.GetByUUID(ctx, req.Msg.Uuid)
	if err != nil {
		log.Err(err).Msg("getting-followed")
		return nil, apiserver.InvalidArg(err.Error())
	}

	err = ss.userStore.AddFollower(ctx, followed.ID, user.ID)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	err = ss.presenceStore.UpdateFollower(ctx, followed, user, true)
	if err != nil {
		// we cannot rollback the follow
		// XXX: why?
		return nil, apiserver.InvalidArg(err.Error())
	}

	return connect.NewResponse(&pb.OKResponse{}), nil
}

func (ss *SocializeService) RemoveFollow(ctx context.Context, req *connect.Request[pb.RemoveFollowRequest],
) (*connect.Response[pb.OKResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, apiserver.InternalErr(err)
	}

	unfollowed, err := ss.userStore.GetByUUID(ctx, req.Msg.Uuid)
	if err != nil {
		log.Err(err).Msg("getting-unfollowed")
		return nil, apiserver.InvalidArg(err.Error())
	}

	err = ss.userStore.RemoveFollower(ctx, unfollowed.ID, user.ID)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	err = ss.presenceStore.UpdateFollower(ctx, unfollowed, user, false)
	if err != nil {
		// we cannot rollback the unfollow
		return nil, apiserver.InvalidArg(err.Error())
	}

	return connect.NewResponse(&pb.OKResponse{}), nil
}

func (ss *SocializeService) GetFollows(ctx context.Context, req *connect.Request[pb.GetFollowsRequest],
) (*connect.Response[pb.GetFollowsResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, apiserver.InternalErr(err)
	}

	users, err := ss.userStore.GetFollows(ctx, user.ID)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	uuids := make([]string, 0, len(users))
	for _, u := range users {
		uuids = append(uuids, u.UUID)
	}
	channels, err := ss.presenceStore.BatchGetChannels(ctx, uuids)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	basicFollowedUsers := make([]*pb.BasicFollowedUser, len(users))
	for i, u := range users {
		basicFollowedUsers[i] = &pb.BasicFollowedUser{
			Uuid:     u.UUID,
			Username: u.Username,
			Channel:  channels[i],
		}
	}

	return connect.NewResponse(&pb.GetFollowsResponse{Users: basicFollowedUsers}), nil
}

// blocks
func (ss *SocializeService) AddBlock(ctx context.Context, req *connect.Request[pb.AddBlockRequest],
) (*connect.Response[pb.OKResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, apiserver.InternalErr(err)
	}

	blocked, err := ss.userStore.GetByUUID(ctx, req.Msg.Uuid)
	if err != nil {
		log.Err(err).Msg("getting-blocked")
		return nil, apiserver.InvalidArg(err.Error())
	}
	privilegedUser, err := ss.queries.HasPermission(ctx, models.HasPermissionParams{
		UserID:     int32(blocked.ID),
		Permission: string(rbac.CanModerateUsers),
	})
	if err != nil {
		return nil, err
	}
	if privilegedUser {
		log.Error().Msg("blocking-admin-or-mod")
		return nil, apiserver.InvalidArg("you cannot block that user")
	}

	err = ss.userStore.AddBlock(ctx, blocked.ID, user.ID)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.OKResponse{}), nil
}

func (ss *SocializeService) RemoveBlock(ctx context.Context, req *connect.Request[pb.RemoveBlockRequest],
) (*connect.Response[pb.OKResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, apiserver.InternalErr(err)
	}

	unblocked, err := ss.userStore.GetByUUID(ctx, req.Msg.Uuid)
	if err != nil {
		log.Err(err).Msg("getting-unblocked")
		return nil, apiserver.InvalidArg(err.Error())
	}

	err = ss.userStore.RemoveBlock(ctx, unblocked.ID, user.ID)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.OKResponse{}), nil
}

func (ss *SocializeService) GetBlocks(ctx context.Context, req *connect.Request[pb.GetBlocksRequest],
) (*connect.Response[pb.GetBlocksResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, apiserver.InternalErr(err)
	}

	users, err := ss.userStore.GetBlocks(ctx, user.ID)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	basicUsers := make([]*pb.BasicUser, len(users))
	for i, u := range users {
		basicUsers[i] = &pb.BasicUser{
			Uuid:     u.UUID,
			Username: u.Username,
		}
	}

	return connect.NewResponse(&pb.GetBlocksResponse{Users: basicUsers}), nil
}

func (ss *SocializeService) GetFullBlocks(ctx context.Context, req *connect.Request[pb.GetFullBlocksRequest],
) (*connect.Response[pb.GetFullBlocksResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	user, err := ss.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, apiserver.InternalErr(err)
	}

	blocks, err := ss.userStore.GetBlocks(ctx, user.ID)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	blockedBy, err := ss.userStore.GetBlockedBy(ctx, user.ID)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}

	seen := map[string]bool{}
	userIDs := []string{}
	blockedIDs := make([]string, len(blocks))
	for i, u := range blocks {
		blockedIDs[i] = u.UUID
		if !seen[u.UUID] {
			seen[u.UUID] = true
			userIDs = append(userIDs, u.UUID)
		}
	}
	for _, u := range blockedBy {
		if !seen[u.UUID] {
			seen[u.UUID] = true
			userIDs = append(userIDs, u.UUID)
		}
	}

	return connect.NewResponse(&pb.GetFullBlocksResponse{
		UserIds:        userIDs,
		BlockedUserIds: blockedIDs,
	}), nil
}

func (ss *SocializeService) GetActiveChatChannels(ctx context.Context, req *connect.Request[pb.GetActiveChatChannelsRequest],
) (*connect.Response[pb.ActiveChatChannels], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := ss.chatStore.LatestChannels(ctx, int(req.Msg.Number), int(req.Msg.Offset), sess.UserUUID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
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

// maxChatPage is the most messages GetChatsForChannel returns at once.
const maxChatPage = 200

func (ss *SocializeService) GetChatsForChannel(ctx context.Context, req *connect.Request[pb.GetChatsRequest],
) (*connect.Response[pb.ChatHistory], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("get-session-get-chats-for-channel")
		// Don't exit on error. We should allow unauthenticated users to get
		// chats, just not private ones.
	}
	if strings.HasPrefix(req.Msg.Channel, "chat.pm.") {
		// Verify that this chat channel is well formed and we have access to it.
		if sess == nil {
			return nil, err
		}
		_, err := ChatChannelReceiver(sess.UserUUID, req.Msg.Channel)
		if err != nil {
			return nil, err
		}
	}
	cursors := 0
	for _, c := range []string{req.Msg.BeforeId, req.Msg.AfterId, req.Msg.AroundId} {
		if c != "" {
			cursors++
		}
	}
	if cursors > 1 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("give at most one of before_id, after_id and around_id"))
	}
	if req.Msg.Limit < 0 || req.Msg.Limit > maxChatPage {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("limit must be 0 to %d", maxChatPage))
	}
	q := user.ChatQuery{
		Before: req.Msg.BeforeId,
		After:  req.Msg.AfterId,
		Around: req.Msg.AroundId,
		Limit:  int(req.Msg.Limit),
	}
	lastRead := ""
	if sess != nil {
		lastRead, err = ss.chatStore.LastRead(ctx, sess.UserUUID, req.Msg.Channel)
		if err != nil {
			return nil, err
		}
		if req.Msg.AroundLastRead && cursors == 0 {
			q.Around = lastRead
		}
	}
	page, err := ss.chatStore.History(ctx, req.Msg.Channel, q)
	if err != nil {
		return nil, err
	}
	chats := page.Messages
	// Clients that don't page (no cursor, not around the read position)
	// fetch a private chat when the user opens it and never call
	// MarkChannelRead; for them, fetching it marks it read.
	if sess != nil && cursors == 0 && !req.Msg.AroundLastRead && len(chats) > 0 &&
		strings.HasPrefix(req.Msg.Channel, "chat.pm.") {
		newest := chats[len(chats)-1].Id
		if err := ss.chatStore.MarkRead(ctx, sess.UserUUID, req.Msg.Channel, newest); err != nil {
			log.Err(err).Str("channel", req.Msg.Channel).Msg("mark-read-on-fetch")
		} else {
			lastRead = newest
		}
	}
	if len(chats) > 0 {
		chatterUuids := make([]string, 0, len(chats))
		for _, chatMessage := range chats {
			chatterUuids = append(chatterUuids, chatMessage.UserId)
		}
		sort.Strings(chatterUuids)
		w := 1
		for r := 1; r < len(chatterUuids); r++ {
			if chatterUuids[r] != chatterUuids[r-1] {
				chatterUuids[w] = chatterUuids[r]
				w++
			}
		}
		chatterUuids = chatterUuids[:w]
		chatterBriefProfiles, err := ss.userStore.GetBriefProfiles(ctx, chatterUuids)
		if err != nil {
			return nil, err
		}
		for _, chatMessage := range chats {
			if chatterBriefProfile, ok := chatterBriefProfiles[chatMessage.UserId]; ok {
				chatMessage.CountryCode = chatterBriefProfile.CountryCode
				chatMessage.AvatarUrl = chatterBriefProfile.AvatarUrl
			}
		}
	}
	return connect.NewResponse(&pb.ChatHistory{
		Messages:   chats,
		HasOlder:   page.HasOlder,
		HasNewer:   page.HasNewer,
		LastReadId: lastRead,
	}), nil
}

func (ss *SocializeService) MarkChannelRead(ctx context.Context, req *connect.Request[pb.MarkChannelReadRequest],
) (*connect.Response[pb.OKResponse], error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.Channel == "" || req.Msg.MessageId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("channel and message_id are required"))
	}
	if strings.HasPrefix(req.Msg.Channel, "chat.pm.") {
		if _, err := ChatChannelReceiver(sess.UserUUID, req.Msg.Channel); err != nil {
			return nil, err
		}
	}
	if err := ss.chatStore.MarkRead(ctx, sess.UserUUID, req.Msg.Channel, req.Msg.MessageId); err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.OKResponse{}), nil
}
