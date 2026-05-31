BEGIN;

DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE code = 'can_create_broadcasts');
DELETE FROM roles WHERE name = 'Broadcast Creator';
DELETE FROM permissions WHERE code = 'can_create_broadcasts';

DROP INDEX IF EXISTS idx_broadcast_games_broadcast_round;
DROP INDEX IF EXISTS idx_broadcasts_slug;
DROP INDEX IF EXISTS idx_broadcasts_active;

DROP TABLE IF EXISTS broadcast_directors;
DROP TABLE IF EXISTS broadcast_annotators;
DROP TABLE IF EXISTS broadcast_games;
DROP TABLE IF EXISTS broadcasts;

COMMIT;
