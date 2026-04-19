BEGIN;

-- active_game_events was added in 202502280432 but never used in production.
DROP TABLE IF EXISTS active_game_events;

-- game_turns: ephemeral per-turn event log for in-progress games.
-- Each row holds one protojson-marshaled ipc.GameEvent (stored as jsonb for debuggability).
-- Rows are deleted atomically when a game ends and its GameHistory is confirmed uploaded to S3.
CREATE TABLE game_turns (
    game_uuid  text        NOT NULL,
    turn_idx   int4        NOT NULL,
    event      jsonb       NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (game_uuid, turn_idx)
);

CREATE INDEX idx_game_turns_game ON game_turns (game_uuid, turn_idx DESC);

COMMIT;
