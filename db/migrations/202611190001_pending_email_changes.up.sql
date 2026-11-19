BEGIN;

-- An email change a user asked for, waiting for them to follow the link sent
-- to the new address. A user has at most one; asking again replaces it.
CREATE TABLE public.pending_email_changes (
    user_id integer PRIMARY KEY REFERENCES public.users(id) ON DELETE CASCADE,
    new_email text NOT NULL,
    token text NOT NULL UNIQUE,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT now()
);

COMMIT;
