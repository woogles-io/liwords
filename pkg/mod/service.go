package mod

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"

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

func (ms *ModService) GetNotorietyReport(ctx context.Context, req *pb.GetNotorietyReportRequest) (*pb.NotorietyReport, error) {
	user, err := sessionUser(ctx, ms)
	if err != nil {
		return nil, err
	}
	if !(user.IsAdmin || user.IsMod) {
		return nil, twirp.NewError(twirp.Unauthenticated, errNotAuthorized.Error())
	}
	// Default to only getting 50 notorious games, which is probably much more than
	// needed anyway.
	score, games, err := GetNotorietyReport(ctx, ms.userStore, ms.notorietyStore, req.UserId, 50)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.NotorietyReport{Score: int32(score), Games: games}, nil
}

func (ms *ModService) ResetNotoriety(ctx context.Context, req *pb.ResetNotorietyRequest) (*pb.ResetNotorietyResponse, error) {
	user, err := sessionUser(ctx, ms)
	if err != nil {
		return nil, err
	}
	if !(user.IsAdmin || user.IsMod) {
		return nil, twirp.NewError(twirp.Unauthenticated, errNotAuthorized.Error())
	}
	err = ResetNotoriety(ctx, ms.userStore, ms.notorietyStore, req.UserId)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.ResetNotorietyResponse{}, nil
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
	removerUserId, err := authenticateMod(ctx, ms, req)
	if err != nil {
		return nil, err
	}
	err = RemoveActions(ctx, ms.userStore, removerUserId, req.Actions)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.ModActionResponse{}, nil
}

func (ms *ModService) ApplyActions(ctx context.Context, req *pb.ModActionsList) (*pb.ModActionResponse, error) {
	applierUserId, err := authenticateMod(ctx, ms, req)
	if err != nil {
		return nil, err
	}
	err = ApplyActions(ctx, ms.userStore, ms.chatStore, applierUserId, req.Actions)
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
		return "", twirp.NewError(twirp.Unauthenticated, errNotAuthorized.Error())
	}
	return user.UUID, nil
}
