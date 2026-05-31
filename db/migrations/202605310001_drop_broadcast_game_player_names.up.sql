-- Drop player1_name and player2_name from broadcast_games.
-- These columns have been obsolete since 20260529: the application no longer
-- writes to or reads from them (player names are sourced from game_documents).
ALTER TABLE broadcast_games
    DROP COLUMN player1_name,
    DROP COLUMN player2_name;
