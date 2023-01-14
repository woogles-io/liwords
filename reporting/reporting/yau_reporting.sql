-- OMGWords
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

yau_omgwords_report AS
(SELECT
	DATE_TRUNC('year',duplicated_games.created_at) AS year,
	COUNT(DISTINCT player) AS yau_omgwords
FROM duplicated_games
LEFT JOIN public.users ON duplicated_games.player = users.id
WHERE NOT (users.internal_bot OR users.id IN (42,43,44,45,46))
GROUP BY 1),

-- we now have a flag for users who are bots, but our earliest
-- bots predate the existence of that flag, hence this query
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
yau_omgwords_vs_human_report AS
(SELECT
    DATE_TRUNC('year',created_at) AS year,
	COUNT(DISTINCT player) AS yau_omgwords_vs_human
FROM duplicated_games_between_humans
GROUP BY 1),

-- Puzzles
yau_puzzles AS
(SELECT
   created_at,
   user_id AS player
FROM public.puzzle_attempts
WHERE correct IS NOT NULL),

yau_puzzles_report AS
(SELECT
	DATE_TRUNC('year',yau_puzzles.created_at) AS year,
	COUNT(DISTINCT player) AS yau_puzzles
FROM yau_puzzles
LEFT JOIN public.users ON yau_puzzles.player = users.id
WHERE NOT users.internal_bot
GROUP BY 1),

-- Joint reporting
games_plus_puzzle_attempts AS
((SELECT
    created_at,
    player0_id AS player
FROM public.games)
UNION ALL
(SELECT
    created_at,
    player1_id AS player
FROM public.games)
UNION ALL
(SELECT
   created_at,
   user_id AS player
FROM public.puzzle_attempts
WHERE correct IS NOT NULL)),

omgwords_plus_puzzles_report AS
(SELECT
	DATE_TRUNC('year',games_plus_puzzle_attempts.created_at) AS year,
	COUNT(DISTINCT player) AS yau
FROM games_plus_puzzle_attempts
LEFT JOIN public.users ON games_plus_puzzle_attempts.player = users.id
WHERE NOT (users.internal_bot OR users.id IN (42,43,44,45,46))
GROUP BY 1)

SELECT
  yau_omgwords_report.year,
  yau_omgwords_report.yau_omgwords,
  yau_omgwords_vs_human_report.yau_omgwords_vs_human,
  TRUNC(100.0*yau_omgwords_vs_human_report.yau_omgwords_vs_human/yau_omgwords_report.yau_omgwords,1)
    AS pct_of_omgwords_dau_who_played_a_human,
  yau_puzzles_report.yau_puzzles,
  omgwords_plus_puzzles_report.yau
FROM yau_omgwords_report
LEFT JOIN yau_omgwords_vs_human_report ON yau_omgwords_report.year = yau_omgwords_vs_human_report.year
LEFT JOIN yau_puzzles_report ON yau_omgwords_report.year = yau_puzzles_report.year
LEFT JOIN omgwords_plus_puzzles_report ON yau_omgwords_report.year = omgwords_plus_puzzles_report.year
ORDER BY 1 DESC