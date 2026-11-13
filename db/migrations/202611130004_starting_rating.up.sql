BEGIN;

-- Fail rather than queue: an ALTER waiting on a lock blocks every query on the
-- table behind it, and the daily pg_dump holds one on every table for hours.
SET LOCAL lock_timeout = '10s';

-- Each player's rating in the game's variant when the game was created, and
-- whether it was provisional (shown with a "?"). Set when the game is created
-- and never changed. NULL for a player with no rating, such as an anonymous
-- one, and for games created before these columns.
ALTER TABLE ongoing_games
    ADD COLUMN player0_starting_rating real,
    ADD COLUMN player0_starting_rating_provisional boolean,
    ADD COLUMN player1_starting_rating real,
    ADD COLUMN player1_starting_rating_provisional boolean;

-- The same, copied from the game when it ends. Not backfilled: games finished
-- before this column existed have NULL.
ALTER TABLE game_players
    ADD COLUMN starting_rating real,
    ADD COLUMN starting_rating_provisional boolean;

COMMIT;
