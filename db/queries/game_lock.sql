-- Cross-node serialization for a single game's write path.
--
-- Taken at the start of every write transaction that touches a game, and
-- released automatically on commit or rollback. This is what makes it safe to
-- reconstruct a position from game_turns: without it, AppendTurns commits a
-- terminal event before Set commits game_end_reason, so a concurrent read sees
-- a finished game that still claims to be in progress. That race is why the
-- turns read path was disabled.
--
-- It also replaces Cache.LockGame, a process-local mutex map that serializes
-- nothing between app servers.
--
-- Deliberately not tied to any one table: the lock is on the game id, and it
-- outlives whatever storage happens to hold the game.

-- name: LockGameForWrite :exec
SELECT pg_advisory_xact_lock(hashtextextended(@game_uuid::text, 0));
