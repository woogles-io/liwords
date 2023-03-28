BEGIN;

CREATE INDEX IF NOT EXISTS idx_user_actions_user_id ON public.user_actions
    USING btree(user_id);

COMMIT;