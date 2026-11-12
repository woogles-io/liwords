-- The last feed each broadcast's poller fetched, so pages keep working when
-- the director's feed URL goes away, and request paths never fetch it.
CREATE TABLE broadcast_feeds (
    broadcast_id         INTEGER PRIMARY KEY REFERENCES broadcasts(id) ON DELETE CASCADE,
    -- The format raw_gz was parsed with when it was fetched.
    format               TEXT NOT NULL DEFAULT '',
    -- The feed as fetched, gzipped. NULL until the first successful fetch.
    raw_gz               BYTEA,
    -- sha256 of format and the raw feed. Servers cache the parsed feed under it.
    content_hash         TEXT NOT NULL DEFAULT '',
    -- The first division's current round, for the listings.
    current_round        INTEGER NOT NULL DEFAULT 0,
    fetched_at           TIMESTAMPTZ,
    changed_at           TIMESTAMPTZ,
    last_error           TEXT NOT NULL DEFAULT '',
    last_error_at        TIMESTAMPTZ,
    consecutive_failures INTEGER NOT NULL DEFAULT 0
);

-- next_poll_at is claimed by one server at a time (see ClaimBroadcastForPolling).
-- games_version goes up whenever claims, slots or annotations change, so
-- servers' short response caches are keyed by it.
ALTER TABLE broadcasts
    ADD COLUMN next_poll_at  TIMESTAMPTZ,
    ADD COLUMN games_version BIGINT NOT NULL DEFAULT 0;

UPDATE broadcasts
SET next_poll_at = last_polled_at + make_interval(secs => poll_interval_seconds)
WHERE last_polled_at IS NOT NULL;
