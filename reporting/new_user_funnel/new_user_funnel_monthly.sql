-- New-user funnel by signup month: of the users who joined in month M, how many
-- played a game, played a human, played 2+ distinct humans, and annotated a
-- game - all within 14 days of signing up.
--
-- updated 2/27 to only include games played within 14 days of signup.
-- updated 9/4 to correctly determine when opponent is a bot.
-- updated 8/26: single-scan rewrite (see PERFORMANCE below), and annotated games
--   split out of the play counts into their own engagement measure.
--
-- Bot rule: internal_bot flag, plus ids 42-46 and 6216 (early bots that predate
-- the flag). It is applied to the OPPONENT only - a bot's own signup still counts
-- toward new_user_count, as it always has.
--
-- ANNOTATED GAMES. Uploading an annotated game is its own vector of early
-- engagement, not a proxy for playing, so:
--   * games.type = 1 (annotated) no longer counts toward the three play columns.
--     Those numbers are therefore slightly lower than before this change.
--   * annotated_at_least_one_game_count is measured from annotated_game_metadata
--     and credited to the ANNOTATOR (creator_uuid), which is the person who did
--     the work - games.player0_id/player1_id on an annotated game are the players
--     in the transcribed game, who often aren't the uploader and are frequently
--     synthesized ids that match no account at all.
--   * IS DISTINCT FROM 1 rather than <> 1 so a NULL type counts as not-annotated,
--     matching how omgwords/games_per_month.sql buckets it.
--
-- PERFORMANCE. Two separate problems, both fixed here.
--
-- (1) The original scanned public.games four times: all_games and
-- human_vs_human_games were each a two-branch UNION ALL, and the same player-rows
-- were then aggregated twice. Now games is scanned once and unpivoted into one
-- row per player per game by CROSS JOIN LATERAL (VALUES ...) - the same technique
-- as reporting/mau_reporting.sql - with the two stat sets collapsed into one
-- aggregation of conditional CASE WHEN counts.
--
-- (2) The first attempt at (1) was still catastrophically slow (19+ min, killed),
-- for a reason worth recording. It ended with
--     FROM public.users LEFT JOIN player_stats ON users.id = player_stats.player
-- and the planner estimated player_stats at *2 rows* when the truth is one row
-- per player who has played, so it chose a Nested Loop with a Materialize on the
-- inner side and rescanned ~100k materialized rows once per each of ~240k users.
-- The estimate is wrong because the GROUP BY key comes from a VALUES list
-- ("*VALUES*".column1), for which Postgres has no n_distinct statistic and falls
-- back to a hardcoded guess - the same class of trap as the materialized-CTE note
-- atop omgwords/games_per_month.sql.
--
-- The fix is to never join at player grain. Each side is aggregated to month
-- grain first and the joins happen on ~70 month rows, where any row-estimate
-- error is harmless. COUNT(DISTINCT ...) still forces a sort-based GroupAggregate
-- over the unpivoted rows, and the full scan of games is unavoidable because the
-- funnel has no date predicate to push down; those are the remaining floor.
--
-- The LEFT JOINs to users are kept rather than tightened to inner joins because
-- the original used LEFT JOIN: a game whose player has no users row still
-- contributes the other player's row, and an opponent with no users row yields a
-- NULL bot flag that drops the row from the human-vs-human counts without
-- dropping it from the all-games counts. Inner joins would change both.
WITH new_user_games AS (
    SELECT
        participants.player,
        DATE_TRUNC('month', participants.player_created_at) AS month_joined,
        participants.opponent,
        participants.opponent_is_bot
    FROM public.games g
    LEFT JOIN public.users u0 ON u0.id = g.player0_id
    LEFT JOIN public.users u1 ON u1.id = g.player1_id
    -- One row per player per game in a single scan. Each row carries the player's
    -- own signup time (for the 14-day window) and whether that row's OPPONENT is
    -- a bot. Three-valued on purpose: NULL when the opponent has no users row,
    -- matching the old NOT (u2.internal_bot OR u2.id IN (...)).
    CROSS JOIN LATERAL (VALUES
        (g.player0_id, g.player1_id, u0.created_at,
         (u1.internal_bot OR u1.id IN (42,43,44,45,46,6216))),
        (g.player1_id, g.player0_id, u1.created_at,
         (u0.internal_bot OR u0.id IN (42,43,44,45,46,6216)))
    ) AS participants(player, opponent, player_created_at, opponent_is_bot)
    WHERE g.type IS DISTINCT FROM 1
      AND g.created_at - participants.player_created_at < interval '14 days'
),
-- The original grouped by (player, username) via an extra join to users. username
-- was never referenced downstream and player -> username is functional, so
-- dropping that join changes no row and no count.
player_stats AS (
    SELECT
        player,
        month_joined,
        COUNT(*) AS games_played,
        COUNT(DISTINCT opponent) AS num_of_opponents,
        -- opponent_is_bot is NULL when the opponent has no users row, so
        -- NOT opponent_is_bot is NULL: the row falls through to ELSE 0, and to a
        -- NULL that COUNT DISTINCT ignores. Same rows drop out as before.
        SUM(CASE WHEN NOT opponent_is_bot THEN 1 ELSE 0 END)
            AS games_played_against_humans,
        COUNT(DISTINCT CASE WHEN NOT opponent_is_bot THEN opponent END)
            AS number_of_human_opponents
    FROM new_user_games
    GROUP BY 1, 2
),
-- Aggregating to month grain HERE, before any join, is the whole point of the
-- rewrite. A user who never played simply has no player_stats row and adds 0 to
-- these sums, exactly as the old LEFT JOIN + "NULL > 0 is false" did.
funnel AS (
    SELECT
        month_joined,
        SUM(CASE WHEN games_played > 0 THEN 1
	     ELSE 0 END) AS played_at_least_one_game_count,
        SUM(CASE WHEN games_played_against_humans > 0 THEN 1
	     ELSE 0 END) AS played_at_least_one_human_count,
        SUM(CASE WHEN number_of_human_opponents > 1 THEN 1
	     ELSE 0 END) AS played_at_least_two_different_people_count
    FROM player_stats
    GROUP BY 1
),
cohorts AS (
    SELECT
        DATE_TRUNC('month', created_at) AS month_joined,
        COUNT(DISTINCT id) AS new_user_count
    FROM public.users
    GROUP BY 1
),
annotators AS (
    SELECT
        DATE_TRUNC('month', au.created_at) AS month_joined,
        COUNT(DISTINCT au.id) AS annotated_at_least_one_game_count
    FROM public.annotated_game_metadata agm
    JOIN public.users au ON au.uuid = agm.creator_uuid
    WHERE agm.created_at - au.created_at < interval '14 days'
    GROUP BY 1
),
reporting AS (
-- Every player and every annotator is a user, so cohorts is the complete spine
-- and these joins can only fail to match for a month where nobody converted -
-- hence the COALESCE to 0 rather than leaving a NULL.
SELECT
    cohorts.month_joined,
    cohorts.new_user_count,
    COALESCE(funnel.played_at_least_one_game_count, 0)
        AS played_at_least_one_game_count,
    COALESCE(funnel.played_at_least_one_human_count, 0)
        AS played_at_least_one_human_count,
    COALESCE(funnel.played_at_least_two_different_people_count, 0)
        AS played_at_least_two_different_people_count,
    COALESCE(annotators.annotated_at_least_one_game_count, 0)
        AS annotated_at_least_one_game_count
FROM cohorts
LEFT JOIN funnel ON funnel.month_joined = cohorts.month_joined
LEFT JOIN annotators ON annotators.month_joined = cohorts.month_joined)

SELECT
    *,
	TRUNC(100.0*played_at_least_one_game_count/new_user_count,1) AS played_at_least_one_game_frac,
	TRUNC(100.0*played_at_least_one_human_count/new_user_count,1) AS played_at_least_one_human_frac,
	TRUNC(100.0*played_at_least_two_different_people_count/new_user_count,1) AS played_at_least_two_different_people_frac,
	TRUNC(100.0*annotated_at_least_one_game_count/new_user_count,1) AS annotated_at_least_one_game_frac
FROM reporting
ORDER BY 1 DESC
