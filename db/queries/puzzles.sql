-- puzzle generation

-- name: GetPotentialPuzzleGamesAvoidBots :many
SELECT games.uuid FROM games
LEFT JOIN puzzles on puzzles.game_id = games.id
WHERE puzzles.id IS NULL
    AND games.created_at BETWEEN $1 AND $2
    AND (stats->'d1'->'Challenged Phonies'->'t' = '0')
    AND (stats->'d2'->'Challenged Phonies'->'t' = '0')
    AND (stats->'d1'->'Unchallenged Phonies'->'t' = '0')
    AND (stats->'d2'->'Unchallenged Phonies'->'t' = '0')
    AND games.request LIKE $3  -- %lexicon%
    AND games.request NOT LIKE '%classic_super%'
    AND games.request NOT LIKE '%wordsmog%'
    -- 0: none, 5: aborted, 7: cancelled
    AND game_end_reason not in (0, 5, 7)
    AND NOT (quickdata @> '{"pi": [{"is_bot": true}]}'::jsonb)
    AND type = 0

    ORDER BY games.id DESC
    LIMIT $4 OFFSET $5;


-- name: GetPotentialPuzzleGames :many
SELECT games.uuid FROM games
LEFT JOIN puzzles on puzzles.game_id = games.id
WHERE puzzles.id IS NULL
    AND games.created_at BETWEEN $1 AND $2
    AND (stats->'d1'->'Challenged Phonies'->'t' = '0')
    AND (stats->'d2'->'Challenged Phonies'->'t' = '0')
    AND (stats->'d1'->'Unchallenged Phonies'->'t' = '0')
    AND (stats->'d2'->'Unchallenged Phonies'->'t' = '0')
    AND games.request LIKE $3  -- %lexicon%
    AND games.request NOT LIKE '%classic_super%'
    AND games.request NOT LIKE '%wordsmog%'
    -- 0: none, 5: aborted, 7: cancelled
    AND game_end_reason not in (0, 5, 7)
    AND type = 0

    ORDER BY games.id DESC
    LIMIT $4 OFFSET $5;