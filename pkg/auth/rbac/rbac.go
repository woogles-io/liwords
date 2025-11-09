package rbac

import (
	"context"

	"github.com/samber/lo"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// These Permissions should be defined in the database. See the rbac.up.sql file.
type Permission string

const (
	AdminAllAccess                  Permission = "admin_all_access"
	CanCreateTournaments            Permission = "can_create_tournaments"
	CanManageTournaments            Permission = "can_manage_tournaments"
	CanPlayEliteBot                 Permission = "can_bypass_elitebot_paywall"
	CanModerateUsers                Permission = "can_moderate_users"
	CanModifyAnnouncements          Permission = "can_modify_announcements"
	CanCreatePuzzles                Permission = "can_create_puzzles"
	CanResetAndDeleteAccounts       Permission = "can_reset_and_delete_accounts"
	CanManageBadges                 Permission = "can_manage_badges"
	CanSeePrivateUserData           Permission = "can_see_private_user_data"
	CanManageAppRolesAndPermissions Permission = "can_manage_app_roles_and_permissions"
	CanManageUserRoles              Permission = "can_manage_user_roles"
	CanViewUserRoles                Permission = "can_view_user_roles"
	CanManageLeagues                Permission = "can_manage_leagues"
	CanPlayLeagues                  Permission = "can_play_leagues"
	CanInviteToLeagues              Permission = "can_invite_to_leagues"
)

// These Roles are also defined in the database.
type Role string

const (
	// There are three special roles for the application that have some level
	// of management power. These roles should follow a hierarchy:
	// Admin is the highest role. It should have a single permission associated
	// with it: admin_all_access, which gives it access to all the permissions
	// listed above without specificially specifying them.
	Admin Role = "Admin"
	// Manager can manage several aspects of the site without having all the permissions.
	// It shouldn't necessarily have user management permissions.
	Manager Role = "Manager"
	// Moderator can moderate users for the most part. Of course, this all depends
	// on what permissions they are assigned.
	Moderator Role = "Moderator"

	// All other roles
	// Although TournamentCreator and TournamentManager have some level of management
	// power as well, these roles _should_ have very focused permissions.
	TournamentCreator   Role = "Tournament Creator"
	TournamentManager   Role = "Tournament Manager"
	SpecialAccessPlayer Role = "Special Access Player"
	LeaguePlayer        Role = "League Player"
	LeaguePromoter      Role = "League Promoter"
)

const LowHierarchyRoleValue = 0

var RoleHierarchyMap = map[Role]int{
	Admin:     100000,
	Manager:   10000,
	Moderator: 1000,
}

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
	roles, err := q.GetUserRoles(ctx, username)
	if err != nil {
		return nil, err
	}
	return lo.Map(roles, func(item models.Role, idx int) string {
		return item.Name
	}), nil
}
