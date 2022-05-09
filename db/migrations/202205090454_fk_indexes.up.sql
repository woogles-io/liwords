BEGIN;

-- a one-to-one between profiles <-> users
CREATE UNIQUE INDEX IF NOT EXISTS
    profiles_user_id_idx ON profiles(user_id);

COMMIT;