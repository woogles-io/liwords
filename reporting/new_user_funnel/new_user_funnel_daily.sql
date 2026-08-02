-- New-user funnel by signup day: of the users who joined on day D, how many
-- played a game, played a human, and played 2+ distinct humans - all within 14
-- days of signing up.
--
-- updated 2/27 to only include games played within 14 days of signup.
-- updated 9/4 to correctly determine when opponent is a bot.
-- updated 8/26: performance refactor only. Same numbers, one scan of games
--   instead of four. Verified column-for-column against the old query.
--
-- Bot rule: internal_bot flag, plus ids 42-46 and 6216 (early bots that predate
-- the flag). It is applied to the OPPONENT only - a bot's own signup still counts
-- toward new_user_count, exactly as before.
--
-- What changed: the old version built all_games and human_vs_human_games as
-- two-branch UNION ALLs, so public.games was scanned four times and the same
-- player-rows were aggregated twice. Now games is scanned once and unpivoted
-- into one row per player per game by CROSS JOIN LATERAL (VALUES ...) - the same
-- technique as reporting/mau_reporting.sql - and the two stat sets collapse into
-- one aggregation of conditional CASE WHEN counts. Joining public.users directly
-- rather than through a bot_users CTE is deliberate; see the note atop
-- omgwords/games_per_month.sql for what a twice-referenced CTE did there.
--
-- The LEFT JOINs to users are kept (rather than tightened to inner joins) because
-- the old query used LEFT JOIN: a game whose player has no users row still
-- contributes the other player's row, and an opponent with no users row makes
-- opponent_is_bot NULL, which drops that row from the human-vs-human counts
-- without dropping it from the all-games counts. Inner joins would quietly change
-- both behaviours.
WITH new_user_games AS (
    SELECT
        participants.player,
        participants.opponent,
        participants.opponent_is_bot
    FROM public.games g
    LEFT JOIN public.users u0 ON u0.id = g.player0_id
    LEFT JOIN public.users u1 ON u1.id = g.player1_id
    -- One row per player per game in a single scan. Each row carries the
    -- player's own signup time (for the 14-day window) and whether that row's
    -- OPPONENT is a bot. Three-valued on purpose: NULL when the opponent has no
    -- users row, matching the old NOT (u2.internal_bot OR u2.id IN (...)).
    CROSS JOIN LATERAL (VALUES
        (g.player0_id, g.player1_id, u0.created_at,
         (u1.internal_bot OR u1.id IN (42,43,44,45,46,6216))),
        (g.player1_id, g.player0_id, u1.created_at,
         (u0.internal_bot OR u0.id IN (42,43,44,45,46,6216)))
    ) AS participants(player, opponent, player_created_at, opponent_is_bot)
    WHERE g.created_at - participants.player_created_at < interval '14 days'
),
-- The old query grouped all_stats/human_vs_human_stats by (player, username) via
-- an extra join to users. username was never referenced downstream and player ->
-- username is functional, so dropping that join changes no row and no count.
player_stats AS (
    SELECT
        player,
        COUNT(*) AS games_played,
        COUNT(DISTINCT opponent) AS num_of_opponents,
        -- opponent_is_bot is NULL when the opponent has no users row, so
        -- NOT opponent_is_bot is NULL and the row falls through to ELSE 0 /
        -- to a NULL that COUNT DISTINCT ignores - dropping it from the
        -- human-vs-human counts exactly as the old NOT (...) predicate did.
        SUM(CASE WHEN NOT opponent_is_bot THEN 1 ELSE 0 END)
            AS games_played_against_humans,
        COUNT(DISTINCT CASE WHEN NOT opponent_is_bot THEN opponent END)
            AS number_of_human_opponents
    FROM new_user_games
    GROUP BY 1
),
reporting AS (
-- A player with no human games had no human_vs_human_stats row before (NULL, so
-- NULL > 0 was false); now the conditional counts yield 0, and 0 > 0 is false.
-- Same result either way.
SELECT
  DATE_TRUNC('day', u.created_at) AS day_joined,
  COUNT(DISTINCT u.id) AS new_user_count,
  SUM(CASE WHEN player_stats.games_played > 0 THEN 1
	 ELSE 0 END) AS played_at_least_one_game_count,
  SUM(CASE WHEN player_stats.games_played_against_humans > 0 THEN 1
	 ELSE 0 END) AS played_at_least_one_human_count,
  SUM(CASE WHEN player_stats.number_of_human_opponents > 1 THEN 1
	 ELSE 0 END) AS played_at_least_two_different_people_count
FROM public.users u
LEFT JOIN player_stats ON u.id = player_stats.player
GROUP BY 1)

SELECT
    *,
	TRUNC(100.0*played_at_least_one_game_count/new_user_count,1) AS played_at_least_one_game_frac,
	TRUNC(100.0*played_at_least_one_human_count/new_user_count,1) AS played_at_least_one_human_frac,
	TRUNC(100.0*played_at_least_two_different_people_count/new_user_count,1) AS played_at_least_two_different_people_frac
FROM reporting
ORDER BY 1 DESC
