BEGIN;

ALTER TABLE tournaments
ALTER COLUMN scheduled_start_time DROP NOT NULL,
ALTER COLUMN scheduled_start_time SET DEFAULT NULL,
ALTER COLUMN scheduled_end_time DROP NOT NULL,
ALTER COLUMN scheduled_end_time SET DEFAULT NULL;

-- Set all scheduled_start_time and scheduled_end_time to NULL if they are the default value
UPDATE tournaments
SET scheduled_start_time = NULL WHERE scheduled_start_time = '1970-01-01 00:00:00+00';
UPDATE tournaments
SET scheduled_end_time = NULL WHERE scheduled_end_time = '1970-01-01 00:00:00+00';

COMMIT;