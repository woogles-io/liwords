import React from "react";
import { Link } from "react-router";
import { Alert, Button, Card, Divider, Spin, Tag } from "antd";
import { CrownOutlined, SafetyOutlined, ToolOutlined } from "@ant-design/icons";
import { useQuery } from "@connectrpc/connect-query";
import {
  getSelfRoles,
  getSelfPermissions,
} from "../gen/api/proto/user_service/user_service-AuthorizationService_connectquery";
import { hasAnyPermission, ADMIN_PANEL_PERMS } from "../mod/perms";

// Human-readable permission names
const PERMISSION_LABELS: Record<string, string> = {
  admin_all_access: "Full Admin Access",
  can_create_tournaments: "Create Tournaments",
  can_manage_tournaments: "Manage Tournaments",
  can_bypass_elitebot_paywall: "Bypass Elite Bot Paywall",
  can_moderate_users: "Moderate Users",
  can_modify_announcements: "Modify Announcements",
  can_create_puzzles: "Create Puzzles",
  can_reset_and_delete_accounts: "Reset and Delete Accounts",
  can_manage_badges: "Manage Badges",
  can_see_private_user_data: "View Private User Data",
  can_manage_app_roles_and_permissions: "Manage App Roles & Permissions",
  can_manage_user_roles: "Manage User Roles",
  can_view_user_roles: "View User Roles",
  can_manage_leagues: "Manage Leagues",
  can_play_leagues: "Play in Leagues",
  can_invite_to_leagues: "Invite Users to Leagues",
  can_revoke_from_leagues: "Revoke Users from Leagues",
  can_verify_user_identities: "Verify User Identities",
  can_create_broadcasts: "Create & Manage Broadcasts",
};

export const RolesPermissions = () => {
  const { data: selfRoles, isLoading, error } = useQuery(getSelfRoles, {});
  const { data: selfPermissions, isLoading: permsLoading } = useQuery(
    getSelfPermissions,
    {},
  );

  if (isLoading || permsLoading) {
    return (
      <div style={{ padding: "24px", textAlign: "center" }}>
        <Spin size="large" />
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: "24px" }}>
        <Alert
          message="Error Loading Roles"
          description="Failed to load your roles and permissions."
          type="error"
          showIcon
        />
      </div>
    );
  }

  const roles = selfRoles?.roles || [];
  const allPermissions = selfPermissions?.permissions || [];
  const canAccessAdmin = hasAnyPermission(allPermissions, ADMIN_PANEL_PERMS);

  return (
    <div className="roles-permissions-container" style={{ padding: "24px" }}>
      <h3 style={{ marginBottom: "8px" }}>
        <SafetyOutlined style={{ marginRight: "8px" }} />
        Roles & Permissions
      </h3>
      <p className="description" style={{ marginBottom: "24px" }}>
        View your assigned roles and permissions on Woogles.
      </p>

      {canAccessAdmin && (
        <div style={{ marginBottom: "24px" }}>
          <Link to="/admin">
            <Button type="primary" icon={<ToolOutlined />} size="large">
              Go to Admin Panel
            </Button>
          </Link>
        </div>
      )}

      {roles.length === 0 ? (
        <Alert
          message="No Special Roles"
          description="You don't have any special roles assigned. You can still play games and participate in tournaments!"
          type="info"
          showIcon
        />
      ) : (
        <>
          <Card title="Your Roles" style={{ marginBottom: "24px" }}>
            <div style={{ display: "flex", flexWrap: "wrap", gap: "8px" }}>
              {roles.map((role) => {
                const isLeague = role.includes("League");
                return (
                  <Tag
                    key={role}
                    color={
                      role === "Admin"
                        ? "red"
                        : role === "Manager"
                          ? "orange"
                          : isLeague
                            ? "purple"
                            : "blue"
                    }
                    style={{
                      fontSize: "14px",
                      padding: "4px 12px",
                      borderRadius: "4px",
                    }}
                    icon={role === "Admin" ? <CrownOutlined /> : undefined}
                  >
                    {role}
                  </Tag>
                );
              })}
            </div>
          </Card>

          <Card title="Your Permissions">
            <p
              className="description"
              style={{ marginBottom: "16px", fontSize: "14px" }}
            >
              These permissions are granted by your roles:
            </p>
            {allPermissions.length === 0 ? (
              <p style={{ color: "#999" }}>
                No specific permissions listed for your roles.
              </p>
            ) : (
              <div
                style={{
                  display: "flex",
                  flexDirection: "column",
                  gap: "8px",
                }}
              >
                {[...allPermissions].sort().map((permission) => (
                  <div
                    key={permission}
                    className="permission-item"
                    style={{
                      padding: "8px 12px",
                      borderRadius: "4px",
                      fontSize: "14px",
                    }}
                  >
                    <strong>
                      {PERMISSION_LABELS[permission] || permission}
                    </strong>
                    <span
                      className="permission-code"
                      style={{ fontSize: "12px", marginLeft: "8px" }}
                    >
                      ({permission})
                    </span>
                  </div>
                ))}
              </div>
            )}
          </Card>
        </>
      )}

      <Divider />

      <div className="description" style={{ fontSize: "12px" }}>
        <p>
          <strong>Note:</strong> Roles and permissions are managed by Woogles
          administrators. If you believe you need additional permissions, please
          contact support.
        </p>
      </div>
    </div>
  );
};
