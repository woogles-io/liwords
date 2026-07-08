-- Monthly active users for OMGWords, puzzles, and Board Editor, plus games played per month.
-- Bot rule: internal_bot flag, plus ids 42-46 (early bots that predate the flag).

-- NOTE: As of right now, a user will not count as an MAU if they are only continuing
-- correspondence games that were started in a previous month.

WITH engagement_events AS (
    -- One row per engagement event, built in a single scan of games:
    -- each player from each game, each puzzle attempt with a recorded result,
    -- and each annotated game created (credited to its annotator).
    SELECT
        DATE_TRUNC('month', g.created_at) AS month,
        g.id AS game_id,
        participants.user_id,
        participants.is_human,
        (NOT (u0.internal_bot OR u0.id IN (42, 43, 44, 45, 46))
         AND NOT (u1.internal_bot OR u1.id IN (42, 43, 44, 45, 46))) AS vs_human,
        'game' AS event_type
    FROM public.games g
    JOIN public.users u0 ON u0.id = g.player0_id
    JOIN public.users u1 ON u1.id = g.player1_id
    -- Unpivot each game into one row per player/two rows per game in a single scan 
    -- (vs the previous UNION ALL that scanned the whole games table twice)
    CROSS JOIN LATERAL (VALUES
        (g.player0_id, NOT (u0.internal_bot OR u0.id IN (42, 43, 44, 45, 46))),
        (g.player1_id, NOT (u1.internal_bot OR u1.id IN (42, 43, 44, 45, 46)))
    ) AS participants(user_id, is_human)

    UNION ALL

    SELECT
        DATE_TRUNC('month', p.created_at) AS month,
        NULL AS game_id,
        p.user_id AS user_id,
        TRUE AS is_human,
        FALSE AS vs_human,
        'puzzle' AS event_type
    FROM public.puzzle_attempts p
    JOIN public.users pu ON pu.id = p.user_id
    WHERE p.correct IS NOT NULL
      AND NOT (pu.internal_bot OR pu.id IN (42, 43, 44, 45, 46))

    UNION ALL

    SELECT
        DATE_TRUNC('month', agm.created_at) AS month,
        NULL AS game_id,
        au.id AS user_id,
        TRUE AS is_human,
        FALSE AS vs_human,
        'annotation' AS event_type
    FROM public.annotated_game_metadata agm
    JOIN public.users au ON au.uuid = agm.creator_uuid
)

SELECT
    month,
    COUNT(DISTINCT CASE WHEN is_human AND event_type = 'game' THEN user_id END) AS mau_omgwords,
    COUNT(DISTINCT game_id) AS games_played, --excludes annotated games
    TRUNC(1.0 * COUNT(DISTINCT game_id)
              / NULLIF(COUNT(DISTINCT CASE WHEN is_human AND event_type = 'game' THEN user_id END), 0), 2)
        AS games_played_per_mau,
    COUNT(DISTINCT CASE WHEN is_human AND vs_human THEN user_id END) AS mau_omgwords_vs_human,
    TRUNC(100.0 * COUNT(DISTINCT CASE WHEN is_human AND vs_human THEN user_id END)
                / NULLIF(COUNT(DISTINCT CASE WHEN is_human AND event_type = 'game' THEN user_id END), 0), 1)
        AS pct_of_omgwords_mau_who_played_a_human,
    COUNT(DISTINCT CASE WHEN event_type = 'puzzle' THEN user_id END) AS mau_puzzles,
    COUNT(DISTINCT CASE WHEN event_type = 'annotation' THEN user_id END) AS mau_annotators,
    COUNT(DISTINCT CASE WHEN is_human THEN user_id END) AS mau
FROM engagement_events
GROUP BY month
ORDER BY month DESC;
