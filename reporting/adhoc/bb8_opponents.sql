WITH bb8_games AS (
  SELECT player1_id AS opponent, created_at FROM games WHERE player0_id = 6216
  UNION ALL
  SELECT player0_id AS opponent, created_at FROM games WHERE player1_id = 6216
)
SELECT
  bg.opponent,
  u.username,
  u.internal_bot,
  COUNT(*) AS games_vs_bb8,
  MIN(bg.created_at) AS first_game,
  MAX(bg.created_at) AS last_game
FROM bb8_games bg
JOIN users u ON bg.opponent = u.id
GROUP BY 1,2,3
ORDER BY games_vs_bb8 DESC
LIMIT 20;
