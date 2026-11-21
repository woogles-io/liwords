-- Login rate limiting (pkg/auth/login_limit.go), moved from Redis.

-- Failed logins counted per account, per address, and per both, each for a
-- window that starts at its first failure.
CREATE TABLE login_failures (
    key text PRIMARY KEY,
    failures integer NOT NULL,
    expires_at timestamptz NOT NULL
);

-- Browsers (client IDs) that have logged into an account, which failures
-- from elsewhere don't hold back. username is lowercased.
CREATE TABLE login_known_clients (
    username text NOT NULL,
    client_id text NOT NULL,
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (username, client_id)
);
