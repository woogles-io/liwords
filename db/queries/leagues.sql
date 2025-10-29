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
WHERE league_id = $1 AND status = 'SEASON_COMPLETED'
ORDER BY season_number DESC;

-- name: GetSeasonsByLeague :many
SELECT * FROM league_seasons
WHERE league_id = $1
ORDER BY season_number DESC;

-- name: UpdateSeasonStatus :exec
UPDATE league_seasons
SET status = $2, updated_at = NOW()
WHERE uuid = $1;

-- name: MarkSeasonComplete :exec
UPDATE league_seasons
SET status = 'SEASON_COMPLETED', actual_end_date = NOW(), updated_at = NOW()
WHERE uuid = $1;

-- Division operations

-- name: CreateDivision :one
INSERT INTO league_divisions (uuid, season_id, division_number, division_name, player_count, is_complete)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetDivision :one
SELECT * FROM league_divisions WHERE uuid = $1;

-- name: GetDivisionsBySeason :many
SELECT * FROM league_divisions
WHERE season_id = $1
ORDER BY division_number ASC;

-- name: UpdateDivisionPlayerCount :exec
UPDATE league_divisions
SET player_count = $2, updated_at = NOW()
WHERE uuid = $1;

-- name: MarkDivisionComplete :exec
UPDATE league_divisions
SET is_complete = true, updated_at = NOW()
WHERE uuid = $1;

-- Registration operations

-- name: RegisterPlayer :one
INSERT INTO league_registrations (user_id, season_id, division_id, registration_date, starting_rating, firsts_count, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, season_id)
DO UPDATE SET
    division_id = EXCLUDED.division_id,
    starting_rating = EXCLUDED.starting_rating,
    firsts_count = EXCLUDED.firsts_count,
    status = EXCLUDED.status
RETURNING *;

-- name: UnregisterPlayer :exec
DELETE FROM league_registrations
WHERE season_id = $1 AND user_id = $2;

-- name: GetPlayerRegistration :one
SELECT * FROM league_registrations
WHERE season_id = $1 AND user_id = $2;

-- name: GetSeasonRegistrations :many
SELECT * FROM league_registrations
WHERE season_id = $1
ORDER BY registration_date;

-- name: GetDivisionRegistrations :many
SELECT * FROM league_registrations
WHERE division_id = $1
ORDER BY registration_date;

-- name: UpdatePlayerDivision :exec
UPDATE league_registrations
SET division_id = $1, firsts_count = $2
WHERE user_id = $3 AND season_id = $4;

-- name: UpdateRegistrationDivision :exec
UPDATE league_registrations
SET division_id = $2, firsts_count = $3, updated_at = NOW()
WHERE season_id = $1 AND user_id = $4;

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
SELECT * FROM league_standings
WHERE division_id = $1
ORDER BY rank ASC;

-- name: GetPlayerStanding :one
SELECT * FROM league_standings
WHERE division_id = $1 AND user_id = $2;

-- name: DeleteDivisionStandings :exec
DELETE FROM league_standings
WHERE division_id = $1;

-- Game queries for league games

-- name: GetLeagueGames :many
SELECT * FROM games
WHERE division_id = $1
ORDER BY created_at;

-- name: GetLeagueGamesByStatus :many
SELECT * FROM games
WHERE division_id = $1
  AND (@include_finished::boolean = true OR game_end_reason = 0)
ORDER BY created_at;

-- name: CountDivisionGamesComplete :one
SELECT COUNT(*) FROM games
WHERE division_id = $1 AND game_end_reason != 0;

-- name: CountDivisionGamesTotal :one
SELECT COUNT(*) FROM games
WHERE division_id = $1;
