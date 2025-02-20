-- name: HasPermission :one
SELECT EXISTS (
    -- Check for either:
    -- 1. Direct permission match, OR
    -- 2. Wildcard admin access
    SELECT 1
    FROM user_roles ur
    JOIN role_permissions rp ON ur.role_id = rp.role_id
    JOIN permissions p ON rp.permission_id = p.id
    WHERE ur.user_id = @user_id
    AND (
        p.code = @permission  -- Specific permission
        OR
        p.code = 'admin_all_access'  -- Wildcard
    )
);

-- name: GetUsersWithRoles :many
SELECT
  u.uuid,
  u.username,
  r.name AS role_name
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.id
WHERE r.name = ANY(@role_names::text[])
ORDER BY u.username, r.name;

-- name: GetUserRoles :many
SELECT r.id, r.name, r.description
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = (SELECT id FROM users where username = @username)
ORDER BY r.name ASC;

-- name: AssignRole :exec
INSERT INTO user_roles (user_id, role_id)
VALUES (
    (SELECT id FROM users where username = @username),
    (SELECT id FROM roles WHERE name = @role_name LIMIT 1)
);

-- name: UnassignRole :execrows
DELETE FROM user_roles WHERE
  user_id = (SELECT id FROM users where username = @username)
  AND role_id = (SELECT id from roles WHERE name = @role_name LIMIT 1);

-- name: AddRole :exec
INSERT INTO roles (name, description) VALUES (@name, @description);

-- name: AddPermission :exec
INSERT INTO permissions (code, description) VALUES (@code, @description);

-- name: LinkRoleAndPermission :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES (
    (SELECT id FROM roles WHERE name = @role_name),
    (SELECT id FROM permissions WHERE code = @permission_code)
);

-- name: UnlinkRoleAndPermission :execrows
DELETE FROM role_permissions
WHERE
    role_id = (SELECT id FROM roles WHERE name = @role_name)
AND
    permission_id = (SELECT id FROM permissions WHERE code = @permission_code);

-- name: GetRolesWithPermissions :many
SELECT
    r.name,
    COALESCE(array_agg(p.code) FILTER (WHERE p.code IS NOT NULL), '{}')::text[] AS permissions
FROM roles r
LEFT JOIN role_permissions rp ON r.id = rp.role_id
LEFT JOIN permissions p ON p.id = rp.permission_id
GROUP BY r.name
ORDER BY r.name;