-- name: GetGameBasicInfo :one
SELECT id, uuid, game_end_reason, migration_status, created_at, updated_at, type
FROM games WHERE uuid = @uuid;

-- name: GetGameFullData :one
SELECT * FROM games WHERE uuid = @uuid;

-- name: GetGameOwner :one
SELECT
    agm.creator_uuid,
    u.username
FROM annotated_game_metadata agm
JOIN users u ON agm.creator_uuid = u.uuid
WHERE agm.game_uuid = @game_uuid;

-- name: GetPastGame :one
SELECT * FROM past_games WHERE gid = @gid AND created_at = @created_at;

-- name: GetLiveGameMetadata :one
SELECT uuid, quickdata, game_end_reason, winner_idx, request, created_at, updated_at,
        tournament_data, tournament_id, type
FROM games
WHERE uuid = @uuid;

-- name: GetPastGameMetadata :one
SELECT pg.game_end_reason, pg.winner_idx, gm.game_request, pg.quickdata, pg.type, gm.tournament_data
FROM past_games pg
JOIN game_metadata gm ON gm.game_uuid = pg.gid
WHERE pg.gid = @gid AND pg.created_at = @created_at;

-- name: GetRematchStreak :many
SELECT DISTINCT game_uuid as gid,
       CASE WHEN won = true THEN player_index
            WHEN won = false THEN (1 - player_index)
            ELSE -1 END as winner_idx,
       created_at
FROM game_players
WHERE original_request_id = @orig_req_id::text
    AND game_end_reason <> 5  -- no aborted games
    -- note that cancelled games aren't saved in this table
    -- and neither are ongoing games.
ORDER BY created_at DESC;

-- name: GetRematchStreakOld :many
-- Backward-compatible query that reads from games table instead of game_players
SELECT DISTINCT uuid as gid,
       winner_idx,
       created_at
FROM games
WHERE quickdata->>'o' = @orig_req_id::text
    AND game_end_reason <> 5  -- no aborted games
    AND game_end_reason <> 3  -- no cancelled games
    AND game_end_reason > 0   -- only ended games
ORDER BY created_at DESC;

-- name: CreateGame :exec
INSERT INTO games (
    created_at, updated_at, uuid, player0_id, player1_id, timers,
    started, game_end_reason, winner_idx, loser_idx, request,
    history, stats, quickdata, tournament_data, tournament_id,
    ready_flag, meta_events, type)
VALUES (
    @created_at, @updated_at, @uuid, @player0_id, @player1_id, @timers,
    @started, @game_end_reason, @winner_idx, @loser_idx, @request,
    @history, @stats, @quickdata, @tournament_data, @tournament_id,
    @ready_flag, @meta_events, @type)
RETURNING id;

-- name: UpdateGame :exec
UPDATE games
SET updated_at = @updated_at,
    player0_id = @player0_id,
    player1_id = @player1_id,
    timers = @timers,
    started = @started,
    game_end_reason = @game_end_reason,
    winner_idx = @winner_idx,
    loser_idx = @loser_idx,
    request = @request,
    history = @history,
    stats = @stats,
    quickdata = @quickdata,
    tournament_data = @tournament_data,
    tournament_id = @tournament_id,
    ready_flag = @ready_flag,
    meta_events = @meta_events
WHERE uuid = @uuid;


-- name: CreateRawGame :exec
INSERT INTO games(uuid, request, history, quickdata, timers,
			game_end_reason, type)
VALUES(@uuid, @request, @history, @quickdata, @timers,
            @game_end_reason, @type);

-- name: ListActiveGames :many
SELECT quickdata, request, uuid, started, tournament_data
FROM games
WHERE game_end_reason = 0;

-- name: ListActiveTournamentGames :many
SELECT quickdata, request, uuid, started, tournament_data
FROM games
WHERE game_end_reason = 0
AND tournament_id = @tournament_id;

-- name: SetReady :one
UPDATE games SET ready_flag = ready_flag | (1 << @player_idx::integer)
WHERE uuid = @uuid
RETURNING ready_flag;

-- name: ListAllIDs :many
SELECT uuid FROM games
ORDER BY created_at ASC;

-- name: GetHistory :one
SELECT history FROM games
WHERE uuid = @uuid;

-- name: InsertPastGame :exec
INSERT INTO past_games (
    gid, created_at, game_end_reason, winner_idx,
    game_document, stats, quickdata, type
) VALUES (
    @gid, @created_at, @game_end_reason, @winner_idx,
    @game_document, @stats, @quickdata, @type
);

-- name: InsertGamePlayer :exec
INSERT INTO game_players (
    game_uuid, player_id, player_index, score, won, game_end_reason,
    rating_before, rating_after, rating_delta, created_at, game_type,
    opponent_id, opponent_score, original_request_id
) VALUES (
    @game_uuid, @player_id, @player_index, @score, @won, @game_end_reason,
    @rating_before, @rating_after, @rating_delta, @created_at, @game_type,
    @opponent_id, @opponent_score, @original_request_id
);

-- name: UpdateGameMigrationStatus :exec
UPDATE games
SET migration_status = @migration_status,
    updated_at = NOW()
WHERE uuid = @uuid;

-- name: InsertGameMetadata :exec
INSERT INTO game_metadata (
    game_uuid, created_at, game_request, tournament_data
) VALUES (
    @game_uuid, @created_at, @game_request, @tournament_data
);

-- name: GetGameMetadata :one
SELECT game_uuid, created_at, game_request, tournament_data
FROM game_metadata 
WHERE game_uuid = @game_uuid;

-- name: ClearGameDataAfterMigration :exec
UPDATE games
SET history = NULL,
    stats = NULL,
    quickdata = NULL,
    timers = NULL,
    meta_events = NULL,
    request = NULL,
    tournament_data = NULL,
    player0_id = NULL,
    player1_id = NULL,
    updated_at = NOW()
WHERE uuid = @uuid;

-- name: GetGamePlayers :many
SELECT player_id, player_index
FROM game_players
WHERE game_uuid = @game_uuid
ORDER BY player_index;

-- name: GetRecentGamesByUsername :many
SELECT gp.game_uuid, gp.score, gp.opponent_score, gp.won, gp.game_end_reason,
       gp.created_at, gp.game_type, u.username as opponent_username,
       COALESCE(pg.quickdata, '{}') as quickdata,
       gm.game_request,
       gm.tournament_data,
       COALESCE(pg.winner_idx, CASE WHEN gp.won = true THEN gp.player_index
                                   WHEN gp.won = false THEN (1 - gp.player_index)
                                   ELSE -1 END) as winner_idx
FROM game_players gp
JOIN users u ON u.id = gp.opponent_id
JOIN users player ON player.id = gp.player_id
JOIN game_metadata gm ON gm.game_uuid = gp.game_uuid
LEFT JOIN past_games pg ON pg.gid = gp.game_uuid
WHERE LOWER(player.username) = LOWER(@username)
ORDER BY gp.created_at DESC
LIMIT @num_games OFFSET @offset_games;

-- name: GetRecentGamesByUsernameOld :many
-- Backward-compatible query that reads from games table
SELECT g.uuid as game_uuid,
       CASE WHEN u1.username = @username THEN (g.quickdata->'finalScores'->>0)::int 
            ELSE (g.quickdata->'finalScores'->>1)::int END as score,
       CASE WHEN u1.username = @username THEN (g.quickdata->'finalScores'->>1)::int 
            ELSE (g.quickdata->'finalScores'->>0)::int END as opponent_score,
       CASE WHEN g.winner_idx = 0 AND u1.username = @username THEN true
            WHEN g.winner_idx = 1 AND u2.username = @username THEN true
            WHEN g.winner_idx = -1 THEN NULL
            ELSE false END as won,
       g.game_end_reason,
       g.created_at,
       g.type as game_type,
       CASE WHEN u1.username = @username THEN u2.username 
            ELSE u1.username END as opponent_username,
       g.quickdata,
       g.request as game_request,
       g.winner_idx
FROM games g
LEFT JOIN users u1 ON g.player0_id = u1.id
LEFT JOIN users u2 ON g.player1_id = u2.id
WHERE (LOWER(u1.username) = LOWER(@username) OR LOWER(u2.username) = LOWER(@username))
  AND g.game_end_reason > 0  -- only ended games
ORDER BY g.created_at DESC
LIMIT @num_games OFFSET @offset_games;

-- name: GetRecentTourneyGames :many
SELECT pg.gid, pg.quickdata, gm.game_request, pg.winner_idx, pg.game_end_reason,
       pg.created_at, pg.type, gm.tournament_data
FROM past_games pg
JOIN game_metadata gm ON gm.game_uuid = pg.gid
WHERE gm.tournament_data->>'Id' = @tourney_id::text
ORDER BY pg.created_at DESC
LIMIT @num_games OFFSET @offset_games;

-- name: GetRecentTourneyGamesOld :many
-- Backward-compatible query that reads from games table
SELECT g.uuid as gid, g.quickdata, g.request as game_request, g.winner_idx, g.game_end_reason,
       g.created_at, g.type, g.tournament_data
FROM games g
WHERE g.tournament_id = @tourney_id::text
  AND g.game_end_reason > 0  -- only ended games
ORDER BY g.created_at DESC
LIMIT @num_games OFFSET @offset_games;

-- name: GameExists :one
SELECT EXISTS (
    SELECT 1 FROM games WHERE uuid = @uuid
) AS exists;