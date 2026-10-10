BEGIN;

-- automod_verdicts: one row per (player, game) that automod has judged.
--
-- Automod's effects are not idempotent. It adds to a player's notoriety score
-- when they misbehaved and subtracts from it when they did not, and it appends
-- to notoriousgames -- a table with four plain indexes and no unique
-- constraint, so the same game can be filed against the same player twice.
-- Running it twice for one game therefore penalises a player twice, or forgives
-- them twice.
--
-- Nothing in the database prevented that. What prevented it was the callers of
-- performEndgameDuties checking persisted state before they got that far:
-- AdjudicateGame, ForfeitGame and AbortGame check game_end_reason, TimedOut
-- checks the play state, and the move path rejects moves on finished games.
-- performEndgameDuties itself has no such guard. So the protection was a
-- property of who called it rather than of the operation, and any new path
-- reaching it twice would double-count in silence.
--
-- This table makes it a property of the operation. The verdict is recorded
-- first, with ON CONFLICT DO NOTHING; if the row already existed, the score is
-- left alone and nothing is filed. That holds across restarts, across app
-- servers, and across repeated calls.
--
-- GOOD verdicts are recorded too, and have to be: a good game *decrements*
-- notoriety, so guarding only the bad ones would leave the forgiving path
-- unprotected.
CREATE TABLE automod_verdicts (
    player_id  text        NOT NULL,
    game_id    text        NOT NULL,
    -- mod_service.NotoriousGameType, GOOD included.
    verdict    integer     NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (player_id, game_id)
);

-- For "what has automod done to this game", which is the question asked when
-- a player disputes a suspension.
CREATE INDEX idx_automod_verdicts_game ON automod_verdicts (game_id);

-- NOTE: notoriousgames should also have a unique index on (player_id, game_id),
-- which would stop the same game appearing twice in a player's report. It is
-- not added here because it would fail if production already contains
-- duplicates -- which is exactly what the missing guard would have produced.
-- Check for them first, then add it separately.

COMMIT;
