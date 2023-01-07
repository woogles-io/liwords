WITH base_query AS
(SELECT
	puzzles.lexicon,
	DATE_TRUNC('month',puzzle_attempts.created_at) AS month,
	COUNT(DISTINCT puzzle_attempts.user_id) AS monthly_puzzle_seers,
	COUNT(DISTINCT CONCAT(puzzle_attempts.user_id,'-',puzzle_attempts.puzzle_id)) AS monthly_puzzles_seen,
	COUNT(DISTINCT CASE WHEN puzzle_attempts.correct IS NOT NULL 
		    THEN puzzle_attempts.user_id
		 	ELSE NULL END) AS monthly_active_puzzle_solvers,
	COUNT(DISTINCT CASE WHEN puzzle_attempts.correct IS NOT NULL
		    THEN CONCAT(puzzle_attempts.user_id,'-',puzzle_attempts.puzzle_id)
		 	ELSE NULL END) AS monthly_puzzles_solved
FROM puzzle_attempts
LEFT JOIN puzzles on puzzle_attempts.puzzle_id = puzzles.id
GROUP BY 1,2)

SELECT
  *,
  ROUND(100.0*monthly_active_puzzle_solvers/monthly_puzzle_seers,1) AS puzzle_seers_who_attempt_pct,
  ROUND(100.0*monthly_active_puzzle_solvers/monthly_puzzle_seers,1) AS puzzles_seen_that_are_attempted_pct
FROM base_query
ORDER BY 1,2 DESC