BEGIN;

ALTER TABLE integrations
ADD COLUMN last_updated TIMESTAMPTZ DEFAULT NULL;

UPDATE integrations
SET last_updated = '1970-01-01 00:00:00+00';

ALTER TABLE integrations
ALTER COLUMN last_updated SET DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE integrations
ALTER COLUMN last_updated SET NOT NULL;

CREATE INDEX idx_integrations_last_updated ON integrations(last_updated);

ALTER TABLE integrations_global
ADD COLUMN last_updated TIMESTAMPTZ DEFAULT NULL;

UPDATE integrations_global
SET last_updated = '1970-01-01 00:00:00+00';

ALTER TABLE integrations_global
ALTER COLUMN last_updated SET DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE integrations_global
ALTER COLUMN last_updated SET NOT NULL;

CREATE INDEX idx_integrations_global_last_updated ON integrations_global(last_updated);

COMMIT;