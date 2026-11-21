-- Chat history, moving from Redis streams (which kept 14 days) to Postgres,
-- kept indefinitely. Messages are ordered by (created_at, id): the backfill
-- copies old messages after new ones have started arriving, so id alone
-- isn't in time order.
CREATE TABLE chat_messages (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- The full channel name, e.g. chat.lobby, chat.game.<id>,
    -- chat.pm.<uuid>_<uuid>, chat.tournament.<id>.
    channel text NOT NULL,
    -- NULL for lines the server posts itself (challenge results), and for a
    -- sender whose account is gone.
    sender_id integer REFERENCES users(id) ON DELETE SET NULL,
    -- The sender's username when the message was sent; '' for server lines.
    sender_username text NOT NULL,
    message text NOT NULL,
    created_at timestamptz NOT NULL,
    -- The Redis stream ID ("<ms>-<seq>") of a message also written to
    -- Redis. Unique within a channel, so the backfill can be re-run.
    legacy_id text,
    -- Set by a moderator's delete; the message is kept for review.
    deleted_at timestamptz,
    deleted_by integer REFERENCES users(id) ON DELETE SET NULL
);

-- A channel's messages, in order (history pages).
CREATE INDEX chat_messages_channel_created_idx ON chat_messages (channel, created_at, id);
-- One user's messages (moderator chat review).
CREATE INDEX chat_messages_sender_created_idx ON chat_messages (sender_id, created_at)
    WHERE sender_id IS NOT NULL;
CREATE UNIQUE INDEX chat_messages_channel_legacy_idx ON chat_messages (channel, legacy_id)
    WHERE legacy_id IS NOT NULL;

-- How far each user has read in a channel. Private-chat rows also make the
-- user's list of private chats.
CREATE TABLE chat_reads (
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel text NOT NULL,
    -- The newest message read; NULL if none yet.
    last_read_message_id bigint REFERENCES chat_messages(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, channel)
);
