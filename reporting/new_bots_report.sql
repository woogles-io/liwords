WITH bot_users AS
(SELECT
   id,
   username,
   internal_bot OR id IN (42,43,44,45,46) AS is_bot
FROM public.users),
bot_games as
(SELECT
   created_at,
   games.player0_id,
   games.player1_id,
   b1.username AS player0_username,
   b2.username AS player1_username,
   b1.is_bot AS player0_isbot,
   b2.is_bot AS player1_isbot
 FROM public.games
 LEFT JOIN bot_users b1 ON games.player0_id=b1.id
 LEFT JOIN bot_users b2 ON games.player1_id=b2.id
 WHERE (b1.is_bot OR b2.is_bot)
 ),
duplicated_bot_games AS
((SELECT
    created_at,
    player0_id AS player,
    player0_username AS bot_name
FROM bot_games
WHERE player0_isbot
 )
UNION ALL
(SELECT
    created_at,
    player1_id AS player,
    player1_username AS bot_name 
FROM bot_games
WHERE player1_isbot))

SELECT
  bot_name,
  COUNT(*)
FROM duplicated_bot_games
WHERE created_at > TIMESTAMP '2021-09-02 21:30:00'
GROUP BY bot_name
ORDER BY 2 DESC