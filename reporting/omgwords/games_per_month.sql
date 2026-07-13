-- Bot rule: internal_bot flag, plus ids 42-46 (early bots that predate the flag).
-- Join users directly rather than through a bot_users CTE: a CTE referenced twice
-- gets materialized without stats or indexes, which sent the planner into a
-- merge-join-plus-sort plan with a ~200M-row overestimate (~6 min instead of seconds).
WITH games_with_bot_flags AS (
SELECT
    DATE_TRUNC('month', g.created_at) AS month,
    g.game_end_reason,
    CASE WHEN NOT (u0.internal_bot OR u0.id IN (42,43,44,45,46,6216)
                OR u1.internal_bot OR u1.id IN (42,43,44,45,46,6216)) THEN 1 ELSE 0 END AS pvp_game,
    CASE WHEN (u0.username = 'BestBot' OR u1.username = 'BestBot') THEN 1 ELSE 0 END AS best_bot_game,
    CASE WHEN game_request -> 'initial_time_seconds' = '432000' THEN 1 ELSE 0 END AS correspondence_game,
    CASE WHEN g.league_id IS NOT NULL THEN 1 ELSE 0 END AS league_game,
    CASE WHEN (g.tournament_id IS NOT NULL AND g.tournament_id != '') THEN 1 ELSE 0 END AS tournament_game,
    -- game in progress at query time (no end reason yet), should be almost all correspondence games
    CASE WHEN g.type = 0 AND g.game_end_reason = 0 THEN 1 ELSE 0 END AS ongoing_game,
    -- sanity check: should always be 0 for now
    CASE WHEN ((u0.internal_bot OR u0.id IN (42,43,44,45,46,6216))
           AND (u1.internal_bot OR u1.id IN (42,43,44,45,46,6216))) THEN 1 ELSE 0 END AS bot_vs_bot_game,
    -- annotated games (type=1) have NULL game_end_reason so they never land in any end-reason bucket below
	-- this is also why the sum of the game_end_reason percentages is slightly less than 100%
    CASE WHEN g.type = 1 THEN 1 ELSE 0 END AS annotated_game,
    CASE WHEN g.type = 0 AND COALESCE(g.game_request ->> 'rating_mode', '0') = '0' THEN 1 ELSE 0 END AS rated_game,
    g.game_request ->> 'lexicon' AS lexicon,
    LOWER(REPLACE(COALESCE(g.game_request -> 'rules' ->> 'letter_distribution_name',
                           g.game_request -> 'rules' ->> 'letterDistributionName'), '_super', '')) AS language_norm,
    -- ZOMGWords = Super Scrabble (super board / super tile bag); WordSmog = Clabbers variant.
    -- A super-wordsmog game would match both (none exist yet).
    CASE WHEN COALESCE(g.game_request -> 'rules' ->> 'board_layout_name',
                       g.game_request -> 'rules' ->> 'boardLayoutName') = 'SuperCrosswordGame'
           OR COALESCE(g.game_request -> 'rules' ->> 'letter_distribution_name',
                       g.game_request -> 'rules' ->> 'letterDistributionName') ILIKE '%\_super'
         THEN 1 ELSE 0 END AS zomgwords_game,
    CASE WHEN COALESCE(g.game_request -> 'rules' ->> 'variant_name',
                       g.game_request -> 'rules' ->> 'variantName') LIKE 'wordsmog%'
         THEN 1 ELSE 0 END AS wordsmog_game
FROM public.games g
LEFT JOIN public.users u0 ON g.player0_id = u0.id
LEFT JOIN public.users u1 ON g.player1_id = u1.id
WHERE g.created_at > '2020-01-01'
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
	  triple_challenge_ending_count,
	SUM(rated_game) AS rated_game_count,
	-- CSW = Collins family (CSW*); NWL = North American family (NWL*, NSWL*, OSPD6);
	-- versions folded together. ECWL counts as English but neither CSW nor NWL, so
	-- csw + nwl <= english. Language columns fold CSW and NWL both into English.
	SUM(CASE WHEN lexicon LIKE 'CSW%' THEN 1 ELSE 0 END) AS csw_game_count,
	SUM(CASE WHEN lexicon LIKE 'NWL%' OR lexicon LIKE 'NSWL%' OR lexicon = 'OSPD6' THEN 1 ELSE 0 END) AS nwl_game_count,
	SUM(CASE WHEN language_norm = 'english'   THEN 1 ELSE 0 END) AS english_game_count,
	SUM(CASE WHEN language_norm = 'german'    THEN 1 ELSE 0 END) AS german_game_count,
	SUM(CASE WHEN language_norm = 'french'    THEN 1 ELSE 0 END) AS french_game_count,
	SUM(CASE WHEN language_norm = 'spanish'   THEN 1 ELSE 0 END) AS spanish_game_count,
	SUM(CASE WHEN language_norm = 'polish'    THEN 1 ELSE 0 END) AS polish_game_count,
	SUM(CASE WHEN language_norm = 'norwegian' THEN 1 ELSE 0 END) AS norwegian_game_count,
	SUM(CASE WHEN language_norm = 'catalan'   THEN 1 ELSE 0 END) AS catalan_game_count,
	SUM(zomgwords_game) AS zomgwords_game_count,
	SUM(wordsmog_game) AS wordsmog_game_count
FROM games_with_bot_flags
GROUP BY 1
ORDER BY 1 DESC
