select
  username,
  (select COUNT(*) from puzzle_attempts where user_id = users.id) as puzzles_attempted,
  (select COUNT(*) from puzzle_attempts where user_id = users.id AND correct = true) as correct_puzzles,
  100.0*(select COUNT(*) from puzzle_attempts where user_id = users.id AND correct = true)/(select COUNT(*) from puzzle_attempts where user_id = users.id) AS puzzle_solved_pct,
  ratings->'Data'->'CSW19.puzzle.corres'->'r' as csw_rating,
  ratings->'Data'->'NWL18.puzzle.corres'->'r' as nwl_rating
from users, profiles 
where
  users.id = profiles.user_id 
AND
  users.id in (select distinct(user_id) from puzzle_attempts)
order by csw_rating desc;