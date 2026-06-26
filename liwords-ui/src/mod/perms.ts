// Permission codes — must stay in sync with pkg/auth/rbac/rbac.go and DB migrations.
// These are the granular permission strings returned by GetSelfPermissions.
export const Perm = {
  AdminAllAccess: "admin_all_access",
  CanCreateTournaments: "can_create_tournaments",
  CanManageTournaments: "can_manage_tournaments",
  CanPlayEliteBot: "can_bypass_elitebot_paywall",
  CanModerateUsers: "can_moderate_users",
  CanModifyAnnouncements: "can_modify_announcements",
  CanCreatePuzzles: "can_create_puzzles",
  CanResetAndDeleteAccounts: "can_reset_and_delete_accounts",
  CanManageBadges: "can_manage_badges",
  CanSeePrivateUserData: "can_see_private_user_data",
  CanManageAppRolesAndPermissions: "can_manage_app_roles_and_permissions",
  CanManageUserRoles: "can_manage_user_roles",
  CanViewUserRoles: "can_view_user_roles",
  CanManageLeagues: "can_manage_leagues",
  CanPlayLeagues: "can_play_leagues",
  CanInviteToLeagues: "can_invite_to_leagues",
  CanRevokeFromLeagues: "can_revoke_from_leagues",
  CanVerifyUserIdentities: "can_verify_user_identities",
  CanCreateBroadcasts: "can_create_broadcasts",
} as const;

export type PermCode = (typeof Perm)[keyof typeof Perm];

/**
 * Returns true if the user's permission list includes the given code,
 * or if the user has admin_all_access (which grants every permission).
 */
export const hasPermission = (
  permissions: Array<string>,
  code: PermCode,
): boolean =>
  permissions.includes(Perm.AdminAllAccess) || permissions.includes(code);
