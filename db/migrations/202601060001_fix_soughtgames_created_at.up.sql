BEGIN;

-- Fix soughtgames.created_at to have a default and backfill NULL values
-- This ensures seeks can expire properly based on their creation time

-- First, backfill NULL created_at values with current timestamp
-- (These are old seeks that should have expired already, but we'll give them a timestamp)
UPDATE soughtgames SET created_at = NOW() WHERE created_at IS NULL;

-- Add DEFAULT NOW() for future inserts
ALTER TABLE soughtgames ALTER COLUMN created_at SET DEFAULT NOW();

-- Make the column NOT NULL to prevent future NULL values
ALTER TABLE soughtgames ALTER COLUMN created_at SET NOT NULL;

COMMIT;
