BEGIN;

ALTER TABLE public.profiles DROP COLUMN IF EXISTS "silent_mode";

COMMIT;