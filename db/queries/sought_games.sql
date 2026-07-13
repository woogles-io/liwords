-- name: GetSoughtGameByUUID :one
SELECT request FROM soughtgames WHERE uuid = @uuid;

-- name: GetSoughtGameBySeekerConnID :one
SELECT request FROM soughtgames WHERE seeker_conn_id = @seeker_conn_id;

-- name: GetSoughtGameByReceiverConnID :one
SELECT request FROM soughtgames WHERE receiver_conn_id = @receiver_conn_id;

-- name: GetSoughtGameBySeekerID :one
SELECT request FROM soughtgames WHERE seeker = @seeker_id;

-- name: GetSoughtGameByReceiverID :one
SELECT request FROM soughtgames WHERE receiver = @receiver_id;

-- name: DeleteSoughtGameByUUID :exec
DELETE FROM soughtgames WHERE uuid = @uuid;

-- name: DeleteSoughtGameBySeekerID :exec
DELETE FROM soughtgames WHERE seeker = @seeker_id;

-- name: DeleteSoughtGameBySeekerConnID :exec
DELETE FROM soughtgames WHERE seeker_conn_id = @seeker_conn_id;

-- name: InsertSoughtGame :exec
INSERT INTO soughtgames (uuid, seeker, seeker_conn_id, receiver, receiver_conn_id, request, game_mode)
VALUES (@uuid, @seeker, @seeker_conn_id, @receiver, @receiver_conn_id, @request, @game_mode);

-- name: ExpireOldRealtimeSeeks :execrows
DELETE FROM soughtgames WHERE (game_mode IS NULL OR game_mode = 0) AND created_at < NOW() - INTERVAL '2 hours';

-- name: ExpireOldCorrespondenceSeeks :execrows
DELETE FROM soughtgames WHERE game_mode = 1 AND created_at < NOW() - INTERVAL '60 hours';

-- name: UpdateSoughtGameReceiverAbsentByReceiverID :execrows
UPDATE soughtgames SET request = jsonb_set(request, array['receiver_state'], @receiver_state) WHERE receiver = @receiver_id;

-- name: UpdateSoughtGameReceiverAbsentByReceiverConnID :execrows
UPDATE soughtgames SET request = jsonb_set(request, array['receiver_state'], @receiver_state) WHERE receiver_conn_id = @receiver_conn_id;

-- name: ListOpenSeeksByTourney :many
SELECT request FROM soughtgames WHERE receiver = @receiver_id AND request->>'tournament_id' = @tournament_id::text;

-- name: ListOpenSeeksAll :many
SELECT request FROM soughtgames WHERE receiver = @receiver_id OR receiver = '';

-- name: ListCorrespondenceSeeksForUser :many
SELECT request FROM soughtgames WHERE game_mode = 1 AND (seeker = @user_id OR receiver = @user_id OR receiver = '');

-- name: ExistsSeekForUser :one
SELECT EXISTS(SELECT 1 FROM soughtgames WHERE seeker = @seeker_id);

-- name: CountSeeksForUser :one
SELECT COUNT(*) FROM soughtgames WHERE seeker = @seeker_id;

-- name: CountSeekConflictsForCorrespondence :one
SELECT
    COUNT(*) FILTER (WHERE receiver = '') AS has_open_seek,
    COUNT(*) FILTER (WHERE game_mode IS NULL OR game_mode != 1) AS has_realtime_seek
FROM soughtgames
WHERE seeker = @seeker_id;

-- name: SeekExistsFromMatcher :one
SELECT EXISTS(SELECT 1 FROM soughtgames WHERE receiver = @user_id AND seeker = @matcher);
