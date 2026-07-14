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

    AND games.game_request->>'lexicon' = $3::text
    AND (
        games.game_request->'rules'->>'variant_name' = 'classic'
        OR games.game_request->'rules'->>'variant_name' IS NULL
        OR games.game_request->'rules'->>'variant_name' = ''
    )
    -- 0: none, 5: aborted, 7: canceled
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
    AND games.game_request->>'lexicon' = $3::text
    AND (
        games.game_request->'rules'->>'variant_name' = 'classic'
        OR games.game_request->'rules'->>'variant_name' IS NULL
        OR games.game_request->'rules'->>'variant_name' = ''
    )
    -- 0: none, 5: aborted, 7: canceled
    AND game_end_reason not in (0, 5, 7)
    AND type = 0

    ORDER BY games.id DESC
    LIMIT $4 OFFSET $5;
-- name: GetPuzzleDBIDFromUUID :one
SELECT id FROM puzzles WHERE uuid = @uuid;

-- name: CreatePuzzleGenerationLog :one
INSERT INTO puzzle_generation_logs (request, created_at) VALUES (@request, NOW()) RETURNING id;

-- name: UpdateGenerationLogStatus :execrows
UPDATE puzzle_generation_logs SET completed_at = NOW(), error_status = @error_status, fulfilled = @fulfilled WHERE id = @id;

-- name: UpsertPuzzleVote :execrows
INSERT INTO puzzle_votes (puzzle_id, user_id, vote) VALUES (@puzzle_id, @user_id, @vote)
ON CONFLICT (puzzle_id, user_id) DO UPDATE SET vote = @vote;
