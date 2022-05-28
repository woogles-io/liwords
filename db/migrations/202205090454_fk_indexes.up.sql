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

-- Other missing indexes
-- See https://www.cybertec-postgresql.com/en/index-your-foreign-key/

CREATE INDEX IF NOT EXISTS idx_puzzle_attempts_user ON public.puzzle_attempts
    USING btree(user_id);

CREATE INDEX IF NOT EXISTS idx_puzzle_games ON public.puzzles
    USING btree(game_id);

CREATE INDEX IF NOT EXISTS idx_puzzle_generationids ON public.puzzles
    USING btree(generation_id);

CREATE INDEX IF NOT EXISTS idx_puzzle_authorid ON public.puzzles
    USING btree(author_id);

CREATE INDEX IF NOT EXISTS idx_puzzletags_tagid ON public.puzzle_tags
    USING btree(tag_id);

CREATE INDEX IF NOT EXISTS idx_puzzlevotes_user_id ON public.puzzle_votes
    USING btree(user_id);

COMMIT;