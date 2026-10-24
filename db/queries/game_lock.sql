-- Cross-node serialization for a single game.
--
-- Taken before a game is loaded for a move and released after it is saved, so
-- that only one server at a time can be playing into a given game. It is the
-- prerequisite for serving positions from game_turns and for running more than
-- one API task.
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
-- advisory locks key on. The key only has to agree with itself within one
-- running server, since every caller asks the same database; Postgres does not
-- promise the same value across versions or architectures, and nothing here
-- needs it to, because advisory locks do not survive a restart. It is not a
-- security hash: two game ids that collide would only share a lock -- correct,
-- just slower.

-- name: TryLockGameSession :one
SELECT pg_try_advisory_lock(hashtextextended(@game_uuid::text, 0));

-- name: UnlockGameSession :one
SELECT pg_advisory_unlock(hashtextextended(@game_uuid::text, 0));
