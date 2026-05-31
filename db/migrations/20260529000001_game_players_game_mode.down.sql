DROP INDEX IF EXISTS idx_game_players_player_correspondence;
ALTER TABLE game_players DROP COLUMN IF EXISTS game_mode;
