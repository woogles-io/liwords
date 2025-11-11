BEGIN;

ALTER TABLE games ADD COLUMN history_in_s3 boolean NOT NULL DEFAULT false;

COMMIT;
