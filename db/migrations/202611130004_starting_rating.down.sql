BEGIN;

ALTER TABLE game_players
    DROP COLUMN IF EXISTS starting_rating,
    DROP COLUMN IF EXISTS starting_rating_provisional;

ALTER TABLE ongoing_games
    DROP COLUMN IF EXISTS player0_starting_rating,
    DROP COLUMN IF EXISTS player0_starting_rating_provisional,
    DROP COLUMN IF EXISTS player1_starting_rating,
    DROP COLUMN IF EXISTS player1_starting_rating_provisional;

COMMIT;
