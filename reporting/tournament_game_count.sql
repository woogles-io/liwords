SELECT
	CASE WHEN (tournament_id IS NOT NULL AND tournament_id !='') THEN 1 ELSE 0 END AS is_tournament_game,
	COUNT(*) AS game_count,
	SUM(CASE WHEN ((player0_id != 230) AND
		(player1_id != 230)) THEN 1 ELSE 0 END) AS pvp_game_count
FROM public.games
GROUP BY 1
