-- name: GetLoginFailures :many
-- The failure counts of the given keys whose windows haven't ended.
SELECT key, failures FROM login_failures
WHERE key = ANY(@keys::text[]) AND expires_at > now();

-- name: AddLoginFailure :exec
-- Counts a failure. A window that has ended starts again at one.
INSERT INTO login_failures (key, failures, expires_at)
VALUES (@key, 1, now() + make_interval(secs => @window_secs::float8))
ON CONFLICT (key) DO UPDATE SET
    failures = CASE WHEN login_failures.expires_at <= now() THEN 1
                    ELSE login_failures.failures + 1 END,
    expires_at = CASE WHEN login_failures.expires_at <= now() THEN EXCLUDED.expires_at
                      ELSE login_failures.expires_at END;

-- name: ClearLoginFailures :exec
DELETE FROM login_failures WHERE key = @key;

-- name: IsKnownLoginClient :one
SELECT EXISTS (
    SELECT 1 FROM login_known_clients
    WHERE username = @username AND client_id = @client_id AND expires_at > now()
);

-- name: AddKnownLoginClient :exec
INSERT INTO login_known_clients (username, client_id, expires_at)
VALUES (@username, @client_id, @expires_at)
ON CONFLICT (username, client_id) DO UPDATE SET
    expires_at = GREATEST(login_known_clients.expires_at, EXCLUDED.expires_at);

-- name: CountKnownLoginClients :one
SELECT count(*) FROM login_known_clients;

-- name: DeleteExpiredLoginFailures :execrows
DELETE FROM login_failures WHERE expires_at <= now();

-- name: DeleteExpiredKnownLoginClients :execrows
DELETE FROM login_known_clients WHERE expires_at <= now();
