-- Cross-node serialization for a single game.
--
-- Taken at the start of every write transaction that touches a game, and
-- released automatically on commit or rollback. This is what makes it safe to
-- reconstruct a position from game_turns: without it, AppendTurns commits a
-- terminal event before Set commits game_end_reason, so a concurrent read sees
-- a finished game that still claims to be in progress. That race is why the
-- turns read path was disabled.
--
-- Deliberately not tied to any one table: the lock is on the game id, and it
-- outlives whatever storage happens to hold the game.
--
-- These are SESSION-level, not transaction-level, and that is the whole point.
-- A move is a read-modify-write -- load the position, play into it, save -- and
-- the lock has to span all three or two servers will load the same position,
-- each play a legal move, and each save it. Both saves succeed, both are
-- internally consistent, and the second silently discards the first. A
-- transaction-scoped lock inside the save cannot prevent that, because by then
-- both have already read.
--
-- Session scope means the lock lives on one connection until it is explicitly
-- released, so whoever takes it must hold that connection and must release it.
-- See pkg/stores/game/lock.go, which owns both halves.
--
-- NOT STANDARD SQL: hashtextextended() maps the game id onto the int8 that
-- advisory locks key on. It is stable across servers and versions, which is
-- what matters here; it is not a security hash and a collision between two game
-- ids would only make them share a lock -- correct, just slower.

-- name: TryLockGameSession :one
SELECT pg_try_advisory_lock(hashtextextended(@game_uuid::text, 0));

-- name: UnlockGameSession :one
SELECT pg_advisory_unlock(hashtextextended(@game_uuid::text, 0));
