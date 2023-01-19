BEGIN;

CREATE TABLE IF NOT EXISTS annotated_game_metadata (
    game_uuid text UNIQUE NOT NULL,
    creator_uuid text NOT NULL,
    private_broadcast boolean NOT NULL default TRUE,
    -- done - we are done annotating this game.
    done boolean NOT NULL default FALSE
);

CREATE INDEX IF NOT EXISTS idx_anno_game_creator ON public.annotated_game_metadata USING btree(creator_uuid);


COMMIT;