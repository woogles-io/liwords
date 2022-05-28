-- OMGWords
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

mau_omgwords_report AS
(SELECT
	DATE_TRUNC('month',duplicated_games.created_at) AS month,
    country_code,
	COUNT(DISTINCT player) AS mau_omgwords
FROM duplicated_games
LEFT JOIN public.users ON duplicated_games.player = users.id
WHERE NOT (users.internal_bot OR users.id IN (42,43,44,45,46))
GROUP BY 1,2
ORDER BY 1,3 DESC),

-- we now have a flag for users who are bots, but our earliest
-- bots predate the existence of that flag, hence this query
bot_users AS
(SELECT
   id,
   internal_bot OR id IN (42,43,44,45,46) AS is_bot
FROM public.users),

pvp_games as
(SELECT
   games.created_at,
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
    pvp_games.created_at,
    pvp_games.player0_id AS player,
    profiles.country_code
FROM pvp_games
LEFT JOIN public.profiles ON pvp_games.player0_id = profiles.user_id
)
UNION ALL
(SELECT
    pvp_games.created_at,
    pvp_games.player1_id AS player,
    profiles.country_code
FROM pvp_games
LEFT JOIN public.profiles ON pvp_games.player1_id = profiles.user_id
)),

mau_omgwords_vs_human_report AS
(SELECT
    DATE_TRUNC('month',duplicated_games_between_humans.created_at) AS month,
    country_code,
    COUNT(DISTINCT player) AS mau_omgwords_vs_human
FROM duplicated_games_between_humans
GROUP BY 1,2
ORDER BY 1 DESC,3 DESC),

-- Puzzles
mau_puzzles AS
(SELECT
    puzzle_attempts.created_at,
    puzzle_attempts.user_id AS player,
    profiles.country_code
FROM public.puzzle_attempts
LEFT JOIN public.profiles ON puzzle_attempts.user_id = profiles.user_id
),

mau_puzzles_report AS
(SELECT
	DATE_TRUNC('month',mau_puzzles.created_at) AS month,
    country_code,
	COUNT(DISTINCT player) AS mau_puzzles
FROM mau_puzzles
LEFT JOIN public.users ON mau_puzzles.player = users.id
WHERE NOT users.internal_bot
GROUP BY 1,2
ORDER BY 1,3 DESC),

-- Joint reporting
games_plus_puzzle_attempts AS
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
)
UNION ALL
(SELECT
    puzzle_attempts.created_at,
    puzzle_attempts.user_id AS player,
    profiles.country_code
FROM public.puzzle_attempts
LEFT JOIN public.profiles ON puzzle_attempts.user_id = profiles.user_id
)),

omgwords_plus_puzzles_report AS
(SELECT
	DATE_TRUNC('month',games_plus_puzzle_attempts.created_at) AS month,
    country_code,
	COUNT(DISTINCT player) AS mau
FROM games_plus_puzzle_attempts
LEFT JOIN public.users ON games_plus_puzzle_attempts.player = users.id
WHERE NOT (users.internal_bot OR users.id IN (42,43,44,45,46))
GROUP BY 1,2
ORDER BY 1 DESC,3 DESC)

SELECT
  mau_omgwords_report.month,
  mau_omgwords_report.country_code,
  mau_omgwords_report.mau_omgwords,
  mau_omgwords_vs_human_report.mau_omgwords_vs_human,
  TRUNC(100.0*mau_omgwords_vs_human_report.mau_omgwords_vs_human/mau_omgwords_report.mau_omgwords,1) AS ratio,
  mau_puzzles_report.mau_puzzles,
  omgwords_plus_puzzles_report.mau
FROM mau_omgwords_report
LEFT JOIN mau_omgwords_vs_human_report ON mau_omgwords_report.month = mau_omgwords_vs_human_report.month
  AND mau_omgwords_report.country_code = mau_omgwords_vs_human_report.country_code
LEFT JOIN mau_puzzles_report ON mau_omgwords_report.month = mau_puzzles_report.month
  AND mau_omgwords_report.country_code = mau_puzzles_report.country_code
LEFT JOIN omgwords_plus_puzzles_report ON mau_omgwords_report.month = omgwords_plus_puzzles_report.month
  AND mau_omgwords_report.country_code = omgwords_plus_puzzles_report.country_code
ORDER BY 1 DESC,7 DESC
