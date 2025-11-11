-- Remove email verification columns from users table
DROP INDEX IF EXISTS public.users_unverified_cleanup_idx;
DROP INDEX IF EXISTS public.users_verification_token_idx;

ALTER TABLE public.users DROP COLUMN IF EXISTS verification_expires_at;
ALTER TABLE public.users DROP COLUMN IF EXISTS verification_token;
ALTER TABLE public.users DROP COLUMN IF EXISTS verified;
