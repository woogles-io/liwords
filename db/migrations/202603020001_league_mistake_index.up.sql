ALTER TABLE league_standings
    ADD COLUMN total_mistake_index DOUBLE PRECISION DEFAULT 0,
    ADD COLUMN games_analyzed INT DEFAULT 0;
