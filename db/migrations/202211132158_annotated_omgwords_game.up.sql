BEGIN;

CREATE TABLE IF NOT EXISTS annotated_game_metadata (
    game_uuid text UNIQUE NOT NULL,
    creator_uuid text UNIQUE NOT NULL,
    -- this can be null if the annotated game is not associated with 
    -- any events
    -- event_uuid text UNIQUE,  

    private_broadcast boolean,
    -- done - we are done annotating this game.
    done boolean

    -- gcg_stream_link text, -- a link to a GCG emitting website

    -- an editor should not have more than X unfinished games before
    -- attempting to create another one
    -- finished bool
);

-- CREATE TABLE IF NOT EXISTS annotated_game_event (
--     id BIGSERIAL PRIMARY KEY,
--     uuid text UNIQUE NOT NULL,
--     creator_uuid text UNIQUE NOT NULL,
--     short_description text,
--     long_description text,
--     editor_ids jsonb,
--     created_at timestamptz NOT NULL
-- )

COMMIT;