-- Add extended stats columns to league_standings for richer leaderboard data
ALTER TABLE league_standings
    ADD COLUMN total_score INT DEFAULT 0,
    ADD COLUMN total_opponent_score INT DEFAULT 0,
    ADD COLUMN total_bingos INT DEFAULT 0,
    ADD COLUMN total_opponent_bingos INT DEFAULT 0,
    ADD COLUMN total_turns INT DEFAULT 0,
    ADD COLUMN high_turn INT DEFAULT 0,
    ADD COLUMN high_game INT DEFAULT 0,
    ADD COLUMN timeouts INT DEFAULT 0,
    ADD COLUMN blanks_played INT DEFAULT 0,
    ADD COLUMN total_tiles_played INT DEFAULT 0,
    ADD COLUMN total_opponent_tiles_played INT DEFAULT 0;
