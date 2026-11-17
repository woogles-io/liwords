-- Messages to send (emails, Discord posts), written with the change that
-- calls for them and sent by the send_outbox job (pkg/outbox). A row is
-- marked sent as soon as it goes out, so a send cut off by a deploy carries
-- on where it stopped.
CREATE TABLE outbox (
    id         BIGSERIAL PRIMARY KEY,
    -- Says what the message is for ('league:<season>:started:<user>'), so
    -- that writing the same messages again adds nothing.
    dedupe_key TEXT NOT NULL UNIQUE,
    channel    TEXT NOT NULL,
    recipient  TEXT NOT NULL DEFAULT '',
    subject    TEXT NOT NULL DEFAULT '',
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at    TIMESTAMPTZ,
    attempts   INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT ''
);

CREATE INDEX outbox_unsent_idx ON outbox (id) WHERE sent_at IS NULL;

-- Set when the league runner opens registration for a season, cleared in the
-- transaction that writes its registration-open messages. Seasons opened any
-- other way never had those messages, and still don't.
ALTER TABLE league_seasons
    ADD COLUMN registration_notice_pending BOOLEAN NOT NULL DEFAULT FALSE;
