-- Top players by game count: all-time, last 365 days, last 30 days.
-- Bot rule matches omgwords_top_users.sql: internal_bot flag plus early bot ids 42-46,230.
WITH all_games AS (
  SELECT player0_id AS player, created_at FROM games
  UNION ALL
  SELECT player1_id AS player, created_at FROM games
),
stats AS (
  SELECT
    ag.player,
    u.username,
    COUNT(*) AS all_time_games,
    COUNT(*) FILTER (WHERE ag.created_at > now() - interval '1 year') AS last_year_games,
    COUNT(*) FILTER (WHERE ag.created_at > now() - interval '1 month') AS last_month_games
  FROM all_games ag
  JOIN users u ON ag.player = u.id
  WHERE NOT u.internal_bot AND u.id NOT IN (42,43,44,45,46,230,6216)
  GROUP BY 1,2
)

(SELECT 'all_time' AS period, player, username, all_time_games AS games
 FROM stats ORDER BY all_time_games DESC LIMIT 15)
UNION ALL
(SELECT 'last_year' AS period, player, username, last_year_games AS games
 FROM stats ORDER BY last_year_games DESC LIMIT 15)
UNION ALL
(SELECT 'last_month' AS period, player, username, last_month_games AS games
 FROM stats ORDER BY last_month_games DESC LIMIT 15)
ORDER BY period, games DESC;
