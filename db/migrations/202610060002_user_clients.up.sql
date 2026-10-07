BEGIN;

-- Where an account was created from. Written once at registration.
ALTER TABLE users
    ADD COLUMN registration_ip inet,
    ADD COLUMN registration_client_id text;

CREATE INDEX users_registration_ip_idx ON users (registration_ip);
CREATE INDEX users_registration_client_id_idx ON users (registration_client_id);

-- One row per distinct (user, ip, client) combination an account has been
-- seen from. last_seen is refreshed at most hourly; rows not seen for a while
-- are pruned periodically.
CREATE TABLE user_clients (
    user_id integer NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    ip inet NOT NULL,
    client_id text NOT NULL DEFAULT '',
    first_seen timestamptz NOT NULL DEFAULT now(),
    last_seen timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, ip, client_id)
);

CREATE INDEX user_clients_ip_idx ON user_clients (ip);
CREATE INDEX user_clients_client_id_idx ON user_clients (client_id) WHERE client_id <> '';
CREATE INDEX user_clients_last_seen_idx ON user_clients (last_seen);

COMMIT;
