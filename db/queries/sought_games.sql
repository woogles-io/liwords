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
