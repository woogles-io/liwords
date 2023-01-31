BEGIN;

CREATE INDEX IF NOT EXISTS idx_game_creation_date ON public.games
    USING btree(created_at);

COMMIT;