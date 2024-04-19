package config

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/config_service"
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
	SetAnnouncement(context.Context, string, *pb.Announcement) error
}

type ConfigService struct {
	store     ConfigStore
	userStore user.Store
}

func NewConfigService(cs ConfigStore, userStore user.Store) *ConfigService {
	return &ConfigService{store: cs, userStore: userStore}
}

func (cs *ConfigService) SetGamesEnabled(ctx context.Context, req *connect.Request[pb.EnableGamesRequest],
) (*connect.Response[pb.ConfigResponse], error) {

	allowed, err := isAdmin(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apiserver.Unauthenticated(errRequiresAdmin.Error())
	}

	err = cs.store.SetGamesEnabled(ctx, req.Msg.Enabled)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.ConfigResponse{}), nil
}

func (cs *ConfigService) SetFEHash(ctx context.Context, req *connect.Request[pb.SetFEHashRequest],
) (*connect.Response[pb.ConfigResponse], error) {
	allowed, err := isAdmin(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apiserver.Unauthenticated(errRequiresAdmin.Error())
	}

	err = cs.store.SetFEHash(ctx, req.Msg.Hash)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.ConfigResponse{}), nil
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
			return false, apiserver.InternalErr(err)
		}
		accessMethod = "session"

	} else {
		user, err = us.GetByAPIKey(ctx, apikey)
		if err != nil {
			log.Err(err).Msg("getting-user-by-apikey")
			return false, apiserver.Unauthenticated(err.Error())
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

func (cs *ConfigService) SetUserPermissions(ctx context.Context, req *connect.Request[pb.PermissionsRequest],
) (*connect.Response[pb.ConfigResponse], error) {

	allowed, err := isAdmin(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apiserver.Unauthenticated(errRequiresAdmin.Error())
	}

	err = cs.userStore.SetPermissions(ctx, req.Msg)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}
	return connect.NewResponse(&pb.ConfigResponse{}), nil

}

func (cs *ConfigService) GetUserDetails(ctx context.Context, req *connect.Request[pb.UserRequest],
) (*connect.Response[pb.UserResponse], error) {
	allowed, err := isAdmin(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apiserver.Unauthenticated(errRequiresAdmin.Error())
	}

	u, err := cs.userStore.Get(ctx, req.Msg.Username)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	return connect.NewResponse(&pb.UserResponse{
		Username:   u.Username,
		Uuid:       u.UUID,
		Email:      u.Email,
		IsBot:      u.IsBot,
		IsDirector: u.IsDirector,
		IsMod:      u.IsMod,
		IsAdmin:    u.IsAdmin,
	}), nil
}

func (cs *ConfigService) SetAnnouncements(ctx context.Context, req *connect.Request[pb.SetAnnouncementsRequest],
) (*connect.Response[pb.ConfigResponse], error) {
	allowed, err := isAdmin(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apiserver.Unauthenticated(errRequiresAdmin.Error())
	}
	err = cs.store.SetAnnouncements(ctx, req.Msg.Announcements)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.ConfigResponse{}), nil
}

func (cs *ConfigService) GetAnnouncements(ctx context.Context, req *connect.Request[pb.GetAnnouncementsRequest],
) (*connect.Response[pb.AnnouncementsResponse], error) {
	announcements, err := cs.store.GetAnnouncements(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.AnnouncementsResponse{Announcements: announcements}), nil
}

func (cs *ConfigService) SetSingleAnnouncement(ctx context.Context, req *connect.Request[pb.SetSingleAnnouncementRequest],
) (*connect.Response[pb.ConfigResponse], error) {
	allowed, err := isAdmin(ctx, cs.userStore)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apiserver.Unauthenticated(errRequiresAdmin.Error())
	}
	if req.Msg.LinkSearchString == "" {
		return nil, apiserver.InvalidArg("need a link search string")
	}
	err = cs.store.SetAnnouncement(ctx, req.Msg.LinkSearchString, req.Msg.Announcement)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.ConfigResponse{}), nil
}
