BEGIN;
-- Add email verification columns to users table
ALTER TABLE public.users ADD COLUMN verified BOOLEAN DEFAULT TRUE NOT NULL;
ALTER TABLE public.users ADD COLUMN verification_token VARCHAR(255);
ALTER TABLE public.users ADD COLUMN verification_expires_at TIMESTAMP WITH TIME ZONE;

-- Index for token lookups
CREATE INDEX users_verification_token_idx ON public.users USING btree (verification_token) WHERE verification_token IS NOT NULL;

-- Index for cleanup of unverified users
CREATE INDEX users_unverified_cleanup_idx ON public.users USING btree (created_at) WHERE verified = FALSE;
COMMIT;