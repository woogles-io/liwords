-- Chat messages reported to the moderators. A report points at its message,
-- which stays in chat_messages (a moderator's delete only marks it), so the
-- reported words are there to review. A person reports a message once.
CREATE TABLE chat_reports (
    id bigserial PRIMARY KEY,
    message_id uuid NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    reporter_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    -- Set when a moderator marks the report handled.
    handled_by integer REFERENCES users(id) ON DELETE SET NULL,
    handled_at timestamptz,
    -- a person's one report of a message; its index also finds a message's
    -- reports
    UNIQUE (message_id, reporter_id)
);

-- The open reports, newest first: the moderators' list.
CREATE INDEX chat_reports_open_idx ON chat_reports (created_at)
    WHERE handled_at IS NULL;
