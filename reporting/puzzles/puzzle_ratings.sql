WITH base_query AS
(SELECT
  id,
  uuid,
  lexicon,
  rating->'r' AS rating,
  rating->'rd' AS rating_deviation,
  (SELECT COUNT(*) FROM puzzle_attempts WHERE puzzle_attempts.puzzle_id = puzzles.id) AS times_seen,
  (SELECT COUNT(*) FROM puzzle_attempts WHERE puzzle_attempts.puzzle_id = puzzles.id 
     AND puzzle_attempts.correct IS NOT NULL) AS times_attempted,
  (SELECT COUNT(*) FROM puzzle_attempts WHERE puzzle_attempts.puzzle_id = puzzles.id 
     AND puzzle_attempts.correct = 'true') AS times_correct
FROM puzzles)

SELECT
  *,
  times_seen-times_attempted AS times_skipped,
  CASE WHEN times_attempted != 0 
    THEN ROUND(100.0*(times_seen-times_attempted)/times_seen,1)
	ELSE 0 END AS skip_pct,
  CASE WHEN times_attempted != 0
    THEN ROUND(100.0*times_correct/times_attempted,1)
	ELSE 0 END AS correct_attempt_pct
FROM base_query
ORDER BY skip_pct DESC