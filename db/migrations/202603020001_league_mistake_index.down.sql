ALTER TABLE league_standings
    DROP COLUMN IF EXISTS total_mistake_index,
    DROP COLUMN IF EXISTS games_analyzed;
