-- Same as idx_games_player0_filtered (202611120003), for player1_id.
-- One statement per file: DROP INDEX CONCURRENTLY cannot run inside a
-- transaction block, and golang-migrate sends a file as one query, so a second
-- statement would put both in an implicit transaction and fail.
--
-- CONCURRENTLY so writes to games carry on while it runs. It waits for every
-- transaction that has games open to finish, so it will wait out the daily
-- pg_dump (18:00 to ~22:00 UTC) if started during it; plain DROP INDEX would
-- wait just as long while blocking every query on games behind it.
DROP INDEX CONCURRENTLY IF EXISTS idx_games_player1_filtered;
