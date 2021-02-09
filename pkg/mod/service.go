package mod

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"

	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
)

type ModService struct {
	userStore       user.Store
}

var AdminRequiredMap = map[ms.ModActionType]bool {
	ms.ModActionType_MUTE: false,
	ms.ModActionType_SUSPEND_ACCOUNT: true,
	ms.ModActionType_SUSPEND_RATED_GAMES: true,
	ms.ModActionType_SUSPEND_GAMES: true,
	ms.ModActionType_RESET_RATINGS: true,
	ms.ModActionType_RESET_STATS: true
	ms.ModActionType_RESET_STATS_AND_RATINGS: true
}

func (ms *ModService) ApplyActions(ctx context.Context, req *ms.ModActions) (*ms.ModActionResponse, error) {
	err := authenticateMod(ctx, req.Id, req)
	if err != nil {
		return nil, err
	}
	err = ApplyActions(ctx, ms.userStore, req.Actions)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.ModActionResponse{}, nil
}

func sessionUser(ctx context.Context, ms *ModService) (error) {
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

func modOrAdmin(ctx context.Context, id string, req *ms.ModActions) (*entity.User, error) {

	user, err := sessionUser(ctx, ts)
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
		return twirp.NewError(twirp.Unauthenticated, "this user is not an authorized to perform this action")
	}
	return nil
}