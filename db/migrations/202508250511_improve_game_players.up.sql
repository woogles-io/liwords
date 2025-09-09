BEGIN;

-- Drop the existing empty game_players table
DROP TABLE IF EXISTS game_players;

-- Create improved game_players table
CREATE TABLE game_players (
    game_uuid text NOT NULL,
    player_id integer NOT NULL,
    player_index SMALLINT NOT NULL CHECK (player_index IN (0, 1)),
    
    -- Game outcome data
    score integer NOT NULL,
    won boolean, -- true = won, false = lost, null = tie
    game_end_reason SMALLINT NOT NULL,
    
    -- Rating data (nullable for unrated games)
    rating_before integer,
    rating_after integer,
    rating_delta integer, -- convenience field: rating_after - rating_before
    
    -- Temporal and type data
    created_at timestamp with time zone NOT NULL,
    game_type SMALLINT NOT NULL,
    
    -- Opponent info (denormalized for convenience)
    opponent_id integer NOT NULL,
    opponent_score integer NOT NULL,
    
    FOREIGN KEY (player_id) REFERENCES users (id),
    FOREIGN KEY (opponent_id) REFERENCES users (id),
    PRIMARY KEY (game_uuid, player_id)
);

-- Essential indexes for common queries
CREATE INDEX idx_game_players_player_created ON game_players(player_id, created_at DESC);
CREATE INDEX idx_game_players_opponents ON game_players(player_id, opponent_id, created_at DESC);
CREATE INDEX idx_game_players_rating_change ON game_players(player_id, rating_delta) WHERE rating_delta IS NOT NULL;

-- Add migration status to games table for tracking
ALTER TABLE games ADD COLUMN IF NOT EXISTS migration_status SMALLINT DEFAULT 0;
-- 0 = not migrated, 1 = migrated to past_games, 2 = archived to S3

COMMIT;