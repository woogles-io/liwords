BEGIN;

DROP INDEX IF EXISTS profiles_user_id_idx;
DROP INDEX IF EXISTS idx_followings_follower;
DROP INDEX IF EXISTS idx_blockings_blocker;

COMMIT;