BEGIN;

-- a one-to-one between profiles <-> users
CREATE UNIQUE INDEX IF NOT EXISTS
    profiles_user_id_idx ON profiles(user_id);

-- we already have a unique index on the combination of both columns,
-- but we should add one on the second column for each of these
-- (the first column index is implied by the unique index)

CREATE INDEX IF NOT EXISTS idx_followings_follower ON public.followings
    USING btree(follower_id);

CREATE INDEX IF NOT EXISTS idx_blockings_blocker ON public.blockings
    USING btree(blocker_id);

COMMIT;