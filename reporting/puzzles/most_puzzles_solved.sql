SELECT
  username,
  (SELECT COUNT(*) FROM puzzle_attempts WHERE user_id = users.id) as puzzles_seen,
  (SELECT COUNT(*) FROM puzzle_attempts WHERE user_id = users.id AND correct IS NOT NULL) as puzzles_attempted,
  (SELECT COUNT(*) FROM puzzle_attempts WHERE user_id = users.id AND correct = true) as puzzles_solved,
  100.0*(SELECT COUNT(*) FROM puzzle_attempts WHERE user_id = users.id AND correct = true)/(SELECT COUNT(*) FROM puzzle_attempts WHERE user_id = users.id) AS puzzle_solved_pct,
  CASE WHEN ratings->'Data'->'CSW19.puzzle.corres'->'r' IS NULL THEN '0'
    ELSE ratings->'Data'->'CSW19.puzzle.corres'->'r' END AS csw_rating,
  CASE WHEN ratings->'Data'->'NWL18.puzzle.corres'->'r' IS NULL THEN '0'
    ELSE ratings->'Data'->'NWL18.puzzle.corres'->'r' END AS nwl_rating
from users, profiles 
where
  users.id = profiles.user_id 
AND
  users.id in (select distinct(user_id) from puzzle_attempts)
order by nwl_rating desc;