DROP INDEX idx_tournaments_scheduled_start_time;
DROP INDEX idx_tournaments_scheduled_end_time;

ALTER TABLE tournaments
DROP COLUMN scheduled_start_time,
DROP COLUMN scheduled_end_time;
