BEGIN;

ALTER TABLE profiles DROP COLUMN title_organization;
ALTER TABLE profiles ALTER COLUMN title TYPE character varying(16);

COMMIT;
