-- Games per active user: the distribution behind mau_reporting.sql's headline
-- column (median, one-and-done share) - a rising mean can mean everyone plays
-- more, or a few whales play much more, and only the distribution tells those
-- apart. Also where the definition was settled: a human-vs-human game counts
-- once per player here, not once total like the old games_played_per_mau
-- column, since that denominator moves with the bot/human mix rather than
-- with engagement. Cross-checked against mau_reporting.sql's independently-
-- written version of the same column.
--
-- Bot rule matches games_per_month.sql / dau_reporting.sql (internal_bot flag
-- + ids 42-46, 6216). Excludes annotated games (type=1) - synthesized player
-- ids there are neither a real user nor a game a user played, so games_played
-- here runs slightly below mau_reporting.sql's same-named column, which counts
-- them. Correspondence games count toward the month created, not continued
-- (inherited from mau_reporting.sql).

WITH games_scan AS (
    -- Single scan of games; both bot flags resolved here, before the unpivot, so
    -- nothing downstream has to join the 12M-row table to users again.
    SELECT
        DATE_TRUNC('month', g.created_at) AS month,
        g.player0_id,
        g.player1_id,
        NOT (u0.internal_bot OR u0.id IN (42, 43, 44, 45, 46, 6216)) AS p0_is_human,
        NOT (u1.internal_bot OR u1.id IN (42, 43, 44, 45, 46, 6216)) AS p1_is_human
    FROM public.games g
    JOIN public.users u0 ON u0.id = g.player0_id
    JOIN public.users u1 ON u1.id = g.player1_id
    WHERE g.created_at > '2020-01-01'
      AND g.type = 0
),

player_games AS (
    -- One row per player per game (two rows per game), human slots only.
    -- game_weight exists so the month-level total can recover the DISTINCT game
    -- count without a second scan or a COUNT(DISTINCT) over ~24M rows: a game
    -- with two human players contributes 0.5 from each side, a game against a
    -- bot contributes 1 from its single human side, so the sum is exactly the
    -- number of games with at least one human in them.
    SELECT
        s.month,
        p.user_id,
        p.opponent_is_human,
        CASE WHEN p.opponent_is_human THEN 0.5 ELSE 1.0 END AS game_weight
    FROM games_scan s
    CROSS JOIN LATERAL (VALUES
        (s.player0_id, s.p0_is_human, s.p1_is_human),
        (s.player1_id, s.p1_is_human, s.p0_is_human)
    ) AS p(user_id, is_human, opponent_is_human)
    WHERE p.is_human
),

per_player_month AS (
    -- Fine grain, but it never gets joined to anything at this grain - it is
    -- aggregated up to month below, and only the ~70 monthly rows are joined.
    -- Joining a stats-less fine-grained aggregate to a large table is what cost
    -- new_user_funnel_monthly.sql 19 minutes; see reference/db-notes.md.
    SELECT
        month,
        user_id,
        SUM(game_weight) AS game_weight,
        COUNT(*) AS games_played,
        SUM(CASE WHEN opponent_is_human THEN 1 ELSE 0 END) AS human_games_played
    FROM player_games
    GROUP BY month, user_id
)

SELECT
    month,
    COUNT(*) AS mau_omgwords,
    SUM(CASE WHEN human_games_played > 0 THEN 1 ELSE 0 END) AS mau_omgwords_vs_human,
    ROUND(SUM(game_weight)) AS games_played,
    SUM(games_played) AS player_games_played,

    -- The two headline ratios.
    TRUNC(SUM(game_weight) / NULLIF(COUNT(*), 0), 2) AS games_per_mau,
    TRUNC(1.0 * SUM(games_played) / NULLIF(COUNT(*), 0), 2) AS games_per_active_user,

    -- Split by opponent type: bot play and human play are different products and
    -- have moved differently over the years.
    TRUNC(1.0 * SUM(human_games_played) / NULLIF(COUNT(*), 0), 2)
        AS human_games_per_active_user,
    TRUNC(1.0 * SUM(games_played - human_games_played) / NULLIF(COUNT(*), 0), 2)
        AS bot_games_per_active_user,

    -- Distribution. The mean is pulled hard by a small number of very heavy
    -- players, so the median is the honest "typical active user" number and the
    -- p90 is the whale line.
    PERCENTILE_DISC(0.5) WITHIN GROUP (ORDER BY games_played)
        AS median_games_per_active_user,
    PERCENTILE_DISC(0.9) WITHIN GROUP (ORDER BY games_played)
        AS p90_games_per_active_user,
    MAX(games_played) AS max_games_by_one_user,

    -- Shape of the tail, as a share of the month's actives.
    TRUNC(100.0 * SUM(CASE WHEN games_played = 1 THEN 1 ELSE 0 END)
                / NULLIF(COUNT(*), 0), 1) AS pct_played_exactly_one_game,
    TRUNC(100.0 * SUM(CASE WHEN games_played >= 10 THEN 1 ELSE 0 END)
                / NULLIF(COUNT(*), 0), 1) AS pct_played_10_or_more_games,
    TRUNC(100.0 * SUM(CASE WHEN games_played >= 50 THEN 1 ELSE 0 END)
                / NULLIF(COUNT(*), 0), 1) AS pct_played_50_or_more_games
FROM per_player_month
GROUP BY month
ORDER BY month DESC;
