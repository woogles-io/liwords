WITH puzzle_exhaustion AS
(SELECT
   users.id AS user_id,
   users.username,
   puzzles.lexicon,
   COUNT(DISTINCT puzzle_attempts.puzzle_id) AS puzzles_seen,
   COUNT(DISTINCT CASE WHEN puzzle_attempts.correct IS NOT NULL 
		 THEN puzzle_id ELSE null END) AS puzzles_attempted
FROM puzzle_attempts
LEFT JOIN puzzles ON puzzles.id=puzzle_attempts.puzzle_id
LEFT JOIN users ON puzzle_attempts.user_id=users.id
LEFT JOIN profiles ON users.id = profiles.user_id 
GROUP BY 1,2,3
ORDER BY 5 DESC
LIMIT 100)

SELECT
  puzzle_exhaustion.username,
  puzzle_exhaustion.lexicon,
  puzzle_exhaustion.puzzles_seen,
  puzzle_exhaustion.puzzles_attempted,
  CASE WHEN puzzle_exhaustion.lexicon = 'CSW21' THEN (profiles.ratings->'Data'->'CSW19.puzzle.corres'->'r')::numeric 
       WHEN puzzle_exhaustion.lexicon = 'NWL20' THEN (profiles.ratings->'Data'->'NWL18.puzzle.corres'->'r')::numeric 
       END AS rating
FROM puzzle_exhaustion
LEFT JOIN profiles ON puzzle_exhaustion.user_id = profiles.user_id
