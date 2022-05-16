SELECT
	puzzles.lexicon,
	DATE_TRUNC('day',puzzle_attempts.created_at) AS day,
	COUNT(DISTINCT puzzle_attempts.user_id) AS daily_active_puzzle_solvers,
	COUNT(DISTINCT CONCAT(puzzle_attempts.user_id,'-',puzzle_attempts.puzzle_id)) AS daily_puzzles_attempted
FROM puzzle_attempts
LEFT JOIN puzzles on puzzle_attempts.puzzle_id = puzzles.id
GROUP BY 1,2
ORDER BY 1,2 DESC