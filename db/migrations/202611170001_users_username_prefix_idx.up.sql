-- For UsersByPrefix (username autocomplete), which scanned all of users on
-- every keystroke: username_idx is on lower(username) in the database's
-- en_US collation, whose order does not keep a prefix's matches together, so
-- it cannot serve a prefix search. In byte order ("C") it can, as one range.
--
-- One statement per file: CREATE INDEX CONCURRENTLY cannot run inside a
-- transaction block, and golang-migrate sends a file as one query.
-- CONCURRENTLY waits for every transaction that has users open, so it waits
-- out the daily pg_dump (18:00 to ~22:00 UTC) if started during it.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_username_lower_c
    ON users ((lower(username::text) COLLATE "C"));
