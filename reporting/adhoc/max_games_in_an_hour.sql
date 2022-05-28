WITH q AS
(SELECT
	DATE_TRUNC('hour', created_at) AS hour_of_game,
	COUNT(*) AS game_count,
	SUM(CASE WHEN ((player0_id != 230) AND
		(player1_id != 230)) THEN 1 ELSE 0 END) AS pvp_game_count
FROM public.games
GROUP BY 1)
SELECT
    DATE_TRUNC('month',hour_of_game) AS month,
	MAX(game_count) AS max_game_count,
	MAX(pvp_game_count) AS max_pvp_game_count
FROM q
GROUP BY 1