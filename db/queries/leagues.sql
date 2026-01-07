-- League operations

-- name: CreateLeague :one
INSERT INTO leagues (uuid, name, description, slug, settings, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetLeagueByUUID :one
SELECT * FROM leagues WHERE uuid = $1;

-- name: GetLeagueBySlug :one
SELECT * FROM leagues WHERE LOWER(slug) = LOWER($1);

-- name: GetAllLeagues :many
SELECT * FROM leagues
WHERE (@active_only::boolean = false OR is_active = true)
ORDER BY name;

-- name: UpdateLeagueSettings :exec
UPDATE leagues
SET settings = $2, updated_at = NOW()
WHERE uuid = $1;

-- name: UpdateLeagueMetadata :exec
UPDATE leagues
SET name = $2, description = $3, updated_at = NOW()
WHERE uuid = $1;

-- name: SetCurrentSeason :exec
UPDATE leagues
SET current_season_id = $2, updated_at = NOW()
WHERE uuid = $1;

-- Season operations

-- name: CreateSeason :one
INSERT INTO league_seasons (uuid, league_id, season_number, start_date, end_date, status, promotion_formula)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetSeason :one
SELECT * FROM league_seasons WHERE uuid = $1;

-- name: GetCurrentSeason :one
SELECT ls.* FROM league_seasons ls
JOIN leagues l ON l.current_season_id = ls.uuid
WHERE l.uuid = $1;

-- name: GetPastSeasons :many
SELECT * FROM league_seasons
WHERE league_id = $1 AND status = 2  -- SeasonStatus.SEASON_COMPLETED
ORDER BY season_number DESC;

-- name: GetSeasonsByLeague :many
SELECT * FROM league_seasons
WHERE league_id = $1
ORDER BY season_number DESC;

-- name: GetRecentSeasons :many
SELECT * FROM league_seasons
WHERE league_id = $1
ORDER BY season_number DESC
LIMIT $2;

-- name: GetSeasonChampion :one
-- Get the champion (result = RESULT_CHAMPION in division 1) for a completed season
SELECT u.uuid as user_uuid, u.username
FROM league_standings ls
JOIN league_divisions ld ON ls.division_id = ld.uuid
JOIN users u ON ls.user_id = u.id
WHERE ld.season_id = $1
  AND ld.division_number = 1
  AND ls.result = 4;  -- RESULT_CHAMPION only

-- name: GetSeasonByLeagueAndNumber :one
SELECT * FROM league_seasons
WHERE league_id = $1 AND season_number = $2;

-- name: UpdateSeasonStatus :exec
UPDATE league_seasons
SET status = $2, updated_at = NOW()
WHERE uuid = $1;

-- name: UpdateSeasonDates :exec
UPDATE league_seasons
SET start_date = $2, end_date = $3, updated_at = NOW()
WHERE uuid = $1;

-- name: UpdateSeasonPromotionFormula :exec
UPDATE league_seasons
SET promotion_formula = $2, updated_at = NOW()
WHERE uuid = $1;

-- Task tracking queries for hourly runner idempotency

-- name: MarkSeasonClosed :exec
UPDATE league_seasons
SET closed_at = NOW(), updated_at = NOW()
WHERE uuid = $1;

-- name: MarkDivisionsPrepared :exec
UPDATE league_seasons
SET divisions_prepared_at = NOW(), updated_at = NOW()
WHERE uuid = $1;

-- name: MarkSeasonStarted :exec
UPDATE league_seasons
SET started_at = NOW(), updated_at = NOW()
WHERE uuid = $1;

-- name: MarkRegistrationOpened :exec
-- Marks when registration was opened for this season
UPDATE league_seasons
SET registration_opened_at = NOW(), updated_at = NOW()
WHERE uuid = $1;

-- name: MarkStartingSoonNotificationSent :exec
UPDATE league_seasons
SET starting_soon_notification_sent_at = NOW(), updated_at = NOW()
WHERE uuid = $1;

-- name: MarkSeasonComplete :exec
UPDATE league_seasons
SET status = 2, actual_end_date = NOW(), updated_at = NOW()  -- SeasonStatus.SEASON_COMPLETED
WHERE uuid = $1;

-- Division operations

-- name: CreateDivision :one
INSERT INTO league_divisions (uuid, season_id, division_number, division_name, is_complete)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDivision :one
SELECT * FROM league_divisions WHERE uuid = $1;

-- name: GetDivisionsBySeason :many
SELECT * FROM league_divisions
WHERE season_id = $1
ORDER BY division_number ASC;

-- name: MarkDivisionComplete :exec
UPDATE league_divisions
SET is_complete = true, updated_at = NOW()
WHERE uuid = $1;

-- name: DeleteDivision :exec
DELETE FROM league_divisions
WHERE uuid = $1;

-- name: UpdateDivisionNumber :exec
UPDATE league_divisions
SET division_number = $2, division_name = $3, updated_at = NOW()
WHERE uuid = $1;

-- Registration operations

-- name: RegisterPlayer :one
INSERT INTO league_registrations (user_id, season_id, division_id, registration_date, firsts_count, status, placement_status, previous_division_rank, seasons_away)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, season_id)
DO UPDATE SET
    division_id = EXCLUDED.division_id,
    firsts_count = EXCLUDED.firsts_count,
    status = EXCLUDED.status,
    placement_status = EXCLUDED.placement_status,
    previous_division_rank = EXCLUDED.previous_division_rank,
    seasons_away = EXCLUDED.seasons_away
RETURNING *;

-- name: UnregisterPlayer :exec
DELETE FROM league_registrations
WHERE season_id = $1 AND user_id = $2;

-- name: GetPlayerRegistration :one
SELECT * FROM league_registrations
WHERE season_id = $1 AND user_id = $2;

-- name: GetSeasonRegistrations :many
SELECT
    lr.*,
    u.uuid as user_uuid,
    u.username as username,
    ld.division_number as division_number
FROM league_registrations lr
JOIN users u ON lr.user_id = u.id
LEFT JOIN league_divisions ld ON lr.division_id = ld.uuid
WHERE lr.season_id = $1
ORDER BY lr.registration_date;

-- name: GetDivisionRegistrations :many
SELECT lr.*, u.uuid as user_uuid, u.username FROM league_registrations lr
JOIN users u ON lr.user_id = u.id
WHERE lr.division_id = $1
ORDER BY lr.registration_date;

-- name: UpdatePlayerDivision :exec
UPDATE league_registrations
SET division_id = $1, firsts_count = $2
WHERE user_id = $3 AND season_id = $4;

-- name: UpdateRegistrationDivision :exec
UPDATE league_registrations
SET division_id = $2, firsts_count = $3, updated_at = NOW()
WHERE season_id = $1 AND user_id = $4;

-- name: UpdatePlacementStatus :exec
UPDATE league_registrations
SET placement_status = $2, previous_division_rank = $3, updated_at = NOW()
WHERE user_id = $1 AND season_id = $4;

-- name: UpdatePreviousDivisionRank :exec
UPDATE league_registrations
SET previous_division_rank = $2, updated_at = NOW()
WHERE user_id = $1 AND season_id = $3;

-- name: UpdatePlacementStatusWithSeasonsAway :exec
UPDATE league_registrations
SET placement_status = $2, previous_division_rank = $3, seasons_away = $4, updated_at = NOW()
WHERE user_id = $1 AND season_id = $5;

-- name: GetRegistrationsByDivision :many
SELECT * FROM league_registrations
WHERE division_id = $1
ORDER BY placement_status, previous_division_rank;

-- name: GetPlayerSeasonHistory :many
SELECT lr.*, ls.season_number, ls.league_id
FROM league_registrations lr
JOIN league_seasons ls ON lr.season_id = ls.uuid
WHERE lr.user_id = $1
  AND (@league_id::uuid IS NULL OR ls.league_id = @league_id)
ORDER BY ls.season_number DESC;

-- Standings operations

-- name: UpsertStanding :exec
-- Note: rank column is not upserted - it's calculated on-demand from wins/losses/draws/spread
INSERT INTO league_standings (division_id, user_id, wins, losses, draws, spread, games_played, games_remaining, result,
    total_score, total_opponent_score, total_bingos, total_opponent_bingos, total_turns, high_turn, high_game, timeouts, blanks_played,
    total_tiles_played, total_opponent_tiles_played, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW())
ON CONFLICT (division_id, user_id)
DO UPDATE SET
    wins = EXCLUDED.wins,
    losses = EXCLUDED.losses,
    draws = EXCLUDED.draws,
    spread = EXCLUDED.spread,
    games_played = EXCLUDED.games_played,
    games_remaining = EXCLUDED.games_remaining,
    result = EXCLUDED.result,
    total_score = EXCLUDED.total_score,
    total_opponent_score = EXCLUDED.total_opponent_score,
    total_bingos = EXCLUDED.total_bingos,
    total_opponent_bingos = EXCLUDED.total_opponent_bingos,
    total_turns = EXCLUDED.total_turns,
    high_turn = EXCLUDED.high_turn,
    high_game = EXCLUDED.high_game,
    timeouts = EXCLUDED.timeouts,
    blanks_played = EXCLUDED.blanks_played,
    total_tiles_played = EXCLUDED.total_tiles_played,
    total_opponent_tiles_played = EXCLUDED.total_opponent_tiles_played,
    updated_at = NOW();

-- name: UpdateStandingResult :exec
-- Updates only the result (outcome) field for a standing without touching other stats
UPDATE league_standings
SET result = $3, updated_at = NOW()
WHERE division_id = $1 AND user_id = $2;

-- name: GetStandings :many
-- Note: rank column is deprecated and not queried. Sorting is done in Go code.
SELECT ls.id, ls.division_id, ls.user_id, ls.wins, ls.losses, ls.draws,
       ls.spread, ls.games_played, ls.games_remaining, ls.result, ls.updated_at,
       ls.total_score, ls.total_opponent_score, ls.total_bingos, ls.total_opponent_bingos,
       ls.total_turns, ls.high_turn, ls.high_game, ls.timeouts, ls.blanks_played,
       ls.total_tiles_played, ls.total_opponent_tiles_played,
       u.uuid as user_uuid, u.username
FROM league_standings ls
JOIN users u ON ls.user_id = u.id
WHERE ls.division_id = $1;

-- name: GetPlayerStanding :one
SELECT * FROM league_standings
WHERE division_id = $1 AND user_id = $2;

-- name: DeleteDivisionStandings :exec
DELETE FROM league_standings
WHERE division_id = $1;

-- name: IncrementStandingsAtomic :exec
-- Atomically increment standings for a player after a game completes
-- This avoids race conditions by using database-level arithmetic
INSERT INTO league_standings (division_id, user_id, wins, losses, draws, spread, games_played, games_remaining, result,
    total_score, total_opponent_score, total_bingos, total_opponent_bingos, total_turns, high_turn, high_game, timeouts, blanks_played,
    total_tiles_played, total_opponent_tiles_played, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, 1, $7, 0, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, NOW())
ON CONFLICT (division_id, user_id)
DO UPDATE SET
    wins = league_standings.wins + EXCLUDED.wins,
    losses = league_standings.losses + EXCLUDED.losses,
    draws = league_standings.draws + EXCLUDED.draws,
    spread = league_standings.spread + EXCLUDED.spread,
    games_played = league_standings.games_played + 1,
    games_remaining = GREATEST(league_standings.games_remaining - 1, 0),
    total_score = league_standings.total_score + EXCLUDED.total_score,
    total_opponent_score = league_standings.total_opponent_score + EXCLUDED.total_opponent_score,
    total_bingos = league_standings.total_bingos + EXCLUDED.total_bingos,
    total_opponent_bingos = league_standings.total_opponent_bingos + EXCLUDED.total_opponent_bingos,
    total_turns = league_standings.total_turns + EXCLUDED.total_turns,
    high_turn = GREATEST(league_standings.high_turn, EXCLUDED.high_turn),
    high_game = GREATEST(league_standings.high_game, EXCLUDED.high_game),
    timeouts = league_standings.timeouts + EXCLUDED.timeouts,
    blanks_played = league_standings.blanks_played + EXCLUDED.blanks_played,
    total_tiles_played = league_standings.total_tiles_played + EXCLUDED.total_tiles_played,
    total_opponent_tiles_played = league_standings.total_opponent_tiles_played + EXCLUDED.total_opponent_tiles_played,
    updated_at = NOW();

-- Game queries for league games

-- name: GetLeagueGames :many
SELECT
    id, created_at, updated_at, deleted_at, uuid,
    player0_id, player1_id, timers, started, game_end_reason,
    winner_idx, loser_idx, history, stats, quickdata,
    tournament_data, tournament_id, ready_flag, meta_events, type,
    game_request, player_on_turn, league_id, season_id, league_division_id
FROM games
WHERE league_division_id = $1
ORDER BY created_at;

-- name: GetLeagueGamesByStatus :many
SELECT
    id, created_at, updated_at, deleted_at, uuid,
    player0_id, player1_id, timers, started, game_end_reason,
    winner_idx, loser_idx, history, stats, quickdata,
    tournament_data, tournament_id, ready_flag, meta_events, type,
    game_request, player_on_turn, league_id, season_id, league_division_id
FROM games
WHERE league_division_id = $1
  AND (@include_finished::boolean = true OR game_end_reason = 0)
ORDER BY created_at;

-- name: CountDivisionGamesComplete :one
SELECT COUNT(*) FROM games
WHERE league_division_id = $1 AND game_end_reason != 0;

-- name: CountDivisionGamesTotal :one
SELECT COUNT(*) FROM games
WHERE league_division_id = $1;

-- name: GetUnfinishedLeagueGames :many
SELECT
    uuid as game_id,
    player0_id,
    player1_id
FROM games
WHERE season_id = $1
  AND game_end_reason = 0;

-- name: GetSeasonZeroMoveGames :many
-- Get all in-progress games in a season that have zero moves
-- Uses timers field: if lu (last update) == ts (time started), no moves have been made
-- This helps league managers identify players who haven't started their games
SELECT
    g.uuid as game_uuid,
    g.created_at,
    g.player0_id,
    g.player1_id,
    u_player0.uuid as player0_uuid,
    u_player0.username as player0_username,
    u_player1.uuid as player1_uuid,
    u_player1.username as player1_username,
    g.league_division_id as division_id
FROM games g
INNER JOIN users u_player0 ON g.player0_id = u_player0.id
INNER JOIN users u_player1 ON g.player1_id = u_player1.id
WHERE g.season_id = $1
  AND g.game_end_reason = 0
  AND (g.timers->>'lu')::bigint = (g.timers->>'ts')::bigint
ORDER BY g.created_at ASC;

-- name: GetSeasonPlayersWithUnstartedGames :many
-- Get players who are on turn but haven't made their first move yet
-- Groups by player to show who needs reminders
SELECT
    u.uuid as user_uuid,
    u.username,
    COUNT(*) as unstarted_game_count
FROM games g
INNER JOIN users u ON (
    (g.player_on_turn = 0 AND u.id = g.player0_id) OR
    (g.player_on_turn = 1 AND u.id = g.player1_id)
)
WHERE g.season_id = $1
  AND g.game_end_reason = 0
  AND (g.timers->>'lu')::bigint = (g.timers->>'ts')::bigint
GROUP BY u.uuid, u.username
ORDER BY unstarted_game_count DESC, u.username;

-- name: ForceFinishGame :exec
WITH game_update AS (
    UPDATE games
    SET game_end_reason = 8,  -- FORCE_FORFEIT
        winner_idx = $2,
        loser_idx = $3,
        updated_at = NOW()
    WHERE uuid = $1
    RETURNING uuid, winner_idx, loser_idx
)
UPDATE game_players gp
SET game_end_reason = 8,  -- FORCE_FORFEIT
    won = CASE
        WHEN gu.winner_idx IS NULL THEN NULL  -- Tie: both players get NULL
        WHEN gp.player_index = gu.winner_idx THEN true
        WHEN gp.player_index = gu.loser_idx THEN false
        ELSE NULL
    END
FROM game_update gu
WHERE gp.game_uuid = gu.uuid;

-- name: GetForceFinishedGamesMissingPlayers :many
-- Find force-finished or adjudicated games in a season that are missing game_players rows
-- This is used by the repair tool to backfill missing data
SELECT
    g.uuid as game_id,
    g.player0_id,
    g.player1_id,
    g.game_end_reason
FROM games g
LEFT JOIN game_players gp ON g.uuid = gp.game_uuid
WHERE g.season_id = $1
  AND g.game_end_reason IN (8, 9)  -- FORCE_FORFEIT or ADJUDICATED
  AND gp.game_uuid IS NULL;  -- No game_players rows exist

-- name: GetDivisionGameResults :many
SELECT
    g.uuid,
    g.player0_id,
    g.player1_id,
    gp0.score as player0_score,
    gp1.score as player1_score,
    gp0.won as player0_won,
    gp1.won as player1_won,
    gp0.game_end_reason
FROM games g
INNER JOIN game_players gp0 ON g.uuid = gp0.game_uuid AND gp0.player_index = 0
INNER JOIN game_players gp1 ON g.uuid = gp1.game_uuid AND gp1.player_index = 1
WHERE g.league_division_id = $1
  AND gp0.game_end_reason != 0  -- Only finished games
  AND gp0.game_end_reason != 5  -- Exclude ABORTED
  AND gp0.game_end_reason != 7; -- Exclude CANCELLED

-- name: GetDivisionGamesWithStats :many
-- Get all finished games for a division including the stats JSON blob
-- Used for recalculating extended standings stats from historical games
SELECT
    g.uuid,
    g.player0_id,
    g.player1_id,
    gp0.score as player0_score,
    gp1.score as player1_score,
    gp0.won as player0_won,
    gp1.won as player1_won,
    gp0.game_end_reason,
    g.stats
FROM games g
INNER JOIN game_players gp0 ON g.uuid = gp0.game_uuid AND gp0.player_index = 0
INNER JOIN game_players gp1 ON g.uuid = gp1.game_uuid AND gp1.player_index = 1
WHERE g.league_division_id = $1
  AND gp0.game_end_reason != 0  -- Only finished games
  AND gp0.game_end_reason != 5  -- Exclude ABORTED
  AND gp0.game_end_reason != 7; -- Exclude CANCELLED

-- name: GetGameLeagueInfo :one
SELECT
    g.league_division_id,
    g.season_id,
    g.league_id,
    g.player0_id,
    g.player1_id,
    gp0.score as player0_score,
    gp1.score as player1_score,
    gp0.won as player0_won,
    gp1.won as player1_won,
    gp0.game_end_reason
FROM games g
LEFT JOIN game_players gp0 ON g.uuid = gp0.game_uuid AND gp0.player_index = 0
LEFT JOIN game_players gp1 ON g.uuid = gp1.game_uuid AND gp1.player_index = 1
WHERE g.uuid = $1;

-- name: GetPlayerSeasonGames :many
-- Get finished games for a specific player in a season with scores from game_players table
-- Optimized to use idx_game_players_player_league_season composite index
SELECT
    gp_player.game_uuid,
    g.created_at,
    g.updated_at,
    gp_player.player_id,
    gp_player.score as player_score,
    gp_player.opponent_score,
    gp_player.won,
    gp_player.game_end_reason,
    u_opponent.uuid as opponent_uuid,
    u_opponent.username as opponent_username
FROM game_players gp_player
INNER JOIN games g ON gp_player.game_uuid = g.uuid
INNER JOIN users u_opponent ON gp_player.opponent_id = u_opponent.id
WHERE gp_player.player_id = (SELECT id FROM users WHERE users.uuid = @user_uuid)
  AND gp_player.league_season_id = @season_id
ORDER BY gp_player.created_at DESC;

-- name: GetPlayerSeasonInProgressGames :many
-- Get in-progress games for a specific player in a season (fast query on indexed fields)
SELECT
    g.uuid as game_uuid,
    g.created_at,
    g.updated_at,
    g.player0_id,
    g.player1_id,
    u_player0.uuid as player0_uuid,
    u_player0.username as player0_username,
    u_player1.uuid as player1_uuid,
    u_player1.username as player1_username,
    g.history
FROM games g
INNER JOIN users u_player0 ON g.player0_id = u_player0.id
INNER JOIN users u_player1 ON g.player1_id = u_player1.id
WHERE g.season_id = @season_id
  AND g.game_end_reason = 0
  AND (u_player0.uuid = @user_uuid OR u_player1.uuid = @user_uuid)
ORDER BY g.created_at DESC;

-- name: GetPlayerSeasonOpponents :many
-- Get distinct opponents for a player in a season (from games table)
SELECT DISTINCT
    (CASE
        WHEN u_player0.uuid = @user_uuid THEN u_player1.username
        ELSE u_player0.username
    END)::text as opponent_username
FROM games g
INNER JOIN users u_player0 ON g.player0_id = u_player0.id
INNER JOIN users u_player1 ON g.player1_id = u_player1.id
WHERE g.season_id = @season_id
  AND (u_player0.uuid = @user_uuid OR u_player1.uuid = @user_uuid)
ORDER BY opponent_username;

-- name: GetDivisionTimeBankStatus :many
-- Get time bank status for all players with active games in a division
-- Returns users who have at least one game where it's their turn and their
-- effective time bank (accounting for deficit from main time) is below threshold
-- Effective time bank = stored_tb + MIN(effective_tr, 0)
-- This matches the adjudicator's calculation in timeRanOut()
SELECT
    u.id as user_id,
    u.uuid as user_uuid,
    u.username,
    COUNT(*) as low_timebank_game_count
FROM games g
JOIN users u ON (
    (g.player_on_turn = 0 AND u.id = g.player0_id) OR
    (g.player_on_turn = 1 AND u.id = g.player1_id)
)
WHERE g.league_division_id = @division_id
  AND g.game_end_reason = 0
  AND g.timers->'tb' IS NOT NULL
  AND jsonb_array_length(g.timers->'tb') = 2
  AND (
    -- Effective time bank = stored_tb + MIN(effective_tr, 0)
    (g.timers->'tb'->(g.player_on_turn))::bigint +
    LEAST(
        (g.timers->'tr'->(g.player_on_turn))::bigint -
        (sqlc.arg(now_ms)::bigint - (g.timers->>'lu')::bigint),
        0
    )
  ) < sqlc.arg(threshold_ms)::bigint
GROUP BY u.id, u.uuid, u.username;

-- name: AddTimeBankSinglePlayer :execrows
-- Add time bank to only the specified player's side in their in-progress games
UPDATE games
SET timers = jsonb_set(
    timers,
    CASE WHEN player0_id = @player_id THEN '{tb,0}' ELSE '{tb,1}' END,
    to_jsonb((timers->'tb'->(CASE WHEN player0_id = @player_id THEN 0 ELSE 1 END))::bigint + @additional_ms::bigint)
),
updated_at = NOW()
WHERE season_id = @season_id
  AND game_end_reason = 0
  AND (player0_id = @player_id OR player1_id = @player_id)
  AND timers->'tb' IS NOT NULL
  AND jsonb_array_length(timers->'tb') = 2;

-- name: AddTimeBankPlayerAndOpponent :execrows
-- Add time bank to both sides of a player's in-progress games
UPDATE games
SET timers = jsonb_set(
    jsonb_set(timers, '{tb,0}', to_jsonb((timers->'tb'->0)::bigint + @additional_ms::bigint)),
    '{tb,1}',
    to_jsonb((timers->'tb'->1)::bigint + @additional_ms::bigint)
),
updated_at = NOW()
WHERE season_id = @season_id
  AND game_end_reason = 0
  AND (player0_id = @player_id OR player1_id = @player_id)
  AND timers->'tb' IS NOT NULL
  AND jsonb_array_length(timers->'tb') = 2;

-- name: AddTimeBankAllPlayers :execrows
-- Add time bank to all in-progress games in a season
UPDATE games
SET timers = jsonb_set(
    jsonb_set(timers, '{tb,0}', to_jsonb((timers->'tb'->0)::bigint + @additional_ms::bigint)),
    '{tb,1}',
    to_jsonb((timers->'tb'->1)::bigint + @additional_ms::bigint)
),
updated_at = NOW()
WHERE season_id = @season_id
  AND game_end_reason = 0
  AND timers->'tb' IS NOT NULL
  AND jsonb_array_length(timers->'tb') = 2;
