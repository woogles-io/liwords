BEGIN;

CREATE INDEX IF NOT EXISTS idx_tournaments_uuid ON tournaments USING btree (lower(uuid));

DROP INDEX IF EXISTS idx_tournaments_uuid_unique;

COMMIT;
