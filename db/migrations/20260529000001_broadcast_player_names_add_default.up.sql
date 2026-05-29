-- player1_name and player2_name are now obsolete: the application no longer
-- writes to or reads from them (player names are sourced from game_documents).
-- Add DEFAULT '' so existing NOT NULL constraint does not reject the new INSERT
-- that omits these columns. They can be dropped in a future migration.
ALTER TABLE broadcast_games
    ALTER COLUMN player1_name SET DEFAULT '',
    ALTER COLUMN player2_name SET DEFAULT '';
