-- name: GetCommentsForGame :many
SELECT game_comments.id, games.uuid as game_uuid, users.uuid as user_uuid, 
    users.username, event_number, edited_at, comment
from game_comments
join games on game_comments.game_id = games.id
join users on game_comments.author_id = users.id
where games.uuid = $1
ORDER BY game_comments.created_at ASC;

-- name: AddComment :one
INSERT INTO game_comments (
    id, game_id, author_id, event_number, comment
) SELECT gen_random_uuid(), games.id, $2, $3, $4
FROM games WHERE games.uuid = $1
RETURNING game_comments.id;

-- name: DeleteComment :exec
DELETE FROM game_comments
WHERE id = $1 and author_id = $2;

-- name: DeleteCommentNoAuthorSpecified :exec
DELETE FROM game_comments WHERE id = $1;

-- name: UpdateComment :exec
UPDATE game_comments SET comment = $1, edited_at = now()
WHERE id = $2 and author_id = $3;

-- name: GetCommentsForAllGames :many
SELECT game_comments.id, games.uuid as game_uuid, users.uuid as user_uuid,
    users.username, event_number, edited_at, quickdata
FROM game_comments
JOIN games on game_comments.game_id = games.id
JOIN users on game_comments.author_id = users.id
ORDER BY game_comments.created_at DESC
LIMIT $1 OFFSET $2;