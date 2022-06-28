WITH
  attempts
AS
(SELECT
  id,
  uuid,
  lexicon,
  rating->'r' AS rating,
  rating->'rd' AS rating_deviation,
  (SELECT COUNT(*) from puzzle_attempts where puzzle_attempts.puzzle_id = puzzles.id
    AND puzzle_attempts.correct IS NOT NULL) AS number_of_attempts
FROM puzzles
ORDER BY number_of_attempts DESC)
SELECT
  lexicon,
  number_of_attempts,
  COUNT(*) AS hist
FROM attempts
GROUP BY 1,2
ORDER BY 1,2 ASC