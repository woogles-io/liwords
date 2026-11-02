BEGIN;

-- Every tournament lookup filters on uuid = $1, but the only index on the
-- column was on lower(uuid), which that filter can't use. Each load and each
-- SELECT ... FOR UPDATE in DBStore.Set was a sequential scan of the table,
-- about 4ms in production. Nothing queries lower(uuid).
--
-- Checked before adding this, across all 6,106 rows: zero duplicates and zero
-- NULLs. The table is small enough to build the index inline.
CREATE UNIQUE INDEX IF NOT EXISTS idx_tournaments_uuid_unique ON tournaments (uuid);

DROP INDEX IF EXISTS idx_tournaments_uuid;

COMMIT;
