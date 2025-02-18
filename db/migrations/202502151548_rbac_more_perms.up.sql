BEGIN;

INSERT INTO permissions (code, description)
VALUES
    ('can_manage_badges', 'Can assign and unassign user badges.'),
    ('can_see_private_user_data', 'Can see private user data, such as emails.'),
    ('can_manage_app_roles_and_permissions', 'Can create and update roles and permissions.'),
    ('can_manage_user_roles', 'Can assign and unassign user roles.'),
    ('can_view_user_roles', 'Can view user roles.');

INSERT INTO roles(name, description)
VALUES
    ('Manager', 'Site manager. Has a lot of access to site internals.');

INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ((SELECT id FROM roles WHERE name = 'Manager'),
     (SELECT id FROM permissions WHERE code = 'can_manage_badges')),
    ((SELECT id FROM roles WHERE name = 'Manager'),
     (SELECT id FROM permissions WHERE code = 'can_see_private_user_data')),
    ((SELECT id FROM roles WHERE name = 'Manager'),
     (SELECT id FROM permissions WHERE code = 'can_manage_user_roles')),
    ((SELECT id FROM roles WHERE name = 'Moderator'),
     (SELECT id FROM permissions WHERE code = 'can_see_private_user_data')),
    ((SELECT id FROM roles WHERE name = 'Manager'),
     (SELECT id FROM permissions WHERE code = 'can_view_user_roles')),
    ((SELECT id FROM roles WHERE name = 'Moderator'),
     (SELECT id FROM permissions WHERE code = 'can_view_user_roles'));

COMMIT;