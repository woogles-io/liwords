-- Add game_mode column to soughtgames table to differentiate correspondence from real-time seeks
ALTER TABLE soughtgames ADD COLUMN game_mode INTEGER;

-- Create index for efficient querying by game_mode
CREATE INDEX idx_soughtgames_game_mode ON soughtgames(game_mode);
