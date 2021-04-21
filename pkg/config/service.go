package config

import (
	"context"
	"errors"

	"github.com/twitchtv/twirp"

	"github.com/domino14/liwords/pkg/apiserver"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	pb "github.com/domino14/liwords/rpc/api/proto/config_service"
	"github.com/rs/zerolog/log"
)

var (
	errRequiresAdmin = errors.New("this api endpoint requires an administrator")
)

type ConfigStore interface {
	SetGamesEnabled(context.Context, bool) error
	GamesEnabled(context.Context) (bool, error)

	SetFEHash(context.Context, string) error
	FEHash(context.Context) (string, error)

	SetAnnouncements(context.Context, []*pb.Announcement) error
	GetAnnouncements(context.Context) ([]*pb.Announcement, error)
}

type ConfigService struct {
	store     ConfigStore
	userStore user.Store
}

func NewConfigService(cs ConfigStore, userStore user.Store) *ConfigService {
	return &ConfigService{store: cs, userStore: userStore}
}

func (cs *ConfigService) SetGamesEnabled(ctx context.Context, req *pb.EnableGamesRequest) (*pb.ConfigResponse, error) {

	allowed, err := isAdmin(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, twirp.NewError(twirp.Unauthenticated, errRequiresAdmin.Error())
	}

	err = cs.store.SetGamesEnabled(ctx, req.Enabled)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.ConfigResponse{}, nil
}

func (cs *ConfigService) SetFEHash(ctx context.Context, req *pb.SetFEHashRequest) (*pb.ConfigResponse, error) {
	allowed, err := isAdmin(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, twirp.NewError(twirp.Unauthenticated, errRequiresAdmin.Error())
	}

	err = cs.store.SetFEHash(ctx, req.Hash)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.ConfigResponse{}, nil
}

func isAdmin(ctx context.Context, us user.Store) (bool, error) {
	var user *entity.User
	var accessMethod string
	// Look for an API key first:
	apikey, err := apiserver.GetAPIKey(ctx)
	if err != nil {
		// Look for a session
		sess, err := apiserver.GetSession(ctx)
		if err != nil {
			return false, err
		}
		user, err = us.Get(ctx, sess.Username)
		if err != nil {
			log.Err(err).Msg("getting-user-by-session")
			return false, twirp.InternalErrorWith(err)
		}
		accessMethod = "session"

	} else {
		user, err = us.GetByAPIKey(ctx, apikey)
		if err != nil {
			log.Err(err).Msg("getting-user-by-apikey")
			return false, twirp.NewError(twirp.Unauthenticated, err.Error())
		}
		accessMethod = "apikey"
	}
	if !user.IsAdmin {
		return false, nil
	}
	log.Info().Str("access-method", accessMethod).Str("username", user.Username).
		Msg("admin-call")
	return true, nil
}

func (cs *ConfigService) SetUserPermissions(ctx context.Context, req *pb.PermissionsRequest) (*pb.ConfigResponse, error) {

	allowed, err := isAdmin(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, twirp.NewError(twirp.Unauthenticated, errRequiresAdmin.Error())
	}

	err = cs.userStore.SetPermissions(ctx, req)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	return &pb.ConfigResponse{}, nil

}

func (cs *ConfigService) GetUserDetails(ctx context.Context, req *pb.UserRequest) (*pb.UserResponse, error) {
	allowed, err := isAdmin(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, twirp.NewError(twirp.Unauthenticated, errRequiresAdmin.Error())
	}

	u, err := cs.userStore.Get(ctx, req.Username)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	return &pb.UserResponse{
		Username:   u.Username,
		Uuid:       u.UUID,
		Email:      u.Email,
		IsBot:      u.IsBot,
		IsDirector: u.IsDirector,
		IsMod:      u.IsMod,
		IsAdmin:    u.IsAdmin,
	}, nil
}

func (cs *ConfigService) SetAnnouncements(ctx context.Context, req *pb.SetAnnouncementsRequest) (*pb.ConfigResponse, error) {
	allowed, err := isAdmin(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, twirp.NewError(twirp.Unauthenticated, errRequiresAdmin.Error())
	}
	err = cs.store.SetAnnouncements(ctx, req.Announcements)
	if err != nil {
		return nil, err
	}
	return &pb.ConfigResponse{}, nil
}

func (cs *ConfigService) GetAnnouncements(ctx context.Context, req *pb.GetAnnouncementsRequest) (*pb.AnnouncementsResponse, error) {
	announcements, err := cs.store.GetAnnouncements(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.AnnouncementsResponse{Announcements: announcements}, nil
}
