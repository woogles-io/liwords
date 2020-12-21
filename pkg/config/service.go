package config

import (
	"context"

	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/config_service"
	"github.com/rs/zerolog/log"
)

type ConfigStore interface {
	SetGamesEnabled(context.Context, bool) error
	GamesEnabled(context.Context) (bool, error)

	SetFEHash(context.Context, string) error
	FEHash(context.Context) (string, error)
}

type ConfigService struct {
	store     ConfigStore
	userStore user.Store
}

func NewConfigService(cs ConfigStore, userStore user.Store) *ConfigService {
	return &ConfigService{store: cs, userStore: userStore}
}

func (cs *ConfigService) SetGamesEnabled(ctx context.Context, req *pb.EnableGamesRequest) (*pb.ConfigResponse, error) {

	apikey, err := apiserver.GetAPIKey(ctx)
	if err != nil {
		return nil, err
	}

	user, err := cs.userStore.GetByAPIKey(ctx, apikey)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, twirp.NewError(twirp.Unauthenticated, err.Error())
	}

	if !user.IsAdmin {
		return nil, twirp.NewError(twirp.Unauthenticated, "this api endpoint requires an administrator")
	}

	err = cs.store.SetGamesEnabled(ctx, req.Enabled)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.ConfigResponse{}, nil
}

func (cs *ConfigService) SetFEHash(ctx context.Context, req *pb.SetFEHashRequest) (*pb.ConfigResponse, error) {
	apikey, err := apiserver.GetAPIKey(ctx)
	if err != nil {
		return nil, err
	}

	user, err := cs.userStore.GetByAPIKey(ctx, apikey)
	if err != nil {
		log.Err(err).Msg("getting-user")
		return nil, twirp.NewError(twirp.Unauthenticated, err.Error())
	}

	if !user.IsAdmin {
		return nil, twirp.NewError(twirp.Unauthenticated, "this api endpoint requires an administrator")
	}

	err = cs.store.SetFEHash(ctx, req.Hash)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.ConfigResponse{}, nil
}
