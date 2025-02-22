BEGIN;

ALTER TABLE users ADD COLUMN entity_uuid uuid DEFAULT public.uuid_generate_v4() NOT NULL UNIQUE;

COMMIT;