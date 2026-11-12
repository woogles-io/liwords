ALTER TABLE broadcasts
    DROP COLUMN next_poll_at,
    DROP COLUMN games_version;

DROP TABLE broadcast_feeds;
