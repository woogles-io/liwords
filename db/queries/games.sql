-- name: GetGame :one
SELECT
    id, created_at, updated_at, deleted_at, uuid,
    player0_id, player1_id, timers, started, game_end_reason,
    winner_idx, loser_idx, history, stats, quickdata,
    tournament_data, tournament_id, ready_flag, meta_events, type,
    game_request, player_on_turn, league_id, season_id, league_division_id,
    history_s3_key, last_known_racks
FROM games
WHERE uuid = @uuid; -- this is not even a uuid, sigh.

-- name: GetGameOwner :one
SELECT
    agm.creator_uuid,
    u.username
FROM annotated_game_metadata agm
JOIN users u ON agm.creator_uuid = u.uuid
WHERE agm.game_uuid = @game_uuid;

-- name: GetUserUUIDByUsername :one
SELECT uuid FROM users WHERE lower(username) = lower(@username);

-- name: GetLatestAnnotatedGameForUsername :one
SELECT agm.game_uuid, agm.creator_uuid
FROM annotated_game_metadata agm
JOIN games g ON g.uuid = agm.game_uuid
WHERE agm.creator_uuid = (SELECT uuid FROM users WHERE lower(username) = lower(@username))
ORDER BY g.updated_at DESC
LIMIT 1;

-- name: GetGameMetadata :one
SELECT
    id, uuid, type, player0_id, player1_id,
    timers, started, game_end_reason, winner_idx, loser_idx,
    quickdata, tournament_data, tournament_id,
    created_at, updated_at, game_request,
    league_id, season_id, league_division_id
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

-- name: GetRecentGamesByUserId :many
WITH recent_game_uuids AS (
  SELECT gp.game_uuid, gp.updated_at
  FROM game_players gp
  WHERE gp.player_id = @user_id
    AND gp.game_end_reason NOT IN (0, 5, 7)  -- NONE, ABORTED, CANCELLED
  ORDER BY gp.updated_at DESC
  LIMIT @num_games::integer
  OFFSET @offset_games::integer
)
SELECT g.id, g.uuid, g.type, g.player0_id, g.player1_id,
       g.timers, g.started, g.game_end_reason, g.winner_idx, g.loser_idx,
       g.quickdata, g.tournament_data, g.created_at, g.updated_at,
       g.game_request, g.league_id, g.season_id, g.league_division_id
FROM recent_game_uuids rgu
JOIN games g ON rgu.game_uuid = g.uuid
ORDER BY rgu.updated_at DESC;

-- name: GetRecentCorrespondenceGamesByUserId :many
WITH recent_game_uuids AS (
  SELECT gp.game_uuid, gp.updated_at
  FROM game_players gp
  JOIN games g ON gp.game_uuid = g.uuid
  WHERE gp.player_id = @user_id
    AND gp.game_end_reason NOT IN (0, 5, 7)  -- NONE, ABORTED, CANCELLED
    AND (g.game_request->>'game_mode')::int = 1  -- CORRESPONDENCE only
  ORDER BY gp.updated_at DESC
  LIMIT @num_games::integer
)
SELECT g.id, g.uuid, g.type, g.player0_id, g.player1_id,
       g.timers, g.started, g.game_end_reason, g.winner_idx, g.loser_idx,
       g.quickdata, g.tournament_data, g.created_at, g.updated_at,
       g.game_request, g.league_id, g.season_id, g.league_division_id,
       l.slug as league_slug
FROM recent_game_uuids rgu
JOIN games g ON rgu.game_uuid = g.uuid
LEFT JOIN leagues l ON g.league_id = l.uuid
ORDER BY rgu.updated_at DESC;

-- name: GetRecentTourneyGames :many
SELECT
    id, uuid, type, player0_id, player1_id,
    timers, started, game_end_reason, winner_idx, loser_idx,
    quickdata, tournament_data, created_at, updated_at, game_request,
    league_id, season_id, league_division_id
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
    league_id, season_id, league_division_id, last_known_racks
) VALUES (
    @created_at, @updated_at, @uuid, @type, @player0_id, @player1_id,
    @ready_flag, @timers, @started, @game_end_reason, @winner_idx, @loser_idx,
    @quickdata, @history, @meta_events, @stats, @tournament_id, @tournament_data, @game_request, @player_on_turn,
    @league_id, @season_id, @league_division_id, @last_known_racks
);

-- name: UpdateGameTimers :exec
UPDATE games SET timers = @timers, updated_at = now()
WHERE uuid = @uuid;

-- name: UpdateGameMetaEvents :exec
UPDATE games SET meta_events = @meta_events, updated_at = now()
WHERE uuid = @uuid;

-- name: UpdateGameStarted :exec
UPDATE games SET started = @started, timers = @timers, updated_at = now()
WHERE uuid = @uuid;

-- name: UpdateGameAfterMove :exec
UPDATE games SET
    history = @history,
    timers = @timers,
    player_on_turn = @player_on_turn,
    updated_at = now()
WHERE uuid = @uuid;

-- name: UpdateGameEnd :exec
UPDATE games SET
    game_end_reason = @game_end_reason,
    winner_idx = @winner_idx,
    loser_idx = @loser_idx,
    history = @history,
    stats = @stats,
    quickdata = @quickdata,
    timers = @timers,
    updated_at = now()
WHERE uuid = @uuid;

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
    player_on_turn = @player_on_turn,
    league_id = @league_id,
    season_id = @season_id,
    league_division_id = @league_division_id,
    last_known_racks = @last_known_racks
WHERE uuid = @uuid;

-- name: CreateRawGame :exec
INSERT INTO games (
    uuid, history, quickdata, timers, game_end_reason, type, game_request
) VALUES (
    @uuid, @history, @quickdata, @timers, @game_end_reason, @type, @game_request
);

-- name: ListActiveGames :many
SELECT quickdata, uuid, started, tournament_data, game_request, player_on_turn, league_id, season_id, league_division_id
FROM games
WHERE game_end_reason = 0 -- NONE (ongoing games)
    AND COALESCE((game_request->>'game_mode')::int, 0) != 1 -- Exclude CORRESPONDENCE games
ORDER BY id;

-- name: ListActiveTournamentGames :many
SELECT quickdata, uuid, started, tournament_data, game_request, player_on_turn, league_id, season_id, league_division_id
FROM games
WHERE game_end_reason = 0 -- NONE (ongoing games)
    AND tournament_id = @tournament_id::text
    AND COALESCE((game_request->>'game_mode')::int, 0) != 1 -- Exclude CORRESPONDENCE games
ORDER BY id;

-- name: ListActiveCorrespondenceGames :many
SELECT quickdata, uuid, started, tournament_data, game_request, player_on_turn, timers, type, updated_at, league_id, season_id, league_division_id
FROM games
WHERE game_end_reason = 0 -- NONE (ongoing games)
    AND (game_request->>'game_mode')::int = 1 -- Only CORRESPONDENCE games
ORDER BY id;

-- name: ListActiveCorrespondenceGamesForUser :many
SELECT quickdata, uuid, started, tournament_data, game_request, player_on_turn, updated_at, league_id, season_id, league_division_id, history
FROM games
WHERE game_end_reason = 0 -- NONE (ongoing games)
    AND (game_request->>'game_mode')::int = 1 -- Only CORRESPONDENCE games
    AND (
        player0_id = (SELECT id FROM users WHERE uuid = @user_uuid::text)
        OR player1_id = (SELECT id FROM users WHERE uuid = @user_uuid::text)
    )
ORDER BY id;

-- name: ListActiveCorrespondenceGamesForUserAndLeague :many
SELECT quickdata, uuid, started, tournament_data, game_request, player_on_turn, updated_at, league_id, season_id, league_division_id, history
FROM games
WHERE league_id = @league_id::uuid
    AND game_end_reason = 0 -- NONE (ongoing games)
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
SELECT history, history_s3_key, game_end_reason, last_known_racks,
       quickdata, game_request, uuid
FROM games
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
    original_request_id,
    league_season_id,
    updated_at
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
        @original_request_id,
        @league_season_id,
        @updated_at
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
        @original_request_id,
        @league_season_id,
        @updated_at
    )
ON CONFLICT (game_uuid, player_id) DO NOTHING;

-- name: SetGameHistoryS3Key :exec
UPDATE games SET history_s3_key = @history_s3_key WHERE uuid = @uuid;

-- name: ListPendingArchival :many
-- Returns finished games that have game_turns rows but no S3 archive yet.
-- Excludes CANCELLED (7) — those are deleted by the maintenance task after 2 days.
-- ABORTED (5) is included: those games had real play and should be archived for audit.
SELECT DISTINCT g.uuid FROM games g
JOIN game_turns gt ON gt.game_uuid = g.uuid
WHERE g.history_s3_key IS NULL
  AND g.game_end_reason NOT IN (0, 7)
ORDER BY g.uuid;

-- name: ListByteaBackfillBatch :many
-- Finished games needing direct bytea→S3 archival (no game_turns assembly).
-- Includes ABORTED (5) for audit; excludes CANCELLED (7) which the maintenance
-- task deletes after 2 days. Keyset-paginated by uuid via
-- idx_games_history_s3_key_pending.
SELECT uuid, history, created_at FROM games
WHERE history_s3_key IS NULL
  AND game_end_reason NOT IN (0, 7)
  AND history IS NOT NULL
  AND created_at IS NOT NULL
  AND uuid > @after_uuid
ORDER BY uuid
LIMIT @lim;

-- name: ListActiveCorrespondenceForArchivalAudit :many
-- Active started correspondence games with their game_turns count, keyset-paginated.
-- Used by cmd/find-pre-archival-games to identify games where game_turns rows are
-- fewer than the bytea history's event count (they predate or crossed the dual-write
-- cutover and will hit the bytea-fallback path in ArchiveAndCleanup when they end).
SELECT g.uuid, g.created_at, g.updated_at, g.history,
       COALESCE(gt.cnt, 0)::int AS turns_count
FROM games g
LEFT JOIN (
    SELECT game_uuid, COUNT(*)::int AS cnt
    FROM game_turns
    GROUP BY game_uuid
) gt ON gt.game_uuid = g.uuid
WHERE g.game_end_reason = 0
  AND (g.game_request->>'game_mode')::int = 1
  AND g.started = true
  AND g.uuid > @after_uuid
ORDER BY g.uuid
LIMIT @lim;

