BEGIN;

CREATE INDEX IF NOT EXISTS idx_puzzles_lexicon ON public.puzzles USING btree (lexicon);

COMMIT;