BEGIN;
DROP INDEX IF EXISTS idx_games_history_s3_key_pending;
ALTER TABLE games DROP COLUMN IF EXISTS history_s3_key;
COMMIT;
