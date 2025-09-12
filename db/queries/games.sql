-- name: GetGame :one
SELECT * FROM games WHERE uuid = @uuid; -- this is not even a uuid, sigh.

-- name: GetGameOwner :one
SELECT
    agm.creator_uuid,
    u.username
FROM annotated_game_metadata agm
JOIN users u ON agm.creator_uuid = u.uuid
WHERE agm.game_uuid = @game_uuid;

-- name: GetGameMetadata :one
SELECT
    id, uuid, type, player0_id, player1_id,
    timers, started, game_end_reason, winner_idx, loser_idx,
    quickdata, request, tournament_data, tournament_id,
    created_at, updated_at, game_request
FROM games
WHERE uuid = @uuid;

-- name: GetRematchStreak :many
SELECT uuid, winner_idx, quickdata
FROM games
WHERE quickdata->>'o' = @original_request_id::text
    AND game_end_reason NOT IN (0, 5, 7) -- NONE, ABORTED, CANCELLED
ORDER BY created_at DESC;

-- name: GetRecentGamesByUsername :many
WITH user_id AS (
    SELECT id FROM users WHERE lower(username) = lower(@username)
)
SELECT g.id, g.uuid, g.type, g.player0_id, g.player1_id,
        g.timers, g.started, g.game_end_reason, g.winner_idx, g.loser_idx,
        g.quickdata, g.request, g.tournament_data, g.created_at, g.updated_at,
        g.game_request
FROM games g, user_id u
WHERE (g.player0_id = u.id OR g.player1_id = u.id)
AND g.game_end_reason NOT IN (0, 5, 7)
ORDER BY g.id DESC
LIMIT @num_games::integer
OFFSET @offset_games::integer;

-- name: GetRecentTourneyGames :many
SELECT
    id, uuid, type, player0_id, player1_id,
    timers, started, game_end_reason, winner_idx, loser_idx,
    quickdata, request, tournament_data, created_at, updated_at, game_request
FROM games
WHERE tournament_id = @tourney_id::text
    AND game_end_reason NOT IN (0, 5, 7) -- NONE, ABORTED, CANCELLED
ORDER BY updated_at DESC
LIMIT @num_games::integer
OFFSET @offset_games::integer;

-- name: CreateGame :exec
INSERT INTO games (
    created_at, updated_at, uuid, type, player0_id, player1_id,
    ready_flag, timers, started, game_end_reason, winner_idx, loser_idx,
    quickdata, request, history, meta_events, stats, tournament_id, tournament_data, game_request
) VALUES (
    @created_at, @updated_at, @uuid, @type, @player0_id, @player1_id,
    @ready_flag, @timers, @started, @game_end_reason, @winner_idx, @loser_idx,
    @quickdata, @request, @history, @meta_events, @stats, @tournament_id, @tournament_data, @game_request
);

-- name: UpdateGame :exec
UPDATE games SET
    updated_at = @updated_at,
    player0_id = @player0_id,
    player1_id = @player1_id,
    timers = @timers,
    started = @started,
    game_end_reason = @game_end_reason,
    winner_idx = @winner_idx,
    loser_idx = @loser_idx,
    quickdata = @quickdata,
    request = @request,
    history = @history,
    meta_events = @meta_events,
    stats = @stats,
    tournament_data = @tournament_data,
    tournament_id = @tournament_id,
    ready_flag = @ready_flag,
    game_request = @game_request
WHERE uuid = @uuid;

-- name: CreateRawGame :exec
INSERT INTO games (
    uuid, request, history, quickdata, timers, game_end_reason, type, game_request
) VALUES (
    @uuid, @request, @history, @quickdata, @timers, @game_end_reason, @type, @game_request
);

-- name: ListActiveGames :many
SELECT quickdata, request, uuid, started, tournament_data, game_request
FROM games
WHERE game_end_reason = 0 -- NONE (ongoing games)
ORDER BY id;

-- name: ListActiveTournamentGames :many
SELECT quickdata, request, uuid, started, tournament_data, game_request
FROM games
WHERE game_end_reason = 0 -- NONE (ongoing games)
    AND tournament_id = @tournament_id::text
ORDER BY id;

-- name: SetReady :one
UPDATE games
SET ready_flag = ready_flag | (1 << @player_idx::integer)
WHERE uuid = @uuid
    AND ready_flag & (1 << @player_idx::integer) = 0
RETURNING ready_flag;

-- name: ListAllIDs :many
SELECT uuid FROM games
ORDER BY created_at ASC;

-- name: GetHistory :one
SELECT history FROM games
WHERE uuid = @uuid;

-- name: GameExists :one
SELECT EXISTS (
    SELECT 1 FROM games WHERE uuid = @uuid
) AS exists;

-- name: GameCount :one
SELECT COUNT(*) FROM games;

