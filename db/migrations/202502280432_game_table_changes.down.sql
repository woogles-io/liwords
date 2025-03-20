BEGIN;

ALTER TABLE games DROP COLUMN game_request;
ALTER TABLE games DROP COLUMN history_in_s3;

DROP TABLE game_players;
DROP TABLE active_game_events;

COMMIT;