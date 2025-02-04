package config

import (
	"context"

	"connectrpc.com/connect"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/config_service"
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
	q         *models.Queries
}

func NewConfigService(cs ConfigStore, userStore user.Store, q *models.Queries) *ConfigService {
	return &ConfigService{store: cs, userStore: userStore, q: q}
}

func (cs *ConfigService) SetGamesEnabled(ctx context.Context, req *connect.Request[pb.EnableGamesRequest],
) (*connect.Response[pb.ConfigResponse], error) {

	err := apiserver.AuthenticateAdmin(ctx, cs.userStore, cs.q)
	if err != nil {
		return nil, err
	}

	err = cs.store.SetGamesEnabled(ctx, req.Msg.Enabled)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.ConfigResponse{}), nil
}

func (cs *ConfigService) SetFEHash(ctx context.Context, req *connect.Request[pb.SetFEHashRequest],
) (*connect.Response[pb.ConfigResponse], error) {
	err := apiserver.AuthenticateAdmin(ctx, cs.userStore, cs.q)
	if err != nil {
		return nil, err
	}

	err = cs.store.SetFEHash(ctx, req.Msg.Hash)
	if err != nil {
		return nil, apiserver.InternalErr(err)
	}
	return connect.NewResponse(&pb.ConfigResponse{}), nil
}

func (cs *ConfigService) SetAnnouncements(ctx context.Context, req *connect.Request[pb.SetAnnouncementsRequest],
) (*connect.Response[pb.ConfigResponse], error) {
	err := apiserver.AuthenticateAdmin(ctx, cs.userStore, cs.q)
	if err != nil {
		return nil, err
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
	err := apiserver.AuthenticateAdmin(ctx, cs.userStore, cs.q)
	if err != nil {
		return nil, err
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

func (cs *ConfigService) SetGlobalIntegration(ctx context.Context, req *connect.Request[pb.SetGlobalIntegrationRequest]) (
	*connect.Response[pb.ConfigResponse], error) {

	err := apiserver.AuthenticateAdmin(ctx, cs.userStore, cs.q)
	if err != nil {
		return nil, err
	}

	err = cs.q.AddOrUpdateGlobalIntegration(ctx, models.AddOrUpdateGlobalIntegrationParams{
		IntegrationName: req.Msg.IntegrationName,
		Data:            []byte(req.Msg.JsonData),
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.ConfigResponse{}), nil
}
