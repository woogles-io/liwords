package rbac

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/samber/lo"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// These Permissions should be defined in the database. See the rbac.up.sql file.
type Permission string

const (
	AdminAllAccess         Permission = "admin_all_access"
	CanCreateTournaments              = "can_create_tournaments"
	CanPlayEliteBot                   = "can_bypass_elitebot_paywall"
	CanModerateUsers                  = "can_moderate_users"
	CanModifyAnnouncements            = "can_modify_announcements"
)

// These Roles are also defined in the database.
type Role string

const (
	Admin               Role = "Admin"
	Moderator                = "Moderator"
	TournamentCreator        = "Tournament Creator"
	SpecialAccessPlayer      = "Special Access Player"
)

func HasPermission(ctx context.Context, q *models.Queries, userID uint, permission Permission) (bool, error) {
	hp, err := q.HasPermission(ctx, models.HasPermissionParams{
		UserID:     int32(userID),
		Permission: string(permission),
	})
	if err != nil {
		return false, err
	}
	return hp, nil
}

func UserRoles(ctx context.Context, q *models.Queries, username string) ([]string, error) {
	roles, err := q.GetUserRoles(ctx, pgtype.Text{Valid: true, String: username})
	if err != nil {
		return nil, err
	}
	return lo.Map(roles, func(item models.Role, idx int) string {
		return item.Name
	}), nil
}
