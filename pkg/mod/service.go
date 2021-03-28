package mod

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"

	pb "github.com/domino14/liwords/rpc/api/proto/mod_service"
)

var (
	errNotAuthorized = errors.New("this user is not authorized to perform this action")
)

type ctxkey string

const rtchankey ctxkey = "realtimechan"

type ModService struct {
	userStore    user.Store
	chatStore    user.ChatStore
	mailgunKey   string
	discordToken string
}

func NewModService(us user.Store, cs user.ChatStore, mailgunKey string, discordToken string) *ModService {
	return &ModService{userStore: us, chatStore: cs, mailgunKey: mailgunKey, discordToken: discordToken}
}

var AdminRequiredMap = map[pb.ModActionType]bool{
	pb.ModActionType_MUTE:                    false,
	pb.ModActionType_SUSPEND_ACCOUNT:         false,
	pb.ModActionType_SUSPEND_RATED_GAMES:     false,
	pb.ModActionType_SUSPEND_GAMES:           false,
	pb.ModActionType_RESET_RATINGS:           true,
	pb.ModActionType_RESET_STATS:             true,
	pb.ModActionType_RESET_STATS_AND_RATINGS: true,
	pb.ModActionType_REMOVE_CHAT:             false,
}

func (ms *ModService) GetActions(ctx context.Context, req *pb.GetActionsRequest) (*pb.ModActionsMap, error) {
	user, err := sessionUser(ctx, ms)
	if err != nil {
		return nil, err
	}
	if !(user.IsAdmin || user.IsMod) {
		return nil, twirp.NewError(twirp.Unauthenticated, errNotAuthorized.Error())
	}
	actions, err := GetActions(ctx, ms.userStore, req.UserId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.ModActionsMap{Actions: actions}, nil
}

func (ms *ModService) GetActionHistory(ctx context.Context, req *pb.GetActionsRequest) (*pb.ModActionsList, error) {
	user, err := sessionUser(ctx, ms)
	if err != nil {
		return nil, err
	}
	if !(user.IsAdmin || user.IsMod) {
		return nil, twirp.NewError(twirp.Unauthenticated, errNotAuthorized.Error())
	}
	history, err := GetActionHistory(ctx, ms.userStore, req.UserId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.ModActionsList{Actions: history}, nil
}

func (ms *ModService) RemoveActions(ctx context.Context, req *pb.ModActionsList) (*pb.ModActionResponse, error) {
	err := authenticateMod(ctx, ms, req)
	if err != nil {
		return nil, err
	}
	err = RemoveActions(ctx, ms.userStore, req.Actions)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.ModActionResponse{}, nil
}

func (ms *ModService) ApplyActions(ctx context.Context, req *pb.ModActionsList) (*pb.ModActionResponse, error) {
	err := authenticateMod(ctx, ms, req)
	if err != nil {
		return nil, err
	}
	err = ApplyActions(ctx, ms.userStore, ms.chatStore, ms.mailgunKey, ms.discordToken, req.Actions)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.ModActionResponse{}, nil
}

func sessionUser(ctx context.Context, ms *ModService) (*entity.User, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ms.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, twirp.InternalErrorWith(err)
	}
	return user, nil
}

func authenticateMod(ctx context.Context, ms *ModService, req *pb.ModActionsList) error {
	user, err := sessionUser(ctx, ms)
	if err != nil {
		return err
	}

	isAdminRequired := false
	for _, action := range req.Actions {
		if AdminRequiredMap[action.Type] {
			isAdminRequired = true
			break
		}
	}

	if !user.IsAdmin && (isAdminRequired || !user.IsMod) {
		return twirp.NewError(twirp.Unauthenticated, errNotAuthorized.Error())
	}
	return nil
}
