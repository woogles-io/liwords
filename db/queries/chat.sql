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

-- name: LockChatSender :exec
-- Serializes one sender's messages within a transaction, so two at once
-- can't both pass the cooldown check.
SELECT pg_advisory_xact_lock(hashtext('chat-sender:' || @sender_uuid::text));

-- name: LastRegulatedChat :one
-- The sender's newest message in a channel where chat is rate-limited (all
-- but private and game chats), for the cooldown and repeat checks.
SELECT m.message, m.created_at
FROM chat_messages m
WHERE m.sender_id = (SELECT u.id FROM users u WHERE u.uuid = @sender_uuid::text)
  AND m.channel NOT LIKE 'chat.pm.%' AND m.channel NOT LIKE 'chat.game.%'
ORDER BY m.id DESC
LIMIT 1;

-- name: ChatMessagesBefore :many
-- A channel's messages older than before (or the newest, if before is
-- NULL), newest first.
SELECT m.id, m.sender_username, m.message, m.created_at, COALESCE(u.uuid, '')::text AS sender_uuid
FROM chat_messages m
LEFT JOIN users u ON u.id = m.sender_id
WHERE m.channel = @channel AND m.deleted_at IS NULL
  AND (sqlc.narg('before')::uuid IS NULL OR m.id < sqlc.narg('before')::uuid)
ORDER BY m.id DESC
LIMIT @lim;

-- name: ChatMessagesAfter :many
-- A channel's messages newer than after, oldest first.
SELECT m.id, m.sender_username, m.message, m.created_at, COALESCE(u.uuid, '')::text AS sender_uuid
FROM chat_messages m
LEFT JOIN users u ON u.id = m.sender_id
WHERE m.channel = @channel AND m.deleted_at IS NULL AND m.id > @after
ORDER BY m.id
LIMIT @lim;

-- name: GetChatMessage :one
SELECT m.id, m.sender_username, m.message, m.created_at, COALESCE(u.uuid, '')::text AS sender_uuid
FROM chat_messages m
LEFT JOIN users u ON u.id = m.sender_id
WHERE m.id = @id AND m.channel = @channel AND m.deleted_at IS NULL;

-- name: AddChatMembers :exec
-- Makes users members of a channel (the two people in a private chat), with
-- nothing read yet if they weren't already.
INSERT INTO chat_reads (user_id, channel)
SELECT u.id, @channel FROM users u WHERE u.uuid = ANY(@user_uuids::text[])
ON CONFLICT (user_id, channel) DO NOTHING;

-- name: MarkChatRead :execrows
-- Moves the user's read position in a channel forward to message_id, a
-- message in that channel. It never moves it back.
INSERT INTO chat_reads (user_id, channel, last_read_message_id, updated_at)
SELECT u.id, @channel, @message_id, now()
FROM users u
WHERE u.uuid = @user_uuid::text
  AND EXISTS (SELECT 1 FROM chat_messages m WHERE m.id = @message_id AND m.channel = @channel)
ON CONFLICT (user_id, channel) DO UPDATE
SET last_read_message_id = EXCLUDED.last_read_message_id, updated_at = now()
WHERE chat_reads.last_read_message_id IS NULL
   OR chat_reads.last_read_message_id < EXCLUDED.last_read_message_id;

-- name: GetChatRead :one
SELECT r.last_read_message_id
FROM chat_reads r JOIN users u ON u.id = r.user_id
WHERE u.uuid = @user_uuid::text AND r.channel = @channel;

-- name: ListPrivateChats :many
-- The user's private chats with at least one message, most recent first,
-- with the newest message, the user's read position and both usernames.
-- Paged by the newest message: only chats whose newest message is older
-- than before_id, if given.
SELECT r.channel, r.last_read_message_id,
       l.id AS last_id, l.message AS last_message, l.created_at AS last_at,
       (l.sender_id IS NOT DISTINCT FROM r.user_id)::bool AS last_from_user,
       ARRAY(SELECT p.username::text FROM users p
             WHERE p.uuid = ANY(string_to_array(substr(r.channel, 9), '_')))::text[] AS usernames
FROM chat_reads r
JOIN users me ON me.id = r.user_id
CROSS JOIN LATERAL (
    SELECT m.id, m.message, m.created_at, m.sender_id
    FROM chat_messages m
    WHERE m.channel = r.channel AND m.deleted_at IS NULL
    ORDER BY m.id DESC
    LIMIT 1) l
WHERE me.uuid = @user_uuid::text AND r.channel LIKE 'chat.pm.%'
  AND (sqlc.narg(before_id)::uuid IS NULL OR l.id < sqlc.narg(before_id)::uuid)
ORDER BY l.id DESC
LIMIT @lim;

-- name: ListUnreadPrivateChats :many
-- The user's private chats whose newest message is from someone else and
-- newer than the user's read position, most recent first.
SELECT r.channel
FROM chat_reads r
JOIN users me ON me.id = r.user_id
CROSS JOIN LATERAL (
    SELECT m.id, m.sender_id
    FROM chat_messages m
    WHERE m.channel = r.channel AND m.deleted_at IS NULL
    ORDER BY m.id DESC
    LIMIT 1) l
WHERE me.uuid = @user_uuid::text AND r.channel LIKE 'chat.pm.%'
  AND l.sender_id IS DISTINCT FROM r.user_id
  AND (r.last_read_message_id IS NULL OR l.id > r.last_read_message_id)
ORDER BY l.id DESC
LIMIT @lim;

-- name: ScanChatChannels :many
-- Channels whose names start with prefix, in name order after after, for
-- moderator chat review. The prefix holds no LIKE wildcards (it is one of a
-- fixed set).
SELECT DISTINCT m.channel
FROM chat_messages m
WHERE m.channel LIKE @prefix::text || '%' AND m.channel > @after::text
ORDER BY m.channel
LIMIT @lim;
