BEGIN;

ALTER TABLE public.tournaments
    ADD COLUMN created_by integer REFERENCES public.users(id);

COMMIT;