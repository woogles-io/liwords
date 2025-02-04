package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/auth/rbac"
	"github.com/woogles-io/liwords/pkg/entitlements"
	"github.com/woogles-io/liwords/pkg/integrations"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

type AuthorizationService struct {
	userStore user.Store
	q         *models.Queries
}

func NewAuthorizationService(u user.Store, q *models.Queries) *AuthorizationService {
	return &AuthorizationService{u, q}
}

func (as *AuthorizationService) GetModList(ctx context.Context, r *connect.Request[pb.GetModListRequest]) (
	*connect.Response[pb.GetModListResponse], error) {

	// This endpoint should work without login.

	users, err := as.q.GetUsersWithRoles(ctx,
		[]string{string(rbac.Admin), string(rbac.Moderator)})
	if err != nil {
		return nil, err
	}

	resp := &pb.GetModListResponse{}
	for _, u := range users {
		if u.RoleName == string(rbac.Admin) {
			resp.AdminUserIds = append(resp.AdminUserIds, u.Uuid.String)
		}
		if u.RoleName == string(rbac.Moderator) {
			resp.ModUserIds = append(resp.ModUserIds, u.Uuid.String)
		}
	}
	return connect.NewResponse(resp), nil
}

func (as *AuthorizationService) GetSubscriptionCriteria(ctx context.Context, r *connect.Request[pb.GetSubscriptionCriteriaRequest]) (
	*connect.Response[pb.GetSubscriptionCriteriaResponse], error) {

	user, err := apiserver.AuthUser(ctx, as.userStore)
	if err != nil {
		return nil, err
	}
	tierData, err := integrations.DetermineUserTier(ctx, user.UUID, as.q)
	if err != nil {
		return nil, err
	}
	tierName := ""
	entitled := false
	lastChargeDate := timestamppb.New(time.Time{})
	if tierData != nil {
		tierName = tierData.TierName
		entitled, err = entitlements.EntitledToBestBot(ctx, as.q, tierData, user.ID, time.Now())
		if err != nil {
			return nil, err
		}
		lastChargeDate = timestamppb.New(tierData.LastChargeDate)
	}
	return connect.NewResponse(&pb.GetSubscriptionCriteriaResponse{
		TierName:           tierName,
		EntitledToBotGames: entitled,
		LastChargeDate:     lastChargeDate,
	}), nil
}

func (as *AuthorizationService) AddRole(ctx context.Context, r *connect.Request[pb.AddRoleRequest]) (
	*connect.Response[pb.AddRoleResponse], error) {

	err := apiserver.AuthenticateAdmin(ctx, as.userStore, as.q)
	if err != nil {
		return nil, err
	}

	err = as.q.AddRole(ctx, models.AddRoleParams{
		Name:        r.Msg.Name,
		Description: r.Msg.Description,
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.AddRoleResponse{}), nil
}

func (as *AuthorizationService) AddPermission(ctx context.Context, r *connect.Request[pb.AddPermissionRequest]) (
	*connect.Response[pb.AddPermissionResponse], error) {

	err := apiserver.AuthenticateAdmin(ctx, as.userStore, as.q)
	if err != nil {
		return nil, err
	}

	err = as.q.AddPermission(ctx, models.AddPermissionParams{
		Code:        r.Msg.Code,
		Description: r.Msg.Description,
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.AddPermissionResponse{}), nil
}

func (as *AuthorizationService) LinkRoleAndPermission(ctx context.Context, r *connect.Request[pb.LinkRoleAndPermissionRequest]) (
	*connect.Response[pb.LinkRoleAndPermissionResponse], error) {

	err := apiserver.AuthenticateAdmin(ctx, as.userStore, as.q)
	if err != nil {
		return nil, err
	}

	err = as.q.LinkRoleAndPermission(ctx, models.LinkRoleAndPermissionParams{
		RoleName:       r.Msg.RoleName,
		PermissionCode: r.Msg.PermissionCode,
	})
	if err != nil {
		if IsForeignKeyViolation(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role or permission not found"))
		}
		return nil, err
	}
	return connect.NewResponse(&pb.LinkRoleAndPermissionResponse{}), nil
}

func (as *AuthorizationService) AssignRole(ctx context.Context, r *connect.Request[pb.AssignRoleRequest]) (
	*connect.Response[pb.AssignRoleResponse], error) {

	err := apiserver.AuthenticateAdmin(ctx, as.userStore, as.q)
	if err != nil {
		return nil, err
	}

	err = as.q.AssignRole(ctx, models.AssignRoleParams{
		Username: pgtype.Text{Valid: true, String: r.Msg.Username},
		RoleName: r.Msg.RoleName,
	})
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("role already assigned to user"))
		}
		if IsForeignKeyViolation(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role or user not found"))
		}
		return nil, err
	}
	return connect.NewResponse(&pb.AssignRoleResponse{}), nil
}

func (as *AuthorizationService) UnassignRole(ctx context.Context, r *connect.Request[pb.UnassignRoleRequest]) (
	*connect.Response[pb.UnassignRoleResponse], error) {

	err := apiserver.AuthenticateAdmin(ctx, as.userStore, as.q)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := as.q.UnassignRole(ctx, models.UnassignRoleParams{
		Username: pgtype.Text{Valid: true, String: r.Msg.Username},
		RoleName: r.Msg.RoleName,
	})
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		return nil, apiserver.NotFound("role assignment not found")
	}

	return connect.NewResponse(&pb.UnassignRoleResponse{}), nil
}

func (as *AuthorizationService) GetUserRoles(ctx context.Context, r *connect.Request[pb.GetUserRolesRequest]) (
	*connect.Response[pb.UserRolesResponse], error) {

	err := apiserver.AuthenticateAdmin(ctx, as.userStore, as.q)
	if err != nil {
		return nil, err
	}
	roles, err := rbac.UserRoles(ctx, as.q, r.Msg.Username)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.UserRolesResponse{
		Roles: roles,
	}), nil
}

// Helper functions to detect specific PostgreSQL errors
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	return false
}

func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return true
	}
	return false
}
