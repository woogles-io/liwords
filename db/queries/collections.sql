-- name: CreateCollection :one
INSERT INTO collections (uuid, title, description, creator_id, public)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, uuid, created_at;

-- name: GetCollectionByUUID :one
SELECT * FROM collections WHERE uuid = $1;

-- name: GetCollectionGames :many
SELECT * FROM collection_games 
WHERE collection_id = $1 
ORDER BY chapter_number;

-- name: UpdateCollection :exec
UPDATE collections 
SET title = $2, description = $3, public = $4, updated_at = NOW()
WHERE uuid = $1;

-- name: DeleteCollection :exec
DELETE FROM collections WHERE uuid = $1;

-- name: AddGameToCollection :exec
INSERT INTO collection_games (collection_id, game_id, chapter_number, chapter_title, is_annotated)
VALUES ($1, $2, $3, $4, $5);

-- name: RemoveGameFromCollection :exec
DELETE FROM collection_games 
WHERE collection_id = $1 AND game_id = $2;

-- name: GetUserCollections :many
SELECT c.*, u.uuid as creator_uuid, u.username as creator_username,
       COALESCE(game_counts.game_count, 0) as game_count
FROM collections c
JOIN users u ON c.creator_id = u.id
LEFT JOIN (
    SELECT collection_id, COUNT(*) as game_count
    FROM collection_games
    GROUP BY collection_id
) game_counts ON c.id = game_counts.collection_id
WHERE c.creator_id = (SELECT id FROM users WHERE users.uuid = $1)
ORDER BY c.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPublicCollections :many
SELECT c.*, u.uuid as creator_uuid, u.username as creator_username,
       COALESCE(game_counts.game_count, 0) as game_count
FROM collections c
JOIN users u ON c.creator_id = u.id
LEFT JOIN (
    SELECT collection_id, COUNT(*) as game_count
    FROM collection_games
    GROUP BY collection_id
) game_counts ON c.id = game_counts.collection_id
WHERE c.public = true 
ORDER BY c.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetMaxChapterNumber :one
SELECT COALESCE(MAX(chapter_number), 0) as max_chapter
FROM collection_games 
WHERE collection_id = $1;

-- name: ReorderCollectionGames :exec
UPDATE collection_games 
SET chapter_number = $2
WHERE collection_id = $1 AND game_id = $3;

-- name: SetTempChapterNumbers :exec
UPDATE collection_games 
SET chapter_number = -chapter_number
WHERE collection_id = $1;

-- name: ResetChapterNumbers :exec
UPDATE collection_games 
SET chapter_number = -chapter_number
WHERE collection_id = $1 AND chapter_number < 0;

-- name: UpdateChapterTitle :exec
UPDATE collection_games 
SET chapter_title = $3
WHERE collection_id = $1 AND game_id = $2;

-- name: GetCollectionWithGames :one
SELECT c.*, u.uuid as creator_uuid, u.username as creator_username
FROM collections c
JOIN users u ON c.creator_id = u.id
WHERE c.uuid = $1;

-- name: CheckCollectionOwnership :one
SELECT EXISTS(
    SELECT 1 FROM collections 
    WHERE uuid = $1 AND creator_id = $2
) as owns;

-- name: GetCollectionsForGame :many
SELECT c.uuid, c.title, c.description, c.creator_id, c.public, 
       u.uuid as creator_uuid, u.username as creator_username,
       cg.chapter_number, cg.chapter_title
FROM collections c
JOIN collection_games cg ON c.id = cg.collection_id
JOIN users u ON c.creator_id = u.id
WHERE cg.game_id = $1
ORDER BY c.created_at DESC;