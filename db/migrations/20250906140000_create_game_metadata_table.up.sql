-- Create game_metadata table to store essential game information for completed games
-- This table stays unpartitioned for fast queries and is never archived
CREATE TABLE IF NOT EXISTS game_metadata (
    game_uuid TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    
    -- Full GameRequest as JSONB (protojson format)
    -- Contains: lexicon, rules, time settings, etc.
    game_request JSONB NOT NULL,
    
    -- Tournament info (moved from past_games for fast access)
    tournament_data JSONB DEFAULT NULL,
    
    -- Creation timestamp for ordering
    created_at_idx TIMESTAMPTZ NOT NULL DEFAULT created_at
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_game_metadata_created_at ON game_metadata (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_game_metadata_tournament ON game_metadata 
    USING GIN ((tournament_data->'Id')) WHERE tournament_data IS NOT NULL;