BEGIN;

ALTER TABLE profiles ALTER COLUMN title TYPE character varying(32);
ALTER TABLE profiles ADD COLUMN title_organization character varying(16);

COMMIT;
