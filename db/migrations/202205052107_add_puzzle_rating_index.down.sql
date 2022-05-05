BEGIN;

CREATE INDEX idx_puzzles_rating ON public.puzzles USING btree ((rating->>'r')::FLOAT);

COMMIT;