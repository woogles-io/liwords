-- Remove game_request and tournament_data from past_games table
-- These are moved to game_metadata for better query performance
ALTER TABLE past_games DROP COLUMN IF EXISTS game_request;
ALTER TABLE past_games DROP COLUMN IF EXISTS tournament_data;