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

-- name: GetAllBroadcasts :many
SELECT b.id, b.uuid, b.slug, b.name, b.description, b.broadcast_url, b.broadcast_url_format,
       b.poll_interval_seconds, b.poll_start_time, b.poll_end_time,
       b.lexicon, b.board_layout, b.letter_distribution, b.challenge_rule,
       b.last_polled_at, b.creator_id, b.active, b.created_at, b.updated_at,
       u.username as creator_username
FROM broadcasts b
JOIN users u ON b.creator_id = u.id
ORDER BY b.active DESC, b.created_at DESC;

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
    annotator_user_id, claimed_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
RETURNING id;

-- name: GetBroadcastGamesForRound :many
SELECT bg.id, bg.broadcast_id, bg.game_uuid, bg.division, bg.round, bg.table_number,
       bg.annotator_user_id, bg.claimed_at, bg.created_at,
       bg.stats, bg.stats_computed_at, bg.completed_at,
       u.username AS annotator_username,
       COALESCE(gd.document->'players'->0->>'realName', '') AS player1_name,
       COALESCE(gd.document->'players'->1->>'realName', '') AS player2_name
FROM broadcast_games bg
LEFT JOIN users u ON bg.annotator_user_id = u.id
LEFT JOIN game_documents gd ON gd.game_id = bg.game_uuid
WHERE bg.broadcast_id = $1 AND bg.division = $2 AND bg.round = $3
ORDER BY bg.table_number;

-- name: GetBroadcastGameByTableRound :one
SELECT bg.id, bg.broadcast_id, bg.game_uuid, bg.division, bg.round, bg.table_number,
       bg.annotator_user_id, bg.claimed_at, bg.created_at,
       bg.stats, bg.stats_computed_at, bg.completed_at,
       u.username as annotator_username
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
SELECT bg.id, bg.broadcast_id, bg.game_uuid, bg.division, bg.round, bg.table_number,
       bg.annotator_user_id, bg.claimed_at, bg.created_at,
       bg.stats, bg.stats_computed_at, bg.completed_at,
       u.username AS annotator_username,
       COALESCE(gd.document->'players'->0->>'realName', '') AS player1_name,
       COALESCE(gd.document->'players'->1->>'realName', '') AS player2_name
FROM broadcast_games bg
LEFT JOIN users u ON bg.annotator_user_id = u.id
LEFT JOIN game_documents gd ON gd.game_id = bg.game_uuid
WHERE bg.broadcast_id = $1 AND bg.annotator_user_id = $2
ORDER BY bg.claimed_at DESC
LIMIT $3;

-- name: GetBroadcastGameByUUID :one
SELECT bg.id, bg.broadcast_id, bg.game_uuid, bg.division, bg.round, bg.table_number,
       bg.annotator_user_id, bg.claimed_at, bg.created_at,
       bg.stats, bg.stats_computed_at, bg.completed_at,
       b.uuid as broadcast_uuid, b.slug as broadcast_slug, b.name as broadcast_name,
       b.broadcast_url, b.broadcast_url_format
FROM broadcast_games bg
JOIN broadcasts b ON bg.broadcast_id = b.id
WHERE bg.game_uuid = $1;

-- name: GetBroadcastGameStatsBySlug :many
-- Returns all broadcast_games for a slug, ordered by completed_at desc (NULLs last = in-progress first).
-- Player names are read from the game_document JSON (source of truth).
SELECT bg.id, bg.broadcast_id, bg.game_uuid, bg.division, bg.round, bg.table_number,
       bg.annotator_user_id, bg.claimed_at, bg.created_at,
       bg.stats, bg.stats_computed_at, bg.completed_at,
       COALESCE(gd.document->'players'->0->>'realName', '') AS player1_name,
       COALESCE(gd.document->'players'->1->>'realName', '') AS player2_name
FROM broadcast_games bg
LEFT JOIN game_documents gd ON gd.game_id = bg.game_uuid
JOIN broadcasts b ON bg.broadcast_id = b.id
WHERE b.slug = $1
ORDER BY bg.completed_at DESC NULLS LAST;

-- name: UpdateBroadcastGameStats :exec
UPDATE broadcast_games
SET stats             = $2,
    stats_computed_at = NOW(),
    completed_at      = $3
WHERE game_uuid = $1;

-- name: CreateBroadcastSlot :exec
INSERT INTO broadcast_slots (broadcast_id, slot_name, division, round, table_number)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT DO NOTHING;

-- name: ListBroadcastSlots :many
-- stream_slot_count is how many user stream slots currently mirror this slot;
-- the UI warns before deleting a slot somebody is streaming from.
SELECT bs.slot_name, bs.division, bs.round, bs.table_number, bs.updated_at,
       (SELECT COUNT(*) FROM user_obs_slots us
         WHERE us.target_broadcast_id = bs.broadcast_id
           AND us.target_slot_name    = bs.slot_name) AS stream_slot_count
FROM broadcast_slots bs
WHERE bs.broadcast_id = $1
ORDER BY bs.slot_name;

-- name: UpdateBroadcastSlotTarget :exec
UPDATE broadcast_slots
SET division = $3, round = $4, table_number = $5, updated_at = NOW()
WHERE broadcast_id = $1 AND slot_name = $2;

-- name: DeleteBroadcastSlot :exec
DELETE FROM broadcast_slots
WHERE broadcast_id = $1 AND slot_name = $2;

-- name: GetSlotCurrentGame :one
-- Resolves (slug, slot_name) → current game with player names, for UI confirmation warnings.
-- Player names are read from the game_document JSON (source of truth); empty string when unclaimed.
SELECT
    bs.division,
    bs.round,
    bs.table_number,
    COALESCE(bg.game_uuid, '')                              AS game_uuid,
    COALESCE(gd.document->'players'->0->>'realName', '')   AS player1_name,
    COALESCE(gd.document->'players'->1->>'realName', '')   AS player2_name
FROM broadcast_slots bs
JOIN broadcasts b ON bs.broadcast_id = b.id
LEFT JOIN broadcast_games bg
    ON  bg.broadcast_id  = bs.broadcast_id
    AND bg.division      = bs.division
    AND bg.round         = bs.round
    AND bg.table_number  = bs.table_number
LEFT JOIN game_documents gd ON gd.game_id = bg.game_uuid
WHERE b.slug = $1 AND bs.slot_name = $2;

-- name: GetBroadcastSlotGame :one
-- Resolves (slug, slot_name) → current game at that (div, round, table), if any.
SELECT
    bs.division,
    bs.round,
    bs.table_number,
    bg.game_uuid,
    b.uuid  AS broadcast_uuid,
    b.id    AS broadcast_id
FROM broadcast_slots bs
JOIN broadcasts b ON bs.broadcast_id = b.id
LEFT JOIN broadcast_games bg
    ON  bg.broadcast_id  = bs.broadcast_id
    AND bg.division      = bs.division
    AND bg.round         = bs.round
    AND bg.table_number  = bs.table_number
WHERE b.slug = $1 AND bs.slot_name = $2;

-- name: GetSlotsByGame :many
-- Returns all slots currently pointing at a given game's (div, round, table).
SELECT bs.slot_name, b.slug, b.uuid AS broadcast_uuid
FROM broadcast_slots bs
JOIN broadcasts b ON bs.broadcast_id = b.id
JOIN broadcast_games bg
    ON  bg.broadcast_id  = bs.broadcast_id
    AND bg.division      = bs.division
    AND bg.round         = bs.round
    AND bg.table_number  = bs.table_number
WHERE bg.game_uuid = $1;

-- ---------------------------------------------------------------------------
-- Stream slots (user_obs_slots): user-owned pointers at a broadcast slot.
-- Every column of the target chain is LEFT JOINed and COALESCEd, because a
-- stream slot is legitimately unpointed, or pointed at a broadcast slot that
-- has since been renamed/deleted. Those resolve to zero values, which the OBS
-- handler renders as the "(no game assigned)" placeholder rather than a 404.
-- ---------------------------------------------------------------------------

-- name: GetUserSlotGame :one
-- Resolves (username, slot_name) → target broadcast + current game, in one trip.
-- Only meaningful for target_kind = 'broadcast_slot'; the caller branches on
-- target_kind and uses GetLatestAnnotationForUser for the other kind.
SELECT
    u.uuid::text                                          AS user_uuid,
    us.target_kind,
    COALESCE(b.uuid::text, '')::text                      AS broadcast_uuid,
    COALESCE(b.slug, '')                                  AS broadcast_slug,
    COALESCE(b.name, '')                                  AS broadcast_name,
    COALESCE(b.broadcast_url, '')                         AS broadcast_url,
    COALESCE(b.broadcast_url_format, '')                  AS broadcast_url_format,
    COALESCE(us.target_slot_name, '')                     AS target_slot_name,
    COALESCE(bs.division, '')                             AS division,
    COALESCE(bs.round, 0)::int                            AS round,
    COALESCE(bs.table_number, 0)::int                     AS table_number,
    COALESCE(bg.game_uuid, '')                            AS game_uuid
FROM user_obs_slots us
JOIN users u ON u.id = us.user_id
LEFT JOIN broadcasts b ON b.id = us.target_broadcast_id
LEFT JOIN broadcast_slots bs
    ON  bs.broadcast_id = us.target_broadcast_id
    AND bs.slot_name    = us.target_slot_name
LEFT JOIN broadcast_games bg
    ON  bg.broadcast_id  = bs.broadcast_id
    AND bg.division      = bs.division
    AND bg.round         = bs.round
    AND bg.table_number  = bs.table_number
WHERE lower(u.username) = lower(@username) AND us.slot_name = @slot_name;

-- name: ListUserObsSlots :many
-- Same resolution as GetUserSlotGame, for every slot the user owns, plus player
-- names so the management UI can show what each slot is currently pointed at.
SELECT
    us.slot_name,
    us.target_kind,
    COALESCE(b.slug, '')                                  AS broadcast_slug,
    COALESCE(b.name, '')                                  AS broadcast_name,
    COALESCE(us.target_slot_name, '')                     AS target_slot_name,
    COALESCE(bs.division, '')                             AS division,
    COALESCE(bs.round, 0)::int                            AS round,
    COALESCE(bs.table_number, 0)::int                     AS table_number,
    COALESCE(bg.game_uuid, '')                            AS game_uuid,
    COALESCE(gd.document->'players'->0->>'realName', '')  AS player1_name,
    COALESCE(gd.document->'players'->1->>'realName', '')  AS player2_name
FROM user_obs_slots us
LEFT JOIN broadcasts b ON b.id = us.target_broadcast_id
LEFT JOIN broadcast_slots bs
    ON  bs.broadcast_id = us.target_broadcast_id
    AND bs.slot_name    = us.target_slot_name
LEFT JOIN broadcast_games bg
    ON  bg.broadcast_id  = bs.broadcast_id
    AND bg.division      = bs.division
    AND bg.round         = bs.round
    AND bg.table_number  = bs.table_number
LEFT JOIN game_documents gd ON gd.game_id = bg.game_uuid
WHERE us.user_id = $1
ORDER BY us.slot_name;

-- name: GetLatestAnnotationForUser :one
-- Resolution for target_kind = 'latest_annotation': the user's most recently
-- edited annotated game, plus its broadcast context when it happens to belong
-- to one. That LEFT JOIN is what lets the standings fields work for someone
-- annotating a broadcast game without owning a stream slot pointed at it —
-- when the game is a plain annotated game the columns come back empty and the
-- standings fields fall back to the placeholder.
SELECT
    COALESCE(agm.game_uuid, '')                           AS game_uuid,
    COALESCE(b.uuid::text, '')::text                      AS broadcast_uuid,
    COALESCE(b.slug, '')                                  AS broadcast_slug,
    COALESCE(b.name, '')                                  AS broadcast_name,
    COALESCE(b.broadcast_url, '')                         AS broadcast_url,
    COALESCE(b.broadcast_url_format, '')                  AS broadcast_url_format,
    COALESCE(bg.division, '')                             AS division,
    COALESCE(bg.round, 0)::int                            AS round,
    COALESCE(bg.table_number, 0)::int                     AS table_number
FROM annotated_game_metadata agm
JOIN games g ON g.uuid = agm.game_uuid
LEFT JOIN broadcast_games bg ON bg.game_uuid = agm.game_uuid
LEFT JOIN broadcasts b ON b.id = bg.broadcast_id
WHERE agm.creator_uuid = (SELECT uuid FROM users WHERE lower(username) = lower(@username))
ORDER BY g.updated_at DESC
LIMIT 1;

-- name: CreateUserObsSlot :execrows
-- Returns 0 rows when the user already has a slot by that name.
INSERT INTO user_obs_slots (user_id, slot_name)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: UpdateUserObsSlotTarget :execrows
-- Returns 0 rows when the slot doesn't exist. Callers pass a NULL broadcast and
-- an empty slot name for the 'none' and 'latest_annotation' kinds.
UPDATE user_obs_slots
SET target_kind         = @target_kind,
    target_broadcast_id = sqlc.narg('target_broadcast_id'),
    target_slot_name    = @target_slot_name,
    updated_at          = NOW()
WHERE user_id = @user_id AND slot_name = @slot_name;

-- name: DeleteUserObsSlot :execrows
DELETE FROM user_obs_slots
WHERE user_id = $1 AND slot_name = $2;

-- name: CountUserObsSlots :one
SELECT COUNT(*) FROM user_obs_slots WHERE user_id = $1;
