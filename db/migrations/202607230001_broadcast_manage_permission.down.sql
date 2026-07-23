BEGIN;

DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'can_manage_broadcasts');
DELETE FROM permissions WHERE code = 'can_manage_broadcasts';

COMMIT;
