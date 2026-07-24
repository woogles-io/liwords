BEGIN;

INSERT INTO permissions (code, description)
VALUES ('can_manage_broadcasts', 'Can edit, deactivate, and otherwise administer any broadcast event.');

INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ((SELECT id FROM roles WHERE name = 'Manager'),
     (SELECT id FROM permissions WHERE code = 'can_manage_broadcasts'));

COMMIT;
