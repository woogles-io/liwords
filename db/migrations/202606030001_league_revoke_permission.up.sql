BEGIN;

INSERT INTO permissions (code, description)
VALUES ('can_revoke_from_leagues', 'Can revoke league player access from users.');

INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ((SELECT id FROM roles WHERE name = 'League Promoter'),
     (SELECT id FROM permissions WHERE code = 'can_revoke_from_leagues')),
    ((SELECT id FROM roles WHERE name = 'Manager'),
     (SELECT id FROM permissions WHERE code = 'can_revoke_from_leagues'));

COMMIT;
