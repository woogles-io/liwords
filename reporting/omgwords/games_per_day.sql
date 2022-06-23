WITH bot_users AS
(SELECT
   id,
   internal_bot OR id IN (42,43,44,45,46) AS is_bot
FROM public.users),
games_with_bot_flags AS
(SELECT
   DATE_TRUNC('day', created_at) AS date,
   game_end_reason,
   CASE WHEN NOT(b1.is_bot OR b2.is_bot) THEN 1 ELSE 0 END AS bot_game
 FROM public.games
 LEFT JOIN bot_users b1 ON games.player0_id=b1.id
 LEFT JOIN bot_users b2 ON games.player1_id=b2.id
 )
 
SELECT
	date,
	COUNT(*) AS game_count,
	SUM(bot_game) AS pvp_game_count,
	SUM(CASE WHEN game_end_reason=2 THEN 1 ELSE 0 END) AS
	  regular_game_end_count,
	SUM(CASE WHEN game_end_reason=1 THEN 1 ELSE 0 END) AS
	  timed_out_count,
	SUM(CASE WHEN game_end_reason=3 THEN 1 ELSE 0 END) AS
	  consecutive_zero_ending_count,
	SUM(CASE WHEN game_end_reason=4 THEN 1 ELSE 0 END) AS
	  resigned_count,
	SUM(CASE WHEN game_end_reason=5 THEN 1 ELSE 0 END) AS
	  aborted_count,
	SUM(CASE WHEN game_end_reason=6 THEN 1 ELSE 0 END) AS
	  triple_challenge_ending_count,
	SUM(CASE WHEN game_end_reason=7 THEN 1 ELSE 0 END) AS
	  cancelled_count,
	SUM(CASE WHEN game_end_reason=8 THEN 1 ELSE 0 END) AS
	  game_ended_with_forced_forfeit_count
FROM games_with_bot_flags
GROUP BY 1
ORDER BY 1 DESC