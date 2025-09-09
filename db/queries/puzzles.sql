-- puzzle generation

-- name: GetPotentialPuzzleGamesAvoidBots :many
SELECT past_games.gid FROM past_games
JOIN games ON past_games.gid = games.uuid
JOIN game_metadata ON game_metadata.game_uuid = past_games.gid
LEFT JOIN puzzles ON puzzles.game_id = games.id
WHERE puzzles.id IS NULL 
    AND past_games.created_at BETWEEN $1 AND $2
    AND (past_games.stats->'d1'->'Challenged Phonies'->'t' = '0')
    AND (past_games.stats->'d2'->'Challenged Phonies'->'t' = '0')
    AND (past_games.stats->'d1'->'Unchallenged Phonies'->'t' = '0')
    AND (past_games.stats->'d2'->'Unchallenged Phonies'->'t' = '0')
    AND game_metadata.game_request->>'lexicon' = $3::text
    AND game_metadata.game_request->'rules'->>'variantName' = 'classic'
    -- 0: none, 5: aborted, 7: canceled
    AND past_games.game_end_reason NOT IN (0, 5, 7)
    AND NOT (past_games.quickdata @> '{"pi": [{"is_bot": true}]}'::jsonb)
    AND past_games.type = 0

    ORDER BY games.id DESC 
    LIMIT $4 OFFSET $5;


-- name: GetPotentialPuzzleGames :many
SELECT past_games.gid FROM past_games
JOIN games ON past_games.gid = games.uuid
JOIN game_metadata ON game_metadata.game_uuid = past_games.gid
LEFT JOIN puzzles ON puzzles.game_id = games.id
WHERE puzzles.id IS NULL 
    AND past_games.created_at BETWEEN $1 AND $2
    AND (past_games.stats->'d1'->'Challenged Phonies'->'t' = '0')
    AND (past_games.stats->'d2'->'Challenged Phonies'->'t' = '0')
    AND (past_games.stats->'d1'->'Unchallenged Phonies'->'t' = '0')
    AND (past_games.stats->'d2'->'Unchallenged Phonies'->'t' = '0')
    AND game_metadata.game_request->>'lexicon' = $3::text
    AND game_metadata.game_request->'rules'->>'variantName' = 'classic'
    -- 0: none, 5: aborted, 7: canceled
    AND past_games.game_end_reason NOT IN (0, 5, 7)
    AND past_games.type = 0
    
    ORDER BY games.id DESC 
    LIMIT $4 OFFSET $5;