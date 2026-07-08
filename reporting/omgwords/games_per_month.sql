WITH bot_users AS (
SELECT
   id,
   username,
   internal_bot OR id IN (42,43,44,45,46) AS is_bot
FROM public.users
),

games_with_bot_flags AS (
SELECT
    DATE_TRUNC('month', g.created_at) AS month,
    g.game_end_reason,
    CASE WHEN NOT (b1.is_bot OR b2.is_bot) THEN 1 ELSE 0 END AS pvp_game,
    CASE WHEN (b1.username = 'BestBot' OR b2.username = 'BestBot') THEN 1 ELSE 0 END AS best_bot_game,
    CASE WHEN game_request -> 'initial_time_seconds' = '432000' THEN 1 ELSE 0 END AS correspondence_game,
    CASE WHEN g.league_id IS NOT NULL THEN 1 ELSE 0 END AS league_game,
    CASE WHEN (g.tournament_id IS NOT NULL AND g.tournament_id != '') THEN 1 ELSE 0 END AS tournament_game,
    -- game in progress at query time (no end reason yet), should be almost all correspondence games
    CASE WHEN g.type = 0 AND g.game_end_reason = 0 THEN 1 ELSE 0 END AS ongoing_game,
    -- sanity check: should always be 0 for now
    CASE WHEN (b1.is_bot AND b2.is_bot) THEN 1 ELSE 0 END AS bot_vs_bot_game,
    -- annotated games (type=1) have NULL game_end_reason so they never land in any end-reason bucket below
	-- this is also why the sum of the game_end_reason percentages is slightly less than 100%
    CASE WHEN g.type = 1 THEN 1 ELSE 0 END AS annotated_game
FROM public.games g
LEFT JOIN bot_users b1 ON g.player0_id = b1.id
LEFT JOIN bot_users b2 ON g.player1_id = b2.id
WHERE created_at > '2020-01-01'
)

SELECT
	month,
	COUNT(*) AS game_count,
	SUM(pvp_game) AS pvp_game_count,
	ROUND(100.0*SUM(pvp_game)/COUNT(*),1) AS pvp_game_pct,
	COUNT(*)-SUM(pvp_game) AS bot_game_count,
	SUM(best_bot_game) AS best_bot_game_count,
	SUM(correspondence_game) AS correspondence_game_count,
	SUM(league_game) AS league_game_count,
	SUM(tournament_game) AS tournament_game_count,
	SUM(annotated_game) AS annotated_count,
	SUM(ongoing_game) AS ongoing_count,
	SUM(bot_vs_bot_game) AS bot_vs_bot_count,
	-- game end reasons are arranged roughly in descending order of frequency
	ROUND(100.0*SUM(CASE WHEN game_end_reason=2 THEN 1 ELSE 0 END)/COUNT(*),1) AS
	  regular_game_end_pct,
	ROUND(100.0*SUM(CASE WHEN game_end_reason=1 THEN 1 ELSE 0 END)/COUNT(*),1) AS
	  timed_out_pct,
	ROUND(100.0*SUM(CASE WHEN game_end_reason=4 THEN 1 ELSE 0 END)/COUNT(*),1) AS
	  resigned_pct,
	ROUND(100.0*SUM(CASE WHEN game_end_reason=3 THEN 1 ELSE 0 END)/COUNT(*),1) AS
	  consecutive_zero_ending_pct,
	ROUND(100.0*SUM(CASE WHEN game_end_reason=7 THEN 1 ELSE 0 END)/COUNT(*),2) AS
	  cancelled_pct,
	ROUND(100.0*SUM(CASE WHEN game_end_reason=8 THEN 1 ELSE 0 END)/COUNT(*),2) AS
	  game_ended_with_forced_forfeit_pct,
	ROUND(100.0*SUM(CASE WHEN game_end_reason=5 THEN 1 ELSE 0 END)/COUNT(*),2) AS
	  aborted_pct,
	ROUND(100.0*SUM(CASE WHEN game_end_reason=6 THEN 1 ELSE 0 END)/COUNT(*),2) AS
	  triple_challenge_ending_pct,
	SUM(CASE WHEN game_end_reason=2 THEN 1 ELSE 0 END) AS
	  regular_game_end_count,
	SUM(CASE WHEN game_end_reason=1 THEN 1 ELSE 0 END) AS
	  timed_out_count,
	SUM(CASE WHEN game_end_reason=4 THEN 1 ELSE 0 END) AS
	  resigned_count,
	SUM(CASE WHEN game_end_reason=3 THEN 1 ELSE 0 END) AS
	  consecutive_zero_ending_count,
	SUM(CASE WHEN game_end_reason=7 THEN 1 ELSE 0 END) AS
	  cancelled_count,
	SUM(CASE WHEN game_end_reason=8 THEN 1 ELSE 0 END) AS
	  game_ended_with_forced_forfeit_count,
	SUM(CASE WHEN game_end_reason=5 THEN 1 ELSE 0 END) AS
	  aborted_count,
	SUM(CASE WHEN game_end_reason=6 THEN 1 ELSE 0 END) AS
	  triple_challenge_ending_count
FROM games_with_bot_flags
GROUP BY 1
ORDER BY 1 DESC
