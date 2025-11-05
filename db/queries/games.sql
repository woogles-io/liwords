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
    quickdata, tournament_data, tournament_id,
    created_at, updated_at, game_request
FROM games
WHERE uuid = @uuid;

-- name: GetRematchStreak :many
SELECT g.uuid, g.winner_idx, g.quickdata
FROM games g
WHERE g.uuid IN (
  SELECT DISTINCT gp.game_uuid
  FROM game_players gp
  WHERE gp.original_request_id = @original_request_id::text
    AND gp.game_end_reason NOT IN (0, 5, 7) -- NONE, ABORTED, CANCELLED
)
ORDER BY g.created_at DESC;

-- name: GetRecentGamesByUsername :many
WITH recent_game_uuids AS (
  SELECT gp.game_uuid, gp.created_at
  FROM game_players gp
  WHERE gp.player_id = (SELECT id FROM users WHERE lower(username) = lower(@username))
    AND gp.game_end_reason NOT IN (0, 5, 7)  -- NONE, ABORTED, CANCELLED
  ORDER BY gp.created_at DESC
  LIMIT @num_games::integer
  OFFSET @offset_games::integer
)
SELECT g.id, g.uuid, g.type, g.player0_id, g.player1_id,
       g.timers, g.started, g.game_end_reason, g.winner_idx, g.loser_idx,
       g.quickdata, g.tournament_data, g.created_at, g.updated_at,
       g.game_request
FROM recent_game_uuids rgu
JOIN games g ON rgu.game_uuid = g.uuid
ORDER BY rgu.created_at DESC;

-- name: GetRecentCorrespondenceGamesByUsername :many
WITH recent_game_uuids AS (
  SELECT gp.game_uuid, gp.created_at
  FROM game_players gp
  JOIN games g ON gp.game_uuid = g.uuid
  WHERE gp.player_id = (SELECT id FROM users WHERE lower(username) = lower(@username))
    AND gp.game_end_reason NOT IN (0, 5, 7)  -- NONE, ABORTED, CANCELLED
    AND (g.game_request->>'game_mode')::int = 1  -- CORRESPONDENCE only
  ORDER BY gp.created_at DESC
  LIMIT @num_games::integer
)
SELECT g.id, g.uuid, g.type, g.player0_id, g.player1_id,
       g.timers, g.started, g.game_end_reason, g.winner_idx, g.loser_idx,
       g.quickdata, g.tournament_data, g.created_at, g.updated_at,
       g.game_request
FROM recent_game_uuids rgu
JOIN games g ON rgu.game_uuid = g.uuid
ORDER BY rgu.created_at DESC;

-- name: GetRecentGamesByPlayerID :many
WITH recent_game_uuids AS (
  SELECT gp.game_uuid, gp.created_at
  FROM game_players gp
  WHERE gp.player_id = @player_id::integer
    AND gp.game_end_reason NOT IN (0, 5, 7)  -- NONE, ABORTED, CANCELLED
  ORDER BY gp.created_at DESC
  LIMIT @num_games::integer
  OFFSET @offset_games::integer
)
SELECT g.id, g.uuid, g.type, g.player0_id, g.player1_id,
       g.timers, g.started, g.game_end_reason, g.winner_idx, g.loser_idx,
       g.quickdata, g.tournament_data, g.created_at, g.updated_at,
       g.game_request
FROM recent_game_uuids rgu
JOIN games g ON rgu.game_uuid = g.uuid
ORDER BY rgu.created_at DESC;

-- name: GetRecentGamesByUsernameOptimized :many
WITH recent_game_uuids AS (
  SELECT gp.game_uuid, gp.created_at
  FROM game_players gp
  WHERE gp.player_id = (SELECT id FROM users WHERE lower(username) = lower(@username))
    AND gp.game_end_reason NOT IN (0, 5, 7)  -- NONE, ABORTED, CANCELLED
  ORDER BY gp.created_at DESC
  LIMIT @num_games::integer
  OFFSET @offset_games::integer
)
SELECT g.id, g.uuid, g.type, g.player0_id, g.player1_id,
       g.timers, g.started, g.game_end_reason, g.winner_idx, g.loser_idx,
       g.quickdata, g.tournament_data, g.created_at, g.updated_at,
       g.game_request
FROM recent_game_uuids rgu
JOIN games g ON rgu.game_uuid = g.uuid
ORDER BY rgu.created_at DESC;

-- name: GetRecentTourneyGames :many
SELECT
    id, uuid, type, player0_id, player1_id,
    timers, started, game_end_reason, winner_idx, loser_idx,
    quickdata, tournament_data, created_at, updated_at, game_request
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
    quickdata, history, meta_events, stats, tournament_id, tournament_data, game_request, player_on_turn,
    league_id, season_id, league_division_id
) VALUES (
    @created_at, @updated_at, @uuid, @type, @player0_id, @player1_id,
    @ready_flag, @timers, @started, @game_end_reason, @winner_idx, @loser_idx,
    @quickdata, @history, @meta_events, @stats, @tournament_id, @tournament_data, @game_request, @player_on_turn,
    @league_id, @season_id, @league_division_id
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
    history = @history,
    meta_events = @meta_events,
    stats = @stats,
    tournament_data = @tournament_data,
    tournament_id = @tournament_id,
    ready_flag = @ready_flag,
    game_request = @game_request,
    player_on_turn = @player_on_turn
WHERE uuid = @uuid;

-- name: CreateRawGame :exec
INSERT INTO games (
    uuid, history, quickdata, timers, game_end_reason, type, game_request
) VALUES (
    @uuid, @history, @quickdata, @timers, @game_end_reason, @type, @game_request
);

-- name: ListActiveGames :many
SELECT quickdata, uuid, started, tournament_data, game_request, player_on_turn
FROM games
WHERE game_end_reason = 0 -- NONE (ongoing games)
    AND COALESCE((game_request->>'game_mode')::int, 0) != 1 -- Exclude CORRESPONDENCE games
ORDER BY id;

-- name: ListActiveTournamentGames :many
SELECT quickdata, uuid, started, tournament_data, game_request, player_on_turn
FROM games
WHERE game_end_reason = 0 -- NONE (ongoing games)
    AND tournament_id = @tournament_id::text
    AND COALESCE((game_request->>'game_mode')::int, 0) != 1 -- Exclude CORRESPONDENCE games
ORDER BY id;

-- name: ListActiveCorrespondenceGames :many
SELECT quickdata, uuid, started, tournament_data, game_request, player_on_turn, timers, type, updated_at
FROM games
WHERE game_end_reason = 0 -- NONE (ongoing games)
    AND (game_request->>'game_mode')::int = 1 -- Only CORRESPONDENCE games
ORDER BY id;

-- name: ListActiveCorrespondenceGamesForUser :many
SELECT quickdata, uuid, started, tournament_data, game_request, player_on_turn, updated_at
FROM games
WHERE game_end_reason = 0 -- NONE (ongoing games)
    AND (game_request->>'game_mode')::int = 1 -- Only CORRESPONDENCE games
    AND (
        player0_id = (SELECT id FROM users WHERE uuid = @user_uuid::text)
        OR player1_id = (SELECT id FROM users WHERE uuid = @user_uuid::text)
    )
ORDER BY id;

-- name: CountActiveCorrespondenceGames :one
SELECT COUNT(*)::int
FROM games
WHERE game_end_reason = 0 -- NONE (ongoing games)
    AND (game_request->>'game_mode')::int = 1; -- Only CORRESPONDENCE games

-- name: ListActiveCorrespondenceGamesWithBotOnTurn :many
SELECT g.uuid
FROM games g
WHERE g.game_end_reason = 0 -- NONE (ongoing games)
    AND (g.game_request->>'game_mode')::int = 1 -- Only CORRESPONDENCE games
    AND g.player_on_turn IS NOT NULL
    AND (
        (g.player_on_turn = 0 AND EXISTS (
            SELECT 1 FROM users u WHERE u.id = g.player0_id AND u.internal_bot = true AND lower(u.username) != 'bestbot'
        ))
        OR
        (g.player_on_turn = 1 AND EXISTS (
            SELECT 1 FROM users u WHERE u.id = g.player1_id AND u.internal_bot = true AND lower(u.username) != 'bestbot'
        ))
    )
ORDER BY g.id;

-- name: ListActiveRealtimeGamesWithBotOnTurn :many
SELECT g.uuid
FROM games g
WHERE g.game_end_reason = 0 -- NONE (ongoing games)
    AND COALESCE((g.game_request->>'game_mode')::int, 0) != 1 -- Exclude CORRESPONDENCE games
    AND g.player_on_turn IS NOT NULL
    AND (
        (g.player_on_turn = 0 AND EXISTS (
            SELECT 1 FROM users u WHERE u.id = g.player0_id AND u.internal_bot = true AND lower(u.username) != 'bestbot'
        ))
        OR
        (g.player_on_turn = 1 AND EXISTS (
            SELECT 1 FROM users u WHERE u.id = g.player1_id AND u.internal_bot = true AND lower(u.username) != 'bestbot'
        ))
    )
ORDER BY g.id;

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

-- name: InsertGamePlayers :exec
INSERT INTO game_players (
    game_uuid,
    player_id,
    player_index,
    score,
    won,
    game_end_reason,
    created_at,
    game_type,
    opponent_id,
    opponent_score,
    original_request_id
) VALUES
    -- Player 0
    (
        @game_uuid,
        @player0_id,
        0,
        @player0_score,
        @player0_won,
        @game_end_reason,
        @created_at,
        @game_type,
        @player1_id,
        @player1_score,
        @original_request_id
    ),
    -- Player 1
    (
        @game_uuid,
        @player1_id,
        1,
        @player1_score,
        @player1_won,
        @game_end_reason,
        @created_at,
        @game_type,
        @player0_id,
        @player0_score,
        @original_request_id
    )
ON CONFLICT (game_uuid, player_id) DO NOTHING;

