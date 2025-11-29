BEGIN;

-- Add task tracking columns to league_seasons for hourly runner idempotency
-- These track when specific tasks have been run for this season
ALTER TABLE league_seasons ADD COLUMN closed_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE league_seasons ADD COLUMN divisions_prepared_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE league_seasons ADD COLUMN started_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE league_seasons ADD COLUMN registration_opened_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE league_seasons ADD COLUMN starting_soon_notification_sent_at TIMESTAMP WITH TIME ZONE;

COMMIT;
