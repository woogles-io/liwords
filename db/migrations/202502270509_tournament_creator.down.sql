BEGIN;

ALTER TABLE public.tournaments DROP COLUMN created_by;

COMMIT;