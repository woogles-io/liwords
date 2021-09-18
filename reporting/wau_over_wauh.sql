WITH duplicated_games AS
((SELECT
    created_at,
    player0_id AS player
FROM public.games)
UNION ALL
(SELECT
    created_at,
    player1_id AS player
FROM public.games)),
wau_report AS
(SELECT
	DATE_TRUNC('week',duplicated_games.created_at) AS week,
	COUNT(DISTINCT player) AS wau
FROM duplicated_games
LEFT JOIN public.users ON duplicated_games.player = users.id
WHERE NOT (users.internal_bot OR users.id IN (42,43,44,45,46))
GROUP BY 1),
bot_users AS
(SELECT
   id,
   internal_bot OR id IN (42,43,44,45,46) AS is_bot
FROM public.users),
pvp_games as
(SELECT
   created_at,
   games.player0_id,
   games.player1_id
 FROM public.games
 LEFT JOIN bot_users b1 ON games.player0_id=b1.id
 LEFT JOIN bot_users b2 ON games.player1_id=b2.id
 WHERE (NOT b1.is_bot)
   AND (NOT b2.is_bot)
 ),
duplicated_games_between_humans AS
((SELECT
    created_at,
    player0_id AS player
FROM pvp_games)
UNION ALL
(SELECT
    created_at,
    player1_id AS player
FROM pvp_games)),
wauh_report AS
(SELECT
    DATE_TRUNC('week',created_at) AS week,
	COUNT(DISTINCT player) AS wau_h
FROM duplicated_games_between_humans
GROUP BY 1)

SELECT
  wau_report.week,
  wau_report.wau,
  wauh_report.wau_h,
  TRUNC(100.0*wauh_report.wau_h/wau_report.wau,1) AS ratio
FROM wau_report
LEFT JOIN wauh_report ON wau_report.week = wauh_report.week
ORDER BY 1 DESC
