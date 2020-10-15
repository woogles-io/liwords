package config

import (
	"context"
	"errors"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/config_service"
	"github.com/rs/zerolog/log"
)

type ConfigStore interface {
	SetGamesEnabled(context.Context, bool) error
	GamesEnabled(context.Context) (bool, error)
}

type ConfigService struct {
	store     ConfigStore
	userStore user.Store
}

func NewConfigService(cs ConfigStore, userStore user.Store) *ConfigService {
	return &ConfigService{store: cs, userStore: userStore}
}

func (cs *ConfigService) SetGamesEnabled(ctx context.Context, req *pb.EnableGamesRequest) (*pb.ConfigResponse, error) {

	sess, err := apiserver.GetSession(ctx)
	if err != nil {
		return nil, err
	}

	user, err := cs.userStore.Get(ctx, sess.Username)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, err
	}

	if !user.IsAdmin {
		return nil, errors.New("this api endpoint requires an administrator")
	}

	err = cs.store.SetGamesEnabled(ctx, req.Enabled)
	if err != nil {
		return nil, err
	}
	return &pb.ConfigResponse{}, nil
}
