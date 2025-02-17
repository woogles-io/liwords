BEGIN;

ALTER TABLE tournaments
DROP COLUMN scheduled_start_time,
DROP COLUMN scheduled_end_time;

COMMIT;
