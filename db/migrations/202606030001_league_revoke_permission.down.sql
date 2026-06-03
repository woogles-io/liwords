BEGIN;

DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'can_revoke_from_leagues');

DELETE FROM permissions WHERE code = 'can_revoke_from_leagues';

COMMIT;
