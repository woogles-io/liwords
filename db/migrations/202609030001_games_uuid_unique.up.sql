BEGIN;

-- games.uuid is the game id every other table and every URL refers to, and
-- nothing in the database prevented two rows sharing one. Only the plain btree
-- idx_games_uuid from the initial migration existed, and a plain index permits
-- duplicates.
--
-- Checked before adding this, across all 12,692,837 rows: zero duplicates and
-- zero NULLs. The scan took 20 seconds because the planner can answer it from
-- the index alone rather than the 17 GB heap.
--
-- IMPORTANT for anyone reading this on a fresh database: the name is
-- deliberately NOT idx_games_uuid. That name is already taken by the non-unique
-- index created in 202203290423_initial, so `CREATE UNIQUE INDEX IF NOT EXISTS
-- idx_games_uuid` would find the name in use, skip, and leave a fresh database
-- silently without the constraint. Distinct name, then drop the old one.
--
-- Production has both statements applied by hand, with CONCURRENTLY, because
-- building a 705 MB index inside this transaction would lock out writers on the
-- 12M-row table for the duration:
--
--     CREATE UNIQUE INDEX CONCURRENTLY idx_games_uuid_unique ON games (uuid);
--     DROP INDEX CONCURRENTLY idx_games_uuid;
--
-- Both are therefore no-ops there. Everywhere else the table is small enough
-- that doing it inline costs nothing.
CREATE UNIQUE INDEX IF NOT EXISTS idx_games_uuid_unique ON games (uuid);

-- Redundant once the unique index exists: same column, same lookups, and it
-- was 705 MB in production. The partial index on uuid used by the
-- pending-archival scan is a different thing and stays.
DROP INDEX IF EXISTS idx_games_uuid;

COMMIT;
