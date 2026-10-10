-- name: AppendGameTurn :exec
INSERT INTO game_turns (game_uuid, turn_idx, event)
VALUES (@game_uuid, @turn_idx, @event);

-- Append a whole move's events in one statement.
--
-- NOT STANDARD SQL -- read this before editing it.
--
-- `unnest(array) WITH ORDINALITY AS t(evt, ord)` turns one array parameter into
-- one row per element, with `ord` counting 1, 2, 3... So passing three events
-- with start_idx = 7 inserts turn_idx 7, 8 and 9. Postgres-specific, and the
-- reason it is worth the strangeness is round trips: a move can produce three
-- events (six scoreless turns writes a PASS and two END_RACK_PENALTY events),
-- and this writes them in one call instead of three while a per-game advisory
-- lock is held.
--
-- Three plain INSERTs inside the same transaction would be equally *correct* --
-- a transaction is what makes a move atomic, not this statement. This is only
-- fewer round trips. If it ever gets in the way, replacing it with a loop of
-- AppendGameTurn inside the same transaction loses nothing but latency.
--
-- The indices are derived from `ord` rather than passed in as a second array,
-- because a move's events are always consecutive from start_idx. A parallel
-- array of indices would add a way for the indices and the events to disagree.
--
-- sqlc note: the arithmetic must be parenthesised and use sqlc.arg(), or sqlc
-- silently rewrites the expression -- it emitted `($2::int4 - 1)`, dropping the
-- ordinal entirely, which would insert every event at the same index. Check the
-- generated SQL in pkg/stores/models/game_turns.sql.go after touching this.
-- name: AppendGameTurns :exec
INSERT INTO game_turns (game_uuid, turn_idx, event)
SELECT @game_uuid, (sqlc.arg(start_idx)::int4 + t.ord::int4 - 1), t.evt
FROM unnest(@events::jsonb[]) WITH ORDINALITY AS t(evt, ord);

-- name: GetGameTurns :many
SELECT turn_idx, event FROM game_turns
WHERE game_uuid = @game_uuid
ORDER BY turn_idx ASC;

-- name: GetLastGameTurn :one
SELECT turn_idx, event FROM game_turns
WHERE game_uuid = @game_uuid
ORDER BY turn_idx DESC
LIMIT 1;

-- name: DeleteGameTurns :exec
DELETE FROM game_turns WHERE game_uuid = @game_uuid;

-- name: CountGameTurns :one
SELECT COUNT(*)::int4 FROM game_turns WHERE game_uuid = @game_uuid;
