-- Throwaway: user counts by nationality x first letter of username,
-- with total/avg games played (from profiles.stats) and avg rating (from profiles.ratings).
WITH per_user AS (
  SELECT
    coalesce(nullif(lower(p.country_code), ''), '??') AS country,
    CASE WHEN lower(left(u.username, 1)) BETWEEN 'a' AND 'z'
         THEN lower(left(u.username, 1)) ELSE '#' END AS first_letter,
    (SELECT sum((v.value->'d1'->'Games'->>'t')::int)
       FROM jsonb_each(CASE WHEN jsonb_typeof(p.stats->'Data') = 'object'
                            THEN p.stats->'Data' ELSE '{}'::jsonb END) v) AS games,
    (SELECT avg((r.value->>'r')::numeric)
       FROM jsonb_each(CASE WHEN jsonb_typeof(p.ratings->'Data') = 'object'
                            THEN p.ratings->'Data' ELSE '{}'::jsonb END) r) AS avg_rating
  FROM users u
  JOIN profiles p ON p.user_id = u.id
  WHERE u.deleted_at IS NULL
    AND NOT coalesce(u.internal_bot, false)
)
SELECT
  country,
  first_letter,
  count(*) AS n_users,
  coalesce(sum(games), 0) AS total_games,
  round(avg(games), 1) AS avg_games,
  round(avg(avg_rating)) AS avg_rating
FROM per_user
GROUP BY 1, 2
ORDER BY 1, 2;
