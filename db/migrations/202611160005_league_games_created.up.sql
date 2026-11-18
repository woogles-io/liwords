-- Set once every game of a season exists. Starting a season creates its
-- games one pair at a time, which can take minutes; a start cut off partway
-- leaves the season ACTIVE with this unset, and the next run creates the
-- games still missing.
ALTER TABLE league_seasons ADD COLUMN games_created_at TIMESTAMPTZ;

-- Every season already past its start had its games created (or never will:
-- cancelled), so none of them is picked up as a start to finish.
-- 0 SCHEDULED, 1 ACTIVE, 2 COMPLETED, 3 CANCELLED, 4 REGISTRATION_OPEN.
UPDATE league_seasons
SET games_created_at = COALESCE(started_at, updated_at, NOW())
WHERE status IN (1, 2, 3);
