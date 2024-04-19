package mod

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"

	pb "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
)

var (
	errNotAuthorized = errors.New("this user is not authorized to perform this action")
)

type ctxkey string

const rtchankey ctxkey = "realtimechan"

type ModService struct {
	userStore      user.Store
	notorietyStore NotorietyStore
	chatStore      user.ChatStore
	mailgunKey     string
	discordToken   string
}

func NewModService(us user.Store, cs user.ChatStore) *ModService {
	return &ModService{userStore: us, chatStore: cs}
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
	pb.ModActionType_DELETE_ACCOUNT:          true,
}

func (ms *ModService) GetNotorietyReport(ctx context.Context, req *connect.Request[pb.GetNotorietyReportRequest],
) (*connect.Response[pb.NotorietyReport], error) {
	user, err := sessionUser(ctx, ms)
	if err != nil {
		return nil, err
	}
	if !(user.IsAdmin || user.IsMod) {
		return nil, apiserver.Unauthenticated(errNotAuthorized.Error())
	}
	// Default to only getting 50 notorious games, which is probably much more than
	// needed anyway.
	score, games, err := GetNotorietyReport(ctx, ms.userStore, ms.notorietyStore, req.Msg.UserId, 50)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.NotorietyReport{Score: int32(score), Games: games}), nil
}

func (ms *ModService) ResetNotoriety(ctx context.Context, req *connect.Request[pb.ResetNotorietyRequest],
) (*connect.Response[pb.ResetNotorietyResponse], error) {
	user, err := sessionUser(ctx, ms)
	if err != nil {
		return nil, err
	}
	if !(user.IsAdmin || user.IsMod) {
		return nil, apiserver.Unauthenticated(errNotAuthorized.Error())
	}
	err = ResetNotoriety(ctx, ms.userStore, ms.notorietyStore, req.Msg.UserId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.ResetNotorietyResponse{}), nil
}

func (ms *ModService) GetActions(ctx context.Context, req *connect.Request[pb.GetActionsRequest],
) (*connect.Response[pb.ModActionsMap], error) {
	user, err := sessionUser(ctx, ms)
	if err != nil {
		return nil, err
	}
	if !(user.IsAdmin || user.IsMod) {
		return nil, apiserver.Unauthenticated(errNotAuthorized.Error())
	}
	actions, err := GetActions(ctx, ms.userStore, req.Msg.UserId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.ModActionsMap{Actions: actions}), nil
}

func (ms *ModService) GetActionHistory(ctx context.Context, req *connect.Request[pb.GetActionsRequest],
) (*connect.Response[pb.ModActionsList], error) {
	user, err := sessionUser(ctx, ms)
	if err != nil {
		return nil, err
	}
	if !(user.IsAdmin || user.IsMod) {
		return nil, apiserver.Unauthenticated(errNotAuthorized.Error())
	}
	history, err := GetActionHistory(ctx, ms.userStore, req.Msg.UserId)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.ModActionsList{Actions: history}), nil
}

func (ms *ModService) RemoveActions(ctx context.Context, req *connect.Request[pb.ModActionsList],
) (*connect.Response[pb.ModActionResponse], error) {
	removerUserId, err := authenticateMod(ctx, ms, req.Msg)
	if err != nil {
		return nil, err
	}
	err = RemoveActions(ctx, ms.userStore, removerUserId, req.Msg.Actions)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.ModActionResponse{}), nil
}

func (ms *ModService) ApplyActions(ctx context.Context, req *connect.Request[pb.ModActionsList],
) (*connect.Response[pb.ModActionResponse], error) {
	applierUserId, err := authenticateMod(ctx, ms, req.Msg)
	if err != nil {
		return nil, err
	}
	err = ApplyActions(ctx, ms.userStore, ms.chatStore, applierUserId, req.Msg.Actions)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.ModActionResponse{}), nil
}

func sessionUser(ctx context.Context, ms *ModService) (*entity.User, error) {
	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := ms.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, apiserver.InternalErr(err)
	}
	return user, nil
}

func authenticateMod(ctx context.Context, ms *ModService, req *pb.ModActionsList) (string, error) {
	user, err := sessionUser(ctx, ms)
	if err != nil {
		return "", err
	}

	isAdminRequired := false
	for _, action := range req.Actions {
		if AdminRequiredMap[action.Type] {
			isAdminRequired = true
			break
		}
	}

	if !user.IsAdmin && (isAdminRequired || !user.IsMod) {
		return "", apiserver.Unauthenticated(errNotAuthorized.Error())
	}
	return user.UUID, nil
}
