BEGIN;

ALTER TABLE league_seasons DROP COLUMN IF EXISTS closed_at;
ALTER TABLE league_seasons DROP COLUMN IF EXISTS divisions_prepared_at;
ALTER TABLE league_seasons DROP COLUMN IF EXISTS started_at;
ALTER TABLE league_seasons DROP COLUMN IF EXISTS registration_opened_at;
ALTER TABLE league_seasons DROP COLUMN IF EXISTS starting_soon_notification_sent_at;

COMMIT;
