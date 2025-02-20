BEGIN;

DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE name = 'Manager');
DELETE FROM role_permissions WHERE permission_id = (SELECT id FROM permissions WHERE code = 'can_see_private_user_data');
DELETE FROM role_permissions WHERE permission_id = (SELECT id FROM permissions WHERE code = 'can_view_user_roles');

DELETE FROM permissions
WHERE code IN ('can_manage_badges',
    'can_see_private_user_data',
    'can_manage_app_roles_and_permissions',
    'can_manage_user_roles',
    'can_view_user_roles');

DELETE FROM roles WHERE name = 'Manager';


COMMIT;