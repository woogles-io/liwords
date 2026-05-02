-- name: AppendGameTurn :exec
INSERT INTO game_turns (game_uuid, turn_idx, event)
VALUES (@game_uuid, @turn_idx, @event);

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
