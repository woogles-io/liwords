-- Rebuild every league standing's mistake index columns from the stored
-- analysis results. The old running counters drifted (a rejected late
-- submission decremented without re-adding; requeues before the JIT change
-- re-added without subtracting; earlier rebuilds skipped perfect games).
-- Same selection as the RefreshDivisionMistakeIndex query.
WITH latest AS (
    SELECT DISTINCT ON (aj.game_id)
        g.league_division_id AS division_id,
        g.player0_id,
        g.player1_id,
        COALESCE(aj.result->'playerSummaries'->0->>'mistakeIndex', '0')::DOUBLE PRECISION AS mi0,
        COALESCE(aj.result->'playerSummaries'->1->>'mistakeIndex', '0')::DOUBLE PRECISION AS mi1
    FROM analysis_jobs aj
    JOIN games g ON g.uuid = aj.game_id
    WHERE g.league_division_id IS NOT NULL
      AND aj.result->'playerSummaries'->0 IS NOT NULL
      AND aj.result->'playerSummaries'->1 IS NOT NULL
    ORDER BY aj.game_id, aj.created_at DESC
),
per_player AS (
    SELECT division_id, player0_id AS user_id, mi0 AS mi FROM latest WHERE player0_id IS NOT NULL
    UNION ALL
    SELECT division_id, player1_id AS user_id, mi1 AS mi FROM latest WHERE player1_id IS NOT NULL
),
totals AS (
    SELECT division_id, user_id, SUM(mi) AS total, COUNT(*)::INT AS n
    FROM per_player
    GROUP BY division_id, user_id
)
UPDATE league_standings ls
SET total_mistake_index = COALESCE(t.total, 0),
    games_analyzed = COALESCE(t.n, 0),
    updated_at = NOW()
FROM league_standings cur
LEFT JOIN totals t ON t.division_id = cur.division_id AND t.user_id = cur.user_id
WHERE ls.id = cur.id
  -- Skip rows that differ only by float summation order.
  AND (ls.games_analyzed IS DISTINCT FROM COALESCE(t.n, 0)
       OR ls.total_mistake_index IS NULL
       OR ABS(ls.total_mistake_index - COALESCE(t.total, 0)) > 1e-6);
