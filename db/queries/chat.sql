-- name: AddChatMessages :execrows
-- Adds messages; one whose id is already here is left as it is. Messages
-- copied from Redis have ids computed from their stream IDs, so the dual
-- write and the backfill can overlap and the backfill can be re-run. An
-- empty or unknown sender UUID gives a NULL sender_id.
INSERT INTO chat_messages (id, channel, sender_id, sender_username, message, created_at)
SELECT m.id, m.channel, u.id, m.sender_username, m.message, m.created_at
FROM (SELECT unnest(@ids::uuid[]) AS id,
             unnest(@channels::text[]) AS channel,
             unnest(@sender_uuids::text[]) AS sender_uuid,
             unnest(@sender_usernames::text[]) AS sender_username,
             unnest(@messages::text[]) AS message,
             unnest(@created_ats::timestamptz[]) AS created_at) m
LEFT JOIN users u ON m.sender_uuid <> '' AND u.uuid = m.sender_uuid
ON CONFLICT (id) DO NOTHING;

-- name: DeleteChatMessage :execrows
-- Soft-deletes a message; it stays for review.
UPDATE chat_messages
SET deleted_at = now(),
    deleted_by = (SELECT u.id FROM users u WHERE u.uuid = @deleted_by_uuid::text)
WHERE chat_messages.id = @id AND chat_messages.channel = @channel
  AND chat_messages.deleted_at IS NULL;
