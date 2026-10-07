BEGIN;

-- Sessions store the user in a JSON blob; index it so all of a user's
-- sessions can be found.
CREATE INDEX db_sessions_user_uuid_idx ON db_sessions ((data ->> 'uuid'));

COMMIT;
