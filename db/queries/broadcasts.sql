-- name: CreateBroadcast :one
INSERT INTO broadcasts (uuid, slug, name, description, broadcast_url, broadcast_url_format,
    poll_interval_seconds, poll_start_time, poll_end_time,
    lexicon, board_layout, letter_distribution, challenge_rule, creator_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING id, uuid, created_at;

-- name: GetBroadcastBySlug :one
SELECT b.id, b.uuid, b.slug, b.name, b.description, b.broadcast_url, b.broadcast_url_format,
       b.poll_interval_seconds, b.poll_start_time, b.poll_end_time,
       b.lexicon, b.board_layout, b.letter_distribution, b.challenge_rule,
       b.last_polled_at, b.creator_id, b.active, b.created_at, b.updated_at,
       u.username as creator_username
FROM broadcasts b
JOIN users u ON b.creator_id = u.id
WHERE b.slug = $1;

-- name: GetBroadcastByUUID :one
SELECT b.id, b.uuid, b.slug, b.name, b.description, b.broadcast_url, b.broadcast_url_format,
       b.poll_interval_seconds, b.poll_start_time, b.poll_end_time,
       b.lexicon, b.board_layout, b.letter_distribution, b.challenge_rule,
       b.last_polled_at, b.creator_id, b.active, b.created_at, b.updated_at,
       u.username as creator_username
FROM broadcasts b
JOIN users u ON b.creator_id = u.id
WHERE b.uuid = $1;

-- name: UpdateBroadcast :exec
UPDATE broadcasts
SET name = $2,
    description = $3,
    broadcast_url = $4,
    broadcast_url_format = $5,
    poll_interval_seconds = $6,
    poll_start_time = $7,
    poll_end_time = $8,
    lexicon = $9,
    board_layout = $10,
    letter_distribution = $11,
    challenge_rule = $12,
    active = $13,
    updated_at = NOW()
WHERE slug = $1;

-- name: UpdateBroadcastLastPolled :exec
UPDATE broadcasts
SET last_polled_at = NOW()
WHERE id = $1;

-- name: GetActiveBroadcasts :many
SELECT b.id, b.uuid, b.slug, b.name, b.description, b.broadcast_url, b.broadcast_url_format,
       b.poll_interval_seconds, b.poll_start_time, b.poll_end_time,
       b.lexicon, b.board_layout, b.letter_distribution, b.challenge_rule,
       b.last_polled_at, b.creator_id, b.active, b.created_at, b.updated_at,
       u.username as creator_username
FROM broadcasts b
JOIN users u ON b.creator_id = u.id
WHERE b.active = true
ORDER BY b.created_at DESC;

-- name: GetBroadcastsForPolling :many
SELECT id, uuid, slug, broadcast_url, broadcast_url_format, poll_interval_seconds,
       poll_start_time, poll_end_time, last_polled_at
FROM broadcasts
WHERE active = true;

-- name: AddBroadcastDirector :exec
INSERT INTO broadcast_directors (broadcast_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveBroadcastDirector :exec
DELETE FROM broadcast_directors
WHERE broadcast_id = $1 AND user_id = $2;

-- name: GetBroadcastDirectors :many
SELECT u.uuid, u.username
FROM broadcast_directors bd
JOIN users u ON bd.user_id = u.id
WHERE bd.broadcast_id = $1;

-- name: IsBroadcastDirector :one
SELECT EXISTS(
    SELECT 1 FROM broadcast_directors
    WHERE broadcast_id = $1 AND user_id = $2
) as is_director;

-- name: AddBroadcastAnnotator :exec
INSERT INTO broadcast_annotators (broadcast_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveBroadcastAnnotator :exec
DELETE FROM broadcast_annotators
WHERE broadcast_id = $1 AND user_id = $2;

-- name: GetBroadcastAnnotators :many
SELECT u.uuid, u.username
FROM broadcast_annotators ba
JOIN users u ON ba.user_id = u.id
WHERE ba.broadcast_id = $1;

-- name: IsBroadcastAnnotator :one
SELECT EXISTS(
    SELECT 1 FROM broadcast_annotators
    WHERE broadcast_id = $1 AND user_id = $2
) as is_annotator;

-- name: CreateBroadcastGame :one
INSERT INTO broadcast_games (broadcast_id, game_uuid, division, round, table_number,
    player1_name, player2_name, annotator_user_id, claimed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
RETURNING id;

-- name: GetBroadcastGamesForRound :many
SELECT bg.*, u.username as annotator_username
FROM broadcast_games bg
LEFT JOIN users u ON bg.annotator_user_id = u.id
WHERE bg.broadcast_id = $1 AND bg.division = $2 AND bg.round = $3
ORDER BY bg.table_number;

-- name: GetBroadcastGameByTableRound :one
SELECT bg.*, u.username as annotator_username
FROM broadcast_games bg
LEFT JOIN users u ON bg.annotator_user_id = u.id
WHERE bg.broadcast_id = $1 AND bg.division = $2 AND bg.round = $3 AND bg.table_number = $4;

-- name: UnclaimBroadcastGame :one
DELETE FROM broadcast_games
WHERE broadcast_id = $1 AND division = $2 AND round = $3 AND table_number = $4
RETURNING game_uuid;

-- name: GetBroadcastGameAnnotatorInfo :one
SELECT annotator_user_id, game_uuid
FROM broadcast_games
WHERE broadcast_id = $1 AND division = $2 AND round = $3 AND table_number = $4;

-- name: GetMyClaimedGames :many
SELECT bg.*, u.username as annotator_username
FROM broadcast_games bg
LEFT JOIN users u ON bg.annotator_user_id = u.id
WHERE bg.broadcast_id = $1 AND bg.annotator_user_id = $2
ORDER BY bg.claimed_at DESC
LIMIT $3;

-- name: GetBroadcastGameByUUID :one
SELECT bg.*, b.slug as broadcast_slug, b.name as broadcast_name
FROM broadcast_games bg
JOIN broadcasts b ON bg.broadcast_id = b.id
WHERE bg.game_uuid = $1;
