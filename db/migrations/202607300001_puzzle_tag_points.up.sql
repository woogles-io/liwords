BEGIN;

-- macondo v0.13.3 added a ninth PuzzleTag, POINTS (enum value 8), which fires
-- when the answer is the top-scoring play by at least score_margin points.
--
-- Puzzle tags are stored by *name*: StorePuzzles resolves tag.String() against
-- puzzle_tag_titles to get an id. A tag with no row here makes that lookup
-- return NULL, and puzzle_tags.tag_id is NOT NULL -- so puzzle generation dies
-- with a 23502 rather than skipping the unknown tag.
--
-- puzzle_tag_titles.tag_title has no unique constraint, so this is guarded with
-- NOT EXISTS rather than ON CONFLICT DO NOTHING (which would only catch a
-- primary-key collision on the serial id, and never fire).
INSERT INTO puzzle_tag_titles (tag_title)
SELECT 'POINTS'
WHERE NOT EXISTS (
    SELECT 1 FROM puzzle_tag_titles WHERE tag_title = 'POINTS'
);

COMMIT;
