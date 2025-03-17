BEGIN;

ALTER TABLE games ADD COLUMN game_request jsonb NOT NULL DEFAULT '{}';
ALTER TABLE games ADD COLUMN history_in_s3 boolean NOT NULL DEFAULT false;

CREATE TABLE game_players (
    game_id integer NOT NULL,
    player_id integer NOT NULL,
    player_index SMALLINT, -- 0 or 1 for first or second
    FOREIGN KEY (game_id) REFERENCES games (id),
    FOREIGN KEY (player_id) REFERENCES users (id),
    PRIMARY KEY (game_id, player_id)
);

CREATE TABLE active_game_events (
    game_id integer NOT NULL,
    event_idx integer NOT NULL,
    event JSONB NOT NULL DEFAULT '{}',

    FOREIGN KEY (game_id) REFERENCES games (id)
);


COMMIT;