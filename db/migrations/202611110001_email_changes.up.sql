BEGIN;

-- Every change of a user's email, by the user or by an admin. Only admins
-- read it.
CREATE TABLE IF NOT EXISTS email_changes (
    id BIGSERIAL PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users (id),
    old_email text NOT NULL,
    new_email text NOT NULL,
    changed_at timestamptz NOT NULL DEFAULT NOW(),
    changed_by integer NOT NULL REFERENCES users (id)
);

CREATE INDEX IF NOT EXISTS email_changes_user_id_changed_at_idx
    ON email_changes (user_id, changed_at DESC);

COMMIT;
