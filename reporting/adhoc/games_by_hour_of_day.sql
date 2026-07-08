-- Server-load profile: games started per hour of day (UTC), last 90 complete days.
-- Used to pick a low-traffic window for heavy reporting queries.
SELECT
    EXTRACT(HOUR FROM g.created_at)::int AS hour_utc,
    COUNT(*) AS games_90d,
    ROUND(COUNT(*) / 90.0, 1) AS avg_games_per_day,
    COUNT(*) FILTER (WHERE EXTRACT(ISODOW FROM g.created_at) <= 5) AS weekday_games,
    COUNT(*) FILTER (WHERE EXTRACT(ISODOW FROM g.created_at) >= 6) AS weekend_games,
    ROUND(100.0 * COUNT(*) / SUM(COUNT(*)) OVER (), 1) AS pct_of_total
FROM public.games g
WHERE g.created_at >= DATE_TRUNC('day', NOW()) - INTERVAL '90 days'
  AND g.created_at < DATE_TRUNC('day', NOW())
GROUP BY 1
ORDER BY 1;
