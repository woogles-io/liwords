-- Throwaway: per-user nationality + first letter of username, plus
-- total games played (summed across all rating variants) and average
-- rating (averaged across all rating variants present).
WITH game_totals AS (
    SELECT p.user_id,
           SUM((v.value -> 'd1' -> 'Games' ->> 't')::int) AS total_games
    FROM profiles p,
         LATERAL jsonb_each(
             CASE WHEN jsonb_typeof(p.stats -> 'Data') = 'object'
                  THEN p.stats -> 'Data' ELSE '{}'::jsonb END
         ) AS v
    GROUP BY p.user_id
),
rating_avgs AS (
    SELECT p.user_id,
           AVG((v.value ->> 'r')::float) AS avg_rating
    FROM profiles p,
         LATERAL jsonb_each(
             CASE WHEN jsonb_typeof(p.ratings -> 'Data') = 'object'
                  THEN p.ratings -> 'Data' ELSE '{}'::jsonb END
         ) AS v
    GROUP BY p.user_id
)
SELECT
    u.username,
    upper(left(u.username, 1))       AS first_letter,
    coalesce(p.country_code, '')     AS country_code,
    coalesce(gt.total_games, 0)      AS total_games,
    ra.avg_rating
FROM users u
JOIN profiles p ON p.user_id = u.id
LEFT JOIN game_totals gt ON gt.user_id = u.id
LEFT JOIN rating_avgs ra ON ra.user_id = u.id
WHERE u.deleted_at IS NULL
ORDER BY u.id;
