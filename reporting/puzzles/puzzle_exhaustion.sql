SELECT
   users.username,
   puzzles.lexicon,
   COUNT(DISTINCT puzzle_attempts.puzzle_id)
FROM puzzle_attempts
LEFT JOIN puzzles ON puzzles.id=puzzle_attempts.puzzle_id
LEFT JOIN users ON puzzle_attempts.user_id=users.id
GROUP BY 1,2
ORDER BY 3 DESC
LIMIT 20