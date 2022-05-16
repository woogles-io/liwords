SELECT
   users.username,
   puzzles.lexicon,
   COUNT(DISTINCT puzzle_attempts.puzzle_id)
FROM puzzle_attempts
LEFT JOIN puzzles ON puzzles.id=puzzle_attempts.puzzle_id
LEFT JOIN users ON puzzle_attempts.user_id=users.id
WHERE puzzle_attempts.created_at > CURRENT_TIMESTAMP-INTERVAL '1' day
GROUP BY 1,2
ORDER BY 3 DESC
LIMIT 50