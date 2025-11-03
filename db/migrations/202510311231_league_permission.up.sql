BEGIN;

INSERT INTO permissions (code, description)
VALUES ('can_manage_leagues', 'Can manage league seasons, registrations, and divisions.');

INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ((SELECT id FROM roles WHERE name = 'Manager'),
     (SELECT id FROM permissions WHERE code = 'can_manage_leagues'));

COMMIT;
