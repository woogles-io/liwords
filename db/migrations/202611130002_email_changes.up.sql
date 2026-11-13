BEGIN;

-- Every change of a user's email, by the user or by an admin. Only admins
-- read it.
CREATE TABLE IF NOT EXISTS email_changes (
    id BIGSERIAL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users (id),
    old_email text NOT NULL,
    new_email text NOT NULL,
    -- clock_timestamp, not NOW: NOW is when the transaction began, and a
    -- change that waited on the user's row lock would sort before the one it
    -- waited for.
    changed_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    changed_by integer NOT NULL REFERENCES users (id)
);

CREATE INDEX IF NOT EXISTS email_changes_user_id_changed_at_idx
    ON email_changes (user_id, changed_at DESC);

COMMIT;
