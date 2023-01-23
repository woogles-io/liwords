-- name: GetCommentsForGame :many
SELECT game_comments.id, games.uuid as game_uuid, users.uuid as user_uuid, 
    users.username, event_number, edited_at, comment
from game_comments
join games on game_comments.game_id = games.id
join users on game_comments.author_id = users.id
where games.uuid = $1
ORDER BY created_at ASC;

-- name: AddComment :one
INSERT INTO game_comments (
    id, game_id, author_id, event_number, comment
) SELECT gen_random_uuid(), games.id, $2, $3, $4
FROM games
JOIN games on games.uuid = $1
RETURNING game_comments.id;

-- name: DeleteComment :exec
DELETE FROM game_comments
WHERE id = $1;

-- name: UpdateComment :exec
UPDATE game_comments SET comment = $1
WHERE id = $2;
