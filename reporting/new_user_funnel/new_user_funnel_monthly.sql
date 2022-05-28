--updated 2/27 to only include games played within 14 days of signup.
--updated 9/4 to correctly determine when opponent is a bot.
WITH all_games AS
((SELECT
  games.player0_id AS player,
  games.player1_id AS opponent
FROM games LEFT JOIN users ON games.player0_id = users.id
WHERE games.created_at-users.created_at < interval '14 days')
UNION ALL
(SELECT
  games.player1_id AS player,
  games.player0_id AS opponent
FROM games LEFT JOIN users ON games.player1_id = users.id
WHERE games.created_at-users.created_at < interval '14 days')),
human_vs_human_games AS
((SELECT
  games.player0_id AS player,
  games.player1_id AS opponent
FROM games LEFT JOIN users u1 ON games.player0_id = u1.id
  LEFT JOIN users u2 ON games.player1_id = u2.id
  WHERE games.created_at-u1.created_at < interval '14 days'
  AND NOT (u2.internal_bot OR u2.id IN (42,43,44,45,46)))
UNION ALL
(SELECT
  games.player1_id AS player,
  games.player0_id AS opponent
FROM games LEFT JOIN users u1 ON games.player1_id = u1.id
  LEFT JOIN users u2 ON games.player0_id = u2.id
  WHERE games.created_at-u1.created_at < interval '14 days'
  AND NOT (u2.internal_bot OR u2.id IN (42,43,44,45,46)))
 ),
all_stats AS
(SELECT
  all_games.player,
  users.username,
  COUNT(*) AS games_played,
  COUNT(DISTINCT opponent) AS num_of_opponents  
FROM all_games
LEFT JOIN users ON all_games.player = users.id
GROUP BY 1,2),
human_vs_human_stats AS
(SELECT
  human_vs_human_games.player,
  users.username,
  COUNT(*) AS games_played_against_humans,
  COUNT(DISTINCT opponent) AS number_of_human_opponents  
FROM human_vs_human_games
LEFT JOIN users ON human_vs_human_games.player = users.id
GROUP BY 1,2),
reporting AS
(SELECT
  DATE_TRUNC('month',created_at) AS month_joined,
  COUNT(DISTINCT id) AS new_user_count,
  SUM(CASE WHEN all_stats.games_played > 0 THEN 1
	 ELSE 0 END) AS played_at_least_one_game_count,
  SUM(CASE WHEN human_vs_human_stats.games_played_against_humans > 0 THEN 1
	 ELSE 0 END) AS played_at_least_one_human_count,
  SUM(CASE WHEN human_vs_human_stats.number_of_human_opponents > 1 THEN 1
	 ELSE 0 END) AS played_at_least_two_different_people_count
FROM public.users 
LEFT JOIN all_stats ON users.id = all_stats.player
LEFT JOIN human_vs_human_stats ON users.id = human_vs_human_stats.player
GROUP BY 1)

SELECT
    *,
	TRUNC(100.0*played_at_least_one_game_count/new_user_count,1) AS played_at_least_one_game_frac,
	TRUNC(100.0*played_at_least_one_human_count/new_user_count,1) AS played_at_least_one_human_frac,
	TRUNC(100.0*played_at_least_two_different_people_count/new_user_count,1) AS played_at_least_two_different_people_frac
FROM reporting
ORDER BY 1 DESC