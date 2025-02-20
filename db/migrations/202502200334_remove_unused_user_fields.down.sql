BEGIN;

ALTER TABLE public.users
ADD COLUMN is_admin boolean default false,
ADD COLUMN is_director boolean default false,
ADD COLUMN is_mod boolean default false;

COMMIT;