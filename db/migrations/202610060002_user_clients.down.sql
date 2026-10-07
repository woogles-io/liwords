BEGIN;

DROP TABLE IF EXISTS user_clients;
DROP INDEX IF EXISTS users_registration_client_id_idx;
DROP INDEX IF EXISTS users_registration_ip_idx;
ALTER TABLE users
    DROP COLUMN IF EXISTS registration_client_id,
    DROP COLUMN IF EXISTS registration_ip;

COMMIT;
