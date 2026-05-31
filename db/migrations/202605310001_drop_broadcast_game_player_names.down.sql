ALTER TABLE broadcast_games
    ADD COLUMN player1_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN player2_name TEXT NOT NULL DEFAULT '';
