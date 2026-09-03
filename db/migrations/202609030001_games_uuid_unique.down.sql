BEGIN;
CREATE INDEX IF NOT EXISTS idx_games_uuid ON games (uuid);
DROP INDEX IF EXISTS idx_games_uuid_unique;
COMMIT;
