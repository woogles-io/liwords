WITH bot_users AS
(SELECT
   id,
   internal_bot OR id IN (42,43,44,45,46) AS is_bot
FROM public.users),
games_with_bot_flags AS
(SELECT
   DATE_TRUNC('day', created_at) AS date,
   b1.is_bot OR b2.is_bot AS bot_game,
   game_end_reason
 FROM public.games
 LEFT JOIN bot_users b1 ON games.player0_id=b1.id
 LEFT JOIN bot_users b2 ON games.player1_id=b2.id
 )
 
SELECT
	date,
	COUNT(*) AS game_count,
	SUM(CASE WHEN bot_game THEN 0 ELSE 1 END) AS pvp_game_count,
	SUM(CASE WHEN game_end_reason = 2 THEN 1 ELSE 0 END) AS regular_ending_count,
	TRUNC(100.0*SUM(CASE WHEN game_end_reason=2 THEN 1 ELSE 0 END)/COUNT(*),1) AS normal_ending_pct,
	SUM(CASE WHEN game_end_reason IN (4,5,8) THEN 1 ELSE 0 END) AS ended_early_count,
	SUM(CASE WHEN game_end_reason = 1 THEN 1 ELSE 0 END) AS timed_out_count,
	SUM(CASE WHEN game_end_reason = 3 THEN 1 ELSE 0 END) AS consecutive_zero_ending_count,
	SUM(CASE WHEN game_end_reason = 4 THEN 1 ELSE 0 END) AS resigned_count,
	SUM(CASE WHEN game_end_reason = 5 THEN 1 ELSE 0 END) AS aborted_count,
	SUM(CASE WHEN game_end_reason = 6 THEN 1 ELSE 0 END) AS triple_challenge_ending_count,
	SUM(CASE WHEN game_end_reason = 7 THEN 1 ELSE 0 END) AS cancelled_count,
	SUM(CASE WHEN game_end_reason = 8 THEN 1 ELSE 0 END) AS force_forfeit_count
FROM games_with_bot_flags
GROUP BY 1
ORDER BY 1 DESC