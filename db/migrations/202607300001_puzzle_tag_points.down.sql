BEGIN;

-- puzzle_tags.tag_id references puzzle_tag_titles(id), so drop the taggings
-- before the title itself.
DELETE FROM puzzle_tags
WHERE tag_id IN (SELECT id FROM puzzle_tag_titles WHERE tag_title = 'POINTS');

DELETE FROM puzzle_tag_titles WHERE tag_title = 'POINTS';

COMMIT;
