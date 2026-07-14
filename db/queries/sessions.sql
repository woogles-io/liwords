-- name: GetSession :one
SELECT expires_at, data FROM db_sessions WHERE uuid = @uuid;

-- name: CreateSession :exec
INSERT INTO db_sessions (uuid, expires_at, data) VALUES (@uuid, @expires_at, @data);

-- name: DeleteSession :exec
DELETE FROM db_sessions WHERE uuid = @uuid;

-- name: ExtendSessionExpiry :execrows
UPDATE db_sessions SET expires_at = @expires_at WHERE uuid = @uuid;

-- name: SetSessionCSRFToken :execrows
UPDATE db_sessions
   SET data = jsonb_set(data, '{csrf_token}', to_jsonb(@csrf_token::text))
 WHERE uuid = @uuid;
