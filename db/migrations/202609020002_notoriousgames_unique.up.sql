BEGIN;

-- One notoriety verdict per player per game.
--
-- automod files a row here when it judges a player badly, and adds to their
-- notoriety score at the same time. Neither is idempotent, and until
-- automod_verdicts (202609020001) nothing stopped it happening twice for one
-- game: the guards against re-judging a finished game all read game_end_reason
-- or the play state, and automod ran *before* the games row was written, so for
-- the length of that window there was nothing for them to read.
--
-- It happened. Six rows in five years of production, four of them in a single
-- incident on 2026-08-07 where the window stayed open for 32 seconds while a
-- 10-second adjudicator ticked, and four players were penalised twice for the
-- same game. Those six rows were deleted before this index could be created,
-- keeping the earliest of each pair.
--
-- WHAT THIS ENCODES: a player gets at most one verdict per game. That is what
-- the code does -- Classify assigns a single verdict, and the phony and sandbag
-- checks only run when it is still GOOD, so a player who both sat on the clock
-- and played phonies is filed as SITTING alone -- and it is what the data has
-- always shown: across 284,179 rows, zero (player, game) pairs carry two
-- different reasons.
--
-- If a game should ever be notorious for several reasons at once, this becomes
-- (player_id, game_id, type) -- and automod_verdicts.PRIMARY KEY has to widen in
-- the same change, or automod will still decline to file the second reason.
--
-- One game may name BOTH players; player_id is part of the key. 3,850 games do.
--
-- IF NOT EXISTS because production already has this index, created by hand with
-- CREATE UNIQUE INDEX CONCURRENTLY after the duplicates were cleared -- 284k
-- rows is too many to lock out writers for. Everywhere else the table is small
-- enough that building it inside this transaction costs nothing. The name here
-- matches the one production has, so this is a no-op there rather than a second
-- index doing the same job.
CREATE UNIQUE INDEX IF NOT EXISTS idx_notoriousgames_player_game
    ON notoriousgames (player_id, game_id);

COMMIT;
