-- name: AddChatMessages :execrows
-- Adds messages already written to Redis, identified there by legacy_id. A
-- message whose (channel, legacy_id) is already here is left as it is, so
-- the dual write and the backfill can overlap and the backfill can be re-run.
-- An empty or unknown sender UUID gives a NULL sender_id.
INSERT INTO chat_messages (id, channel, sender_id, sender_username, message, created_at, legacy_id)
SELECT m.id, m.channel, u.id, m.sender_username, m.message, m.created_at, m.legacy_id
FROM (SELECT unnest(@ids::uuid[]) AS id,
             unnest(@channels::text[]) AS channel,
             unnest(@sender_uuids::text[]) AS sender_uuid,
             unnest(@sender_usernames::text[]) AS sender_username,
             unnest(@messages::text[]) AS message,
             unnest(@created_ats::timestamptz[]) AS created_at,
             unnest(@legacy_ids::text[]) AS legacy_id) m
LEFT JOIN users u ON m.sender_uuid <> '' AND u.uuid = m.sender_uuid
ON CONFLICT (channel, legacy_id) WHERE legacy_id IS NOT NULL DO NOTHING;

-- name: DeleteChatMessageByLegacyID :execrows
-- Soft-deletes a message by its Redis stream ID; it stays for review.
UPDATE chat_messages
SET deleted_at = now(),
    deleted_by = (SELECT id FROM users WHERE uuid = @deleted_by_uuid::text)
WHERE channel = @channel AND legacy_id = @legacy_id AND deleted_at IS NULL;
