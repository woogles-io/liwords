ALTER TABLE broadcast_games
    ADD COLUMN stats            JSONB,
    ADD COLUMN stats_computed_at TIMESTAMPTZ,
    ADD COLUMN completed_at     TIMESTAMPTZ;

CREATE INDEX idx_broadcast_games_completed_at
    ON broadcast_games (broadcast_id, completed_at DESC NULLS LAST);
