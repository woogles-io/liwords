-- Private chats are listed from chat_reads, which the chat store fills as
-- messages are sent. This adds the two people of every private chat that
-- already has messages, marked read up to its newest message, so nobody
-- sees every old private chat as unread.
INSERT INTO chat_reads (user_id, channel, last_read_message_id)
SELECT u.id, c.channel, c.last_id
FROM (
    SELECT DISTINCT ON (channel) channel, id AS last_id
    FROM chat_messages
    WHERE channel LIKE 'chat.pm.%' AND deleted_at IS NULL
    ORDER BY channel, id DESC
) c
CROSS JOIN LATERAL unnest(string_to_array(substr(c.channel, length('chat.pm.') + 1), '_')) AS p(user_uuid)
JOIN users u ON u.uuid = p.user_uuid
ON CONFLICT (user_id, channel) DO NOTHING;
