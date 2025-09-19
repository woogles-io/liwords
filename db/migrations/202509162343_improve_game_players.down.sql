BEGIN;

-- Revert to original game_players structure
DROP TABLE IF EXISTS game_players;

CREATE TABLE game_players (
    game_id integer NOT NULL,
    player_id integer NOT NULL,
    player_index SMALLINT,
    FOREIGN KEY (game_id) REFERENCES games (id),
    FOREIGN KEY (player_id) REFERENCES users (id),
    PRIMARY KEY (game_id, player_id)
);

COMMIT;