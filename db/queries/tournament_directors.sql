-- name: GetTournamentDirectorByUUID :one
-- Returns the director's role for the given tournament UUID and user ID.
-- Returns pgx.ErrNoRows if the user is not a director of this tournament.
SELECT td.role
FROM tournaments t
JOIN tournament_directors td ON td.tournament_id = t.id AND td.user_id = $2
WHERE t.uuid = $1;

-- name: GetTournamentNumericID :one
SELECT id FROM tournaments WHERE uuid = $1;

-- name: AddTournamentDirector :exec
INSERT INTO tournament_directors (tournament_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (tournament_id, user_id) DO UPDATE SET role = EXCLUDED.role;

-- name: RemoveTournamentDirector :exec
DELETE FROM tournament_directors
WHERE tournament_id = $1 AND user_id = $2;

-- name: UpdateTournamentDirectorRole :exec
UPDATE tournament_directors
SET role = $3
WHERE tournament_id = $1 AND user_id = $2;

-- name: ListTournamentDirectors :many
SELECT u.id AS user_id, u.uuid AS user_uuid, u.username, td.role
FROM tournament_directors td
JOIN users u ON td.user_id = u.id
WHERE td.tournament_id = $1
ORDER BY td.created_at ASC;

-- name: ListTournamentsByDirector :many
SELECT t.uuid, t.name, t.slug, t.description, t.type, t.directors, t.extra_meta,
       t.is_started, t.is_finished, t.scheduled_start_time, t.scheduled_end_time,
       t.created_at, t.parent
FROM tournaments t
JOIN tournament_directors td ON td.tournament_id = t.id
WHERE td.user_id = $1
ORDER BY t.created_at DESC
LIMIT $2;
