BEGIN;

CREATE TABLE IF NOT EXISTS annotated_game_editors (
    game_uuid text UNIQUE NOT NULL,
    player_uuid text UNIQUE NOT NULL,
    -- an editor should not have more than X unfinished games before
    -- attempting to create another one
    finished bool
);

COMMIT;