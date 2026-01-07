BEGIN;

-- Rollback changes to soughtgames.created_at
ALTER TABLE soughtgames ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE soughtgames ALTER COLUMN created_at DROP DEFAULT;

COMMIT;
