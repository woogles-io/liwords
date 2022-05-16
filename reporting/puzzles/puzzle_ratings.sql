SELECT
  id,
  uuid,
  lexicon,
  rating->'r' AS rating,
  rating->'rd' AS rating_deviation,
  (SELECT COUNT(*) from puzzle_attempts where puzzle_attempts.puzzle_id = puzzles.id) AS number_of_attempts
FROM puzzles
ORDER BY number_of_attempts DESC