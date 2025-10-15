package config

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/stores/common"
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

	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.AdminAllAccess)
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
	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.AdminAllAccess)
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
	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.CanModifyAnnouncements)
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
	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.CanModifyAnnouncements)
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

	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.AdminAllAccess)
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

func (cs *ConfigService) AddBadge(ctx context.Context, req *connect.Request[pb.AddBadgeRequest]) (
	*connect.Response[pb.ConfigResponse], error) {

	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.AdminAllAccess)
	if err != nil {
		return nil, err
	}
	err = cs.q.AddBadge(ctx, models.AddBadgeParams{
		Code:        req.Msg.Code,
		Description: req.Msg.Description,
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.ConfigResponse{}), nil
}

func (cs *ConfigService) AssignBadge(ctx context.Context, req *connect.Request[pb.AssignBadgeRequest]) (
	*connect.Response[pb.ConfigResponse], error) {

	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.CanManageBadges)
	if err != nil {
		return nil, err
	}
	err = cs.q.AddUserBadge(ctx, models.AddUserBadgeParams{
		Code:     req.Msg.Code,
		Username: req.Msg.Username,
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.ConfigResponse{}), nil
}

func (cs *ConfigService) UnassignBadge(ctx context.Context, req *connect.Request[pb.AssignBadgeRequest]) (
	*connect.Response[pb.ConfigResponse], error) {

	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.CanManageBadges)
	if err != nil {
		return nil, err
	}
	err = cs.q.RemoveUserBadge(ctx, models.RemoveUserBadgeParams{
		Code:     req.Msg.Code,
		Username: req.Msg.Username,
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.ConfigResponse{}), nil
}

func (cs *ConfigService) GetUsersForBadge(ctx context.Context, req *connect.Request[pb.GetUsersForBadgeRequest]) (
	*connect.Response[pb.Usernames], error) {

	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.CanManageBadges)
	if err != nil {
		return nil, err
	}
	users, err := cs.q.GetUsersForBadge(ctx, req.Msg.Code)
	if err != nil {
		return nil, err
	}
	usernames := &pb.Usernames{Usernames: make([]string, len(users))}
	for i := range users {
		usernames.Usernames[i] = users[i].String
	}

	return connect.NewResponse(usernames), nil
}

func (cs *ConfigService) GetUserDetails(ctx context.Context, req *connect.Request[pb.GetUserDetailsRequest]) (
	*connect.Response[pb.UserDetailsResponse], error) {

	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.CanSeePrivateUserData)
	if err != nil {
		return nil, err
	}
	if len(req.Msg.Username) == 0 {
		return nil, apiserver.InvalidArg("need to specify a username")
	}
	deetz, err := cs.q.GetUserDetails(ctx, common.ToPGTypeText(strings.ToLower(req.Msg.Username)))
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.UserDetailsResponse{
		Uuid:      deetz.Uuid.String,
		Email:     deetz.Email.String,
		Created:   timestamppb.New(deetz.CreatedAt.Time),
		BirthDate: deetz.BirthDate.String,
		Username:  deetz.Username.String,
	}), nil
}

func (cs *ConfigService) SearchEmail(ctx context.Context, req *connect.Request[pb.SearchEmailRequest]) (
	*connect.Response[pb.SearchEmailResponse], error) {

	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.CanSeePrivateUserData)
	if err != nil {
		return nil, err
	}
	if len(req.Msg.PartialEmail) < 2 {
		return nil, apiserver.InvalidArg("need to specify a partial email address to match")
	}
	emailStr := fmt.Sprintf("%%%s%%", strings.ToLower(req.Msg.PartialEmail))

	matches, err := cs.q.GetMatchingEmails(ctx, common.ToPGTypeText(emailStr))
	if err != nil {
		return nil, err
	}
	matchesPb := make([]*pb.UserDetailsResponse, len(matches))

	for i := range matches {
		matchesPb[i] = &pb.UserDetailsResponse{
			Uuid:      matches[i].Uuid.String,
			Email:     matches[i].Email.String,
			Created:   timestamppb.New(matches[i].CreatedAt.Time),
			BirthDate: matches[i].BirthDate.String,
			Username:  matches[i].Username.String,
		}
	}

	return connect.NewResponse(&pb.SearchEmailResponse{
		Users: matchesPb,
	}), nil
}

func (cs *ConfigService) GetCorrespondenceGameCount(ctx context.Context, req *connect.Request[pb.GetCorrespondenceGameCountRequest]) (
	*connect.Response[pb.CorrespondenceGameCountResponse], error) {

	_, err := apiserver.AuthenticateWithPermission(ctx, cs.userStore, cs.q, rbac.AdminAllAccess)
	if err != nil {
		return nil, err
	}

	count, err := cs.q.CountActiveCorrespondenceGames(ctx)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&pb.CorrespondenceGameCountResponse{
		Count: count,
	}), nil
}
