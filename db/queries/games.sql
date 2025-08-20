-- name: GetGame :one
SELECT * FROM games WHERE uuid = @uuid; -- this is not even a uuid, sigh.

-- name: GetGameOwner :one
SELECT 
    agm.creator_uuid,
    u.username 
FROM annotated_game_metadata agm
JOIN users u ON agm.creator_uuid = u.uuid
WHERE agm.game_uuid = @game_uuid;

