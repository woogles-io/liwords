-- name: GetNotoriousGames :many
SELECT game_id, type, timestamp FROM notoriousgames
WHERE player_id = $1
ORDER BY timestamp DESC LIMIT $2;

-- name: GetActionsBatch :many
-- Get current actions for multiple users in a single query
-- Returns one row per (user_uuid, action_type) combination
SELECT DISTINCT ON (users.uuid, user_actions.action_type)
    users.uuid as user_uuid,
    user_actions.id,
    user_actions.user_id,
    user_actions.action_type,
    user_actions.start_time,
    user_actions.end_time,
    user_actions.removed_time,
    user_actions.message_id,
    user_actions.applier_id,
    user_actions.remover_id,
    user_actions.note,
    user_actions.removal_note,
    user_actions.chat_text,
    user_actions.email_type
FROM users
JOIN user_actions ON users.id = user_actions.user_id
WHERE users.uuid = ANY(@user_uuids::text[])
    AND user_actions.removed_time IS NULL
    AND (user_actions.end_time IS NULL OR user_actions.end_time > NOW())
ORDER BY users.uuid, user_actions.action_type, user_actions.start_time DESC;