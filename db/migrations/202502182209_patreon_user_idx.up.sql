BEGIN;

CREATE UNIQUE INDEX unique_patreon_user_id
ON integrations ((data->>'patreon_user_id'));

COMMIT;