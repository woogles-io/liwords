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

    -- Temporal and type data
    created_at timestamp with time zone NOT NULL,
    game_type SMALLINT NOT NULL,

    -- Opponent info (denormalized for convenience)
    opponent_id integer NOT NULL,
    opponent_score integer NOT NULL,

    -- For rematch tracking
    original_request_id text,

    FOREIGN KEY (player_id) REFERENCES users (id),
    FOREIGN KEY (opponent_id) REFERENCES users (id),
    PRIMARY KEY (game_uuid, player_id)
);

-- Essential indexes for common queries
CREATE INDEX idx_game_players_player_created ON game_players(player_id, created_at DESC);
CREATE INDEX idx_game_players_opponents ON game_players(player_id, opponent_id, created_at DESC);
CREATE INDEX idx_game_players_orig_req ON game_players(original_request_id);

-- Drop the old rematch indexes since we'll query game_players instead of past_games/games
-- XXX: Drop these later, once we've completed the full migration.
-- DROP INDEX IF EXISTS idx_past_games_rematch_req_idx;
-- DROP INDEX IF EXISTS rematch_req_idx;

COMMIT;