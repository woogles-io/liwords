BEGIN;

ALTER TABLE integrations DROP COLUMN last_updated;
ALTER TABLE integrations_global DROP COLUMN last_updated;

COMMIT;