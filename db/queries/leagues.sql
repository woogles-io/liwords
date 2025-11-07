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

-- name: SetCurrentSeason :exec
UPDATE leagues
SET current_season_id = $2, updated_at = NOW()
WHERE uuid = $1;

-- Season operations

-- name: CreateSeason :one
INSERT INTO league_seasons (uuid, league_id, season_number, start_date, end_date, status)
VALUES ($1, $2, $3, $4, $5, $6)
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

-- name: GetSeasonByLeagueAndNumber :one
SELECT * FROM league_seasons
WHERE league_id = $1 AND season_number = $2;

-- name: UpdateSeasonStatus :exec
UPDATE league_seasons
SET status = $2, updated_at = NOW()
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
SELECT lr.*, u.uuid as user_uuid, u.username as username FROM league_registrations lr
JOIN users u ON lr.user_id = u.id
WHERE lr.season_id = $1
ORDER BY lr.registration_date;

-- name: GetDivisionRegistrations :many
SELECT lr.*, u.uuid as user_uuid FROM league_registrations lr
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
INSERT INTO league_standings (division_id, user_id, rank, wins, losses, draws, spread, games_played, games_remaining, result, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
ON CONFLICT (division_id, user_id)
DO UPDATE SET
    rank = EXCLUDED.rank,
    wins = EXCLUDED.wins,
    losses = EXCLUDED.losses,
    draws = EXCLUDED.draws,
    spread = EXCLUDED.spread,
    games_played = EXCLUDED.games_played,
    games_remaining = EXCLUDED.games_remaining,
    result = EXCLUDED.result,
    updated_at = NOW();

-- name: GetStandings :many
SELECT ls.*, u.uuid as user_uuid, u.username FROM league_standings ls
JOIN users u ON ls.user_id = u.id
WHERE ls.division_id = $1
ORDER BY ls.rank ASC;

-- name: GetPlayerStanding :one
SELECT * FROM league_standings
WHERE division_id = $1 AND user_id = $2;

-- name: DeleteDivisionStandings :exec
DELETE FROM league_standings
WHERE division_id = $1;

-- name: IncrementStandingsAtomic :exec
-- Atomically increment standings for a player after a game completes
-- This avoids race conditions by using database-level arithmetic
INSERT INTO league_standings (division_id, user_id, rank, wins, losses, draws, spread, games_played, games_remaining, result, updated_at)
VALUES ($1, $2, 0, $3, $4, $5, $6, 1, $7, 0, NOW())
ON CONFLICT (division_id, user_id)
DO UPDATE SET
    wins = league_standings.wins + EXCLUDED.wins,
    losses = league_standings.losses + EXCLUDED.losses,
    draws = league_standings.draws + EXCLUDED.draws,
    spread = league_standings.spread + EXCLUDED.spread,
    games_played = league_standings.games_played + 1,
    games_remaining = GREATEST(league_standings.games_remaining - 1, 0),
    updated_at = NOW();

-- name: RecalculateRanks :exec
-- Recalculate ranks for all players in a division
-- Ranks are based on: wins (DESC), then spread (DESC)
WITH ranked AS (
    SELECT
        ls.division_id,
        ls.user_id,
        ROW_NUMBER() OVER (
            PARTITION BY ls.division_id
            ORDER BY ls.wins DESC, ls.spread DESC
        ) as new_rank
    FROM league_standings ls
    WHERE ls.division_id = $1
)
UPDATE league_standings
SET rank = ranked.new_rank,
    updated_at = NOW()
FROM ranked
WHERE league_standings.division_id = ranked.division_id
  AND league_standings.user_id = ranked.user_id;

-- Game queries for league games

-- name: GetLeagueGames :many
SELECT * FROM games
WHERE league_division_id = $1
ORDER BY created_at;

-- name: GetLeagueGamesByStatus :many
SELECT * FROM games
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

-- name: ForceFinishGame :exec
UPDATE games
SET game_end_reason = 8,  -- FORCE_FORFEIT
    winner_idx = $2,
    loser_idx = $3,
    updated_at = NOW()
WHERE uuid = $1;

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
-- Get all games for a specific player in a season with scores from game_players table
SELECT
    g.uuid as game_uuid,
    g.created_at,
    gp_player.player_id,
    gp_player.score as player_score,
    gp_player.opponent_score,
    gp_player.won,
    gp_player.game_end_reason,
    u_opponent.uuid as opponent_uuid,
    u_opponent.username as opponent_username
FROM games g
INNER JOIN game_players gp_player ON g.uuid = gp_player.game_uuid
INNER JOIN users u_player ON gp_player.player_id = u_player.id
INNER JOIN game_players gp_opponent ON g.uuid = gp_opponent.game_uuid AND gp_opponent.player_index = (1 - gp_player.player_index)
INNER JOIN users u_opponent ON gp_opponent.player_id = u_opponent.id
WHERE g.season_id = @season_id
  AND u_player.uuid = @user_uuid
ORDER BY g.created_at DESC;
