WITH all_games AS
((SELECT
  player0_id AS player,
  player1_id AS opponent
FROM games)
UNION ALL
(SELECT
  player1_id AS player, 
  player0_id AS opponent
FROM games)),
human_vs_human_games AS
((SELECT
  player0_id AS player,
  player1_id AS opponent
FROM games
WHERE player1_id NOT IN (42,43,44,45,46,230))
UNION ALL
(SELECT
  player1_id AS player, 
  player0_id AS opponent
FROM games
WHERE player0_id NOT IN (42,43,44,45,46,230))),
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
  COUNT(DISTINCT opponent) AS num_of_human_opponents  
FROM human_vs_human_games
LEFT JOIN users ON human_vs_human_games.player = users.id
GROUP BY 1,2)

SELECT
  all_stats.player,
  all_stats.username,
  all_stats.games_played,
  all_stats.num_of_opponents,
  COALESCE(human_vs_human_stats.games_played_against_humans,0)
    AS games_played_against_humans,
  COALESCE(all_stats.games_played-
	human_vs_human_stats.games_played_against_humans,0) AS games_played_against_bots,
  COALESCE(human_vs_human_stats.num_of_human_opponents,0) AS number_of_human_opponents
FROM all_stats LEFT JOIN human_vs_human_stats
  ON all_stats.player = human_vs_human_stats.player
ORDER by 3 DESC