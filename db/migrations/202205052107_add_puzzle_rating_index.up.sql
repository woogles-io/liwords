BEGIN;

CREATE INDEX IF NOT EXISTS idx_puzzles_rating ON public.puzzles USING btree (((rating->>'r')::FLOAT));

COMMIT;