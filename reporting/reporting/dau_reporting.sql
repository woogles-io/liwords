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

dau_omgwords_report AS
(SELECT
	DATE_TRUNC('day',duplicated_games.created_at) AS day,
	COUNT(DISTINCT player) AS dau_omgwords
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
dau_omgwords_vs_human_report AS
(SELECT
    DATE_TRUNC('day',created_at) AS day,
	COUNT(DISTINCT player) AS dau_omgwords_vs_human
FROM duplicated_games_between_humans
GROUP BY 1),

-- Puzzles
-- note that seeing a puzzle creates a row in  puzzle attempts table,
-- even if the user never inputs a solution. Query now accounts for this.
dau_puzzles AS
(SELECT
   created_at,
   user_id AS player
FROM public.puzzle_attempts
WHERE correct IS NOT NULL),

dau_puzzles_report AS
(SELECT
	DATE_TRUNC('day',dau_puzzles.created_at) AS day,
	COUNT(DISTINCT player) AS dau_puzzles
FROM dau_puzzles
LEFT JOIN public.users ON dau_puzzles.player = users.id
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
	DATE_TRUNC('day',games_plus_puzzle_attempts.created_at) AS day,
	COUNT(DISTINCT player) AS dau
FROM games_plus_puzzle_attempts
LEFT JOIN public.users ON games_plus_puzzle_attempts.player = users.id
WHERE NOT (users.internal_bot OR users.id IN (42,43,44,45,46))
GROUP BY 1)

SELECT
  dau_omgwords_report.day,
  dau_omgwords_report.dau_omgwords,
  dau_omgwords_vs_human_report.dau_omgwords_vs_human,
  TRUNC(100.0*dau_omgwords_vs_human_report.dau_omgwords_vs_human/dau_omgwords_report.dau_omgwords,1)
    AS pct_of_omgwords_dau_who_played_a_human,
  dau_puzzles_report.dau_puzzles,
  omgwords_plus_puzzles_report.dau
FROM dau_omgwords_report
LEFT JOIN dau_omgwords_vs_human_report ON dau_omgwords_report.day = dau_omgwords_vs_human_report.day
LEFT JOIN dau_puzzles_report ON dau_omgwords_report.day = dau_puzzles_report.day
LEFT JOIN omgwords_plus_puzzles_report ON dau_omgwords_report.day = omgwords_plus_puzzles_report.day
ORDER BY 1 DESC
