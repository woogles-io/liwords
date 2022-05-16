BEGIN;

DROP INDEX IF EXISTS profiles_user_id_idx;
DROP INDEX IF EXISTS idx_followings_follower;
DROP INDEX IF EXISTS idx_blockings_blocker;

DROP INDEX IF EXISTS idx_puzzle_attempts_user;
DROP INDEX IF EXISTS idx_puzzle_games;
DROP INDEX IF EXISTS idx_puzzle_generationids;
DROP INDEX IF EXISTS idx_puzzle_authorid;
DROP INDEX IF EXISTS idx_puzzletags_tagid;
DROP INDEX IF EXISTS idx_puzzlevotes_user_id;

COMMIT;