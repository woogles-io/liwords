BEGIN;

-- Remove role permission associations
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN ('can_play_leagues', 'can_invite_to_leagues')
);

-- Remove roles
DELETE FROM roles WHERE name IN ('League Player', 'League Promoter');

-- Remove permissions
DELETE FROM permissions WHERE code IN ('can_play_leagues', 'can_invite_to_leagues');

COMMIT;
