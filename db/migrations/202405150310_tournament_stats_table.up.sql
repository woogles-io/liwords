BEGIN;

CREATE TABLE IF NOT EXISTS tournament_stats  (
    tournament_id bigint NOT NULL,
    division_name text NOT NULL,
    player_id text NOT NULL,
    stats jsonb NOT NULL DEFAULT '{}',
    FOREIGN KEY (tournament_id) REFERENCES tournaments(id),
    UNIQUE (tournament_id, division_name, player_id)
);

COMMIT;