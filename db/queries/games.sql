-- name: GetLiveGame :one
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
SELECT game_end_reason, winner_idx, game_request, quickdata, type, tournament_data
FROM past_games
WHERE gid = @gid AND created_at = @created_at;

-- name: GetRematchStreak :many
SELECT gid, winner_idx, quickdata FROM past_games
    WHERE quickdata->>'o' = @orig_req_id::text
	AND game_end_reason <> 5  -- no aborted games
    -- note that cancelled games aren't saved in this table
    -- and neither are ongoing games.
    ORDER BY created_at desc;

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
AND ready_flag & (1 << @player_idx::integer) = 0
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
    game_request, game_document, stats, quickdata, type, tournament_data
) VALUES (
    @gid, @created_at, @game_end_reason, @winner_idx,
    @game_request, @game_document, @stats, @quickdata, @type, @tournament_data
);

-- name: InsertGamePlayer :exec
INSERT INTO game_players (
    game_uuid, player_id, player_index, score, won, game_end_reason,
    rating_before, rating_after, rating_delta, created_at, game_type,
    opponent_id, opponent_score
) VALUES (
    @game_uuid, @player_id, @player_index, @score, @won, @game_end_reason,
    @rating_before, @rating_after, @rating_delta, @created_at, @game_type,
    @opponent_id, @opponent_score
);

-- name: UpdateGameMigrationStatus :exec
UPDATE games 
SET migration_status = @migration_status,
    updated_at = NOW()
WHERE uuid = @uuid;

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
       COALESCE(pg.game_request, '{}') as game_request,
       COALESCE(pg.winner_idx, CASE WHEN gp.won = true THEN gp.player_index 
                                   WHEN gp.won = false THEN (1 - gp.player_index)
                                   ELSE -1 END) as winner_idx
FROM game_players gp
JOIN users u ON u.id = gp.opponent_id
JOIN users player ON player.id = gp.player_id
LEFT JOIN past_games pg ON pg.gid = gp.game_uuid
WHERE LOWER(player.username) = LOWER(@username)
ORDER BY gp.created_at DESC
LIMIT @num_games OFFSET @offset_games;

-- name: GetRecentTourneyGames :many
SELECT pg.gid, pg.quickdata, pg.game_request, pg.winner_idx, pg.game_end_reason,
       pg.created_at, pg.type, pg.tournament_data
FROM past_games pg
WHERE pg.tournament_data->>'Id' = @tourney_id::text
ORDER BY pg.created_at DESC
LIMIT @num_games OFFSET @offset_games;
