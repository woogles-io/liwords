BEGIN;

ALTER TABLE public.users
DROP COLUMN is_admin,
DROP COLUMN is_director,
DROP COLUMN is_mod;

COMMIT;