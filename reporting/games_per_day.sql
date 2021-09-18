WITH bot_users AS
(SELECT
   id,
   internal_bot OR id IN (42,43,44,45,46) AS is_bot
FROM public.users),
games_with_bot_flags AS
(SELECT
   DATE_TRUNC('day', created_at) AS date,
   CASE WHEN NOT(b1.is_bot OR b2.is_bot) THEN 1 ELSE 0 END AS bot_game
 FROM public.games
 LEFT JOIN bot_users b1 ON games.player0_id=b1.id
 LEFT JOIN bot_users b2 ON games.player1_id=b2.id
 )
 
SELECT
	date,
	COUNT(*) AS game_count,
	SUM(bot_game) AS pvp_game_count
FROM games_with_bot_flags
GROUP BY 1
ORDER BY 1 DESC