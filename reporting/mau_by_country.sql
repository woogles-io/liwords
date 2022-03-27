WITH duplicated_games AS
((SELECT
    games.created_at,
    games.player0_id AS player,
    profiles.country_code
FROM public.games
LEFT JOIN public.profiles ON games.player0_id = profiles.user_id
)
UNION ALL
(SELECT
    games.created_at,
    games.player1_id AS player,
    profiles.country_code
FROM public.games
LEFT JOIN public.profiles ON games.player1_id = profiles.user_id
)),
by_month AS
(SELECT
	DATE_TRUNC('month',created_at) AS month,
    country_code,
	COUNT(DISTINCT player) AS mau
FROM duplicated_games
GROUP BY 1,2
ORDER BY 3 DESC),
by_day AS
(SELECT
	DATE_TRUNC('day',created_at) AS day,
    country_code,
	COUNT(DISTINCT player) AS mau
FROM duplicated_games
GROUP BY 1,2
ORDER BY 3 DESC),
total AS
(SELECT
    country_code,
	COUNT(DISTINCT player) AS total_users
FROM duplicated_games
GROUP BY 1
ORDER BY 2 DESC)

SELECT
  *
FROM by_day
WHERE country_code IN ('ca','my','us','','id','in','lk','pk','sg','th','de')
ORDER BY 1 DESC,2
