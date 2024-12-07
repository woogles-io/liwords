BEGIN;

CREATE TABLE integrations_global (
    id BIGSERIAL PRIMARY KEY,
    integration_name TEXT NOT NULL UNIQUE,
    data JSONB NOT NULL DEFAULT '{}'::jsonb
);

COMMIT;