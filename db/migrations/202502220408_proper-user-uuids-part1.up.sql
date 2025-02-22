BEGIN;

ALTER TABLE users ADD COLUMN entity_uuid uuid DEFAULT gen_random_uuid() NOT NULL UNIQUE;

COMMIT;