-- New-user funnel by signup day: of the users who joined on day D, how many
-- played a game, played a human, played 2+ distinct humans, and annotated a
-- game - all within 14 days of signing up.
--
-- updated 2/27 to only include games played within 14 days of signup.
-- updated 9/4 to correctly determine when opponent is a bot.
-- updated 8/26 to scan games once, and to score annotating separately from playing.
--
-- Bot rule: internal_bot flag, plus ids 42-46 and 6216 (early bots that predate
-- the flag). Applied to the opponent only - a bot's own signup still counts toward
-- new_user_count.
WITH new_user_games AS (
    -- One row per player per game, in a single scan of games.
    SELECT
        participants.player,
        DATE_TRUNC('day', participants.player_created_at) AS day_joined,
        participants.opponent,
        participants.opponent_is_bot
    FROM public.games g
    LEFT JOIN public.users u0 ON u0.id = g.player0_id
    LEFT JOIN public.users u1 ON u1.id = g.player1_id
    CROSS JOIN LATERAL (VALUES
        (g.player0_id, g.player1_id, u0.created_at,
         (u1.internal_bot OR u1.id IN (42,43,44,45,46,6216))),
        (g.player1_id, g.player0_id, u1.created_at,
         (u0.internal_bot OR u0.id IN (42,43,44,45,46,6216)))
    ) AS participants(player, opponent, player_created_at, opponent_is_bot)
    -- IS DISTINCT FROM 1 so a NULL type counts as not-annotated, matching
    -- omgwords/games_per_month.sql.
    WHERE g.type IS DISTINCT FROM 1
      AND g.created_at - participants.player_created_at < interval '14 days'
),
player_stats AS (
    SELECT
        player,
        day_joined,
        COUNT(*) AS games_played,
        COUNT(DISTINCT opponent) AS num_of_opponents,
        -- opponent_is_bot is NULL when the opponent has no users row, which drops
        -- the row from these two counts but not from the two above.
        SUM(CASE WHEN NOT opponent_is_bot THEN 1 ELSE 0 END)
            AS games_played_against_humans,
        COUNT(DISTINCT CASE WHEN NOT opponent_is_bot THEN opponent END)
            AS number_of_human_opponents
    FROM new_user_games
    GROUP BY 1, 2
),
-- Each side is aggregated to cohort grain BEFORE joining. Joining users to
-- player_stats at player grain instead makes the planner nested-loop ~240k users
-- against the whole aggregate, which takes it from seconds to many minutes.
funnel AS (
    SELECT
        day_joined,
        SUM(CASE WHEN games_played > 0 THEN 1
	     ELSE 0 END) AS played_at_least_one_game_count,
        SUM(CASE WHEN games_played_against_humans > 0 THEN 1
	     ELSE 0 END) AS played_at_least_one_human_count,
        SUM(CASE WHEN number_of_human_opponents > 1 THEN 1
	     ELSE 0 END) AS played_at_least_two_different_people_count
    FROM player_stats
    GROUP BY 1
),
-- verified_count is current state, not 14-day-windowed like the columns above -
-- unverified users are blocked at login, and pile up here when the 48h cleanup
-- job stalls.
cohorts AS (
    SELECT
        DATE_TRUNC('day', created_at) AS day_joined,
        COUNT(DISTINCT id) AS new_user_count,
        SUM(CASE WHEN verified THEN 1 ELSE 0 END) AS verified_count
    FROM public.users
    GROUP BY 1
),
-- Credited to the annotator, not to either player_id on the annotated game -
-- those are the players in the transcribed game, often synthesized ids.
annotators AS (
    SELECT
        DATE_TRUNC('day', au.created_at) AS day_joined,
        COUNT(DISTINCT au.id) AS annotated_at_least_one_game_count
    FROM public.annotated_game_metadata agm
    JOIN public.users au ON au.uuid = agm.creator_uuid
    WHERE agm.created_at - au.created_at < interval '14 days'
    GROUP BY 1
),
reporting AS (
SELECT
    cohorts.day_joined,
    cohorts.new_user_count,
    COALESCE(funnel.played_at_least_one_game_count, 0)
        AS played_at_least_one_game_count,
    COALESCE(funnel.played_at_least_one_human_count, 0)
        AS played_at_least_one_human_count,
    COALESCE(funnel.played_at_least_two_different_people_count, 0)
        AS played_at_least_two_different_people_count,
    COALESCE(annotators.annotated_at_least_one_game_count, 0)
        AS annotated_at_least_one_game_count,
    cohorts.verified_count
FROM cohorts
LEFT JOIN funnel ON funnel.day_joined = cohorts.day_joined
LEFT JOIN annotators ON annotators.day_joined = cohorts.day_joined)

-- Denominator is verified_count, not new_user_count: unverified users can't
-- play/annotate, so they'd otherwise deflate every rate below.
SELECT
    *,
	TRUNC(100.0*played_at_least_one_game_count/verified_count,1) AS played_at_least_one_game_frac,
	TRUNC(100.0*played_at_least_one_human_count/verified_count,1) AS played_at_least_one_human_frac,
	TRUNC(100.0*played_at_least_two_different_people_count/verified_count,1) AS played_at_least_two_different_people_frac,
	TRUNC(100.0*annotated_at_least_one_game_count/verified_count,1) AS annotated_at_least_one_game_frac,
	TRUNC(100.0*verified_count/new_user_count,1) AS verified_frac
FROM reporting
ORDER BY 1 DESC
