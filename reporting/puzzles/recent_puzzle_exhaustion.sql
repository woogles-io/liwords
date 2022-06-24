WITH base_query AS
(SELECT
   users.username,
   puzzles.lexicon,
   COUNT(DISTINCT puzzle_attempts.puzzle_id) AS puzzles_seen,
   COUNT(DISTINCT 
     CASE WHEN puzzle_attempts.correct IS NOT NULL 
		  THEN puzzle_attempts.puzzle_id ELSE NULL END) AS puzzles_attempted,
   COUNT(DISTINCT 
     CASE WHEN puzzle_attempts.correct = 'true'
		  THEN puzzle_attempts.puzzle_id ELSE NULL END) AS puzzles_solved
FROM puzzle_attempts
LEFT JOIN puzzles ON puzzles.id=puzzle_attempts.puzzle_id
LEFT JOIN users ON puzzle_attempts.user_id=users.id
WHERE puzzle_attempts.created_at > CURRENT_TIMESTAMP-INTERVAL '1' day
GROUP BY 1,2
ORDER BY 4 DESC)

SELECT
  *,
  puzzles_seen-puzzles_attempted AS puzzles_skipped,
  ROUND(100.0*puzzles_attempted/puzzles_seen,1) AS seen_puzzle_attempted_pct,
  CASE WHEN base_query.puzzles_attempted != 0 THEN ROUND(100.0*puzzles_solved/puzzles_attempted,1) 
       ELSE 0 END AS puzzle_solved_pct
FROM base_query
ORDER BY 3 DESC
LIMIT 100