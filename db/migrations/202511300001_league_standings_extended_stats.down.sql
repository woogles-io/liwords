-- Remove extended stats columns from league_standings
ALTER TABLE league_standings
    DROP COLUMN IF EXISTS total_score,
    DROP COLUMN IF EXISTS total_opponent_score,
    DROP COLUMN IF EXISTS total_bingos,
    DROP COLUMN IF EXISTS total_opponent_bingos,
    DROP COLUMN IF EXISTS total_turns,
    DROP COLUMN IF EXISTS high_turn,
    DROP COLUMN IF EXISTS high_game,
    DROP COLUMN IF EXISTS timeouts,
    DROP COLUMN IF EXISTS blanks_played;
