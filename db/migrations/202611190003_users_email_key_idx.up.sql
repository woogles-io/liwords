-- For finding an account whose email has the same email_key as a new
-- address. Not unique: some existing accounts already share a key.
--
-- One statement per file: CREATE INDEX CONCURRENTLY cannot run inside a
-- transaction block, and golang-migrate sends a file as one query.
-- CONCURRENTLY waits for every transaction that has users open, so it waits
-- out the daily pg_dump (18:00 to ~22:00 UTC) if started during it.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email_key
    ON users (email_key(email));
